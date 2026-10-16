package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRoomMetadataPreservesOptionalGeneratedRegionName(t *testing.T) {
	for _, named := range []bool{false, true} {
		selection := `"generatedMapSelection":{"previewId":"generated-example","playerCount":2,"seed":"seed"`
		if named {
			selection += `,"regionName":"The Amber Highlands","regionNameVersion":1`
		}
		var room RoomMetadata
		if err := json.Unmarshal([]byte(`{`+selection+`}}`), &room); err != nil {
			t.Fatal(err)
		}
		if room.GeneratedMapSelection == nil || room.GeneratedMapSelection.PreviewID != "generated-example" {
			t.Fatal("lost map selection")
		}
		raw, err := json.Marshal(room)
		if err != nil {
			t.Fatal(err)
		}
		if named {
			if room.GeneratedMapSelection.RegionName != "The Amber Highlands" || room.GeneratedMapSelection.RegionNameVersion != 1 || !strings.Contains(string(raw), `"regionName":"The Amber Highlands"`) {
				t.Fatalf("room-list round trip stripped region name: %s", raw)
			}
		} else if strings.Contains(string(raw), "regionName") {
			t.Fatalf("old room list gained region metadata: %s", raw)
		}
	}
}
