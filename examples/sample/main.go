package main

import (
	"fmt"
	"github.com/bttown/metadata"
	"log"
	"os"
	"strconv"
)

var c = metadata.NewCollector()

func saveTorrentFile(req *metadata.Request, meta *metadata.Metadata) {
	torrentName := fmt.Sprintf("%s.torrent", meta.Name)
	f, err := os.Create(torrentName)
	if err != nil {
		log.Println("fail to create torrent file", err)
		return
	}
	defer f.Close()

	f.Write(meta.Torrent())
}

func main() {
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
