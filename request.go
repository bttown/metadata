package metadata

import (
	"fmt"
	"time"
)

// Request is the metadata request.
type Request struct {
	IP       string
	Port     int
	HashInfo string
	PeerID   string

	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// DailTimeout returns dail timeout.
func (req *Request) DailTimeout() time.Duration {
	if req.dialTimeout == 0 {
		req.dialTimeout = 5 * time.Second
	}

	return req.dialTimeout
}

// PeerAddress returns net address of the querying peer.
func (req *Request) PeerAddress() string {
	return fmt.Sprintf("%s:%d", req.IP, req.Port)
}
