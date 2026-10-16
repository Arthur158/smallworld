import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock

spec=importlib.util.spec_from_file_location('remove_model',Path(__file__).parents[1]/'remove_model.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class RemoveModelTests(unittest.TestCase):
    def record(self, model_id='training-2', name='Training 2'):
        return m.model_record(model_id+'.json',json.dumps(dict(id=model_id,name=name)).encode())

    def test_exact_selection_ambiguity_and_reserved_models(self):
        first=self.record();second=self.record('different-id')
        self.assertEqual(m.resolve_model([first,second],model_id='training-2'),first)
        with self.assertRaisesRegex(ValueError,'ambiguous'):m.resolve_model([first,second],name='Training 2')
        for key in ('random','greedy'):
            with self.assertRaisesRegex(ValueError,'built-in'):m.resolve_model([],model_id=key)
        with self.assertRaisesRegex(ValueError,'not found'):m.resolve_model([first],name='Training')
        with self.assertRaises(ValueError):m.model_record('../model.json',first['data'])

    def test_archive_round_trip_preserves_other_models_and_checkpoints(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);models=root/'models';models.mkdir();archive=root/'runs/removed'
            selected=self.record();other=self.record('training-1','Training 1')
            for r in (selected,other):(models/r['filename']).write_bytes(r['data'])
            (root/'latest.pt').write_bytes(b'checkpoint')
            result=m.archive_model(selected,models,archive)
            self.assertFalse((models/selected['filename']).exists())
            self.assertEqual((models/other['filename']).read_bytes(),other['data'])
            self.assertEqual((Path(result['archive'])/selected['filename']).read_bytes(),selected['data'])
            self.assertEqual((root/'latest.pt').read_bytes(),b'checkpoint')
            self.assertEqual(result['status'],'removed')

    def test_deployment_failure_keeps_local_weights_and_recovery_copy(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);r=self.record();(root/r['filename']).write_bytes(r['data'])
            cluster=Mock();cluster.remove.side_effect=ValueError('in use')
            with self.assertRaisesRegex(ValueError,'in use'):m.archive_model(r,root,root/'archive',cluster)
            self.assertEqual((root/r['filename']).read_bytes(),r['data'])
            self.assertEqual(len(list((root/'archive').glob('*/training-2.json'))),1)

    def test_room_usage_fails_closed(self):
        cluster=object.__new__(m.Cluster);cluster.redis_eval=Mock(return_value=['Test room [abc]'])
        with self.assertRaisesRegex(ValueError,'Test room'):cluster.assert_unused('training-2')
        cluster.redis_eval.return_value='unexpected'
        with self.assertRaisesRegex(ValueError,'could not check'):cluster.assert_unused('training-2')
        cluster.redis_eval.return_value=[];cluster.assert_unused('training-2')

    def test_failed_post_rollout_verification_restores_previous_image(self):
        cluster=object.__new__(m.Cluster);cluster.previous='old';cluster.replicas=2
        cluster.models=[self.record()];cluster.args=Mock(cluster='game-local',context='kind-game-local')
        cluster.assert_unused=Mock();cluster.switch_image=Mock();cluster.refresh=Mock()
        cluster.read_models=Mock(return_value=cluster.models)
        from unittest.mock import patch
        with patch.object(m,'run',return_value=Mock(stdout='game')):
            with self.assertRaisesRegex(ValueError,'deployed model files'):cluster.remove(cluster.models[0])
        self.assertEqual(cluster.switch_image.call_count,2)
        new=cluster.switch_image.call_args_list[0].args[1]
        self.assertEqual(cluster.switch_image.call_args_list[1].args,(new,'old'))
        cluster.refresh.assert_called_once_with(cluster.models)

if __name__=='__main__':unittest.main()
