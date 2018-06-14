package main

import (
	"fmt"
	"github.com/bttown/metadata"
	"log"
	"os"
	"strconv"
)

func saveTorrentFile(req metadata.Request, torrent metadata.Torrent) {
	torrentName := fmt.Sprintf("%s.torrent", torrent.Info.Name)
	f, err := os.Create(torrentName)
	if err != nil {
		log.Println("fail to create torrent file", err)
		return
	}
	defer f.Close()

	f.Write(torrent.Bytes())
}

func main() {
	var c = metadata.NewCollector(metadata.Options{})
	defer c.Close()

	ip, peerID, hashInfo := os.Args[1], os.Args[3], os.Args[4]
	port, _ := strconv.Atoi(os.Args[2])

	req := metadata.Request{
		IP:       ip,
		Port:     port,
		HashInfo: hashInfo,
		PeerID:   peerID,
	}

	log.Println(c.GetSync(&req, saveTorrentFile, nil))
}
