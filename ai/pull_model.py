#!/usr/bin/env python3
"""Freeze and import a GPU checkpoint; optionally deploy it to local kind."""
import argparse
import hashlib
import inspect
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def select_checkpoint(run, choice='auto'):
    """Auto uses an evaluated incumbent for league runs, never silently latest."""
    champion = run/'champion.json'
    if choice != 'latest' and champion.exists():
        record = json.loads(champion.read_text())
        source = Path(record['checkpoint'])
        if record.get('status') != 'evaluated' or not source.resolve().is_relative_to((run/'champions').resolve()):
            raise ValueError('invalid evaluated champion record')
        with source.open('rb') as stream:
            actual = hashlib.file_digest(stream, 'sha256').hexdigest()
        if actual != record['sha256']:
            raise ValueError('champion checkpoint checksum mismatch')
        return source, 'champion', record
    if choice == 'champion' or (choice == 'auto' and (run/'league.json').exists()):
        raise ValueError('No evaluated champion is ready yet. Wait for the benchmark, or use --checkpoint latest to explicitly export experimental weights.')
    return run/'latest.pt', 'latest', None


REMOTE_EXPORT = r'''
import hashlib,json,shutil,sys,time
from pathlib import Path
import torch
''' + inspect.getsource(select_checkpoint) + r'''
args=json.loads(sys.argv[1])
root=Path(args['root']).expanduser()
run=Path(args['run']).expanduser() if args['run'] else root/'ai/runs/active'
if not args['run'] and not run.exists():run=root/'ai/runs/surface-v2-20261015'
run=run.resolve()
if args['release']:
    release=Path(args['release']).expanduser()
    manifest=json.loads((release/'manifest.json').read_text())
    if manifest['name']!=args['name']:raise ValueError('release name differs from --name')
else:
    selected,selection,benchmark=select_checkpoint(run,args.get('checkpoint','auto'))
    release=run/'previews'/('export-'+str(time.time_ns()));release.mkdir(parents=True)
    # latest.pt is atomically replaced by training. Opening it once yields one
    # complete checkpoint even while the next update is being saved.
    shutil.copyfile(selected,release/'checkpoint.pt')
    checkpoint=release/'checkpoint.pt'
    data=torch.load(checkpoint,map_location='cpu',weights_only=True)
    source=Path(args['source']).expanduser() if args['source'] else Path(data['config']['engine']).resolve().parents[1]
    sys.path.insert(0,str(source))
    from smallworld_ai.model import load_checkpoint,export_policy
    policy,data=load_checkpoint(checkpoint)
    digest=hashlib.sha256(checkpoint.read_bytes()).hexdigest()
    if benchmark and digest!=benchmark['sha256']:raise ValueError('champion changed while exporting')
    model_id=args['slug']+'-s'+str(data['step'])+'-'+digest[:8]
    model=release/(model_id+'.json')
    export_policy(policy,model,model_id,args['name'],f'PPO policy at {data["step"]:,} training decisions; {policy.schema}.')
    manifest={'id':model_id,'name':args['name'],'step':data['step'],'iteration':data['iteration'],'schema':policy.schema,'release':str(release),'checkpoint_sha256':digest,'model_sha256':hashlib.sha256(model.read_bytes()).hexdigest(),'model_bytes':model.stat().st_size,'parameters':sum(p.numel() for p in policy.parameters()),'checkpoint_selection':selection,'benchmark_report':benchmark.get('report') if benchmark else None}
    (release/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps(manifest))
'''


def run(command, **kwargs):
    return subprocess.run([str(v) for v in command], check=True, **kwargs)


def ssh(host, command, ssh_command=None):
    if ssh_command is not None:
        return shlex.split(ssh_command) + [command]
    return ['ssh', '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=yes',
            '-o', 'ConnectTimeout=10', '--', host, command]


def reject_nonfinite(value):
    raise ValueError('non-finite JSON constant: '+value)


