import { WikiArticle } from './wikiTypes';
import { raceArticles } from './wikiRaces';
import { traitArticles } from './wikiTraits';
import assets from './wikiAssets.json';
import { PieceStack, Player, Tribe } from '../types/Board';

const base = { label: 'Small World · official rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7901-SW-rules-EN-2018_lowres.pdf' };
const underground = { label: 'Small World Underground · official rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7909-SWU-Rules-EN.pdf' };
export const terrainNames: Record<string, string> = { Forest: 'Forest / mushroom forest', Hill: 'Hills', Field: 'Farmland', Swamp: 'Swamp / mudpool', Water: 'Sea & lake', River: 'River', Mountain: 'Mountain', MineArea: 'Mine region', CrystalArea: 'Mystic crystal region', AbyssalChasm: 'Chasm', Magic: 'Magic source', Mine: 'Mine symbol', Cave: 'Cavern' };
const terrainRules: Record<string, [string, string[]]> = {
  Forest: ['Woodland terrain used by forest-related abilities.', ['The Forest trait earns an extra coin here. Underground mushroom forests use the Forest terrain category in this game.']],
  Hill: ['Hilly terrain, valuable to races and traits that favor hills.', ['Hills do not add defense on their own. Check the Hill trait and the race occupying this region for bonuses.']],
  Field: ['Open farmland, favored by Humans.', ['Farmland has the normal region conquest cost. Humans earn an additional coin here while active.']],
  Swamp: ['Wet terrain: swamps above ground, mudpools below.', ['The Swamp and Muddy traits reward occupying this terrain. Mudmen can grow their army from mudpools.']],
  Water: ['Seas and lakes require an ability that grants water access.', ['Ordinary troops cannot conquer these regions. Seafaring or an appropriate race ability changes that restriction. Water does not make two land regions adjacent.']],
  River: ['A route through Underground, normally crossed rather than held.', ['An empty river region costs one troop before other modifiers. Ordinary troops must leave it during redeployment; an explicit ability is needed to remain there and score.']],
  Mountain: ['High ground adds one to the normal conquest cost.', ['An empty mountain normally needs three troops. The extra defense remains when ownership changes; other defenses and attacking discounts still apply.']],
  MineArea: ['An Underground region rich in ore.', ['Mining, Dwarves and Iron Dwarves check the Mine symbol in this game. Most Mine regions carry it, but check the region symbols rather than relying on the terrain name alone. A mine gives no automatic coin bonus to other races.']],
  CrystalArea: ['An Underground region containing mystic crystals.', ['These regions normally carry a Magic symbol, which is what Mystic, Wizards and Will-o\'-Wisps check in this game. Crystals do not change the ordinary conquest cost.']],
  AbyssalChasm: ['An impassable Underground region.', ['Ordinary troops cannot occupy or cross a chasm. Volcano and race-specific effects are described on their own pages.']],
  Magic: ['A magic-source symbol on a region.', ['This symbol matters to Wizards and other magic-related abilities. It does not grant extra income or defense by itself.']],
  Mine: ['A mine symbol on a region.', ['Dwarves and other mine-related abilities use this symbol. The underlying terrain still determines the normal region rules.']],
  Cave: ['A cavern symbol connects to specific movement abilities.', ['A cavern is not a free connection for every race. Underworld and other explicit cavern abilities explain when distant caverns can be reached.']],
};
export const terrainArticles: WikiArticle[] = Object.entries(terrainRules).map(([key, [summary, details]]) => ({ id: `terrain:${key}`, kind: 'terrain', name: terrainNames[key], summary, details, pieces: ['Magic', 'Mine', 'Cave'].includes(key) ? [key] : [], sources: [base, underground] }));

