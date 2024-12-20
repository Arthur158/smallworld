package gamestate

// Map1 defines tiles and their adjacency relationships
func Map1() map[string]*Tile {
    // Step 1: Create tiles without adjacency
    tileMap := map[string]*Tile{
        "0":  {Id: "0", Biome: Swamp, Attributes: []Attribute{}, IsEdge: true},
        "1":  {Id: "1", Biome: Forest, Attributes: []Attribute{Cave}, IsEdge: true},
        "2":  {Id: "2", Biome: Hill, Attributes: []Attribute{}},
        "3":  {Id: "3", Biome: Swamp, Attributes: []Attribute{Magic}},
        "4":  {Id: "4", Biome: Field, Attributes: []Attribute{}},
        "5":  {Id: "5", Biome: Mountain, Attributes: []Attribute{Cave}},
        "6":  {Id: "6", Biome: Forest, Attributes: []Attribute{}},
        "7":  {Id: "7", Biome: Mountain, Attributes: []Attribute{}},
        "8":  {Id: "8", Biome: Mountain, Attributes: []Attribute{}},
        "9":  {Id: "9", Biome: Hill, Attributes: []Attribute{}},
        "10": {Id: "10", Biome: Field, Attributes: []Attribute{Magic}},
        "11": {Id: "11", Biome: Forest, Attributes: []Attribute{Mine}},
        "12": {Id: "12", Biome: Mountain, Attributes: []Attribute{Cave, Mine}},
        "14": {Id: "14", Biome: Forest, Attributes: []Attribute{Mine}},
        "15": {Id: "15", Biome: Swamp, Attributes: []Attribute{}},
        "16": {Id: "16", Biome: Hill, Attributes: []Attribute{Magic}},
        "17": {Id: "17", Biome: Mountain, Attributes: []Attribute{}},
    }

    // Step 2: Set adjacency by adding pointers to AdjacentTiles
    tileMap["9"].AdjacentTiles = []*Tile{tileMap["8"], tileMap["10"]}

    // Step 3: Convert the map of pointers to a map of values
    result := make(map[string]*Tile, len(tileMap))
    for id, tile := range tileMap {
        result[id] = tile // Dereference pointer to get a Tile value
    }

    return result
}


// MapRegistry stores map definitions for dynamic loading
var MapRegistry = map[string]func() map[string]*Tile{
	"Map1": Map1,
}
