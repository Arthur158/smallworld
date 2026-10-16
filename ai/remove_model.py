#!/usr/bin/env python3
"""Archive a bot model locally and optionally remove it from the local kind game."""
import argparse
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
BUILTINS = [dict(id='random', name='Random bot', description='Chooses randomly from available moves. A baseline opponent.'),
            dict(id='greedy', name='Greedy bot', description='Prefers cheap conquests and larger armies. A scripted benchmark.')]


def run(command, **kwargs):
    return subprocess.run([str(v) for v in command], check=True, **kwargs)


def model_record(filename, data):
    if Path(filename).name != filename or not filename.endswith('.json'):
        raise ValueError('unexpected model filename: '+filename)
    value = json.loads(data)
    if not re.fullmatch(r'[a-z0-9][a-z0-9_-]{0,63}', value['id']):
        raise ValueError('invalid model ID')
    return dict(id=value['id'], name=value['name'], description=value.get('description', ''),
                filename=filename, sha256=hashlib.sha256(data).hexdigest(), data=data)


def local_models(directory):
    return [model_record(p.name, p.read_bytes()) for p in sorted(directory.glob('*.json'))]


def resolve_model(records, model_id=None, name=None):
    if model_id in ('random', 'greedy') or name in ('Random bot', 'Greedy bot'):
        raise ValueError('built-in random and greedy bots cannot be removed')
    matches = [m for m in records if m['id'] == model_id] if model_id else [m for m in records if m['name'] == name]
    identities = {(m['id'], m['sha256']) for m in matches}
    if not matches:
        raise ValueError('model not found; use --list to see exact names and IDs')
    if len(identities) != 1 or len({m['filename'] for m in matches}) != 1:
        raise ValueError('ambiguous or inconsistent model; choose --id and reconcile differing copies first')
    selected = matches[0]
    if selected['id'] in ('random', 'greedy'):
        raise ValueError('built-in bots cannot be removed')
    return selected


# Include snapshots missing from the lobby and rewind/forward history. Catalog
# entries are availability, not usage, and must not count as room references.
ROOM_USAGE = '''
local id=ARGV[1]
local function uses(v)
  if type(v)~='table' then return false end
  if v.botModel==id or v.modelId==id then return true end
  for k,x in pairs(v) do if k~='botModels' and uses(x) then return true end end
  return false
end
local found={};local rows=redis.call('HGETALL','game:rooms')
for i=1,#rows,2 do
  local r=cjson.decode(rows[i+1])
  if uses(r) then found[rows[i]]=r.name or rows[i] end
end
local cursor='0'
repeat
  local page=redis.call('SCAN',cursor,'MATCH','game:room:*:snapshot','COUNT',100)
  cursor=page[1]
  for _,key in ipairs(page[2]) do
    local raw=redis.call('GET',key)
    if raw then local r=cjson.decode(raw);if uses(r) then found[r.id or key]=r.name or key end end
  end
until cursor=='0'
local result={};for id,name in pairs(found) do table.insert(result,name..' ['..id..']') end
return result
'''
REFRESH = '''local rows=redis.call('HGETALL','game:rooms');local models=cjson.decode(ARGV[1]);for i=1,#rows,2 do local r=cjson.decode(rows[i+1]);r.botModels=models;redis.call('HSET','game:rooms',rows[i],cjson.encode(r)) end;redis.call('PUBLISH','game:lobby:changed','model-removal');return #rows/2'''