// Marker descriptions are paraphrases, checked against Power.go and TileModifiers.go.
const markerRules: Record<string, [string, string[]]> = {
  'Lost Tribe': ['Neutral remnants that defend a region at the start of the game.', ['They belong to no player. Count their troops when conquering the region; they do not take turns.']],
  'Coin': ['Victory coins are both your score and the currency for choosing a tribe.', ['The total is revealed at game end. Coins on a tribe choice are collected by the player who selects that combination.']],
  'Coin_No_Num': ['Victory coins mark income or the coins waiting on a tribe choice.', ['See the coin total beside the artwork for the actual amount.']],
  'Decline': ['Choose the Priestesses\' tower region after they decline.', ['Select this piece and one of your declined Priestess regions. The remaining Priestesses from their other regions gather there into a single tower, abandoning those regions.', 'The tower scores one coin per Priestess token each turn and can still be conquered. This piece is specific to Priestesses; use the normal decline control for other races.']],
  'Great Beanstalk': ['An entrance linking the ground to a sky island.', ['Use the board connection to reach the island. Control and conquest still follow the normal region rules and any active abilities.']],
  'Great Stairs': ['A second route between the ground and a sky island.', ['Its location determines the linked entrance. It is a board feature, rather than a race or Special Power.']],
  'Scepter of Avarice': ['Gain the scoring value of one of your regions an extra time.', ['Place the Scepter during redeployment. In this game it immediately awards that region\'s current scoring value; the normal end-of-turn scoring still follows.', 'This game also applies the bonus in the Mine of the Lost Dwarf; the printed rules exclude that location.']],
  "Froggy's Ring": ['Collect tribute from neighboring opponents.', ['Place the Ring in an owned region during redeployment. It collects one coin from each opponent with active troops in a neighboring region, once per opponent.']],
  'Shiny Orb': ['Convert a lone active enemy troop into your own.', ['Once per turn, convert a region containing exactly one active enemy race token into your own. Your active race must already be on the board, and the target must be accessible and adjacent under your movement abilities.', 'Normal immunity still protects the target. Unlike the printed rules, this game does not apply the Gnomes’ special defense to this conversion.']],
  'Sword of the Killer Rabbit': ['A relic that makes one conquest cheaper.', ['Use it for a conquest with a two-troop discount. You still need at least one troop and a valid target.']],
  "Stinky Troll's Socks": ['Use the terrain\'s empty-region cost for a special conquest.', ['The displaced troops return without the ordinary casualty. Normal access and adjacency restrictions still apply.', 'This game skips most defending race abilities and extra region defenses for this attack. Gnomes and hidden Skag Attacks can still interrupt it, but do not assume every printed immunity is enforced.']],
  'Flying Doormat': ['Reach a region beyond your usual neighbors.', ['The relic allows a nonadjacent conquest. It does not remove the normal troop cost or override immunity.']],
  'Diamond Fields': ['Earn extra income from matching terrain.', ['The race holding the Diamond Fields gains one bonus coin for each of its regions with the same terrain, including the Fields themselves. This continues in decline.', 'The place stays in its original region. Other races belonging to the same player do not share its bonus.']],
  'Great Brass Pipe': ['Connect regions that share the Pipe\'s terrain.', ['The race holding the Pipe can reach regions with the same terrain type as the Pipe\'s region as if they were adjacent. It must still pay normal conquest costs and obey terrain access restrictions.']],
  'Fountain of Youth': ['Supply an extra token to the controller\'s current active race.', ['When preparing your troops for a turn, receive one additional token of your active race.', 'In this game, the Fountain can be held by one of your declined races and still supply your current active race. If you have no active race, it adds no token.']],
  'Keep on the Motherland': ['A stronghold worth extra income and defense.', ['This permanent place adds one defense and one coin to its region. These region bonuses persist when it changes hands.']],
  'Mine of the Lost Dwarf': ['A rich location that increases the region\'s income.', ['The region gains two extra coins when scored. This is a Popular Place, distinct from an ordinary mine symbol.']],
  'Stonehedge': ['A place that grants another trait.', ['The controlling race gains the trait assigned when this place is revealed. Control of the place determines who receives that additional ability.']],
  'Altar of Souls': ['Sacrifice a declined region to gain three coins.', ['During redeployment, use the Altar on a region occupied by a declined race. You must have an active race of your own. The region is cleared and you immediately receive three coins.', 'Unlike the printed rule, this version does not restrict the target to your own declined race and performs the sacrifice before end-of-turn scoring.']],
  'Crypt of the Tomb-raider': ['A place that provides a protective ghost.', ['During redeployment, place the Tomb-raider in one of your regions other than the Crypt. Its region becomes immune to conquest.', 'The ghost leaves its old region when collected for your next turn. Capturing the Crypt removes the old protection and transfers control to the new owner.']],
  'Tomb-raider': ['The protective ghost supplied by the Crypt of the Tomb-raider.', ['Place the ghost in an owned region during redeployment to protect it from conquest. It cannot protect the region containing its own Crypt.', 'The ghost is removed when the Crypt is conquered or when its controller prepares troops for the next turn.']],
  'Wickedest Pentacle': ['A place that brings the Balrog into play.', ['In this game, discovering the Pentacle places the Balrog on it. The controller may then move the Balrog once per turn to a neighboring region.', 'Moving it clears the destination troops with a two-token casualty and makes that destination immune. The cleared region gives no ordinary territory income. Controlling the Pentacle determines who commands the Balrog.']],
  'Balrog': ['A powerful creature controlled through the Wickedest Pentacle.', ['Move it once per turn to a neighboring region. It clears defending troops, normally costing their owner two tokens, and makes the destination immune to conquest.', 'Its previous region loses the Balrog\'s protection. Protection begins after its first move; the marker\'s initial appearance on the Pentacle does not itself protect that region in this game.']],
};
const neutralSources = (name: string) => name === 'Decline' ? [{ label: 'Small World Power Pack 2 · official rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/720024-SWC2-Helper-EN.pdf' }] : ['Lost Tribe', 'Coin', 'Coin_No_Num'].includes(name) ? [base] : ['Great Beanstalk', 'Great Stairs'].includes(name) ? [{label:'Small World: Sky Islands',url:'https://www.daysofwonder.com/game/small-world-sky-islands/'}] : [underground];
export const specialPieceArticles: WikiArticle[] = Object.entries(markerRules).map(([name, [summary, details]]) => ({ id: `piece:${name}`, kind: 'piece', name: name === 'Coin_No_Num' ? 'Victory coins' : name, summary, details, pieces: [name], sources: neutralSources(name) }));
for (let i = 1; i <= 7; i++) specialPieceArticles.push({ id: `piece:Monster${i}`, kind: 'piece', name: `Monster ${i}`, summary: 'A neutral guardian of an Underground region.', details: ['Monster illustrations have the same rules. Their troops add to the conquest cost. Defeating the guardians can reveal a relic or place when those are enabled.'], pieces: [`Monster${i}`], sources: [underground] });

