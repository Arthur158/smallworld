package server


import (
	"io/ioutil"
	"encoding/base64"
	"fmt"
	"path/filepath"
)

type Map struct {
	Name	string
	Offset float64
	FontSize int
	Capacity int
}

func (m *Map) ImagePath(baseDir string) string {
    return filepath.Join(baseDir, m.Name + ".jpg")
}

func getMapImageAsBase64(path string) (string, error) {
    imgBytes, err := ioutil.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("cannot read file: %w", err)
    }
    return base64.StdEncoding.EncodeToString(imgBytes), nil
}

var Map2 = Map{
	Name: "map2players",
	Offset: 0.728,
	FontSize: 60,
	Capacity: 2,
}

var Map3 = Map{
	Name: "map3players",
	Offset: 1,
	FontSize: 18,
	Capacity: 3,
}

var Map4 = Map{
	Name: "map4players",
	Offset: 0.378,
	FontSize: 44,
	Capacity: 4,
}

var Map5 = Map{
	Name: "map5players",
	Offset: 0.39,
	FontSize: 50,
	Capacity: 2,
}

var Map3Isles2 = Map{
	Name: "map3players2islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}
var Map3Isles3 = Map{
	Name: "map3players3islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}

var Map4Isles2 = Map{
	Name: "map4players2islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}
var Map4Isles3 = Map{
	Name: "map4players3islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}


var Map5Isles2 = Map{
	Name: "map5players2islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}
var Map5Isles3 = Map{
	Name: "map5players3islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}

var Map6Isles2 = Map{
	Name: "map6players2islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}
var Map6Isles3 = Map{
	Name: "map6players3islands",
	Offset: 1.185,
	FontSize: 50,
	Capacity: 2,
}

var mapMap = map[string]Map {
	"2 Players": Map2,
	"3 Players": Map3,
	"4 Players": Map4,
	"5 Players": Map5,
	"4 Players with 2 islands": Map4Isles2,
	"3 Players with 2 islands": Map3Isles2,
	"5 Players with 2 islands": Map5Isles2,
	"6 Players with 2 islands": Map6Isles2,
	"4 Players with 3 islands": Map4Isles3,
	"3 Players with 3 islands": Map3Isles3,
	"5 Players with 3 islands": Map5Isles3,
	"6 Players with 3 islands": Map6Isles3,
}

