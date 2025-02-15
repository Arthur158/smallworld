package gamestate

import (
	"fmt"
)

func mergeMaps(map1, map2 map[string]*Tile) map[string]*Tile {
    merged := make(map[string]*Tile)

    // Copy all elements from map1
    for k, v := range map1 {
        merged[k] = v
    }

    // Copy all elements from map2 (overwrites if key exists)
    for k, v := range map2 {
        merged[k] = v
    }

    return merged
}
// Map1 defines tiles and their adjacency relationships
func Map3(gs *GameState) map[string]*Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0":  {Id: "0", Biome: Swamp, Attributes: []Attribute{}},
        "1":  {Id: "1", Biome: Forest, Attributes: []Attribute{Cave}},
        "2":  {Id: "2", Biome: Hill, Attributes: []Attribute{}},
        "3":  {Id: "3", Biome: Swamp, Attributes: []Attribute{Magic}},
        "4":  {Id: "4", Biome: Field, Attributes: []Attribute{}},
        "5":  {Id: "5", Biome: Mountain, Attributes: []Attribute{Cave}},
        "6":  {Id: "6", Biome: Forest, Attributes: []Attribute{}, IsEdge: true},
        "7":  {Id: "7", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "8":  {Id: "8", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "9":  {Id: "9", Biome: Hill, Attributes: []Attribute{}, IsEdge: true},
        "10": {Id: "10", Biome: Field, Attributes: []Attribute{Magic}, IsEdge: true},
        "11": {Id: "11", Biome: Forest, Attributes: []Attribute{Mine}, IsEdge: true},
        "12": {Id: "12", Biome: Mountain, Attributes: []Attribute{Cave, Mine}, IsEdge: true},
        "13": {Id: "13", Biome: Field, Attributes: []Attribute{Magic}, IsEdge: true},
        "14": {Id: "14", Biome: Forest, Attributes: []Attribute{Mine}, IsEdge: true},
        "15": {Id: "15", Biome: Swamp, Attributes: []Attribute{}},
        "16": {Id: "16", Biome: Hill, Attributes: []Attribute{Magic}},
        "17": {Id: "17", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "18": {Id: "18", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "19": {Id: "19", Biome: Swamp, Attributes: []Attribute{Cave}, IsEdge: true},
        "20": {Id: "20", Biome: Mountain, Attributes: []Attribute{Mine}},
        "21": {Id: "21", Biome: Field, Attributes: []Attribute{}},
        "22": {Id: "22", Biome: Forest, Attributes: []Attribute{}, IsEdge: true},
        "23": {Id: "23", Biome: Swamp, Attributes: []Attribute{Mine}, IsEdge: true},
        "24": {Id: "24", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "25": {Id: "25", Biome: Hill, Attributes: []Attribute{Magic}},
        "26": {Id: "26", Biome: Hill, Attributes: []Attribute{Cave}, IsEdge: true},
        "27": {Id: "27", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "28": {Id: "28", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "29": {Id: "29", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
    }

    for key := range(tileMap) {
        tileMap[key].ModifierDefenses = make(map[string]func(int, error) (int, error))
        tileMap[key].ModifierPoints = make(map[string]func(int) (int))
    }

        // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["0"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["15"], tileMap["16"], tileMap["28"], tileMap["1"], tileMap["10"],
    }

    tileMap["1"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["8"], tileMap["10"], tileMap["12"], tileMap["28"], tileMap["7"], tileMap["2"],
    }

    tileMap["2"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["6"], tileMap["1"], tileMap["7"], tileMap["28"],
    }

    tileMap["3"].AdjacentTiles = []*Tile{
        tileMap["11"], tileMap["6"], tileMap["2"],  tileMap["5"],tileMap["4"], tileMap["28"],
    }

    tileMap["4"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["5"], tileMap["25"], tileMap["26"], tileMap["11"],
    }

    tileMap["5"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["4"], tileMap["25"], tileMap["21"],  tileMap["20"], tileMap["28"],
    }

    tileMap["6"].AdjacentTiles = []*Tile{
        tileMap["2"], tileMap["7"], tileMap["3"], tileMap["11"],
    }

    tileMap["7"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["2"], tileMap["6"], tileMap["1"],
    }

    tileMap["8"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["9"], tileMap["10"], tileMap["1"],
    }

    tileMap["9"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["10"], // ajustez si besoin
    }

    tileMap["10"].AdjacentTiles = []*Tile{
        tileMap["1"], tileMap["8"], tileMap["9"], tileMap["12"],tileMap["0"],
    }

    tileMap["11"].AdjacentTiles = []*Tile{
        tileMap["6"], tileMap["3"], tileMap["4"], tileMap["26"],tileMap["27"],
    }

    tileMap["12"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["13"], tileMap["10"], tileMap["15"], tileMap["29"], 
    }

    tileMap["13"].AdjacentTiles = []*Tile{
        tileMap["29"], tileMap["14"], tileMap["15"], tileMap["12"], 
    }

    tileMap["14"].AdjacentTiles = []*Tile{
        tileMap["13"], tileMap["17"], tileMap["15"], tileMap["29"],
    }

    tileMap["15"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["13"], tileMap["14"], tileMap["17"], tileMap["16"], tileMap["0"],
    }

    tileMap["16"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["15"], tileMap["17"], tileMap["18"], tileMap["20"], tileMap["28"],
    }

    tileMap["17"].AdjacentTiles = []*Tile{
        tileMap["14"], tileMap["15"], tileMap["16"], tileMap["18"],
    }

    tileMap["18"].AdjacentTiles = []*Tile{
        tileMap["16"], tileMap["17"], tileMap["19"], tileMap["20"],
    }

    tileMap["19"].AdjacentTiles = []*Tile{
        tileMap["18"], tileMap["20"], tileMap["21"], tileMap["22"],
    }

    tileMap["20"].AdjacentTiles = []*Tile{
        tileMap["16"], tileMap["19"], tileMap["21"], tileMap["5"], tileMap["28"], tileMap["18"],
    }

    tileMap["21"].AdjacentTiles = []*Tile{
        tileMap["19"], tileMap["20"], tileMap["23"], tileMap["22"], tileMap["5"], tileMap["25"],
    }

    tileMap["22"].AdjacentTiles = []*Tile{
        tileMap["19"], tileMap["23"], tileMap["21"],
    }

    tileMap["23"].AdjacentTiles = []*Tile{
        tileMap["21"], tileMap["22"], tileMap["24"], tileMap["25"],
    }

    tileMap["24"].AdjacentTiles = []*Tile{
        tileMap["23"], tileMap["25"], tileMap["26"], tileMap["27"],
    }

    tileMap["25"].AdjacentTiles = []*Tile{
        tileMap["24"], tileMap["21"], tileMap["5"], tileMap["4"], tileMap["23"], tileMap["26"],
    }

    tileMap["26"].AdjacentTiles = []*Tile{
        tileMap["25"], tileMap["4"], tileMap["24"], tileMap["27"], tileMap["11"],
    }

    tileMap["27"].AdjacentTiles = []*Tile{
        tileMap["26"], tileMap["11"], tileMap["24"],
    }

    tileMap["28"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["1"], tileMap["2"], tileMap["3"], tileMap["5"], tileMap["20"], tileMap["16"],
    }

    tileMap["29"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["13"], tileMap["14"], // selon la frontière visible
    }

    // Step 3: Convert the map of pointers to a map of values
    result := make(map[string]*Tile, len(tileMap))
    for id, tile := range tileMap {
        result[id] = tile // Dereference pointer to get a Tile value
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{ "22", "21", "4", "3", "11", "26", "1", "13", "16", "0"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }


    return result
}

func Map2(gs *GameState) map[string]*Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0":  {Id: "0", Biome: Swamp, Attributes: []Attribute{Magic}, IsEdge: true},
        "1":  {Id: "1", Biome: Hill, Attributes: []Attribute{Cave}, IsEdge: true},
        "2":  {Id: "2", Biome: Swamp, Attributes: []Attribute{Mine}, IsEdge: true},
        "3":  {Id: "3", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "4":  {Id: "4", Biome: Swamp, Attributes: []Attribute{}, IsEdge: true},
        "5":  {Id: "5", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "6":  {Id: "6", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "7":  {Id: "7", Biome: Forest, Attributes: []Attribute{}},
        "8":  {Id: "8", Biome: Field, Attributes: []Attribute{Magic}},
        "9":  {Id: "9", Biome: Hill, Attributes: []Attribute{Cave}},
        "10": {Id: "10", Biome: Forest, Attributes: []Attribute{}, IsEdge: true},
        "11": {Id: "11", Biome: Mountain, Attributes: []Attribute{Mine}, IsEdge: true},
        "12": {Id: "12", Biome: Mountain, Attributes: []Attribute{Cave, Mine}, IsEdge: true},
        "13": {Id: "13", Biome: Hill, Attributes: []Attribute{}},
        "14": {Id: "14", Biome: Water, Attributes: []Attribute{}},
        "15": {Id: "15", Biome: Mountain, Attributes: []Attribute{}},
        "16": {Id: "16", Biome: Field, Attributes: []Attribute{}},
        "17": {Id: "17", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: true},
        "18": {Id: "18", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "19": {Id: "19", Biome: Field, Attributes: []Attribute{Magic}, IsEdge: true},
        "20": {Id: "20", Biome: Forest, Attributes: []Attribute{Mine}, IsEdge: true},
        "21": {Id: "21", Biome: Swamp, Attributes: []Attribute{Cave}, IsEdge: true},
        "22": {Id: "22", Biome: Hill, Attributes: []Attribute{}, IsEdge: true},
    }

    for key := range(tileMap) {
        tileMap[key].ModifierDefenses = make(map[string]func(int, error) (int, error))
        tileMap[key].ModifierPoints = make(map[string]func(int) (int))
    }
        // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["0"].AdjacentTiles = []*Tile{
        tileMap["6"], tileMap["1"], 
    }

    tileMap["1"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["6"], tileMap["7"], tileMap["2"],
    }

    tileMap["2"].AdjacentTiles = []*Tile{
        tileMap["1"], tileMap["7"], tileMap["8"], tileMap["3"], 
    }

    tileMap["3"].AdjacentTiles = []*Tile{
        tileMap["2"], tileMap["8"], tileMap["4"],  
    }

    tileMap["4"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["8"], tileMap["9"], tileMap["10"], tileMap["5"],
    }

    tileMap["5"].AdjacentTiles = []*Tile{
        tileMap["11"], tileMap["10"], tileMap["4"], 
    }

    tileMap["6"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["1"], tileMap["7"], tileMap["13"], tileMap["12"],
    }

    tileMap["7"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["13"], tileMap["6"], tileMap["1"], tileMap["14"], tileMap["2"],
    }

    tileMap["8"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["14"], tileMap["15"], tileMap["4"], tileMap["3"], tileMap["2"], tileMap["9"],
    }

    tileMap["9"].AdjacentTiles = []*Tile{
        tileMap["4"], tileMap["8"], tileMap["15"], tileMap["16"], tileMap["17"], tileMap["11"], tileMap["10"],
    }

    tileMap["10"].AdjacentTiles = []*Tile{
        tileMap["5"], tileMap["11"], tileMap["9"],tileMap["4"],
    }

    tileMap["11"].AdjacentTiles = []*Tile{
        tileMap["5"], tileMap["10"], tileMap["9"], tileMap["17"],
    }

    tileMap["12"].AdjacentTiles = []*Tile{
        tileMap["18"], tileMap["19"], tileMap["13"], tileMap["6"], 
    }

    tileMap["13"].AdjacentTiles = []*Tile{
        tileMap["6"], tileMap["12"], tileMap["19"], tileMap["20"], tileMap["14"], tileMap["7"],
    }

    tileMap["14"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["13"], tileMap["20"], tileMap["15"], tileMap["8"],
    }

    tileMap["15"].AdjacentTiles = []*Tile{
        tileMap["9"], tileMap["21"], tileMap["20"], tileMap["16"], tileMap["14"], tileMap["8"],
    }

    tileMap["16"].AdjacentTiles = []*Tile{
        tileMap["21"], tileMap["22"], tileMap["15"], tileMap["9"], tileMap["17"], 
    }

    tileMap["17"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["16"], tileMap["9"], tileMap["11"],
    }

    tileMap["18"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["19"], 
    }

    tileMap["19"].AdjacentTiles = []*Tile{
        tileMap["18"], tileMap["12"], tileMap["13"], tileMap["20"],
    }

    tileMap["20"].AdjacentTiles = []*Tile{
        tileMap["13"], tileMap["19"], tileMap["21"], tileMap["15"], tileMap["14"], 
    }

    tileMap["21"].AdjacentTiles = []*Tile{
        tileMap["15"], tileMap["20"], tileMap["16"], tileMap["22"], 
    }

    tileMap["22"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["16"], tileMap["21"],
    }
    // Step 3: Convert the map of pointers to a map of values
    result := make(map[string]*Tile, len(tileMap))
    for id, tile := range tileMap {
        result[id] = tile // Dereference pointer to get a Tile value
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{ "0", "2", "6", "7", "8", "9", "13", "21", "17"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }


    return result
}

func Map4(gs *GameState) map[string]*Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0":  {Id: "0", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "1":  {Id: "1", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "2":  {Id: "2", Biome: Hill, Attributes: []Attribute{Mine}, IsEdge: true},
        "3":  {Id: "3", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "4":  {Id: "4", Biome: Mountain, Attributes: []Attribute{Mine}, IsEdge: true},
        "5":  {Id: "5", Biome: Forest, Attributes: []Attribute{}, IsEdge: true},
        "6":  {Id: "6", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "7":  {Id: "7", Biome: Swamp, Attributes: []Attribute{Mine}, IsEdge: true},
        "8":  {Id: "8", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: false},
        "9":  {Id: "9", Biome: Swamp, Attributes: []Attribute{Cave}, IsEdge: false},
        "10": {Id: "10", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: false},
        "11": {Id: "11", Biome: Mountain, Attributes: []Attribute{}, IsEdge: false},
        "12": {Id: "12", Biome: Field, Attributes: []Attribute{}, IsEdge: false},
        "13": {Id: "13", Biome: Mountain, Attributes: []Attribute{}, IsEdge: false},
        "14": {Id: "14", Biome: Field, Attributes: []Attribute{}, IsEdge: false},
        "15": {Id: "15", Biome: Field, Attributes: []Attribute{Magic}, IsEdge: true},
        "16": {Id: "16", Biome: Swamp, Attributes: []Attribute{Cave}, IsEdge: true},
        "17": {Id: "17", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "18": {Id: "18", Biome: Hill, Attributes: []Attribute{}, IsEdge: false},
        "19": {Id: "19", Biome: Mountain, Attributes: []Attribute{Cave, Mine}, IsEdge: false},
        "20": {Id: "20", Biome: Water, Attributes: []Attribute{}},
        "21": {Id: "21", Biome: Swamp, Attributes: []Attribute{Mine}},
        "22": {Id: "22", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "23": {Id: "23", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "24": {Id: "24", Biome: Forest, Attributes: []Attribute{}, IsEdge: true},
        "25": {Id: "25", Biome: Swamp, Attributes: []Attribute{}},
        "26": {Id: "26", Biome: Hill, Attributes: []Attribute{Magic}, IsEdge: false},
        "27": {Id: "27", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: false},
        "28": {Id: "28", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "29": {Id: "29", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "30": {Id: "30", Biome: Swamp, Attributes: []Attribute{}, IsEdge: true},
        "31": {Id: "31", Biome: Hill, Attributes: []Attribute{Cave}, IsEdge: true},
        "32": {Id: "32", Biome: Field, Attributes: []Attribute{}, IsEdge: false},
        "33": {Id: "33", Biome: Mountain, Attributes: []Attribute{Mine}, IsEdge: false},
        "34": {Id: "34", Biome: Field, Attributes: []Attribute{Mine}, IsEdge: true},
        "35": {Id: "35", Biome: Forest, Attributes: []Attribute{Cave}, IsEdge: true},
        "36": {Id: "36", Biome: Hill, Attributes: []Attribute{}, IsEdge: true},
        "37": {Id: "37", Biome: Swamp, Attributes: []Attribute{Magic}, IsEdge: true},
        "38": {Id: "38", Biome: Forest, Attributes: []Attribute{Cave}, IsEdge: true},
    }

    for key := range(tileMap) {
        tileMap[key].ModifierDefenses = make(map[string]func(int, error) (int, error))
        tileMap[key].ModifierPoints = make(map[string]func(int) (int))
    }

        // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["0"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["8"], tileMap["1"], tileMap["2"], 
    }

    tileMap["1"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["8"], tileMap["9"], tileMap["2"], 
    }

    tileMap["2"].AdjacentTiles = []*Tile{
        tileMap["1"], tileMap["9"], tileMap["3"], 
    }

    tileMap["3"].AdjacentTiles = []*Tile{
        tileMap["2"], tileMap["9"], tileMap["10"],  tileMap["4"],
    }

    tileMap["4"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["10"], tileMap["12"], tileMap["13"], tileMap["5"],
    }

    tileMap["5"].AdjacentTiles = []*Tile{
        tileMap["4"], tileMap["13"], tileMap["15"], tileMap["6"],  tileMap["20"], tileMap["28"],
    }

    tileMap["6"].AdjacentTiles = []*Tile{
        tileMap["5"], tileMap["15"], 
    }

    tileMap["7"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["0"], tileMap["17"], tileMap["18"],
    }

    tileMap["8"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["18"], tileMap["0"], tileMap["1"],        tileMap["1"], tileMap["9"], tileMap["11"], tileMap["19"],

    }

    tileMap["9"].AdjacentTiles = []*Tile{
        tileMap["2"], tileMap["3"], tileMap["10"], tileMap["12"],        tileMap["1"], tileMap["8"], tileMap["11"], 

    }

    tileMap["10"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["4"], tileMap["9"], 
    }

    tileMap["11"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["9"], tileMap["12"], tileMap["20"],tileMap["19"],
    }

    tileMap["12"].AdjacentTiles = []*Tile{
        tileMap["4"], tileMap["13"], tileMap["11"], tileMap["10"], tileMap["9"], 
        tileMap["14"], tileMap["20"], tileMap["21"], 
    }

    tileMap["13"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["4"], tileMap["5"], tileMap["10"], tileMap["14"], 
        tileMap["15"], 
    }

    tileMap["14"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["13"], tileMap["15"], tileMap["16"], tileMap["21"], tileMap["22"], 
    }

    tileMap["15"].AdjacentTiles = []*Tile{
        tileMap["5"], tileMap["13"], tileMap["14"], tileMap["6"], tileMap["16"],
    }

    tileMap["16"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["15"], tileMap["14"], 
    }

    tileMap["17"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["8"], tileMap["24"], tileMap["23"],
    }

    tileMap["18"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["17"], tileMap["24"], tileMap["25"],
        tileMap["19"], tileMap["8"],
    }

    tileMap["19"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["18"], tileMap["26"], tileMap["25"],
        tileMap["20"], tileMap["11"],
    }

    tileMap["20"].AdjacentTiles = []*Tile{
        tileMap["11"], tileMap["19"], tileMap["26"], tileMap["32"], tileMap["33"], tileMap["27"], tileMap["21"], tileMap["12"],

    }

    tileMap["21"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["20"], tileMap["27"], tileMap["28"], tileMap["22"], tileMap["14"],
    }

    tileMap["22"].AdjacentTiles = []*Tile{
        tileMap["14"], tileMap["16"], tileMap["21"], tileMap["28"], 
    }

    tileMap["23"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["24"], tileMap["30"], tileMap["29"], tileMap["31"], tileMap["34"], tileMap["35"],
    }

    tileMap["24"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["18"], tileMap["31"], tileMap["25"], tileMap["30"], tileMap["23"], 
    }

    tileMap["25"].AdjacentTiles = []*Tile{
        tileMap["18"], tileMap["24"], tileMap["31"], tileMap["32"], tileMap["19"], tileMap["26"],
    }

    tileMap["26"].AdjacentTiles = []*Tile{
        tileMap["19"], tileMap["25"], tileMap["32"], tileMap["20"], 
    }

    tileMap["27"].AdjacentTiles = []*Tile{
        tileMap["21"], tileMap["28"], tileMap["38"], tileMap["33"], tileMap["20"], 
    }

    tileMap["28"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["21"], tileMap["27"], tileMap["38"], 
    }

    tileMap["29"].AdjacentTiles = []*Tile{
        tileMap["30"], tileMap["23"], tileMap["14"], // selon la frontière visible
    }
    tileMap["30"].AdjacentTiles = []*Tile{
        tileMap["29"], tileMap["23"], tileMap["24"], tileMap["31"], 
    }

    tileMap["31"].AdjacentTiles = []*Tile{
        tileMap["24"], tileMap["25"], tileMap["32"], tileMap["34"], tileMap["30"], tileMap["23"], 
    }

    tileMap["32"].AdjacentTiles = []*Tile{
        tileMap["25"], tileMap["26"], tileMap["20"], tileMap["33"], tileMap["36"], tileMap["34"], tileMap["31"],
    }

    tileMap["33"].AdjacentTiles = []*Tile{
        tileMap["20"], tileMap["32"], tileMap["36"], tileMap["37"], tileMap["38"], tileMap["27"], 
    }

    tileMap["34"].AdjacentTiles = []*Tile{
        tileMap["23"], tileMap["32"], tileMap["31"], tileMap["35"], tileMap["36"], 
    }

    tileMap["35"].AdjacentTiles = []*Tile{
        tileMap["23"], tileMap["34"], tileMap["36"], 
    }

    tileMap["36"].AdjacentTiles = []*Tile{
        tileMap["34"], tileMap["35"], tileMap["32"], tileMap["33"], tileMap["37"], 
    }

    tileMap["37"].AdjacentTiles = []*Tile{
        tileMap["36"], tileMap["33"], tileMap["38"], 
    }

    tileMap["38"].AdjacentTiles = []*Tile{
        tileMap["37"], tileMap["33"], tileMap["27"], tileMap["28"], 
    }

    // Step 3: Convert the map of pointers to a map of values
    result := make(map[string]*Tile, len(tileMap))
    for id, tile := range tileMap {
        result[id] = tile // Dereference pointer to get a Tile value
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{"7", "8", "9", "10", "12", "14", "16", "17", "36", "35", "34", "30", "31", "25", "27"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }


    return result
}

func MapIsles2(gs *GameState) map[string]*Tile {
    tileMap := map[string]*Tile{
        "0i":  {Id: "0i", Biome: Swamp, Attributes: []Attribute{}},
        "1i":  {Id: "1i", Biome: Hill, Attributes: []Attribute{Magic}},
        "2i":  {Id: "2i", Biome: Field, Attributes: []Attribute{}},
        "3i":  {Id: "3i", Biome: Mountain, Attributes: []Attribute{Mine}},
        "4i":  {Id: "4i", Biome: Mountain, Attributes: []Attribute{}},
        "5i":  {Id: "5i", Biome: Field, Attributes: []Attribute{Cave}},
        "6i":  {Id: "6i", Biome: Water, Attributes: []Attribute{}},
        "7i":  {Id: "7i", Biome: Swamp, Attributes: []Attribute{Mine}},
        "8i":  {Id: "8i", Biome: Hill, Attributes: []Attribute{Magic}},
        "9i":  {Id: "9i", Biome: Forest, Attributes: []Attribute{}},
    }

    tileMap["0i"].AdjacentTiles = []*Tile{
        tileMap["2i"], tileMap["3i"], tileMap["1i"], tileMap["8i"],
    }

    tileMap["1i"].AdjacentTiles = []*Tile{
        tileMap["0i"], tileMap["3i"], 
    }

    tileMap["2i"].AdjacentTiles = []*Tile{
        tileMap["0i"], tileMap["3i"], tileMap["4i"], 
    }
    tileMap["3i"].AdjacentTiles = []*Tile{
        tileMap["2i"], tileMap["0i"], tileMap["1i"], 
    }
    tileMap["4i"].AdjacentTiles = []*Tile{
        tileMap["2i"], tileMap["5i"], tileMap["6i"], tileMap["8i"], 
    }
    tileMap["5i"].AdjacentTiles = []*Tile{
        tileMap["4i"], tileMap["5i"], tileMap["6i"], 
    }
    tileMap["6i"].AdjacentTiles = []*Tile{
        tileMap["4i"], tileMap["5i"], tileMap["7i"], tileMap["8i"],
    }
    tileMap["7i"].AdjacentTiles = []*Tile{
        tileMap["5i"], tileMap["6i"], tileMap["8i"], tileMap["9i"], 
    }
    tileMap["8i"].AdjacentTiles = []*Tile{
        tileMap["0i"], tileMap["4i"], tileMap["6i"], 
        tileMap["7i"], tileMap["9i"],
    }
    tileMap["9i"].AdjacentTiles = []*Tile{
        tileMap["7i"], tileMap["8i"], 
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{"0i", "7i", "8i"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }

    gs.ModifierPoints["islands"] = func(i int, p *Player) int {
        if tile, ok := tileMap["0i"]; ok && tile.Presence != None && tile.OwningPlayer == p {
            tribe := tile.OwningTribe
            foundOutlier := false
            for _, id := range []string{"1i", "2i", "3i"} {
                if tile, ok := tileMap[id]; !(ok && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)) {
                    foundOutlier = true
                }
            }
            if !foundOutlier {
                gs.Messages = append(gs.Messages, fmt.Sprintf(
                                "%s owns an entire island!",
                                p.Name,
                ))
                i += 1
            }
        }

        if tile, ok := tileMap["9i"]; ok && tile.Presence != None && tile.OwningPlayer == p {
            tribe := tile.OwningTribe
            foundOutlier := false
            for _, id := range []string{"4i", "5i", "7i", "8i"} {
                if tile, ok := tileMap[id]; !(ok && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)) {
                    foundOutlier = true
                }
            }
            if !foundOutlier {
                gs.Messages = append(gs.Messages, fmt.Sprintf(
                                "%s owns an entire island!",
                                p.Name,
                ))
                i += 1
            }
        }

        return i
    }

    return tileMap
}

func MapIsles3(gs *GameState) map[string]*Tile {
    tileMap := map[string]*Tile{
        "0i":  {Id: "0i", Biome: Hill, Attributes: []Attribute{}},
        "1i":  {Id: "1i", Biome: Swamp, Attributes: []Attribute{}},
        "2i":  {Id: "2i", Biome: Field, Attributes: []Attribute{Magic}},
        "3i":  {Id: "3i", Biome: Mountain, Attributes: []Attribute{Mine}},
        "4i":  {Id: "4i", Biome: Field, Attributes: []Attribute{}},
        "5i":  {Id: "5i", Biome: Forest, Attributes: []Attribute{Magic}},
        "6i":  {Id: "6i", Biome: Swamp, Attributes: []Attribute{}},
        "7i":  {Id: "7i", Biome: Hill, Attributes: []Attribute{Cave}},
        "8i":  {Id: "8i", Biome: Forest, Attributes: []Attribute{Mine}},
    }

    tileMap["0i"].AdjacentTiles = []*Tile{
        tileMap["1i"], tileMap["3i"], 
    }

    tileMap["1i"].AdjacentTiles = []*Tile{
        tileMap["8i"], tileMap["2i"], 
        tileMap["0i"], tileMap["3i"], 
    }

    tileMap["2i"].AdjacentTiles = []*Tile{
        tileMap["1i"], tileMap["3i"], 
    }
    tileMap["3i"].AdjacentTiles = []*Tile{
        tileMap["1i"], tileMap["2i"], 
        tileMap["0i"], tileMap["4i"], 
    }
    tileMap["4i"].AdjacentTiles = []*Tile{
        tileMap["3i"], tileMap["5i"], 
    }
    tileMap["5i"].AdjacentTiles = []*Tile{
        tileMap["4i"], tileMap["6i"], 
    }
    tileMap["6i"].AdjacentTiles = []*Tile{
        tileMap["5i"], tileMap["7i"], tileMap["8i"],
    }
    tileMap["7i"].AdjacentTiles = []*Tile{
        tileMap["6i"], tileMap["8i"],
    }
    tileMap["8i"].AdjacentTiles = []*Tile{
        tileMap["1i"], tileMap["6i"], tileMap["7i"], 
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{"1i", "5i", "7i"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }

    gs.ModifierPoints["islands"] = func(i int, p *Player) int {
        if tile, ok := tileMap["0i"]; ok && tile.Presence != None && tile.OwningPlayer == p {
            tribe := tile.OwningTribe
            foundOutlier := false
            for _, id := range []string{"1i", "2i", "3i"} {
                if tile, ok := tileMap[id]; !(ok && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)) {
                    foundOutlier = true
                }
            }
            if !foundOutlier {
                gs.Messages = append(gs.Messages, fmt.Sprintf(
                                "%s owns an entire island!",
                                p.Name,
                ))
                i += 1
            }
        }

        if tile, ok := tileMap["4i"]; ok && tile.Presence != None && tile.OwningPlayer == p {
            tribe := tile.OwningTribe
            foundOutlier := false
            for _, id := range []string{"5i"} {
                if tile, ok := tileMap[id]; !(ok && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)) {
                    foundOutlier = true
                }
            }
            if !foundOutlier {
                gs.Messages = append(gs.Messages, fmt.Sprintf(
                                "%s owns an entire island!",
                                p.Name,
                ))
                i += 1
            }
        }

        if tile, ok := tileMap["6i"]; ok && tile.Presence != None && tile.OwningPlayer == p {
            tribe := tile.OwningTribe
            foundOutlier := false
            for _, id := range []string{"7i", "8i"} {
                if tile, ok := tileMap[id]; !(ok && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)) {
                    foundOutlier = true
                }
            }
            if !foundOutlier {
                gs.Messages = append(gs.Messages, fmt.Sprintf(
                                "%s owns an entire island!",
                                p.Name,
                ))
                i += 1
            }
        }

        return i
    }

    return tileMap
}

func Map4Isles2(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map3(gs), MapIsles2(gs))
    potentialPositions := []string{"1", "2", "3", "4", "25", "5", "21", "20", "16", "15", "0"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["0i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["9i"])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[a])
    result["9i"].AdjacentTiles = append(result["9i"].AdjacentTiles, result[b])
    return result
}

func Map4Isles3(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map3(gs), MapIsles3(gs))
    potentialPositions := []string{"1", "2", "3", "4", "25", "5", "21", "20", "16", "15", "0"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["8i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["0i"])
    result["8i"].AdjacentTiles = append(result["8i"].AdjacentTiles, result[a])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[b])
    return result
}

func Map3Isles2(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map2(gs), MapIsles2(gs))
    potentialPositions := []string{"7", "8", "9", "13", "15", "16"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["0i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["9i"])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[a])
    result["9i"].AdjacentTiles = append(result["9i"].AdjacentTiles, result[b])
    return result
}
func Map3Isles3(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map2(gs), MapIsles3(gs))
    potentialPositions := []string{"7", "8", "9", "13", "15", "16"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["8i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["0i"])
    result["8i"].AdjacentTiles = append(result["8i"].AdjacentTiles, result[a])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[b])
    return result
}

func Map5Isles2(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map4(gs), MapIsles2(gs))
    potentialPositions := []string{"8", "9", "10", "11", "12", "13", "14", "18", "19", "21", "25", "26", "27", "32", "33"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["0i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["9i"])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[a])
    result["9i"].AdjacentTiles = append(result["9i"].AdjacentTiles, result[b])
    return result
}
func Map5Isles3(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map4(gs), MapIsles3(gs))
    potentialPositions := []string{"8", "9", "10", "11", "12", "13", "14", "18", "19", "21", "25", "26", "27", "32", "33"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["8i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["0i"])
    result["8i"].AdjacentTiles = append(result["8i"].AdjacentTiles, result[a])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[b])
    return result
}

func Map6Isles2(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map5(gs), MapIsles2(gs))
    potentialPositions := []string{"8", "9", "10", "11", "18", "12", "19", "20", "25", "31", "36", "35", "41", "40", "34", "39", "33", "29", "30", "24", "16"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["0i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["9i"])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[a])
    result["9i"].AdjacentTiles = append(result["9i"].AdjacentTiles, result[b])
    return result
}

func Map6Isles3(gs *GameState) map[string]*Tile {
    result := mergeMaps(Map5(gs), MapIsles3(gs))
    potentialPositions := []string{"8", "9", "10", "11", "18", "12", "19", "20", "25", "31", "36", "35", "41", "40", "34", "39", "33", "29", "30", "24", "16"}
    AncientBuilders := CreateBaseTribe()
    a, b, _ := pickTwoRandom(potentialPositions)
    result[a].PieceStacks = AddPieceStacks(result[a].PieceStacks, []PieceStack{{Type: "Great Beanstalk", Amount: 1, Tribe: AncientBuilders}})
    result[b].PieceStacks = AddPieceStacks(result[b].PieceStacks, []PieceStack{{Type: "Great Stairs", Amount: 1, Tribe: AncientBuilders}})
    result[a].AdjacentTiles = append(result[a].AdjacentTiles, result["8i"])
    result[b].AdjacentTiles = append(result[b].AdjacentTiles, result["0i"])
    result["8i"].AdjacentTiles = append(result["8i"].AdjacentTiles, result[a])
    result["0i"].AdjacentTiles = append(result["0i"].AdjacentTiles, result[b])
    return result
}

func Map5(gs *GameState) map[string]*Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0":  {Id: "0", Biome: Swamp, Attributes: []Attribute{Mine}, IsEdge: true},
        "1":  {Id: "1", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: true},
        "2":  {Id: "2", Biome: Hill, Attributes: []Attribute{Mine}, IsEdge: true},
        "3":  {Id: "3", Biome: Mountain, Attributes: []Attribute{Magic}, IsEdge: true},
        "4":  {Id: "4", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "5":  {Id: "5", Biome: Forest, Attributes: []Attribute{Cave}, IsEdge: true},
        "6":  {Id: "6", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "7":  {Id: "7", Biome: Field, Attributes: []Attribute{Cave}, IsEdge: true},
        "8":  {Id: "8", Biome: Forest, Attributes: []Attribute{}},
        "9":  {Id: "9", Biome: Field, Attributes: []Attribute{}},
        "10": {Id: "10", Biome: Swamp, Attributes: []Attribute{Cave}},
        "11": {Id: "11", Biome: Forest, Attributes: []Attribute{}},
        "12": {Id: "12", Biome: Mountain, Attributes: []Attribute{Mine}},
        "13": {Id: "13", Biome: Hill, Attributes: []Attribute{Magic}, IsEdge: true},
        "14": {Id: "14", Biome: Forest, Attributes: []Attribute{Mine}, IsEdge: true},
        "15": {Id: "15", Biome: Swamp, Attributes: []Attribute{}, IsEdge: true},
        "16": {Id: "16", Biome: Hill, Attributes: []Attribute{}},
        "17": {Id: "17", Biome: Water, Attributes: []Attribute{}},
        "18": {Id: "18", Biome: Field, Attributes: []Attribute{Magic}},
        "19": {Id: "19", Biome: Forest, Attributes: []Attribute{}},
        "20": {Id: "20", Biome: Hill, Attributes: []Attribute{}},
        "21": {Id: "21", Biome: Swamp, Attributes: []Attribute{Cave}, IsEdge: true},
        "22": {Id: "22", Biome: Water, Attributes: []Attribute{}, IsEdge: true},
        "23": {Id: "23", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "24": {Id: "24", Biome: Mountain, Attributes: []Attribute{Cave, Mine}},
        "25": {Id: "25", Biome: Swamp, Attributes: []Attribute{Mine}},
        "26": {Id: "26", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "27": {Id: "27", Biome: Hill, Attributes: []Attribute{}, IsEdge: true},
        "28": {Id: "28", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: true},
        "29": {Id: "29", Biome: Swamp, Attributes: []Attribute{}},
        "30": {Id: "30", Biome: Hill, Attributes: []Attribute{Magic}},
        "31": {Id: "31", Biome: Forest, Attributes: []Attribute{}},
        "32": {Id: "32", Biome: Mountain, Attributes: []Attribute{Magic}, IsEdge: true},
        "33": {Id: "33", Biome: Hill, Attributes: []Attribute{}, IsEdge: true},
        "34": {Id: "34", Biome: Field, Attributes: []Attribute{Cave}},
        "35": {Id: "35", Biome: Mountain, Attributes: []Attribute{}},
        "36": {Id: "36", Biome: Swamp, Attributes: []Attribute{Cave}},
        "37": {Id: "37", Biome: Hill, Attributes: []Attribute{Mine}, IsEdge: true},
        "38": {Id: "38", Biome: Swamp, Attributes: []Attribute{}, IsEdge: true},
        "39": {Id: "39", Biome: Field, Attributes: []Attribute{Mine}},
        "40": {Id: "40", Biome: Hill, Attributes: []Attribute{}},
        "41": {Id: "41", Biome: Mountain, Attributes: []Attribute{Mine}},
        "42": {Id: "42", Biome: Field, Attributes: []Attribute{}, IsEdge: true},
        "43": {Id: "43", Biome: Mountain, Attributes: []Attribute{}, IsEdge: true},
        "44": {Id: "44", Biome: Swamp, Attributes: []Attribute{Magic}, IsEdge: true},
        "45": {Id: "45", Biome: Mountain, Attributes: []Attribute{Cave}, IsEdge: true},
        "46": {Id: "46", Biome: Forest, Attributes: []Attribute{Magic}, IsEdge: true},
        "47": {Id: "47", Biome: Field, Attributes: []Attribute{Cave}, IsEdge: true},
    }

    for key := range(tileMap) {
        tileMap[key].ModifierDefenses = make(map[string]func(int, error) (int, error))
        tileMap[key].ModifierPoints = make(map[string]func(int) (int))
    }

        // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["0"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["8"], tileMap["1"], 
    }

    tileMap["1"].AdjacentTiles = []*Tile{
        tileMap["0"], tileMap["8"], tileMap["9"], tileMap["2"], 
    }

    tileMap["2"].AdjacentTiles = []*Tile{
        tileMap["1"], tileMap["9"], tileMap["10"], tileMap["3"], 
    }

    tileMap["3"].AdjacentTiles = []*Tile{
        tileMap["11"], tileMap["10"], tileMap["2"], tileMap["4"],
    }

    tileMap["4"].AdjacentTiles = []*Tile{
        tileMap["3"], tileMap["11"], tileMap["18"], tileMap["12"], 
    }

    tileMap["5"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["4"], tileMap["13"], tileMap["6"],
    }

    tileMap["6"].AdjacentTiles = []*Tile{
        tileMap["5"], tileMap["13"], 
    }

    tileMap["7"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["0"], tileMap["15"], tileMap["14"],
    }

    tileMap["8"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["9"], tileMap["1"], tileMap["0"],
        tileMap["16"], tileMap["15"],
    }

    tileMap["9"].AdjacentTiles = []*Tile{
        tileMap["1"], tileMap["2"], tileMap["8"], tileMap["10"],
        tileMap["16"], 
    }

    tileMap["10"].AdjacentTiles = []*Tile{
        tileMap["2"], tileMap["3"], tileMap["9"], tileMap["11"],tileMap["16"],
        tileMap["17"], tileMap["18"],
    }

    tileMap["11"].AdjacentTiles = []*Tile{
        tileMap["10"], tileMap["3"], tileMap["4"], tileMap["18"],
    }

    tileMap["12"].AdjacentTiles = []*Tile{
        tileMap["4"], tileMap["13"], tileMap["5"], tileMap["18"], tileMap["19"], 
        tileMap["20"],
    }

    tileMap["13"].AdjacentTiles = []*Tile{
        tileMap["6"], tileMap["5"], tileMap["20"], tileMap["12"], tileMap["21"], 
    }

    tileMap["14"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["15"], tileMap["7"],
    }

    tileMap["15"].AdjacentTiles = []*Tile{
        tileMap["7"], tileMap["8"], tileMap["14"], tileMap["22"], tileMap["23"], tileMap["16"],
    }

    tileMap["16"].AdjacentTiles = []*Tile{
        tileMap["8"], tileMap["9"], tileMap["10"], tileMap["15"], tileMap["17"], tileMap["23"], tileMap["24"], 
    }

    tileMap["17"].AdjacentTiles = []*Tile{
        tileMap["10"], tileMap["16"], tileMap["24"], tileMap["30"], tileMap["34"], tileMap["35"], tileMap["31"], 
        tileMap["25"], tileMap["19"], tileMap["18"],
    }

    tileMap["18"].AdjacentTiles = []*Tile{
        tileMap["10"], tileMap["11"], tileMap["4"], tileMap["12"],
        tileMap["19"], tileMap["17"],
    }

    tileMap["19"].AdjacentTiles = []*Tile{
        tileMap["18"], tileMap["20"], tileMap["12"], tileMap["25"],
        tileMap["17"],
    }

    tileMap["20"].AdjacentTiles = []*Tile{
        tileMap["12"], tileMap["19"], tileMap["13"], tileMap["21"], tileMap["25"], tileMap["26"],
    }

    tileMap["21"].AdjacentTiles = []*Tile{
        tileMap["13"], tileMap["20"], tileMap["26"],
    }

    tileMap["22"].AdjacentTiles = []*Tile{
        tileMap["14"], tileMap["15"], tileMap["23"],
        tileMap["28"], tileMap["27"], tileMap["33"],
        tileMap["38"], tileMap["43"], 
    }

    tileMap["23"].AdjacentTiles = []*Tile{
        tileMap["15"], tileMap["16"], tileMap["24"], tileMap["28"],tileMap["29"],
    }

    tileMap["24"].AdjacentTiles = []*Tile{
        tileMap["30"], tileMap["16"], tileMap["23"], tileMap["29"], tileMap["17"],
    }

    tileMap["25"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["19"], tileMap["20"], tileMap["26"], tileMap["32"], tileMap["31"],
    }

    tileMap["26"].AdjacentTiles = []*Tile{
        tileMap["25"], tileMap["21"], tileMap["20"], tileMap["32"],
    }

    tileMap["27"].AdjacentTiles = []*Tile{
        tileMap["28"], tileMap["22"], 
    }

    tileMap["28"].AdjacentTiles = []*Tile{
        tileMap["27"], tileMap["22"], tileMap["23"], tileMap["33"], tileMap["29"],
    }

    tileMap["29"].AdjacentTiles = []*Tile{
        tileMap["23"], tileMap["24"], tileMap["30"], // selon la frontière visible
        tileMap["34"], tileMap["33"], tileMap["28"], // selon la frontière visible
    }
    tileMap["30"].AdjacentTiles = []*Tile{
        tileMap["29"], tileMap["34"], tileMap["24"], tileMap["17"], 
    }

    tileMap["31"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["35"], tileMap["32"], tileMap["36"], tileMap["35"],
    }

    tileMap["32"].AdjacentTiles = []*Tile{
        tileMap["25"], tileMap["26"], tileMap["31"], tileMap["37"], tileMap["36"],
    }

    tileMap["33"].AdjacentTiles = []*Tile{
        tileMap["28"], tileMap["29"], tileMap["34"], tileMap["39"], tileMap["38"], tileMap["22"], 
    }

    tileMap["34"].AdjacentTiles = []*Tile{
        tileMap["29"], tileMap["30"], tileMap["17"], tileMap["35"], tileMap["40"], 
        tileMap["39"], tileMap["33"],
    }

    tileMap["35"].AdjacentTiles = []*Tile{
        tileMap["17"], tileMap["34"], tileMap["40"], 
        tileMap["41"], tileMap["36"], tileMap["41"], 
    }

    tileMap["36"].AdjacentTiles = []*Tile{
        tileMap["31"], tileMap["35"], tileMap["32"], tileMap["41"], tileMap["37"], tileMap["42"],
    }

    tileMap["37"].AdjacentTiles = []*Tile{
        tileMap["36"], tileMap["32"], tileMap["42"], 
    }

    tileMap["38"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["33"], tileMap["39"], tileMap["43"], tileMap["44"],
    }
    tileMap["39"].AdjacentTiles = []*Tile{
        tileMap["34"], tileMap["33"], tileMap["38"], tileMap["40"], tileMap["44"], tileMap["45"],
    }
    tileMap["40"].AdjacentTiles = []*Tile{
        tileMap["34"], tileMap["35"], tileMap["39"], tileMap["41"], tileMap["45"], tileMap["46"], 
    }
    tileMap["41"].AdjacentTiles = []*Tile{
        tileMap["35"], tileMap["36"], tileMap["40"], tileMap["42"], tileMap["46"], tileMap["47"],
    }
    tileMap["42"].AdjacentTiles = []*Tile{
        tileMap["36"], tileMap["37"], tileMap["41"], tileMap["47"],
    }
    tileMap["43"].AdjacentTiles = []*Tile{
        tileMap["22"], tileMap["38"], tileMap["44"],
    }
    tileMap["44"].AdjacentTiles = []*Tile{
        tileMap["45"], tileMap["38"], tileMap["39"], tileMap["43"],
    }
    tileMap["45"].AdjacentTiles = []*Tile{
        tileMap["39"], tileMap["40"], tileMap["44"], tileMap["46"],
    }
    tileMap["46"].AdjacentTiles = []*Tile{
        tileMap["45"], tileMap["47"], tileMap["40"], tileMap["41"], 
    }
    tileMap["47"].AdjacentTiles = []*Tile{
        tileMap["46"], tileMap["41"], tileMap["42"],
    }

    // Step 3: Convert the map of pointers to a map of values
    result := make(map[string]*Tile, len(tileMap))
    for id, tile := range tileMap {
        result[id] = tile // Dereference pointer to get a Tile value
    }

    lostTribe := CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"
    lostPlayer := Player{
        PieceStacks : []PieceStack{},
        ActiveTribe: lostTribe,
    }

    for _, id := range []string{"0", "1", "13", "21", "19", "25", "10", "16", "7", "23", "8", "29", "30", "27", "33", "34", "39", "47", "36", "25"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
        tileMap[id].Presence = Passive
        tileMap[id].OwningPlayer = &lostPlayer
    }


    return result
}

// MapRegistry stores map definitions for dynamic loading
var MapRegistry = map[string]func(*GameState) map[string]*Tile{
	"map2players": Map2,
	"map3players": Map3,
        "map4players": Map4,
        "map5players": Map5,
        "map3players2islands": Map3Isles2,
        "map4players2islands": Map4Isles2,
        "map5players2islands": Map5Isles2,
        "map6players2islands": Map6Isles2,
        "map3players3islands": Map3Isles3,
        "map4players3islands": Map4Isles3,
        "map5players3islands": Map5Isles3,
        "map6players3islands": Map6Isles3,
}
