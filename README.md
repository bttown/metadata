# metadata
a infohash metadata collector

#### Install
    go get -u github.com/bttown/metadata

#### Usage
```go

c := metadata.NewCollector()
	c.OnFinish(func(req *metadata.Request, meta *metadata.Metadata) {
		magnetLink := fmt.Sprintf("magnet:?xt=urn:btih:%s", req.HashInfo)
		log.Println("[onMetadata]", magnetLink, meta.Name)
	})


    c.GetSync(&metadata.Request{
        IP:       ip,
        Port:     port,
        HashInfo: hashInfo,
        PeerID:   peerID,
    })

```