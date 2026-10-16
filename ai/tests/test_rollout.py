from concurrent.futures import ThreadPoolExecutor
import unittest
import torch
from smallworld_ai import SCHEMA, STATE_FEATURES, ACTION_FEATURES
from smallworld_ai.model import Policy
from smallworld_ai.rollout import collect_async

class Env:
    def __init__(self,seat):self.seat=seat;self.turn=0;self.episode=0
    def observation(self):
        state=[0.0]*STATE_FEATURES;state[:3]=[self.seat,self.turn,self.episode]
        return dict(schema=SCHEMA,state=state,candidates=[dict(features=[0.0]*ACTION_FEATURES)])
    def advance(self):
        self.turn+=1
        reward=self.seat*100+self.turn
        done=self.turn==3
        if done:self.turn=0;self.episode+=1
        return self.observation(),reward,done,False,{}

class RolloutTests(unittest.TestCase):
    def test_async_keeps_each_player_and_episode_in_order_and_drains_all_actions(self):
        torch.set_num_threads(1)
        envs=[Env(i) for i in range(4)]
        observations=[e.observation() for e in envs]
        received=[]
        with ThreadPoolExecutor(max_workers=4) as pool:
            sequences=collect_async(Policy(8),envs,observations,pool,37,'cpu',lambda pair:pair[0].advance(),lambda e,r:received.append((e.seat,r[1])))
        self.assertEqual(sum(map(len,sequences)),37)
        self.assertEqual(len(received),37)
        for i,sequence in enumerate(sequences):
            for j,(batch,action,logprob,value,reward,done) in enumerate(sequence):
                self.assertEqual(batch[0][0,:3].tolist(),[i,j%3,j//3])
                self.assertEqual(reward,i*100+j%3+1)
                self.assertEqual(done,float(j%3==2))
            self.assertEqual(observations[i]['state'][:3],[i,len(sequence)%3,len(sequence)//3])

if __name__=='__main__':unittest.main()
