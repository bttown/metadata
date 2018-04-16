package metadata

import (
	"github.com/IncSW/go-bencode"
)

// Metadata contains all details of resource.
type Metadata struct {
	Name string
	raw  interface{}
}

// newMetadata unmarshal bytes to metadata
func newMetadata(data []byte) (m Metadata, err error) {
	i, err := bencode.Unmarshal(data)
	if err != nil {
		return
	}

	m = Metadata{
		raw: i,
	}
	m.Name = string(i.(map[string]interface{})["name"].([]byte))
	return
}

func (meta *Metadata) Torrent() []byte {
	torrentData := map[string]interface{}{
		"info": meta.raw,
	}

	b, _ := bencode.Marshal(torrentData)
	return b
}
