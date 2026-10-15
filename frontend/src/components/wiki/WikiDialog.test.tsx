import React from 'react';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import application from '../../redux/slices/applicationSlice';
import WikiToolbar from './WikiToolbar';
import { WikiProvider, useWiki } from './WikiContext';
import { artwork, findArticle, tribePieces, tribesForPiece, wikiArticles } from '../../data/wikiCatalog';
import { Player } from '../../types/Board';

const player: Player = { name: 'Ada', activeTribe: { race: 'Trolls', trait: 'Dragon Master' }, passiveTribes: [{ race: 'Dwarves', trait: 'Forest' }], pieceStacks: [] };
function setup(gameStarted = true) {
  const initial = application(undefined, { type: 'init' });
  const store = configureStore({ reducer: { application }, preloadedState: { application: { ...initial, isAuthenticated: true, gameStarted, players: [player], availableTribes: [{ race:'Elves',trait:'Hill',pieceCount:10,coinCount:0 }] } } });
  const view = render(<Provider store={store}><WikiProvider><WikiToolbar /><Triggers /></WikiProvider></Provider>);
  return { store, ...view };
}
function Triggers() {
  const {openWiki} = useWiki();
  return <><button onClick={() => openWiki({kind:'piece',stack:{type:'Dragon',amount:1,isActive:true}})}>Inspect dragon</button>
    <button onClick={() => openWiki({kind:'zone',tile:{id:'0',polygon:{coords:[0,0,10,0,10,10],stackX:5,stackY:5},pieceStack:[],terrain:{biome:'Mountain',attributes:['Mine']}}})}>Inspect empty mountain</button></>;
}

test('search, categories, market pair links, related pieces and back navigation', () => {
  setup(); fireEvent.click(screen.getByRole('button',{name:'Game wiki'}));
  fireEvent.change(screen.getByRole('textbox',{name:'Search the wiki'}),{target:{value:'Elves'}});
  fireEvent.click(screen.getByRole('button',{name:'Races'}));
  const nav=screen.getByRole('navigation',{name:'Wiki entries'});
  fireEvent.click(within(nav).getByRole('button',{name:/Elves/}));
  expect(screen.getByRole('heading',{name:'Paired in this game'})).toBeTruthy();
  fireEvent.click(screen.getByRole('button',{name:'Hill Elves →'}));
  expect(screen.getByRole('heading',{name:'Hill Elves'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Hill'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Related pieces'})).toBeTruthy();
  fireEvent.click(screen.getByRole('button',{name:'Back'}));
  expect(screen.getByRole('heading',{name:'Paired in this game'})).toBeTruthy();
  fireEvent.change(screen.getByRole('textbox',{name:'Search the wiki'}),{target:{value:'nonexistent-reference'}});
  expect(screen.getByText('No entries found. Try a race, trait, power or piece name.')).toBeTruthy();
});

test('a special stack opens its actual race, power, and their token gallery together', () => {
  setup(); fireEvent.click(screen.getByRole('button',{name:'Inspect dragon'}));
  expect(screen.getByRole('heading',{name:'Dragon Master Trolls'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Trolls'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Dragon Master'})).toBeTruthy();
  expect(screen.getByRole('button',{name:'Lair'})).toBeTruthy();
  expect(screen.getAllByAltText('Dragon').length).toBeGreaterThan(0);
});

test('an empty zone still explains terrain and attributes', () => {
  setup(); fireEvent.click(screen.getByRole('button',{name:'Inspect empty mountain'}));
  expect(screen.getByRole('heading',{name:'Region 0'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Mountain'})).toBeTruthy();
  expect(screen.getByRole('heading',{name:'Mine symbol'})).toBeTruthy();
  expect(screen.getByText('No pieces occupy this region.')).toBeTruthy();
});

test('the dialog traps focus, closes with Escape, and restores the launch button', () => {
  setup(); const launch=screen.getByRole('button',{name:'Game wiki'});launch.focus();fireEvent.click(launch);
  expect(document.activeElement).toBe(screen.getByRole('textbox',{name:'Search the wiki'}));
  const dialog=screen.getByRole('dialog');
  const focusables=Array.from(dialog.querySelectorAll<HTMLElement>('button:not([disabled]),input,a[href]'));
  focusables[focusables.length-1].focus();fireEvent.keyDown(document.activeElement!,{key:'Tab'});
  expect(document.activeElement).toBe(focusables[0]);
  fireEvent.keyDown(document.activeElement!,{key:'Escape'});
  expect(screen.queryByRole('dialog')).toBeNull();expect(document.activeElement).toBe(launch);
});

test('declined pieces keep their former power, and all documented pieces have artwork', () => {
  expect(tribesForPiece({type:'Dwarves',amount:1,isActive:false},[player])).toEqual([player.passiveTribes[0]]);
  expect(tribePieces(player.activeTribe!)).toEqual(expect.arrayContaining(['Trolls','Lair','Dragon']));
  expect(new Set(wikiArticles.map(entry=>entry.id)).size).toBe(wikiArticles.length);
  for(const article of wikiArticles) for(const piece of article.pieces) expect(artwork('piece',piece)).toBeDefined();
  expect(findArticle('race:Trolls')?.sources.length).toBeGreaterThan(0);
});


test('traits have a dedicated searchable category with their artwork and related pieces', () => {
  setup(); fireEvent.click(screen.getByRole('button', { name: 'Game wiki' }));
  fireEvent.click(screen.getByRole('button', { name: 'Traits' }));
  fireEvent.change(screen.getByRole('textbox', { name: 'Search the wiki' }), { target: { value: 'Dragon Master' } });
  fireEvent.click(within(screen.getByRole('navigation', { name: 'Wiki entries' })).getByRole('button', { name: 'Dragon Master Traits' }));
  expect(screen.getByRole('heading', { name: 'Dragon Master' })).toBeTruthy();
  expect(screen.getByAltText('Dragon Master')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Dragon' })).toBeTruthy();
});

test('the lobby library does not present retained room data as a current game', () => {
  setup(false); fireEvent.click(screen.getByRole('button', { name: 'Game wiki' }));
  expect(screen.queryByRole('heading', { name: 'In your game' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Inspect dragon' }));
  expect(screen.queryByRole('heading', { name: 'Dragon Master Trolls' })).toBeNull();
});


test('relic and place powers are separate from traits and show their associated pieces', () => {
  setup(); fireEvent.click(screen.getByRole('button', { name: 'Game wiki' }));
  fireEvent.click(screen.getByRole('button', { name: 'Powers' }));
  const nav = screen.getByRole('navigation', { name: 'Wiki entries' });
  expect(within(nav).getAllByRole('button')).toHaveLength(15);
  expect(within(nav).queryByRole('button', { name: /Dragon Master/ })).toBeNull();
  fireEvent.click(within(nav).getByRole('button', { name: 'Wickedest Pentacle Powers' }));
  expect(screen.getByRole('heading', { name: 'Wickedest Pentacle' })).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Balrog' })).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Balrog' }));
  expect(screen.getByRole('heading', { name: 'Related power' })).toBeTruthy();
  expect(screen.queryByRole('heading', { name: 'Dragon Master Trolls' })).toBeNull();
});
