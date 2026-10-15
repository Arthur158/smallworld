import type { WikiArticle } from './wikiTypes';

// Mechanics below are checked against the current TraitMap implementation.
// Publisher links provide the corresponding tabletop rules for comparison.
const sources = {
  base: { label: 'Official Small World rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7901-SW-rules-EN-2018_lowres.pdf' },
  underground: { label: 'Official Small World Underground rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7909-SWU-Rules-EN.pdf' },
  packOne: { label: 'Official Power Pack 1 rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/sw_c1_en.pdf' },
  packTwo: { label: 'Official Power Pack 2 rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/720024-SWC2-Helper-EN.pdf' },
  sky: { label: 'Official Sky Islands rules', url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/sw_si_en.pdf' },
};
function trait(name: string, count: number, summary: string, details: string[], source: keyof typeof sources, pieces: string[] = [], note?: string): WikiArticle {
  return { id: `trait:${name}`, kind: 'trait', name, summary,
    details: [`Adds ${count} race tokens to the starting army.`, ...details], pieces,
    sources: [sources[source]], ...(note ? { note } : {}) };
}

export const traitArticles: WikiArticle[] = [
  trait('Alchemist', 4, 'A steady income for an active race.', [
    'Gain 2 extra coins when you score an active turn, independently of how many regions you hold. The bonus ends in decline.',
  ], 'base'),
  trait('Aquatic', 3, 'Coastal territory pays more; inland territory pays less.', [
    'An active region bordering Water earns 1 extra coin. Every other active region earns 1 fewer coin, down to a minimum of zero.',
    'This does not grant water access. Both scoring adjustments end in decline.',
  ], 'packTwo'),
  trait('Barricade', 4, 'Keep a small empire for a flat bonus.', [
    'An active race holding at most 4 regions earns 3 extra coins at scoring. A fifth region removes that bonus.',
    'The bonus ends in decline.',
  ], 'packOne'),
  trait('Behemoth', 4, 'Two creatures turn swamps into military strength.', [
    'The Male and Female Behemoth stacks grow when you gain a Swamp and shrink when you lose one. Each contributes its displayed strength to attack or defense.',
    'Attack or redeploy each creature as one whole stack; accompanying race tokens must still meet the remaining cost and occupation minimum. The creatures disappear on decline.',
  ], 'packTwo', ['Male Behemoth', 'Female Behemoth']),
  trait('Berserk', 4, 'Use a visible reinforcement roll for each conquest.', [
    'The current die result reduces your next attack cost. A new result is rolled at the start of the turn and after the conquest calculation.',
    'You cannot add a second, ordinary last-conquest die roll to the same attack. This benefit belongs to the active race.',
  ], 'base'),
  trait('Bivouacking', 5, 'Spread five encampments across your defensive line.', [
    'Receive 5 Encampments. Each adds 1 defense, and several can reinforce the same region.',
    'Reposition them with your troops. They do not conquer land or replace its occupying race token, and disappear on decline.',
  ], 'base', ['Encampment']),
  trait('Catapult', 4, 'Protect a stronghold and launch a conquest over a region.', [
    'Place the Catapult in a region your race occupies during conquest. That region becomes immune to conquest.',
    'The next eligible jump reaches a region one region away for 1 fewer race token. Recover the Catapult next turn; it disappears on decline.',
  ], 'packOne', ['Catapult']),
  trait('Commando', 4, 'Make ordinary conquests cheaper.', [
    'Each conquest costs 1 fewer race token. The race’s minimum occupation requirement still applies.',
    'This is an attack discount, not an extra defender placed on the board.',
  ], 'base'),
  trait('Copycat', 3, 'Borrow a trait from an available race combination.', [
    'Use the Mirror on an offered combination to give your active race its trait.',
    'The borrowed effect ends at your next turn, or sooner if someone picks that combination. Its special pieces are then removed. Each copied trait keeps its own restrictions.',
  ], 'packOne', ['Mirror']),
  trait('Corrupt', 4, 'Make successful attackers pay for taking your land.', [
    'When an opponent conquers one of your active regions, 1 coin is transferred from the attacker to you.',
    'This does not increase defense and stops protecting the race’s finances in decline.',
  ], 'packOne'),
  trait('Diplomat', 5, 'Choose an opponent who cannot attack you until your next turn.', [
    'During redeployment, give the Diplomat marker to an opponent whose active race you did not attack this turn.',
    'That opponent cannot conquer regions protected by your Diplomat trait. The marker returns at your next turn; it is a pact, not a payment.',
  ], 'base', ['Diplomat']),
  trait('Dragon Master', 5, 'Conquer with one race token and a Dragon.', [
    'Choose the Dragon to attack with 1 race token regardless of ordinary troop cost. Protected regions can still reject the attack.',
    'The Dragon stays in the conquered region, making it immune to conquest. Recover it next turn for another attack. It disappears on decline.',
  ], 'base', ['Dragon']),
  trait('Fireball', 5, 'Store magical ammunition for later attacks.', [
    'At redeployment, receive 1 Fireball for every Magic source you occupy.',
    'A Fireball attack spends one marker to reduce the required race tokens by 2. The race’s occupation minimum still applies. Spent markers disappear; unused ones leave on decline.',
  ], 'packTwo', ['Fireball', 'Magic'], 'This game spends one Fireball per attack. The tabletop option to combine several Fireballs in one attack is not available.'),
  trait('Fisher', 4, 'Connected territory near rivers generates extra income.', [
    'While active, the current game counts links between your occupied non-river regions when an end borders a River. Half that total, rounded down, becomes the bonus.',
    'River regions do not count, and the bonus ends in decline.',
  ], 'underground', [], 'This differs from the official rule: 1 coin per pair of occupied coastal regions, without requiring those regions to be adjacent.'),
  trait('Flocking', 5, 'Keep your entire active empire connected.', [
    'Gain 2 extra coins if every region held by your active race belongs to one connected group. No regions means no bonus.',
    'Connections follow neighboring regions on the map. A remote conquest does not itself join two groups. The bonus ends in decline.',
  ], 'underground'),
  trait('Flying', 5, 'Attack across the map without an adjacent foothold.', [
    'The race ignores the normal adjacency requirement when selecting a conquest target.',
    'Defense costs, occupation minimums and terrain-access restrictions still apply. Flying alone does not open water or chasms.',
  ], 'base'),
  trait('Forest', 4, 'Forests generate extra income.', [
    'Each Forest region held by this active race adds 1 coin to its normal score. The bonus ends in decline.',
  ], 'base'),
  trait('Fortified', 3, 'Build fortresses for defense and income.', [
    'Receive 1 Fortress at redeployment and place it in an occupied region without a Fortress.',
    'Each adds 1 defense and, while the race is active, 1 coin. Defense remains in decline. Clearing or conquering the region removes its Fortress.',
  ], 'base', ['Fortress'], 'The tabletop game has six Fortresses in total. This game creates one each turn without enforcing that total supply limit.'),
  trait('Frightened', 4, 'Concentrate troops for extra regional income.', [
    'A region with at least 3 tokens of this race earns 1 additional coin.',
    'Special pieces such as Dragons or Encampments do not count toward those three race tokens.',
  ], 'underground', [], 'The current scoring check also applies in decline if another effect preserves a stack of at least three race tokens.'),
  trait('Goldsmith', 4, 'Mine regions pay well; other territory loses value.', [
    'Each active region containing a Mine earns 2 extra coins. Every other active region earns 1 fewer coin, down to zero.',
    'Both the bonus and penalty end in decline.',
  ], 'sky', ['Mine']),
  trait('Gunner', 3, 'Protect two positions and bombard their neighbors.', [
    'Place the Left and Right Cannons in your occupied regions. A cannon’s region is immune to conquest.',
    'Move a cannon between your regions or activate it with the Cannon Trigger. Each firing cannon reduces attacks on neighboring regions by 2 race tokens; overlapping fire adds together.',
    'Firing markers reset next turn. Cannons disappear on decline.',
  ], 'sky', ['Left Cannon', 'Right Cannon', 'Cannon Trigger', 'Firing Left Cannon', 'Firing Right Cannon']),
  trait('Haggling', 5, 'Offer trade pacts that reward peace or charge attackers.', [
    'Distribute up to 5 Treaty markers to opponents during redeployment.',
    'An opponent attacking your Haggling regions pays you 1 coin per Treaty and returns them. Outstanding Treaties instead pay that opponent from your coins at your next turn.',
  ], 'sky', ['Treaty'], 'Pacts are assigned during redeployment here. The game does not enforce the tabletop requirement that your coins cover every pact offered.'),
  trait('Heroic', 5, 'Protect important regions with two Heroes.', [
    'Receive 2 Heroes and position them in your occupied regions during redeployment. A Hero makes its region immune to conquest.',
    'Recover them for repositioning at the next conquest phase. Heroes disappear on decline.',
  ], 'base', ['Hero']),
  trait('Hill', 4, 'Hill country brings additional income.', [
    'Each Hill region held by this active race earns 1 extra coin. This ends in decline.',
  ], 'base'),
  trait('Hordes of', 7, 'A larger starting army instead of an ongoing ability.', [
    'The badge contributes 7 ordinary race tokens and has no additional action or scoring bonus.',
    'These troops follow the race’s normal conquest and decline rules.',
  ], 'packTwo', [], 'The tabletop game’s separate Hordes tokens are included in the ordinary race-token count here.'),
  trait('Immortal', 5, 'Recover troops normally lost in a defeat.', [
    'When an active region falls, recover the race token normally lost, together with the surviving force.',
    'You still lose the region and follow the normal return and redeployment sequence. This protection ends in decline.',
  ], 'underground'),
  trait('Imperial', 4, 'Large active empires receive an expansion bonus.', [
    'At scoring, gain 1 extra coin for every occupied region beyond the first three. Five regions give 2 bonus coins.',
    'The bonus ends in decline.',
  ], 'packOne'),
  trait('Lava', 5, 'Clear and temporarily block regions near your mountains.', [
    'At redeployment, each occupied Mountain provides one Lava placement in a neighboring region.',
    'Occupying troops return without normal token loss, and Lava blocks conquest of the region. Lava is removed at your next conquest turn.',
  ], 'packOne', ['Lava']),
  trait('Martyr', 4, 'Receive compensation when an active region falls.', [
    'Gain 1 coin from the supply whenever an opponent conquers a region held by your active race.',
    'Unlike Corrupt, this does not charge the attacker. The bonus ends in decline.',
  ], 'underground'),
  trait('Mercenary', 4, 'Spend coins to reduce a conquest’s troop cost.', [
    'Choose the Mercenary action for an attack: pay 1 coin to reduce the required race tokens by 2.',
    'You need a coin available and must still meet the race’s occupation minimum. The marker is an action, not a defender.',
  ], 'packOne', ['Mercenary']),
  trait('Merchant', 2, 'Every active region earns more.', [
    'Each region held by this active race earns 1 extra coin, regardless of terrain.',
    'The badge provides relatively few starting troops. Its extra income ends in decline.',
  ], 'base'),
  trait('Mining', 4, 'Earn extra coins from mines.', [
    'Each occupied region containing a Mine earns 1 extra coin while this race is active. No separate mining action is needed.',
    'The bonus ends in decline.',
  ], 'underground', ['Mine']),
  trait('Mounted', 5, 'Conquer fields and hills with fewer troops.', [
    'An active race pays 1 fewer race token for a Field or Hill conquest. Other terrain keeps its usual cost, and occupation minimums still apply.',
  ], 'base', [], 'Implemented by the game, but not currently offered among the traits in room settings.'),
  trait('Muddy', 3, 'Swamps keep paying extra after decline.', [
    'Every Swamp region held by this race earns 1 extra coin, including in decline.',
    'This differs from the Swamp badge, whose bonus only applies while active.',
  ], 'underground', [], 'Implemented by the game, but not currently offered among the traits in room settings.'),
  trait('Mystic', 4, 'Magic sources provide additional income.', [
    'Each Magic source held by this active race adds 1 coin at scoring. The bonus stops in decline.',
    'Mystic does not itself produce Fireballs or grant a special attack.',
  ], 'underground', ['Magic']),
  trait('Peace-loving', 4, 'Earn a reward for avoiding active opponents.', [
    'Gain 3 extra coins when your active race scores a turn without conquering an active enemy region.',
    'Conquering empty land or declined races does not remove the bonus. It stops in decline.',
  ], 'packTwo'),
  trait('Pillaging', 5, 'Conquering occupied territory earns an immediate coin.', [
    'Every successful conquest of a non-empty region adds 1 coin from the supply.',
    'Active and declined occupants both qualify. Empty land does not, and the defender does not pay this bonus.',
  ], 'base'),
  trait('Quarreling', 5, 'Separate groups of territory each earn a bonus.', [
    'Gain 1 extra coin per disconnected group of regions held by the active race.',
    'An isolated region counts as a group. Connecting two groups reduces the group count. The bonus ends in decline.',
  ], 'underground'),
  trait('Racketeering', 5, 'Profit when opponents buy expensive race combinations.', [
    'While active, gain coins equal to the number of combinations another player skips when choosing a new race.',
    'After decline, the selection bonus refunds your own skipped-combination cost.',
  ], 'sky', [], 'The refund currently remains registered after the next selection, so later picks can also be refunded. The official rule makes only the next combination free.'),
  trait('Ransacking', 3, 'Take a coin from an active region’s defeated owner.', [
    'A successful conquest of an active region transfers 1 coin from its owner to you.',
    'Empty regions and declined races do not trigger the transfer. It is separate from normal regional scoring.',
  ], 'packTwo'),
  trait('Reborn', 4, 'Turn a declined empire into footholds for its successor.', [
    'Once Reborn is in decline and you have an active race, receive 2 Phoenix actions at the start of your turn.',
    'During the opening decline/abandonment phase, use each Phoenix on a Reborn region to replace its old troops with 1 token of your active race.',
  ], 'underground', ['Phoenix']),
  trait('Royal', 5, 'A Queen protects a region, including in decline.', [
    'Place the Queen in one of your race’s regions to make it immune to conquest.',
    'While active, recover and reposition her during your turn. In decline she stays fixed and continues protecting that region.',
  ], 'underground', ['Queen']),
  trait('Seafaring', 5, 'Occupy water that ordinary races cannot enter.', [
    'Water regions become legal targets. Ordinary conquest costs and adjacency requirements still apply.',
    'Occupied water can remain held and score after the race declines.',
  ], 'base'),
  trait('Shield', 3, 'Generate Mushroom Shields from forests.', [
    'At redeployment, each occupied Forest produces 1 Mushroom Shield. Place them in your regions; several can share a region.',
    'Shields are discarded when their region is cleared.',
  ], 'underground', ['Mushroom Shield'], 'Current limitation: placed shields do not add their intended defense. The official rule grants +1 defense per shield, including in decline.'),
  trait('Soul-touch', 4, 'Bring a previous declined race back into action.', [
    'When Soul-touch declines, your most recent declined race becomes active if one exists. Its surviving regions remain, its army is replenished toward its original size, and initial special pieces return.',
    'Without an earlier declined race, no revival occurs.',
  ], 'packOne', [], 'This game queues the revived race’s turn immediately after the decline sequence. The tabletop description activates it on your next turn.'),
  trait('Spirit', 4, 'Keep this declined race alongside later empires.', [
    'A Spirit race remains on the board when another of your races enters decline.',
    'Its surviving regions keep normal declined scoring. Opponents can still conquer them.',
  ], 'base'),
  trait('Stone', 4, 'Mountains provide extra income.', [
    'Each Mountain region held by the active race earns 1 additional coin. Normal mountain defense is unchanged.',
    'The bonus stops in decline.',
  ], 'underground'),
  trait('Stout', 4, 'Conquer first, then decline during the same turn.', [
    'You may choose decline during redeployment instead of spending your next whole turn on it.',
    'The game preserves the difference between your active score and the declined position’s score for that turn, retaining your active-turn income.',
  ], 'base'),
  trait('Swamp', 4, 'Swamps are worth more to an active race.', [
    'Each occupied Swamp adds 1 coin while this race is active. The bonus ends in decline.',
  ], 'base'),
  trait('Thieving', 4, 'Take coins from neighboring active opponents.', [
    'At scoring, take 1 coin from each different opponent whose active territory borders a region held by this race.',
    'Multiple borders with the same opponent still produce only one payment.',
  ], 'underground', [], 'The current game also applies this effect to declined Thieving troops, unlike the usual active-only trait rule.'),
  trait('Tomb', 5, 'Keep larger defensive stacks after decline.', [
    'Keep the race-token stacks when declining instead of reducing every region to one token. A final redeployment follows.',
    'Later defeats can return survivors for redeployment in remaining regions. Tomb does not allow ordinary conquests in decline.',
  ], 'underground'),
  trait('Underworld', 5, 'Use cave regions as a conquest network.', [
    'Cave regions cost 1 fewer race token to conquer.',
    'Once the race occupies a Cave, it can reach another Cave without a chain of neighboring territory. Other defense and access rules still apply.',
  ], 'base', ['Cave']),
  trait('Vampire', 5, 'Replace a lone active enemy token with your own.', [
    'Use the Vampire action on a reachable active region with one enemy race token. Your race must already occupy the board; protected regions may reject conversion.',
    'Replace the opponent’s token with 1 token of your race. The Vampire action marker refreshes each active turn.',
  ], 'underground', ['Vampire'], 'The game supplies one conversion per turn in total. The official trait allows one conversion per opponent per turn.'),
  trait('Vanishing', 5, 'Trade the whole empire for an immediate payout.', [
    'When declining, remove this race from all its regions and receive 2 coins per cleared region.',
    'The empty regions provide no later declined-race income.',
  ], 'underground'),
  trait('Wealthy', 4, 'Receive a one-time treasury boost.', [
    'The first time this trait scores, gain 7 additional coins. The payment is marked as used and never repeats on later turns.',
  ], 'base'),
  trait('Wise', 4, 'Earn a bonus from a surviving declined empire.', [
    'After decline, gain 2 extra coins each turn while this race still occupies at least one region.',
    'There is no Wise bonus while active or after its last region disappears.',
  ], 'underground'),
  trait('Zeppelined', 5, 'Launch five airborne attacks with a crash risk.', [
    'Select a Zeppelin to attack accessible terrain without ordinary adjacency. At least 1 race token must remain in hand.',
    'A positive die roll reduces the troop requirement. A blank loses an attacking token and leaves an immune Burning Zeppelin region.',
    'Zeppelin markers return next turn. A Zeppelin attack cannot use an additional ordinary reinforcement roll.',
  ], 'sky', ['Zeppelin', 'Burning Zeppelin']),
  trait('Adventurous', 4, 'Earn income from controlled special objects.', [
    'The current game grants 1 extra coin for each object you control in its relic/place collection.',
    'Holding special locations and relics can therefore add income alongside ordinary regions.',
  ], 'underground', [], 'This game counts controlled relics as well as places. The official bonus counts occupied regions containing a Popular Place.'),
];