def validate_model(model, manifest, name):
    if hashlib.sha256(model).hexdigest() != manifest['model_sha256']:
        raise ValueError('downloaded model checksum mismatch')
    data = json.loads(model, parse_constant=reject_nonfinite)
    if data.get('id') != manifest['id'] or data.get('name') != name:
        raise ValueError('model identity does not match the release')
    if not re.fullmatch(r'[a-z0-9][a-z0-9_-]{0,63}', data['id']):
        raise ValueError('invalid model ID')
    if data.get('schema') not in ('smallworld-bot-v1', 'smallworld-bot-v2'):
        raise ValueError('unsupported model schema')
    if len(model) > 384 << 20:
        raise ValueError('model exceeds the server limit')
    return data


def validate_with_engine(directory):
    # Validate dimensions, finite weights and every existing model using the
    # actual Go reader before importing or replacing any deployment image.
    with tempfile.TemporaryDirectory(prefix='smallworld-check-') as tmp:
        binary = Path(tmp)/'botenv'
        run(['go', 'build', '-buildvcs=false', '-mod=mod', '-o', binary, './cmd/botenv'], cwd=ROOT/'backend2')
        command = [binary, '-model-dir', directory]
        needs_python = any(json.loads(path.read_bytes()).get('format_version') == 7 for path in Path(directory).glob('*.json'))
        if needs_python:
            from compare_bots import resolve_evaluator
            command += ['-inference-python', resolve_evaluator('pytorch')['python']]
        result = run(command, input='{"op":"spec"}\n', env=os.environ | {'PYTHONPATH':str(ROOT/'ai')},
                     text=True, capture_output=True, timeout=90 if needs_python else 30)
        if json.loads(result.stdout).get('error'):
            raise ValueError(result.stdout)