class Cluster:
    def __init__(self, args):
        self.args = args
        self.command = ['kubectl', '--context', args.context, '-n', args.namespace]
        obj = self.json('get', 'statefulset', 'room-worker', '-o', 'json')
        containers = obj['spec']['template']['spec']['containers']
        self.index = next(i for i, c in enumerate(containers) if c['name'] == 'room-worker')
        self.previous = containers[self.index]['image']
        self.replicas = obj['spec'].get('replicas', 1)
        if self.replicas < 1 or obj.get('status', {}).get('readyReplicas', 0) != self.replicas:
            raise ValueError('worker deployment must be ready before removing a model')
        redis = self.json('get', 'pods', '-l', 'app=redis', '-o', 'json')['items']
        if len(redis) != 1:
            raise ValueError('expected exactly one Redis pod')
        self.redis = redis[0]['metadata']['name']
        self.models = self.read_models()

    def json(self, *command):
        return json.loads(run(self.command+list(command), capture_output=True, text=True).stdout)

    def redis_eval(self, script, *arguments):
        return self.json('exec', self.redis, '--', 'redis-cli', '--json', 'EVAL', script, '0', *arguments)

    def assert_unused(self, model_id):
        rooms = self.redis_eval(ROOM_USAGE, model_id)
        if not isinstance(rooms, list):
            raise ValueError('could not check room usage: '+str(rooms))
        if rooms:
            raise ValueError('model is used by room(s): '+', '.join(rooms)+'. Remove those bot seats or rooms first (including retained playback history).')

    def read_models(self):
        pods = self.json('get', 'pods', '-l', 'app=room-worker', '-o', 'json')['items']
        if len(pods) != self.replicas:
            raise ValueError('unexpected number of worker pods')
        catalog = None
        for pod in pods:
            blob = run(self.command+['exec', pod['metadata']['name'], '--', 'tar', '-C', '/app/bot-models', '-cf', '-', '.'], capture_output=True).stdout
            records = []
            with tarfile.open(fileobj=io.BytesIO(blob)) as archive:
                for item in archive:
                    if item.isfile() and item.name.endswith('.json'):
                        records.append(model_record(item.name.removeprefix('./'), archive.extractfile(item).read()))
            records.sort(key=lambda m: (m['id'], m['filename']))
            if catalog is not None and records != catalog:
                raise ValueError('worker model catalogs differ; wait for the current deployment')
            catalog = records
        return catalog

    def refresh(self, models):
        result = self.redis_eval(REFRESH, json.dumps(BUILTINS+[{k:m[k] for k in ('id', 'name', 'description')} for m in models]))
        if not isinstance(result, int):
            raise ValueError('could not refresh lobby model catalog: '+str(result))

    def switch_image(self, expected, replacement):
        path = '/spec/template/spec/containers/'+str(self.index)+'/image'
        patch = [dict(op='test', path=path, value=expected), dict(op='replace', path=path, value=replacement)]
        run(self.command+['patch', 'statefulset', 'room-worker', '--type=json', '-p', json.dumps(patch)])
        run(self.command+['rollout', 'status', 'statefulset/room-worker', '--timeout=180s'])

    def remove(self, model):
        self.assert_unused(model['id'])
        wanted = [m for m in self.models if m['id'] != model['id']]
        paths = ['/app/bot-models/'+m['filename'] for m in self.models if m['id'] == model['id']]
        if not paths:
            return dict(already_absent=True, image=self.previous)
        image = 'smallworld-roomworker:remove-'+model['id']+'-'+str(time.time_ns())
        with tempfile.TemporaryDirectory(prefix='smallworld-remove-image-') as tmp:
            # Preserve the configured image user, executable, other weights and
            # format-support labels. A whiteout removes just these catalog files.
            user = run(['docker', 'image', 'inspect', self.previous, '--format', '{{.Config.User}}'], capture_output=True, text=True).stdout.strip() or '0'
            if not re.fullmatch(r'[A-Za-z0-9_.:-]+', user):
                raise ValueError('unexpected base image user')
            Path(tmp, 'Dockerfile').write_text('ARG BASE\nFROM ${BASE}\nUSER 0\nRUN '+json.dumps(['rm', '--']+paths)+'\nUSER '+user+'\n')
            run(['docker', 'build', '--build-arg', 'BASE='+self.previous, '-t', image, tmp])
        run(['kind', 'load', 'docker-image', image, '--name', self.args.cluster])
        self.assert_unused(model['id'])
        try:
            self.switch_image(self.previous, image)
            actual = self.read_models()
            if actual != wanted:
                raise ValueError('deployed model files differ from the expected catalog')
            self.assert_unused(model['id'])
            self.refresh(actual)
        except (OSError, ValueError, subprocess.SubprocessError):
            # Restore only if our image is still current; never overwrite a
            # concurrent deployment. Preserve the original error if rollback fails.
            try:
                self.switch_image(image, self.previous)
                self.refresh(self.models)
            except (OSError, ValueError, subprocess.SubprocessError) as error:
                print('Rollback needs attention: '+str(error), file=sys.stderr)
            raise
        return dict(image=image, previous_image=self.previous, context=self.args.context, replicas_verified=self.replicas)


