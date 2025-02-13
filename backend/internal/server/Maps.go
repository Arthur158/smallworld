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
}

var Map5 = Map{
	Name: "map5players",
	Offset: 1.185,
}

var Map4 = Map{
	Name: "map4players",
	Offset: 0.378,
}

var Map2 = Map{
	Name: "map2players",
	Offset: 0.728,
}

var mapMap = map[int]Map {
	2: Map4,
	3: Map3,
	4: Map5,
	5: Map5,
}

