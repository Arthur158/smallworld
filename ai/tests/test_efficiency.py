import copy
import unittest
import torch
from torch.nn import functional as F
from torch.distributions import Categorical
from smallworld_ai.hierarchical import HierarchicalPolicy, WIDTH
from smallworld_ai.graph import S, A, N

class EfficiencyTests(unittest.TestCase):
    def test_joint_probabilities_values_and_gradients_match_original_dense_equations(self):
        torch.set_num_threads(1);torch.manual_seed(846)
        states=torch.randn(4,S);candidates=torch.randn(4,35,A)
        mask=torch.arange(35)[None,:]<torch.tensor([1,35,7,19])[:,None]
        nodes=torch.randn(4,13,N);adj=torch.softmax(torch.randn(4,13,13),-1)
        node_mask=torch.ones(4,13);node_mask[0,9:]=0
        targets=torch.randint(-1,13,(4,35,2))
        batch=(states,candidates,mask,nodes,adj,node_mask,targets)
        for train in (False,True):
            model=HierarchicalPolicy(48,joint=True).train(train)
            with torch.no_grad():model.group_actor[-1].weight.uniform_(-.1,.1)
            reference=copy.deepcopy(model)
            local,pooled=reference.inputs(*batch);first=reference.actor[0]
            x=F.relu(F.linear(states,first.weight[:,:S],first.bias)[:,None]+F.linear(local,first.weight[:,S:]))
            scores=reference.actor[4](F.relu(reference.actor[2](x))).squeeze(-1)
            context=torch.cat((states,pooled),-1);group=reference.group_actor[0]
            group_hidden=F.relu(F.linear(context,group.weight[:,:S+2*WIDTH],group.bias)[:,None]+F.linear(reference.group_features(candidates),group.weight[:,S+2*WIDTH:]))
            scores+=reference.group_actor[2](group_hidden).squeeze(-1)
            expected=Categorical(logits=scores.masked_fill(~mask,-1e9));value=reference.critic(context).squeeze(-1)
            actual,new_value=model(*batch)
            torch.testing.assert_close(expected.logits,actual.logits,atol=2e-6,rtol=2e-5)
            torch.testing.assert_close(value,new_value)
            self.assertTrue(torch.equal(expected.logits.argmax(-1),actual.logits.argmax(-1)))
            self.assertTrue((actual.probs[~mask]==0).all())
            actions=mask.long().sum(-1)-1
            for dist,v in ((expected,value),(actual,new_value)):
                (-dist.log_prob(actions).mean()+v.square().mean()-.02*dist.entropy().mean()).backward()
            for (name,a),(_,b) in zip(reference.named_parameters(),model.named_parameters()):
                torch.testing.assert_close(a.grad,b.grad,atol=3e-6,rtol=3e-4,msg=lambda msg:name+': '+msg)
            with torch.inference_mode():
                torch.testing.assert_close(model.action_distribution(*batch).logits,actual.logits,atol=2e-6,rtol=2e-5)

if __name__=='__main__':unittest.main()
