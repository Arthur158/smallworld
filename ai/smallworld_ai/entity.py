"""Function-preserving entity expansion: public tiles, players and piece types.

Feature offsets describe the observation schema, never tactical game rules.
The format-5 network remains intact; new actor/value contributions start at zero.
"""
import torch
from torch import nn
from torch.nn import functional as F
from torch.distributions import Categorical
from . import STATE_FEATURES, ACTION_FEATURES
from .graph import S, A, N, SPEC
from .relational import RelationalPolicy

WIDTH, DEPTH, HEADS = 256, 6, 8
R, T, K = (len(SPEC[k]) for k in ('races', 'traits', 'stacks'))
PLAYER_WIDTH = 1+2*R+3*T+K
RESERVE = 1+R+2*T
TILE_STACK = 23+R+2*T
ACTION_STACK = ACTION_FEATURES+len(SPEC['action_kinds'])
assert TILE_STACK+K+len(SPEC['states'])+8 == N


class EntityBlock(nn.Module):
    def __init__(self):
        super().__init__()
        self.norm1, self.norm2 = nn.LayerNorm(WIDTH), nn.LayerNorm(WIDTH)
        self.qkv, self.out = nn.Linear(WIDTH,3*WIDTH), nn.Linear(WIDTH,WIDTH)
        self.ff = nn.Sequential(nn.Linear(WIDTH,4*WIDTH),nn.GELU(),nn.Linear(4*WIDTH,WIDTH))
        self.edge_bias = nn.Parameter(torch.zeros(HEADS))

    def forward(self,x,valid,adjacency):
        b,n,d=x.shape
        q,k,v=self.qkv(self.norm1(x)).reshape(b,n,3,HEADS,d//HEADS).permute(2,0,3,1,4)
        bias=(adjacency[:,None]*self.edge_bias[None,:,None,None]).masked_fill(~valid[:,None,None,:],float('-inf'))
        attended=F.scaled_dot_product_attention(q,k,v,attn_mask=bias).transpose(1,2).reshape(b,n,d)
        x=x+self.out(attended)
        return (x+self.ff(self.norm2(x)))*valid[:,:,None]


class EntityBranch(nn.Module):
    def __init__(self):
        super().__init__()
        self.state,self.tile=nn.Linear(S,WIDTH),nn.Linear(N,WIDTH)
        self.player,self.stack=nn.Linear(PLAYER_WIDTH+6,WIDTH),nn.Linear(14,WIDTH)
        self.stack_id=nn.Embedding(K,WIDTH)
        self.blocks=nn.ModuleList([EntityBlock() for _ in range(DEPTH)])
        self.action=nn.Linear(A+3*WIDTH,WIDTH)
        self.actor=nn.Sequential(nn.Linear(2*WIDTH,WIDTH),nn.GELU(),nn.Linear(WIDTH,1))
        self.critic=nn.Sequential(nn.Linear(WIDTH,WIDTH),nn.GELU(),nn.Linear(WIDTH,1))
        for head in (self.actor[-1],self.critic[-1]):
            nn.init.zeros_(head.weight);nn.init.zeros_(head.bias)

    def encode(self,states,candidates,mask,nodes,adjacency,node_mask):
        b,n,_=nodes.shape
        players=states[:,STATE_FEATURES:STATE_FEATURES+6*PLAYER_WIDTH].reshape(b,6,PLAYER_WIDTH)
        order=torch.eye(6,device=states.device).expand(b,-1,-1)
        reserve=players[:,:,RESERVE:RESERVE+K]*players[:,:,:1]
        counts=nodes[:,:,TILE_STACK:TILE_STACK+K]*node_mask[:,:,None]
        owners=nodes[:,:,14:22]*node_mask[:,:,None]
        stacks=torch.cat((reserve,owners.transpose(1,2)@counts),1).transpose(1,2)
        offered=(candidates[:,:,ACTION_STACK:ACTION_STACK+K]*mask[:,:,None]).sum(1)>0
        valid=torch.cat((torch.ones(b,1,dtype=torch.bool,device=nodes.device),node_mask.bool(),
                         players[:,:,0]>0,(stacks.abs().sum(-1)>0)|offered),1)
        x=torch.cat((self.state(states)[:,None],self.tile(nodes),
                     self.player(torch.cat((players,order),-1)),self.stack(stacks)+self.stack_id.weight[None]),1)
        edge=F.pad((adjacency>0).to(nodes.dtype),(1,x.shape[1]-n-1,1,x.shape[1]-n-1))
        x=x*valid[:,:,None]
        for block in self.blocks:x=block(x,valid,edge)
        return x[:,0],x[:,1:n+1],x[:,n+7:]

    def forward(self,batch,value=True,actions=None,packed=True,return_features=False):
        states,candidates,mask,nodes,adjacency,node_mask,targets=batch
        context,board,stacks=self.encode(states,candidates,mask,nodes,adjacency,node_mask)
        tiles=torch.cat((torch.zeros_like(board[:,:1]),board),1)
        packed=self.training and torch.is_grad_enabled() and packed
        if packed:
            rows,columns=mask.nonzero(as_tuple=True)
            choices=candidates[rows,columns]
            target,source=(tiles[rows,targets[rows,columns,i]+1] for i in (0,1))
            identity=choices[:,ACTION_STACK:ACTION_STACK+K]
            piece=stacks[rows,identity.argmax(-1)]*identity.sum(-1,keepdim=True)
            global_state=context[rows]
        else:
            choices=candidates
            def gather(i):return tiles.gather(1,(targets[:,:,i]+1)[:,:,None].expand(-1,-1,WIDTH))
            target,source=gather(0),gather(1)
            identity=choices[:,:,ACTION_STACK:ACTION_STACK+K]
            piece=stacks.gather(1,identity.argmax(-1)[:,:,None].expand(-1,-1,WIDTH))*identity.sum(-1,keepdim=True)
            global_state=context[:,None].expand(-1,candidates.shape[1],-1)
        query=F.gelu(self.action(torch.cat((choices,target,source,piece),-1)))
        score=self.actor(torch.cat((query,global_state),-1)).squeeze(-1)
        selected=None
        if packed:
            residual=score.new_zeros(mask.shape);residual[rows,columns]=score
            if actions is not None:
                offset=mask.sum(-1).cumsum(0)-mask.sum(-1)
                selected=query[offset+mask.long().cumsum(-1)[torch.arange(mask.shape[0],device=mask.device),actions]-1]
        else:
            residual=score
            if actions is not None:selected=query[torch.arange(mask.shape[0],device=mask.device),actions]
        result = (residual,self.critic(context).squeeze(-1) if value else None,board,selected)
        return (*result,context,query) if return_features else result


class EntityPolicy(RelationalPolicy):
    def __init__(self,hidden=320):
        super().__init__(hidden)
        self.format_version,self.architecture=6,'entity-v1'
        self.entity=EntityBranch()
        # Preserve existing auxiliary heads while teaching the new branch too.
        self.entity_board_feedback=nn.Linear(WIDTH,128)
        self.entity_action_feedback=nn.Linear(WIDTH,hidden)
        for layer in (self.entity_board_feedback,self.entity_action_feedback):
            nn.init.zeros_(layer.weight);nn.init.zeros_(layer.bias)

    def entity_forward(self,batch,value=True,actions=None):
        original=self.relational_forward(*batch,value=value,actions=actions)
        scores,values,board,selected=self.entity(batch,value=value,actions=actions,packed=getattr(self,'pack_candidates',True))
        result=(Categorical(logits=(original[0].logits+scores).masked_fill(~batch[2],-1e9),validate_args=False),
                original[1]+values if value else None)
        if actions is None:return result
        embedding=original[4]+self.entity_board_feedback(board)
        hidden=original[2]+self.entity_action_feedback(selected)
        return (*result,hidden,self.survival_head(embedding).squeeze(-1),embedding)

    def forward(self,*batch):return self.entity_forward(batch)
    def action_distribution(self,*batch):return self.entity_forward(batch,value=False)[0]
    def training_forward(self,batch,actions):return self.entity_forward(batch,actions=actions)

    def warm_start(self,old):
        if getattr(old,'format_version',0)!=5 or old.hidden!=self.hidden:
            raise ValueError('entity expansion requires a relational policy of the same hidden size')
        missing,unexpected=self.load_state_dict(old.state_dict(),strict=False)
        if unexpected or any(not name.startswith('entity') for name in missing):
            raise ValueError('incompatible entity expansion')
