package gamestate

// Map1 defines tiles and their adjacency relationships
func Map1() []Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0": {Id: "0", Biome: Swamp, Attributes: []Attribute{}},
        "1": {Id: "1", Biome: Forest, Attributes: []Attribute{Cave}},
        "2": {Id: "2", Biome: Hill, Attributes: []Attribute{}},
        "3": {Id: "3", Biome: Swamp, Attributes: []Attribute{Magic}},
        "4": {Id: "4", Biome: Field, Attributes: []Attribute{}},
        "5": {Id: "5", Biome: Mountain, Attributes: []Attribute{Cave}},
        "6": {Id: "6", Biome: Forest, Attributes: []Attribute{}},
        "7": {Id: "7", Biome: Mountain, Attributes: []Attribute{}},
        "8": {Id: "8", Biome: Mountain, Attributes: []Attribute{}},
        "9": {Id: "9", Biome: Hill, Attributes: []Attribute{}},
    }

    // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["A1"].AdjacentTiles = []*Tile{tileMap["A2"], tileMap["B1"]}
    tileMap["A2"].AdjacentTiles = []*Tile{tileMap["A1"], tileMap["B2"]}
    tileMap["B1"].AdjacentTiles = []*Tile{tileMap["A1"], tileMap["B2"]}
    tileMap["B2"].AdjacentTiles = []*Tile{tileMap["A2"], tileMap["B1"]}

    // Step 3: Return the tiles as a slice
    tiles := make([]Tile, 0, len(tileMap))
    for _, tile := range tileMap {
        tiles = append(tiles, *tile) // Dereference to return a copy of Tile
    }
    return tiles
}


// MapRegistry stores map definitions for dynamic loading
var MapRegistry = map[string]func() []Tile{
	"Map1": Map1,
}