def archive_model(model, model_dir, archive_dir, cluster=None):
    destination = model_dir/model['filename']
    if destination.exists() and destination.read_bytes() != model['data']:
        raise ValueError('local model changed; refusing removal')
    archive = archive_dir/(str(time.time_ns())+'-'+model['id'])
    archive.mkdir(parents=True)
    (archive/model['filename']).write_bytes(model['data'])
    receipt = {k:model[k] for k in ('id', 'name', 'filename', 'sha256')}
    receipt.update(archive=str(archive), status='archived; removal pending')
    manifest = archive/'manifest.json'
    manifest.write_text(json.dumps(receipt, indent=2)+'\n')
    if cluster:
        receipt['deployment'] = cluster.remove(model)
    # Only remove the local file after the deployed catalog has been verified.
    if destination.exists():
        if destination.read_bytes() != model['data']:
            raise ValueError('local model changed during deployment; leaving it untouched')
        destination.unlink()
    receipt['status'] = 'removed'
    manifest.write_text(json.dumps(receipt, indent=2)+'\n')
    return receipt


def main():
    p = argparse.ArgumentParser(description=__doc__)
    choice = p.add_mutually_exclusive_group(required=True)
    choice.add_argument('--id', help='exact model ID')
    choice.add_argument('--name', help='exact menu name, e.g. "Training 2"')
    choice.add_argument('--list', action='store_true', help='list names and IDs without changing anything')
    p.add_argument('--dry-run', action='store_true', help='resolve the model and check room usage without changing anything')
    p.add_argument('--deploy-kind', action='store_true', help='also remove from the current kind deployment and refresh the lobby')
    p.add_argument('--model-dir', type=Path, default=ROOT/'ai/models')
    p.add_argument('--archive-dir', type=Path, default=ROOT/'ai/runs/removed')
    p.add_argument('--context', default='kind-game-local')
    p.add_argument('--cluster', default='game-local')
    p.add_argument('--namespace', default='game')
    args = p.parse_args()
    cluster = Cluster(args) if args.deploy_kind else None
    records = local_models(args.model_dir)+(cluster.models if cluster else [])
    if args.list:
        for name, model_id in sorted({(m['name'], m['id']) for m in records}):
            print(name+'\t'+model_id)
        return
    model = resolve_model(records, args.id, args.name)
    if cluster:
        cluster.assert_unused(model['id'])
    if args.dry_run:
        print('Would archive '+model['name']+' ['+model['id']+'] and remove it '+('locally and from '+args.context if cluster else 'locally only')+'.')
        return
    receipt = archive_model(model, args.model_dir.resolve(), args.archive_dir.resolve(), cluster)
    print('Removed '+model['name']+(' locally and from the game.' if cluster else ' locally. The running game is unchanged; use --deploy-kind to remove it there too.'))
    print('Recoverable model and receipt: '+receipt['archive'])
    print('Training checkpoints and the GPU run were not changed.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, KeyError, tarfile.TarError, subprocess.SubprocessError) as error:
        if isinstance(error, subprocess.CalledProcessError) and error.stderr:
            print(error.stderr.decode() if isinstance(error.stderr, bytes) else error.stderr, file=sys.stderr)
        print('Removal failed: '+str(error), file=sys.stderr)
        sys.exit(1)
