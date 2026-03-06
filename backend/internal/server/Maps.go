package server

import (
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "path/filepath"
)

type Map struct {
    Name     string
    Offset   float64
    FontSize int
    Capacity int
}

func (m *Map) ImagePath(baseDir string) string {
    return filepath.Join(baseDir, m.Name+".jpg")
}

func getMapImageAsBase64(path string) (string, error) {
    imgBytes, err := ioutil.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("cannot read file: %w", err)
    }
    return base64.StdEncoding.EncodeToString(imgBytes), nil
}

var Map2 = Map{
    Name:     "map2players",
    Offset:   0.728,
    FontSize: 60,
    Capacity: 2,
}

var Map3 = Map{
    Name:     "map3players",
    Offset:   1,
    FontSize: 18,
    Capacity: 3,
}

var Map4 = Map{
    Name:     "map4players",
    Offset:   0.378,
    FontSize: 44,
    Capacity: 4,
}

var Map5 = Map{
    Name:     "map5players",
    Offset:   0.39,
    FontSize: 50,
    Capacity: 5,
}

var Map3Isles2 = Map{
    Name:     "map3players2islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 3,
}
var Map3Isles3 = Map{
    Name:     "map3players3islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 3,
}

var Map4Isles2 = Map{
    Name:     "map4players2islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 4,
}
var Map4Isles3 = Map{
    Name:     "map4players3islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 4,
}

var Map5Isles2 = Map{
    Name:     "map5players2islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 5,
}
var Map5Isles3 = Map{
    Name:     "map5players3islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 5,
}

var Map6Isles2 = Map{
    Name:     "map6players2islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 6,
}
var Map6Isles3 = Map{
    Name:     "map6players3islands",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 6,
}

var Underground2 = Map{
    Name:     "underground2players",
    Offset:   1.185,
    FontSize: 50,
    Capacity: 2,
}

var mapMap = map[string]Map{
    "map2players":         Map2,
    "map3players":         Map3,
    "map4players":         Map4,
    "map5players":         Map5,
    "map3players2islands": Map3Isles2,
    "map4players2islands": Map4Isles2,
    "map5players2islands": Map5Isles2,
    "map6players2islands": Map6Isles2,
    "map3players3islands": Map3Isles3,
    "map4players3islands": Map4Isles3,
    "map5players3islands": Map5Isles3,
    "map6players3islands": Map6Isles3,
    "underground2players": Underground2,
}