// Relics and Popular Places are powers in this game's rules; race-paired abilities are traits.
const powerNames = ['Scepter of Avarice', "Froggy's Ring", 'Shiny Orb', 'Sword of the Killer Rabbit', "Stinky Troll's Socks", 'Flying Doormat', 'Diamond Fields', 'Great Brass Pipe', 'Fountain of Youth', 'Keep on the Motherland', 'Mine of the Lost Dwarf', 'Stonehedge', 'Altar of Souls', 'Crypt of the Tomb-raider', 'Wickedest Pentacle'];
export const powerArticles: WikiArticle[] = powerNames.map(name => {
  const [summary, details] = markerRules[name];
  return { id: `power:${name}`, kind: 'power', name, summary, details,
    pieces: [name, ...(name === 'Wickedest Pentacle' ? ['Balrog'] : name === 'Crypt of the Tomb-raider' ? ['Tomb-raider'] : [])], sources: [underground] };
});

export function relatedArticles(piece: string): WikiArticle[] {
  return [...raceArticles, ...traitArticles, ...powerArticles, ...terrainArticles].filter(article => article.pieces.includes(piece));
}
export function pieceArticle(piece: string): WikiArticle {
  const explicit = specialPieceArticles.find(article => article.id === `piece:${piece}`);
  if (explicit) return explicit;
  const related = relatedArticles(piece);
  return { id: `piece:${piece}`, kind: 'piece', name: piece,
    summary: related.length ? `A token associated with ${related.map(article => article.name).join(' and ')}.` : 'A special marker used on the board or in a player’s reserve.',
    details: related.length ? ['Its effect is described with the linked race or trait below. The number shown on the board is the size of the stack.'] : ['A verified standalone description is not available for this marker. The current tribe and linked rules may provide its context.'],
    pieces: [piece], sources: related.flatMap(article => article.sources) };
}
export const wikiArticles = [...raceArticles, ...traitArticles, ...powerArticles, ...terrainArticles, ...assets.stacks.filter(name => name !== 'Coin_No_Num').map(pieceArticle)];
export const findArticle = (id: string) => wikiArticles.find(article => article.id === id);
export function artwork(kind: 'race' | 'trait' | 'power' | 'piece', name: string): string | undefined {
  const group = kind === 'race' ? 'races' : kind === 'trait' ? 'traits' : 'stacks';
  return assets[group].includes(name) ? `/${group}/${encodeURIComponent(name)}.png` : undefined;
}
export function tribePieces(tribe: Tribe): string[] {
  return Array.from(new Set([...(findArticle(`race:${tribe.race}`)?.pieces ?? []), ...(findArticle(`trait:${tribe.trait}`)?.pieces ?? [])]));
}
export function tribesForPiece(stack: PieceStack, players: Player[]): Tribe[] {
  const isRace = raceArticles.some(article => article.name === stack.type);
  const tribes = players.flatMap(player => isRace
    ? (stack.isActive ? [player.activeTribe] : player.passiveTribes ?? [])
    : [player.activeTribe, ...(player.passiveTribes ?? [])]);
  return tribes.filter((tribe): tribe is Tribe => !!tribe?.race && tribePieces(tribe).includes(stack.type))
    .filter((tribe, index, all) => all.findIndex(other => other.race === tribe.race && other.trait === tribe.trait) === index);
}
