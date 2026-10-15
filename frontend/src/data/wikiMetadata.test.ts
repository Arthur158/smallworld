import application from '../redux/slices/applicationSlice';
import { mapDatabase } from './mapData';
import metadata from './wikiMapMetadata.json';

test('reference terrain ids match every built-in map polygon',()=>{
  for(const [map,regions] of Object.entries(metadata)) {
    if(!mapDatabase[map]) continue;
    expect(Object.keys(regions).sort()).toEqual(mapDatabase[map].Tiles.map(tile=>String(tile.ID)).sort());
  }
});
test('generated-map terrain survives into the region inspection model',()=>{
  URL.createObjectURL=jest.fn(()=> 'blob:test-map');
  const state=application(undefined,{type:'init'});
  const result=application(state,{type:'application/websocketMessageReceived',payload:JSON.stringify({type:'generatedmapvisuals',data:{mapName:'generated-wiki-test',offset:0.808,imageBase64:'',graphics:{Tiles:[{ID:'x',Polygon:{Coords:[0,0,10,0,10,10],StackX:4,StackY:4}}],OffsetStacks:1,Xmult:1,Ymult:1},serverTiles:[{id:'x',biome:'Mountain',attributes:['Magic'],isEdge:true}]}})});
  expect(result.tiles.x.terrain).toEqual({biome:'Mountain',attributes:['Magic'],isEdge:true,isSkyIsland:undefined});
});
