package server


import (
	"io/ioutil"
	"encoding/base64"
	"fmt"
	"path/filepath"
)

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

var Map3 = Map{
	Name: "map3players",
	Offset: 1,
	FontSize: 18,
}

var Map5 = Map{
	Name: "map5players",
	Offset: 1.185,
	FontSize: 50,
}

var Map4 = Map{
	Name: "map4players",
	Offset: 0.378,
	FontSize: 44,
}

var Map2 = Map{
	Name: "map2players",
	Offset: 0.728,
	FontSize: 60,
}

var Map4Isles2 = Map{
	Name: "map4players2islands",
	Offset: 1.185,
	FontSize: 50,
}

var mapMap = map[int]Map {
	2: Map2,
	3: Map3,
	4: Map4,
	5: Map5,
}

