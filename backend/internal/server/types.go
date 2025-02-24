package server

type TilePolygon struct {
    Coords []int `json:"coords"`
    StackX int   `json:"stackX"`
    StackY int   `json:"stackY"`
}

type TileData struct {
    ID      int         `json:"id"`
    Polygon TilePolygon `json:"polygon"`
    // ...
}

type ChoiceEntry struct {
    Choice string `json:"choice"`
    IsChecked bool `json:"isChecked"`
}