def deploy_kind(args, model_dir, model_id):
    kubectl = ['kubectl', '--context', args.context, '-n', args.namespace]
    obj = json.loads(run(kubectl+['get', 'statefulset', 'room-worker', '-o', 'json'], capture_output=True, text=True).stdout)
    containers = obj['spec']['template']['spec']['containers']
    index = next(i for i, c in enumerate(containers) if c['name'] == 'room-worker')
    previous = containers[index]['image']
    formats = {json.loads(p.read_text()).get('format_version', 1) for p in model_dir.glob('*.json')}
    if max(formats, default=1) >= 3:
        label = run(['docker', 'image', 'inspect', previous, '--format',
                     '{{ index .Config.Labels "io.smallworld.bot-formats" }}'], capture_output=True, text=True).stdout.strip()
        supported = {int(v) for v in label.split(',') if v.isdigit()}
        if not formats.issubset(supported):
            raise ValueError('The deployed worker does not advertise support for this model format. Rebuild/deploy the current worker code first; the frozen export is safe to retry.')
    # A unique tag forces a rollout even when republishing the same model.
    image = 'smallworld-roomworker:pull-'+model_id+'-'+str(time.time_ns())
    with tempfile.TemporaryDirectory(prefix='smallworld-model-image-') as tmp:
        tmp = Path(tmp)
        # Inherit the CURRENT deployment, retaining its executable, assets and
        # old models even if the local checkout has only the newly pulled model.
        shutil.copytree(model_dir, tmp/'models', ignore=shutil.ignore_patterns('.gitkeep'))
        (tmp/'Dockerfile').write_text('ARG BASE\nFROM ${BASE}\nCOPY --chown=game:game models/ /app/bot-models/\nENV BOT_MODEL_DIR=/app/bot-models\n')
        run(['docker', 'build', '--build-arg', 'BASE='+previous, '-t', image, tmp])
    run(['kind', 'load', 'docker-image', image, '--name', args.cluster])
    path = '/spec/template/spec/containers/'+str(index)+'/image'
    # Abort if another deployment changed the worker while this image built.
    patch = [{'op':'test','path':path,'value':previous}, {'op':'replace','path':path,'value':image}]
    run(kubectl+['patch','statefulset','room-worker','--type=json','-p',json.dumps(patch)])
    run(kubectl+['rollout','status','statefulset/room-worker','--timeout=180s'])
    pods = json.loads(run(kubectl+['get','pods','-l','app=room-worker','-o','json'],capture_output=True,text=True).stdout)['items']
    expected = hashlib.sha256((model_dir/(model_id+'.json')).read_bytes()).hexdigest()
    catalog = None
    for pod in pods:
        pod_name = pod['metadata']['name']
        actual = run(kubectl+['exec',pod_name,'--','sha256sum','/app/bot-models/'+model_id+'.json'],capture_output=True,text=True).stdout.split()[0]
        if actual != expected: raise ValueError('deployed weights do not match on '+pod_name)
        # Read all installed models, including ones inherited from the old image.
        result = run(kubectl+['exec',pod_name,'--','sh','-c','for f in /app/bot-models/*.json; do cat "$f"; printf "\\n"; done'],capture_output=True,text=True)
        text, decoder, found = result.stdout, json.JSONDecoder(), []
        while text.strip():
            value, end = decoder.raw_decode(text.lstrip());text = text.lstrip()[end:]
            found.append({k:value[k] for k in ('id','name','description')})
        found.sort(key=lambda m:m['id'])
        if catalog is not None and found != catalog: raise ValueError('worker model catalogs differ')
        catalog = found
    if not pods or catalog is None: raise ValueError('no ready worker pods found')
    builtins = [dict(id='random',name='Random bot',description='Chooses randomly from available moves. A baseline opponent.'),dict(id='greedy',name='Greedy bot',description='Prefers cheap conquests and larger armies. A scripted benchmark.')]
    # Old lobby projections can survive a worker restart. Change only catalog
    # metadata; never hydrate, join, start or alter anyone's room/game.
    lua = '''local rows=redis.call('HGETALL',KEYS[1]);local models=cjson.decode(ARGV[1]);local n=0;for i=1,#rows,2 do local ok,r=pcall(cjson.decode,rows[i+1]);if ok then r.botModels=models;redis.call('HSET',KEYS[1],rows[i],cjson.encode(r));n=n+1 end end;redis.call('PUBLISH',KEYS[2],'model-release');return n'''
    redis_pods = json.loads(run(kubectl+['get','pods','-l','app=redis','-o','json'],capture_output=True,text=True).stdout)['items']
    if not redis_pods: raise ValueError('no Redis pod found to refresh the lobby catalog')
    run(kubectl+['exec',redis_pods[0]['metadata']['name'],'--','redis-cli','EVAL',lua,'2','game:rooms','game:lobby:changed',json.dumps(builtins+catalog)])
    return dict(image=image, previous_image=previous, context=args.context, replicas_verified=len(pods))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--name', required=True, help='new menu label, e.g. "Training 6"')
    connection = p.add_mutually_exclusive_group()
    connection.add_argument('--host', default=os.environ.get('SMALLWORLD_GPU_HOST', 'gpu'))
    connection.add_argument('--ssh-command', help='complete SSH connection command including destination, e.g. \"ssh gpu\"')
    p.add_argument('--checkpoint',choices=['auto','champion','latest'],default='auto',help='auto selects the evaluated incumbent for league runs; latest explicitly exports experimental weights')
    p.add_argument('--remote-root', default='~/github/smallworld')
    p.add_argument('--remote-run', help='absolute or ~/ checkpoint directory on GPU host')
    p.add_argument('--remote-source', help='remote ai directory; default: checkpoint engine directory')
    p.add_argument('--remote-release', help='reuse a previously frozen release directory')
    p.add_argument('--output-dir', type=Path, default=ROOT/'ai/models')
    p.add_argument('--with-checkpoint', action='store_true', help='also download resumable weights under ai/runs/imports')
    p.add_argument('--deploy-kind', action='store_true', help='publish to the existing local kind worker deployment')
    p.add_argument('--context', default='kind-game-local')
    p.add_argument('--cluster', default='game-local')
    p.add_argument('--namespace', default='game')
    args = p.parse_args()
    if args.ssh_command is not None:
        try:
            connection_args = shlex.split(args.ssh_command)
        except ValueError as error:
            p.error('invalid --ssh-command: '+str(error))
        if len(connection_args) < 2 or any(not arg or '\0' in arg for arg in connection_args):
            p.error('--ssh-command must include an executable and destination, e.g. "ssh gpu"')
    slug = re.sub(r'[^a-z0-9]+','-',args.name.lower()).strip('-')[:30]
    if not slug or not 1 <= len(args.name) <= 80 or (args.ssh_command is None and args.host.startswith('-')):
        p.error('invalid model name or SSH host')
    config = dict(root=args.remote_root,run=args.remote_run,source=args.remote_source,release=args.remote_release,name=args.name,slug=slug,checkpoint=args.checkpoint)
    if args.remote_root.startswith('~/'):
        remote_python = '"$HOME"/'+shlex.quote(args.remote_root[2:].rstrip('/')+'/ai/.venv/bin/python')
    elif args.remote_root.startswith('/'):
        remote_python = shlex.quote(args.remote_root.rstrip('/')+'/ai/.venv/bin/python')
    else:
        p.error('--remote-root must be absolute or start with ~/')
    # The optional host runtime setup supplies CUDA/compiler paths. The venv
    # belongs to the selected checkout; all user arguments are shell quoted.
    command = 'set -e; if [ -f "$HOME/.local/share/smallworld-training/env.sh" ]; then source "$HOME/.local/share/smallworld-training/env.sh"; fi; exec '+remote_python+' -c '+shlex.quote(REMOTE_EXPORT)+' '+shlex.quote(json.dumps(config))
    connection_label = args.ssh_command if args.ssh_command is not None else args.host
    print('Freezing checkpoint via '+connection_label+' (training continues).',flush=True)
    result = run(ssh(args.host,shlex.join(['bash','-lc',command]),args.ssh_command),capture_output=True,text=True,timeout=180)
    manifest = json.loads(result.stdout)
    print('Selected '+manifest.get('checkpoint_selection','frozen release')+' at step '+str(manifest['step']),flush=True)
    print('Frozen release: '+manifest['release'],flush=True)
    model_id = manifest['id']
    if not re.fullmatch(r'[a-z0-9][a-z0-9_-]{0,63}',model_id): raise ValueError('invalid release ID')
    remote = manifest['release']+'/'+model_id+'.json'
    model = run(ssh(args.host,'cat -- '+shlex.quote(remote),args.ssh_command),capture_output=True,timeout=90).stdout
    validate_model(model,manifest,args.name)
    output = args.output_dir.resolve();output.mkdir(parents=True,exist_ok=True)
    for existing in output.glob('*.json'):
        old = json.loads(existing.read_text())
        if old.get('name')==args.name and old.get('id')!=model_id:
            raise ValueError('That name already exists. Choose a new numbered name to preserve old agents.')
    destination = output/(model_id+'.json')
    if destination.exists() and destination.read_bytes()!=model:
        raise ValueError('Refusing to replace immutable model ID '+model_id)
    with tempfile.TemporaryDirectory(prefix='smallworld-validate-') as tmp:
        tmp=Path(tmp)
        for old in output.glob('*.json'):shutil.copyfile(old,tmp/old.name)
        (tmp/destination.name).write_bytes(model)
        validate_with_engine(tmp)
    receipt = ROOT/'ai/runs/imports'/model_id;receipt.mkdir(parents=True,exist_ok=True)
    if args.with_checkpoint:
        checkpoint = run(ssh(args.host,'cat -- '+shlex.quote(manifest['release']+'/checkpoint.pt'),args.ssh_command),capture_output=True,timeout=90).stdout
        if hashlib.sha256(checkpoint).hexdigest()!=manifest['checkpoint_sha256']:raise ValueError('checkpoint checksum mismatch')
        (receipt/'checkpoint.pt').write_bytes(checkpoint)
    with tempfile.NamedTemporaryFile(dir=output,prefix='.import-',delete=False) as f:
        temporary=Path(f.name);f.write(model)
    try: os.replace(temporary,destination)
    finally:temporary.unlink(missing_ok=True)
    manifest['local_model']=str(destination)
    (receipt/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    print('Imported '+args.name+': '+str(destination),flush=True)
    if args.deploy_kind:
        manifest['deployment']=deploy_kind(args,output,model_id)
        (receipt/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
        print(args.name+' is deployed and available in the bot selector.',flush=True)
    else:
        print('To publish it, rerun with --deploy-kind --remote-release '+shlex.quote(manifest['release']),flush=True)

if __name__=='__main__':
    try:main()
    except (OSError,ValueError,KeyError,subprocess.SubprocessError) as error:
        if isinstance(error,subprocess.CalledProcessError) and error.stderr:
            print(error.stderr.decode() if isinstance(error.stderr,bytes) else error.stderr,file=sys.stderr)
        print('Import failed: '+str(error),file=sys.stderr);sys.exit(1)
