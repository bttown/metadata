package metadata

import (
	"github.com/IncSW/go-bencode"
)

// Metadata contains all details of resource.
type Metadata struct {
	Name string
}

// NewMetadata unmarshal bytes to metadata
func NewMetadata(data []byte) (m Metadata, err error) {
	i, err := bencode.Unmarshal(data)
	if err != nil {
		return
	}

	m = Metadata{}
	m.Name = string(i.(map[string]interface{})["name"].([]byte))
	return
}
