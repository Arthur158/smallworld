package gamestate

// Map1 defines tiles and their adjacency relationships
func Map1() map[string]*Tile {
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
        tileMap["0"], tileMap["13"], tileMap["10"], tileMap["15"], // et éventuellement tileMap["13’"] si c’est distinct
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

    lostTribe := createBaseTribe()

    for _, id := range []string{ "22", "21", "4", "3", "11", "26", "1", "13", "16", "0"} {
        tileMap[id].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
        tileMap[id].OwningTribe = lostTribe
    }


    return result
}


// MapRegistry stores map definitions for dynamic loading
var MapRegistry = map[string]func() map[string]*Tile{
	"Map1": Map1,
}
