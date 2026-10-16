import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from smallworld_ai.lifecycle import JsonlLog, prune_checkpoints
from smallworld_ai.train import parser, run
from smallworld_ai.model import atomic_torch_save, load_checkpoint

class LifecycleTests(unittest.TestCase):
    def test_retention_preserves_latest_parent_exports_and_unrelated_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for name in ('step-2.pt','step-10.pt','step-100.pt','step-manual.pt','latest.pt','parent.pt'):
                (root/name).write_text(name)
            (root/'exports').mkdir();(root/'exports/step-1.pt').write_text('export')
            prune_checkpoints(root,2)
            self.assertEqual({p.name for p in root.iterdir()}, {'step-10.pt','step-100.pt','step-manual.pt','latest.pt','parent.pt','exports'})
            self.assertTrue((root/'exports/step-1.pt').exists())

    def test_logs_rotate_as_complete_json_records(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            with JsonlLog(root/'metrics.jsonl',max_bytes=80,backups=2) as log:
                for i in range(100):log.write(json.dumps(dict(step=i))+'\n')
            files=list(root.iterdir());self.assertEqual(len(files),3)
            for file in files:
                for line in file.read_text().splitlines():self.assertIn('step',json.loads(line))
            self.assertEqual(json.loads((root/'metrics.jsonl').read_text().splitlines()[-1])['step'],99)

    def test_zero_step_limit_keeps_training_until_interrupted_and_resumes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);saved=[]
            args=['--device','cpu','--envs','1','--rollout-steps','4','--total-steps','0','--epochs','1','--batch-size','4','--hidden','8','--save-every','1','--keep-checkpoints','2','--output',str(root)]
            def save(payload,path):
                atomic_torch_save(payload,path)
                if path.name=='latest.pt':
                    saved.append(payload['step'])
                    if len(saved)==3:raise KeyboardInterrupt
            with contextlib.redirect_stdout(io.StringIO()),patch('smallworld_ai.train.atomic_torch_save',side_effect=save):
                with self.assertRaises(KeyboardInterrupt):run(parser().parse_args(args))
            self.assertEqual(saved,[4,8,12])
            _,checkpoint=load_checkpoint(root/'latest.pt');self.assertEqual(checkpoint['step'],12)
            args[args.index('--total-steps')+1]='16';args+=['--resume',str(root/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            _,checkpoint=load_checkpoint(root/'latest.pt');self.assertEqual(checkpoint['step'],16)
            self.assertLessEqual(len(list(root.glob('step-*.pt'))),2)

if __name__=='__main__':unittest.main()
