import hashlib
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("pull_model", Path(__file__).parents[1] / "pull_model.py")
pull_model = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pull_model)


class PullModelTests(unittest.TestCase):
    def setUp(self):
        self.data = dict(id="training-6-s123", name="Training 6", schema="smallworld-bot-v2")
        self.model = json.dumps(self.data).encode()
        self.manifest = dict(id=self.data["id"], model_sha256=hashlib.sha256(self.model).hexdigest())

    def test_corrupted_download_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "checksum"):
            pull_model.validate_model(self.model + b" ", self.manifest, "Training 6")

    def test_wrong_release_identity_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "identity"):
            pull_model.validate_model(self.model, self.manifest, "Training 7")
        self.manifest["id"] = "training-5-s123"
        with self.assertRaisesRegex(ValueError, "identity"):
            pull_model.validate_model(self.model, self.manifest, "Training 6")

    def test_unsupported_schema_is_rejected(self):
        self.data["schema"] = "future-schema"
        model = json.dumps(self.data).encode()
        self.manifest["model_sha256"] = hashlib.sha256(model).hexdigest()
        with self.assertRaisesRegex(ValueError, "unsupported"):
            pull_model.validate_model(model, self.manifest, "Training 6")

    def test_nonfinite_weights_are_rejected(self):
        self.data["weights"] = [float("nan")]
        model = json.dumps(self.data).encode()
        self.manifest["model_sha256"] = hashlib.sha256(model).hexdigest()
        with self.assertRaisesRegex(ValueError, "non-finite"):
            pull_model.validate_model(model, self.manifest, "Training 6")

    def test_ssh_command_handles_quotes_options_and_all_binary_downloads(self):
        # A fake ssh executable receives the actual subprocess argument vector.
        # Exercise export and both downloads without contacting a real host.
        cases = [
            ('ssh gpu', ['gpu']),
            ('ssh "arthurj"@192.168.27.69', ['arthurj@192.168.27.69']),
            ('ssh -p 2222 -i "/tmp/key $(literal)" gpu',
             ['-p', '2222', '-i', '/tmp/key $(literal)', 'gpu']),
            (None, ['-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=yes',
                    '-o', 'ConnectTimeout=10', '--', 'chosen-host']),
        ]
        for connection, expected in cases:
            with self.subTest(connection=connection), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                executable = root/'ssh'
                calls = root/'calls.jsonl'
                checkpoint = b'checkpoint\x00\xff\n'
                manifest = dict(self.manifest, name='Training 6', step=123,
                                release='/remote/frozen release',
                                checkpoint_sha256=hashlib.sha256(checkpoint).hexdigest())
                executable.write_text(f'''#!{sys.executable}
import json, sys
from pathlib import Path
with Path({str(calls)!r}).open('a') as output:
    output.write(json.dumps(sys.argv[1:])+'\\n')
command = sys.argv[-1]
if command.startswith('bash '):
    print({json.dumps(manifest)!r})
elif 'checkpoint.pt' in command:
    sys.stdout.buffer.write({checkpoint!r})
else:
    sys.stdout.buffer.write({self.model!r})
''')
                executable.chmod(0o755)
                args = ['pull-model.sh', '--name', 'Training 6', '--output-dir', str(root/'models'),
                        '--with-checkpoint']
                args += ['--ssh-command', connection] if connection is not None else ['--host', 'chosen-host']
                with patch.object(sys, 'argv', args), patch.object(pull_model, 'ROOT', root), \
                     patch.dict(os.environ, {'PATH': str(root)+os.pathsep+os.environ['PATH']}), \
                     patch.object(pull_model, 'validate_with_engine'), contextlib.redirect_stdout(io.StringIO()):
                    pull_model.main()
                records = [json.loads(line) for line in calls.read_text().splitlines()]
                self.assertEqual(len(records), 3)
                self.assertTrue(all(record[:-1] == expected for record in records), records)
                self.assertEqual((root/'models'/f'{self.data["id"]}.json').read_bytes(), self.model)
                self.assertEqual((root/'ai/runs/imports'/self.data['id']/'checkpoint.pt').read_bytes(), checkpoint)

    def test_invalid_connection_command_fails_before_remote_operations(self):
        for command in ('', 'ssh', 'ssh "gpu'):
            with self.subTest(command=command), \
                 patch.object(sys, 'argv', ['pull-model.sh', '--name', 'Test', '--ssh-command', command]), \
                 patch.object(pull_model, 'run') as execute, contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit) as error:
                    pull_model.main()
                self.assertEqual(error.exception.code, 2)
                execute.assert_not_called()


if __name__ == "__main__":
    unittest.main()
