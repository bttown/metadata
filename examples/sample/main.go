package main

import (
	"github.com/bttown/metadata"
	"os"
	"strconv"
)

var c = metadata.NewCollector()

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

	c.GetSync(&req)
}
