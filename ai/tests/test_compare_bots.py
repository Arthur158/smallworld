import unittest
from unittest.mock import patch
import subprocess
import tempfile
from pathlib import Path
import json
from compare_bots import exposure, schedule, comparison_plan, select_bots, summarize, summarize_counts, report_counts, resolve_evaluator


class ComparisonTests(unittest.TestCase):
    def test_evaluator_selection_and_explicit_failure(self):
        with patch('compare_bots.subprocess.run') as run:
            self.assertEqual(resolve_evaluator('go')['backend'], 'go')
            run.assert_not_called()
        with patch.dict('os.environ', {'BOT_INFERENCE_PYTHON': ''}), patch('compare_bots.shutil.which', return_value=None):
            self.assertEqual(resolve_evaluator('auto')['backend'], 'go')
            with self.assertRaises(ValueError):
                resolve_evaluator('pytorch')
            with self.assertRaises(ValueError):
                resolve_evaluator('auto', '/missing/python')
        with patch('compare_bots.shutil.which', return_value='/cpu/python'), patch('compare_bots.subprocess.run', return_value=subprocess.CompletedProcess([], 0, '2.8.0+cpu\n')):
            choice = resolve_evaluator('auto', '/cpu/python')
            self.assertEqual(choice, dict(backend='pytorch', python='/cpu/python', torch='2.8.0+cpu'))

    def test_balanced_schedules(self):
        for n in range(2, 13):
            with self.subTest(n=n):
                jobs = schedule(n, [101, 102], 1234)
                exposure(jobs, n)
                self.assertEqual(jobs, schedule(n, [101, 102], 1234))
                self.assertTrue(all(len(j['seats']) == 4 for j in jobs))
        self.assertEqual(len(schedule(10, [1, 2, 3], 1)), 720)

    def test_every_player_count_is_balanced_per_map(self):
        for players in range(2, 7):
            for n in range(2, 13):
                with self.subTest(players=players, bots=n):
                    jobs = schedule(n, [71, 72], 42, players=players)
                    self.assertEqual(jobs, schedule(n, [71, 72], 42, players=players))
                    self.assertEqual(len(jobs), len({j['index'] for j in jobs}))
                    for seed in (71, 72):
                        current = [j for j in jobs if j['seed'] == seed]
                        exposure(current, n)
                        self.assertEqual(len(current), len({j['seats'] for j in current}))
                        self.assertTrue(all(len(j['seats']) == players and j['players'] == players for j in current))
                        if n >= players:
                            self.assertTrue(all(len(set(j['seats'])) == players for j in current))
                        else:
                            self.assertTrue(all(max(j['seats'].count(i) for i in range(n))-min(j['seats'].count(i) for i in range(n)) <= 1 for j in current))
        for players in (1, 7):
            with self.assertRaises(ValueError):
                schedule(4, [1], 1, players=players)

    def test_exhaustive_player_counts(self):
        for players in range(2, 7):
            jobs = schedule(6, [1], 42, 'exhaustive', 'all', players)
            expected = 1
            for i in range(6-players+1, 7): expected *= i
            self.assertEqual(len(jobs), expected)
            exposure(jobs, 6)

    def test_normalized_combined_results_and_reports(self):
        bots = [dict(id='a', name='A'), dict(id='b', name='B')]
        head_to_head = dict(players=2, seats=[0,1], seed=1, scores=[10,0], actions=[1,1], rejections=[0,0])
        tied = dict(players=6, seats=[0,0,0,1,1,1], seed=1, scores=[1]*6, actions=[1]*6, rejections=[0]*6)
        rows = [head_to_head]+[tied]*20
        result = summarize_counts(bots, rows, [2,6])
        a, b = result['leaderboard']
        self.assertEqual(a['id'], 'a')
        self.assertAlmostEqual(a['mean_relative_win_share'], 1.5)
        self.assertAlmostEqual(b['mean_relative_win_share'], .5)
        self.assertAlmostEqual(a['mean_normalized_rank'], .75)
        self.assertEqual(result['by_player_count']['2']['leaderboard'][0]['mean_margin'], 10)
        with self.assertRaises(ValueError): summarize(bots, rows)
        manifest = dict(games=21, seeds=[1], rules='surface', games_by_player_count={'2':1,'6':20})
        with tempfile.TemporaryDirectory() as directory, patch('builtins.print'):
            output = Path(directory)
            report_counts(output, result, manifest)
            self.assertIn('6 players: win share', (output/'report.md').read_text())
            self.assertIn('2 players: win share', (output/'report.md').read_text())
            self.assertEqual(len((output/'leaderboard-by-player-count.csv').read_text().splitlines()), 5)
            for players in (2,6):
                saved = json.loads((output/f'players-{players}'/'summary.json').read_text())
                self.assertEqual(saved['players'], players)
                self.assertIn('pairwise', saved)

    def test_comparable_game_counts_use_fresh_balanced_rounds(self):
        for n in range(2, 13):
            for maps in (1, 3):
                with self.subTest(bots=n, maps=maps):
                    plan = comparison_plan(n, [2,3,4,5,6], maps, 100)
                    self.assertEqual(plan, comparison_plan(n, [2,3,4,5,6], maps, 100))
                    totals = []
                    for count, seeds in plan['seeds_by_player_count'].items():
                        self.assertGreaterEqual(len(seeds), maps)
                        self.assertEqual(seeds, list(range(100, 100+len(seeds))))
                        totals.append(len(seeds)*plan['games_per_map'][int(count)])
                    self.assertLessEqual(5*max(totals), 6*min(totals))
        plan = comparison_plan(2, [2,3,4,5,6], 3, 100)
        totals = []
        for count, seeds in plan['seeds_by_player_count'].items():
            jobs = schedule(2, seeds, 100, players=int(count))
            exposure(jobs, 2)
            self.assertEqual(len(jobs), len({(j['seed'], j['seats']) for j in jobs}))
            totals.append(len(jobs))
        self.assertEqual(totals, [36,36,36,40,36])
        single = comparison_plan(4, [4], 3, 100)
        self.assertEqual(single['seeds_by_player_count'], {'4':[100,101,102]})
        with self.assertRaises(ValueError):
            comparison_plan(2, [2,3,4,5,6], 1, 2**63-2)

    def test_report_uses_actual_map_counts(self):
        bots = [dict(id='a', name='A'), dict(id='b', name='B')]
        rows = [dict(players=2, seats=[0,1], seed=1, scores=[1,0], actions=[1,1], rejections=[0,0]),
                dict(players=3, seats=[0,0,1], seed=1, scores=[0,0,1], actions=[1]*3, rejections=[0]*3)]
        result = summarize_counts(bots, rows, [2,3])
        manifest = dict(games=24, seeds=[1,2,3,4,5,6], rules='surface', games_by_player_count={'2':12,'3':12},
                        seeds_by_player_count={'2':[1,2,3,4,5,6], '3':[1,2]})
        with tempfile.TemporaryDirectory() as directory, patch('builtins.print'):
            output = Path(directory)
            report_counts(output, result, manifest)
            self.assertIn('6 generated evaluation maps', (output/'players-2/report.md').read_text())
            self.assertIn('2 generated evaluation maps', (output/'players-3/report.md').read_text())
            self.assertIn('3 players: 12 games / 2 maps', (output/'report.md').read_text())
            self.assertIn('| Bot | Appearances |', (output/'players-3/report.md').read_text())

    def test_exhaustive(self):
        jobs = schedule(5, [1], 42, 'exhaustive', 'all')
        self.assertEqual(len(jobs), 120)
        exposure(jobs, 5)

    def test_identity_resolution(self):
        bots = [dict(id='a',name='Training 1'),dict(id='b',name='Training 2')]
        self.assertEqual(select_bots(bots, ['training 1','b']), bots)
        for names in (['missing'], ['a','Training 1']):
            with self.assertRaises(ValueError):
                select_bots(bots, names)

    def test_ties_rank_and_pairwise(self):
        bots = [dict(id=str(i),name=str(i)) for i in range(4)]
        row = dict(seats=[0,1,2,3],seed=1,scores=[10,10,5,0],actions=[1]*4,rejections=[0]*4)
        summary = summarize(bots, [row])
        by_id = {s['id']:s for s in summary['leaderboard']}
        self.assertEqual(by_id['0']['win_share'], .5)
        self.assertEqual(by_id['0']['mean_rank'], 1.5)
        self.assertEqual(by_id['2']['mean_rank'], 3)
        self.assertEqual(sum(s['win_credit'] for s in by_id.values()), 1)
        self.assertEqual(summary['pairwise'][0][1]['credit'], .5)
        self.assertEqual(summary['pairwise'][0][2]['credit'], 1)

    def test_duplicate_agents_are_individual_competitors(self):
        row = dict(seats=[0,0,1,1],seed=1,scores=[10,10,5,0],actions=[1]*4,rejections=[0]*4)
        result = summarize([dict(id='a',name='a'),dict(id='b',name='b')],[row])
        self.assertEqual(result['leaderboard'][0]['appearances'],2)
        self.assertEqual(result['leaderboard'][0]['win_share'],.5)
        self.assertEqual(result['pairwise'][0][0]['games'],0)

if __name__ == '__main__':
    unittest.main()
