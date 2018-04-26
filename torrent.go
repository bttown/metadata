package metadata

import (
	"encoding/hex"
	"errors"
	"github.com/IncSW/go-bencode"
)

// ErrInvalidMetadata ...
var ErrInvalidMetadata = errors.New("invalid metadata")

// Torrent ...
type Torrent struct {
	raw      interface{}
	Announce string
	Info     TorrentInfo
}

// Bytes torrent file bytes.
func (torrent *Torrent) Bytes() []byte {
	b, err := bencode.Marshal(torrent.raw)
	if err != nil {
		panic(err)
	}
	return b
}

// TorrentInfo ...
type TorrentInfo struct {
	Files       []TorrentSubFile
	Length      int64
	Name        string
	PieceLength int64
	Pieces      string
}

// TorrentSubFile ...
type TorrentSubFile struct {
	Length int64
	Path   string
}

// NewTorrentFrom ...
func NewTorrentFromMetadata(b []byte, torrent *Torrent) error {
	metadata, err := bencode.Unmarshal(b)
	if err != nil {
		return err
	}

	torrent.raw = map[string]interface{}{
		"info": metadata,
	}
	m, ok := torrent.raw.(map[string]interface{})
	if !ok {
		return ErrInvalidMetadata
	}

	if announce, ok := m["announce"].([]byte); ok {
		torrent.Announce = string(announce)
	}

	if info, ok := m["info"].(map[string]interface{}); ok {
		tinfo := TorrentInfo{}
		// name of root file or folder
		if name, ok := info["name"].([]byte); ok {
			tinfo.Name = string(name)
		}

		// size of per piece
		if pieceLength, ok := info["piece length"].(int64); ok {
			tinfo.PieceLength = pieceLength
		}
		// piece's SHA-1 hash
		if pieces, ok := info["pieces"].([]byte); ok {
			tinfo.Pieces = hex.EncodeToString(pieces)
		}
		// sub files
		if files, ok := info["files"].([]interface{}); ok {
			fileNum := len(files)
			tfiles := make([]TorrentSubFile, 0, fileNum)
			for i := 0; i < fileNum; i++ {
				if file, ok := files[i].(map[string]interface{}); ok {
					tfile := TorrentSubFile{}
					if length, ok := file["length"].(int64); ok {
						tfile.Length = length
					}
					if path, ok := file["path"].([]interface{}); ok {
						tfile.Path = string(path[0].([]byte))
						tfiles = append(tfiles, tfile)
					}
				}
			}
			tinfo.Files = tfiles
		}

		torrent.Info = tinfo
	}
	return nil
}
