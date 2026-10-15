import type { WikiArticle } from './wikiTypes';

const baseRules = {
  label: 'Small World rulebook',
  url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7901-SW-rules-EN-2018_lowres.pdf',
};
const undergroundRules = {
  label: 'Small World Underground rulebook',
  url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/7909-SWU-Rules-EN.pdf',
};
const skyRules = {
  label: 'Sky Islands rules',
  url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/sw_si_en.pdf',
};
const packOneRules = {
  label: 'Power Pack 1 rules',
  url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/sw_c1_en.pdf',
};
const packTwoRules = {
  label: 'Power Pack 2 rules',
  url: 'https://cdn.svc.asmodee.net/production-daysofwonder/uploads/2024/07/720024-SWC2-Helper-EN.pdf',
};

function race(
  name: string,
  count: number,
  summary: string,
  details: string[],
  source: WikiArticle['sources'][number],
  markers: string[] = [],
  note?: string,
): WikiArticle {
  return {
    id: `race:${name}`,
    kind: 'race',
    name,
    summary,
    details: [`Starts with ${count} race tokens, plus the tokens supplied by its trait.`, ...details],
    pieces: [name, ...markers],
    sources: [source],
    ...(note ? { note } : {}),
  };
}

// These descriptions follow this game's RaceMap, Tribe and TileModifiers behavior.
// Publisher rulebooks provide background; differences that affect play are noted.
// Keep names and piece basenames identical to the game and public/stacks assets.
export const raceArticles: WikiArticle[] = [
  race('Amazons', 10, 'Four extra attackers must return to reserve before scoring.', [
    'The starting total already includes four temporary Amazons. They can help conquer regions, but cannot remain on the board at the end of your turn.',
    'During redeployment, return at least four Amazons to reserve. If fewer than four survive altogether, return all survivors. You may have to abandon a region to meet this requirement.',
    'The reserved tokens are available on your next turn. Decline follows the usual one-token-per-region rule.',
  ], baseRules),
  race('Barbarians', 9, 'A large army that cannot rearrange its troops after conquest.', [
    'Barbarian race tokens cannot be moved into or out of regions during redeployment. Plan each conquest with the final defenses in mind.',
    'Unused race tokens stay in reserve for your next turn. Special pieces follow their own rules.',
  ], packOneRules),
  race('Cultists', 5, 'The Great Ancient protects one region and supports nearby attacks.', [
    'Your Great Ancient accompanies the first conquest. Its region cannot be conquered while the marker remains there.',
    'A target bordering the Great Ancient costs one fewer Cultist to conquer, with at least one race token still required.',
    'At the beginning of a later turn, move the Great Ancient once between your regions if desired. It is removed when the race declines.',
  ], undergroundRules, ['Great Ancient']),
  race('Drakons', 6, 'Turn race tokens into three powerful, reusable dragon attacks.', [
    'Select a Drakon\'s Dragon to conquer using two Drakons and one dragon marker, regardless of ordinary defense strength. One Drakon is consumed; the other occupies the region.',
    'A dragon protects its region from conquest. The three dragon markers return to reserve at the start of your next turn.',
    'Existing immunity still blocks a dragon attack. Normal movement restrictions apply.',
  ], skyRules, ["Drakon's Dragon"]),
  race('Drows', 4, 'Earn more from regions separated from your other Drows.', [
    'Each Drow region earns one extra coin if none of its neighboring regions contains your Drows.',
    'Enemy neighbors do not cancel this bonus in the current game. The bonus also remains in decline.',
  ], undergroundRules, [], 'This differs from the printed Drow rule, which rewards avoiding other races, rather than avoiding neighboring Drows.'),
  race('Dwarves', 4, 'Mine regions keep earning extra coins, even in decline.', [
    'Each occupied region with a Mine symbol adds one coin to your score.',
    'This race bonus remains after decline, so a surviving Dwarf on a Mine can keep earning two coins before other modifiers.',
  ], baseRules, ['Mine']),
  race('Elves', 6, 'Recover the token normally lost when an active region falls.', [
    'When an active Elf region is conquered, the race returns the usual casualty to your reserve along with the surviving troops.',
    'This recovery stops in decline. A conquered declined region loses its remaining Elf normally.',
  ], baseRules, [], 'This implementation applies the recovery to special conquests too; printed rules contain exceptions, including Sorcerer conversion.'),
  race('Escargots', 12, 'Hold territory until your next turn to earn its basic income.', [
    'While active, your territory income is based on the regions held at the beginning of the turn. Newly conquered territory provides no basic region income that turn.',
    'The game records that starting income and adds it when scoring. Trait bonuses are calculated separately.',
    'Your first turn usually has no starting territory income. In decline, normal region scoring resumes.',
  ], skyRules),
  race('Fauns', 5, 'Grow through conquests of active races without causing their usual casualty.', [
    'At redeployment, gain one Faun for every two active regions conquered this turn, rounding down.',
    'An active defender loses no race token to the ordinary casualty when attacked by Fauns. Attacks against declined races use the normal casualty rule.',
  ], packTwoRules, [], 'The printed rule grants one new token per active conquest to both sides. This game instead uses the growth and casualty rules described above.'),
  race('Flames', 4, 'A Volcano creates an entry point and very cheap neighboring conquests.', [
    'Place your Volcano on an Abyssal Chasm before entering the board. Your first Flames conquest must border it.',
    'A region directly beside the Volcano costs the minimum number of Flames to conquer, normally one. Existing immunity still applies.',
    'The Volcano returns to reserve at the beginning of each turn so you can place it again.',
  ], undergroundRules, ['Volcano'], 'This game discounts only direct Volcano neighbors. The printed rule instead treats regions connected through Flames territory as empty for conquest costs.'),
  race('Ghouls', 5, 'Keep a fighting army after going into decline.', [
    'Decline preserves your Ghoul troops instead of reducing every region to one token.',
    'On later turns, select Ghoul tokens to conquer and redeploy even while you have a different active race.',
    'Declined Ghouls get at most one final reinforcement-die attempt per turn. Their regions continue contributing to your score.',
  ], baseRules, [], 'The tabletop rule calls for Ghoul conquests before the active race. This version exposes their tokens within the shared turn controls.'),
  race('Giants', 6, 'Owned Mountains make neighboring regions easier to conquer.', [
    'Conquering a region next to a Mountain occupied by your Giants costs one fewer Giant.',
    'The target can have any terrain. Several qualifying Mountains do not multiply this race discount, and at least one token is required.',
  ], baseRules),
  race('Gnomes', 6, 'Force attackers to pay ordinary attack costs.', [
    'When Gnomes defend, the game calculates the attacking race-token cost without the attacker\'s usual race and trait discounts.',
    'Terrain and defending tokens still contribute to defense. The attacker\'s access and adjacency checks still run.',
  ], undergroundRules, [], 'The printed immunity covers a broader range of powers and indirect effects. This implementation specifically replaces attack-cost calculations and keeps this defense in decline.'),
  race('Goblins', 6, 'Conquer declined races with fewer troops.', [
    'A region occupied by a declined race costs one fewer Goblin to conquer.',
    'The discount does not apply to an empty region or an active race. At least one token is always needed.',
  ], packTwoRules),
  race('Halflings', 6, 'Enter away from the border and protect the first two conquests.', [
    'For your initial arrival, choose an accessible region without the usual border restriction.',
    'The first two conquered regions each receive a Hole. Those regions cannot be conquered while their Hole remains.',
    'Abandoning a protected region removes its Hole. Decline removes both Holes, and they are not replenished on later turns.',
  ], baseRules, ['Hole']),
  race('Humans', 5, 'Earn extra coins from farmland.', [
    'While active, each Field region occupied by Humans adds one coin to your score.',
    'The bonus ends in decline. Fields are the farmland terrain used by this game.',
  ], baseRules),
  race('Ice Witches', 5, 'Magic regions supply Winter markers that alter defense and income.', [
    'At redeployment, receive one Winter marker for each occupied region with a Magic symbol.',
    'Place Winter in your own regions, with at most one marker in each. It adds one defense and subtracts one coin from that region\'s income.',
    'The markers and their scoring penalty are removed when the Ice Witches decline.',
  ], packOneRules, ['Winter', 'Magic'], 'Unlike the printed rule, this game only accepts Winter placement in your own regions and applies the income penalty there too. A Winter defense modifier can remain after its marker is removed.'),
  race('Iron Dwarves', 7, 'Mine regions supply Hammers for later attacks.', [
    'At redeployment, gain one Hammer for each occupied Mine region. Keep unused Hammers in reserve for later turns.',
    'Selecting a Hammer for conquest consumes one Hammer and reduces the race-token cost by one. At least one Iron Dwarf is still required.',
    'Hammers do not remain on conquered regions or add defense. Unused Hammers are removed at decline.',
  ], undergroundRules, ['Hammer', 'Mine'], 'Printed Silver Hammers are reusable conquest troops. Here each selected Hammer is spent on one discounted attack.'),
  race('Khans', 5, 'Favor Fields and Hills over other terrain.', [
    'While active, each occupied Field or Hill earns one extra coin. Every other occupied terrain earns one fewer coin.',
    'A region\'s final income cannot fall below zero. Both the bonus and the penalty stop in decline.',
  ], skyRules),
  race('Kobolds', 11, 'An abundant race that must travel in pairs.', [
    'Active conquests and deployments require at least two Kobolds in each occupied region. A single spare Kobold cannot conquer by itself.',
    'When they decline, Kobolds leave one token in each region, following the normal decline rule.',
  ], packTwoRules),
  race('Krakens', 5, 'Keep control of River regions after the other races leave.', [
    'At redeployment, Krakens retain a race token in each occupied River region rather than emptying it.',
    'These occupied Rivers score as territory and remain occupied in decline. Other races normally have to conquer them to pass through.',
  ], undergroundRules, [], 'Named Kraken in the printed rules. The game returns River troops during redeployment, then restores one Kraken to each held River.'),
  race('Leprechauns', 6, 'Leave Cauldrons to collect later, or lose them to an attacker.', [
    'You start with 16 Cauldrons. During redeployment, place at most one in each of your regions.',
    'A Cauldron still held at the beginning of your next turn becomes one coin. An opponent conquering it first receives the coin instead.',
    'Collected Cauldrons are spent, so the initial supply is limited.',
  ], packOneRules, ['Cauldron'], 'Cauldrons are the Pot of Gold markers in the printed rules. This game also prevents ordinary Leprechaun race-token placement during redeployment.'),
  race('Liches', 4, 'Charge attackers for conquering your declined regions.', [
    'While active, Liches have no extra race effect.',
    'After decline, each successful conquest of a Lich region transfers one coin from the attacker to the Lich owner.',
  ], undergroundRules, [], 'The printed rule prevents a penniless opponent from attacking. This implementation applies the transfer without that additional check.'),
  race('Lizardmen', 7, 'Reach distant land through River connections.', [
    'The game extends adjacency through chains of empty River regions, letting Lizardmen reach territory without occupying every intervening River.',
    'A River you do conquer is emptied during normal redeployment; move those troops back onto land.',
  ], undergroundRules, [], 'The printed rule permits crossing occupied Rivers too. Current reachability only follows empty River chains, and can also accept some targets two regions away.'),
  race('Mudmen', 5, 'Swamps grow additional troops every turn.', [
    'At redeployment, each Swamp occupied by Mudmen supplies one new Mudman in reserve.',
    'Use those new troops to reinforce your regions. Swamps are this game\'s equivalent of Underground Mudpools.',
  ], undergroundRules),
  race('Mummies', 10, 'A large starting army pays extra for every conquest.', [
    'Each conquest costs one more Mummy than it otherwise would.',
    'Other applicable discounts can offset the penalty. Their advantage is the large initial supply rather than efficient attacks.',
  ], undergroundRules),
  race('Nomads', 6, 'Earn a coin by leaving a region behind.', [
    'Abandoning a Nomad region leaves a Coin marker and records that region for the current turn.',
    'You cannot reconquer it in that turn. At scoring, collect one extra coin for each recorded region and remove the markers.',
  ], packTwoRules, ['Coin']),
  race('Ogres', 5, 'Every conquest needs one fewer race token.', [
    'Reduce the race-token cost of each conquest by one.',
    'This combines with other applicable discounts, but cannot reduce the cost below one token.',
  ], undergroundRules),
  race('Orcs', 5, 'Earn extra coins for taking occupied territory.', [
    'Each successful conquest of a region occupied by another tribe gives one extra coin.',
    'Active and declined defenders both count. Empty terrain does not trigger the bonus.',
  ], baseRules),
  race('Pixies', 11, 'Many attackers, but only one defender per region.', [
    'At the start of redeployment, surplus Pixies return to reserve, leaving one in each region.',
    'You cannot place extra Pixie race tokens back into regions during redeployment. The reserve becomes available for the next turn\'s conquests.',
  ], packOneRules),
  race('Priestesses', 4, 'Decline into one tower whose height determines its income.', [
    'When Priestesses decline, use the Decline piece to choose one of their regions for the tower.',
    'The remaining Priestess from each other occupied region joins that region. All those other regions are abandoned.',
    'The tower earns one coin per Priestess token each turn. Its stacked tokens defend normally; it is not immune to conquest.',
  ], packTwoRules, ['Decline']),
  race('Ratmen', 8, 'More starting troops, with no extra race ability.', [
    'Ratmen use the ordinary conquest, redeployment and decline rules.',
    'Their larger initial army is their racial advantage. Their paired trait supplies any additional ability.',
  ], baseRules),
  race('Scarecrows', 12, 'Numerous troops make tempting targets for rival players.', [
    'An attacker receives one extra coin when conquering a Scarecrow region.',
    'The large starting army helps compensate for this reward offered to opponents.',
  ], skyRules, [], 'The reward also applies to declined Scarecrows in this game; the printed rule limits it to active Scarecrows.'),
  race('Scavengers', 6, 'Share conquered regions with the declined troops underneath.', [
    'When conquering a declined region, leave its old tokens beneath the Scavengers instead of removing them.',
    'The old race remains present for scoring, and its tokens and applicable defenses help defend the shared region.',
    'If Scavengers abandon the region, its underlying tribe takes over again. When Scavengers decline, the underlying pieces are removed from their regions.',
  ], skyRules),
  race('Shadow Mimes', 7, 'Swap your trait with one in the race market.', [
    'Use the Mime piece on an available race-and-trait combination to exchange its trait with yours.',
    'The other race stays in the market, now with your old trait. The exchange consumes the Mime piece and does not exchange coins.',
  ], undergroundRules, ['Mime']),
  race('Shrooms', 5, 'Earn extra income from Forests.', [
    'While active, every Forest region occupied by Shrooms adds one coin to your score.',
    'This game uses Forests for the Mushroom Forest terrain in the Underground rules. The bonus stops in decline.',
  ], undergroundRules),
  race('Shrubmen', 6, 'Forest homes remain protected, even in decline.', [
    'A Forest region occupied by Shrubmen cannot be conquered.',
    'This protection remains after decline. Shrubmen on other terrain defend normally.',
  ], packTwoRules),
  race('Skags', 6, 'Hide coin rewards and dangerous ambushes in conquered territory.', [
    'New conquests receive hidden Loot while the supply lasts. The ten markers contain three Skag Attacks and seven coin values: 0, 0, 1, 1, 2, 2 and 3.',
    'A normal Loot marker rewards the next conqueror. A Skag Attack instead costs that attacker one race token and blocks further conquests of the region for the rest of the turn.',
    'Only the Skag owner receives the initial message revealing a marker. Selecting Loot lets that owner check it again.',
    'The game settles remaining loot values when Skags decline or the game ends.',
  ], packOneRules, ['Loot', 'Skag Attack'], 'Loot settlement in this version scans stored values across the board; it can differ from the printed rule that pays only unrevealed Loot still in your regions.'),
  race('Skeletons', 6, 'Occupied conquests replenish your army.', [
    'At redeployment, gain one new Skeleton for every two occupied regions conquered during this turn, rounding down.',
    'Conquests of active or declined races count. Empty regions do not, and an unmatched conquest is not carried over to the next turn.',
  ], baseRules),
  race('Slingmen', 5, 'Skip an intervening region and earn a coin for the distant conquest.', [
    'When ordinary adjacency does not allow a conquest, Slingmen can reach a target two region connections away from their territory.',
    'A successful conquest made using this extended reach earns one extra coin. Ordinary adjacent conquests earn no race bonus.',
  ], packOneRules, [], 'The printed rules distinguish jumping over a Lake from jumping over Seas. Current reachability follows map connections without that terrain distinction.'),
  race('Sorcerers', 5, 'Replace a lone active defender using the Staff.', [
    'You receive one Staff each active turn. Select it to convert a region containing a single active enemy race token into one Sorcerer.',
    'Your Sorcerers must already be on the board. The target must be reachable, accessible and not protected by immunity.',
    'Using the Staff consumes it for the turn. Regions held by declined races or multiple enemy race tokens cannot be converted.',
  ], baseRules, ['Staff'], 'This version allows one conversion in total per turn. The printed rule allows one per opponent per turn.'),
  race('Spiderines', 7, 'Use Chasm borders as entrances and attack routes.', [
    'A region bordering an Abyssal Chasm counts as reachable for Spiderines, even when it is not next to their existing territory.',
    'This also permits a first conquest beside a Chasm. The target must still be terrain that your race can occupy.',
  ], undergroundRules),
  race('Storm Giants', 6, 'Two Storms let you seize distant Mountains with one troop.', [
    'Receive two Storm markers at the beginning of each active turn. Unused Storms do not accumulate.',
    'Select Storm to conquer any unprotected Mountain with one Storm Giant and one Storm marker. Ordinary defender numbers do not increase the cost.',
    'Storm Giants also have direct access to Sky Island regions, including for their first conquest.',
  ], skyRules, ['Storm'], 'This implementation grants the Sky Island adjacency exception throughout play, not only on the initial conquest.'),
  race('Tritons', 6, 'Conquer coastal territory at a discount.', [
    'A region next to Water costs one fewer Triton to conquer.',
    'This refers to Water terrain, such as Seas and Lakes. The discount does not itself let Tritons occupy Water, and at least one token is still required.',
  ], baseRules),
  race('Trolls', 5, 'Lairs strengthen each region and survive decline.', [
    'Every newly conquered Troll region receives a Lair and has one additional defense.',
    'The extra defense remains in decline. A Lair disappears when its region is abandoned or conquered.',
  ], baseRules, ['Lair']),
  race('Wendigos', 6, 'Clear a Forest anywhere and gain another race token.', [
    'Once per active turn, select the Power piece on a Forest. The Forest can be anywhere on the map, but immunity still protects its occupant.',
    'The action clears the defending race, applies its normal return-and-loss handling, and adds one Wendigo to your reserve.',
    'It does not occupy the Forest. Conquer the emptied region separately if you want it.',
  ], skyRules, ['Power'], 'This version uses each defender\'s normal casualty handling; printed rules specifically require an Elf casualty from this ability.'),
  race('White Ladies', 2, 'A small race whose declined regions cannot be conquered.', [
    'While active, White Ladies conquer and defend normally.',
    'After decline, their occupied regions become immune to conquest. Their remaining tokens continue providing region income.',
  ], packTwoRules),
  race("Will-o'-Wisps", 6, 'Roll for a cheaper conquest of a Magic region.', [
    'When attacking a region with a Magic symbol, the game rolls the reinforcement die and reduces the race-token cost by the result.',
    'This extra roll applies when the attack is already affordable without a final-conquest roll. At least one token is required.',
  ], undergroundRules, ['Magic'], 'The printed rule also covers neighbors of your Mystic Crystal regions and lets you roll before committing the attack. This version only checks the target\'s Magic symbol.'),
  race('Witch Doctors', 6, 'Roll for replacement troops when an active region is conquered.', [
    'After an active Witch Doctor region falls, roll the reinforcement die and add that many replacement tokens to reserve for an ordinary one-token casualty.',
    'When an attack causes a different casualty count, the replacement calculation changes accordingly. The ability stops in decline.',
  ], packOneRules, [], 'This race is named Pygmies in the linked older rulebook. The game calls it Witch Doctors and rolls once per conquered region.'),
  race('Wizards', 5, 'Earn extra coins from Magic regions.', [
    'While active, each occupied region with a Magic symbol adds one coin to your score.',
    'The symbol can appear on different terrain types. The bonus stops in decline.',
  ], baseRules, ['Magic']),
];
