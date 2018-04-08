package metadata

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"sync/atomic"
)

// Collector collects metadata.
type Collector struct {
	handler    Handler
	errHandler ErrHandler
	requsts    chan *Request

	MaxRoutineCount int64
	running         int64
	closed          bool
}

// NewCollector creates and return a new metadata collector.
func NewCollector() *Collector {
	c := Collector{
		requsts:         make(chan *Request, 4000),
		MaxRoutineCount: 800,
	}

	c.handler = func(req *Request, meta *Metadata) {
		magnetLink := fmt.Sprintf("magnet:?xt=urn:btih:%s", req.HashInfo)
		log.Println("[onMetadata]", magnetLink, meta.Name)
	}

	c.errHandler = func(req *Request, err error) {
		// log.Println("[onError]", err.Error())
	}

	return &c
}

// Close the collector.
func (c *Collector) Close() {
	c.closed = true
	close(c.requsts)
}

func (c *Collector) runGoRoutine() {
	var err error
	for req := range c.requsts {
		err = c.collect(req)
		if err != nil {
			c.errHandler(req, err)
		}
	}
}

// GetSync metadata sync.
func (c *Collector) GetSync(req *Request) error {
	err := c.collect(req)
	if err != nil {
		c.errHandler(req, err)
	}

	return nil
}

// Get adds the request to the collector's queue.
func (c *Collector) Get(req *Request) error {
	if c.closed {
		return errors.New("collector closed")
	}

	select {
	case c.requsts <- req:

	default:
		if atomic.AddInt64(&c.running, 1) > c.MaxRoutineCount {
			c.requsts <- req
			return nil
		}

		go c.runGoRoutine()
	}
	return nil
}

// OnFinish registers metadata handler.
func (c *Collector) OnFinish(handler Handler) {
	c.handler = handler
}

// OnError registers error handler.
func (c *Collector) OnError(errHandler ErrHandler) {
	c.errHandler = errHandler
}

func (c *Collector) handleError(req *Request, err error) {
	c.errHandler(req, err)
}

func (c *Collector) collect(req *Request) error {
	defer func() {
		v := recover()
		if v != nil {
			buf := make([]byte, 1024)
			n := runtime.Stack(buf, false)
			log.Println("[panic]", v, string(buf[:n]))
		}
	}()
	conn, err := net.DialTimeout("tcp", req.PeerAddress(), req.DailTimeout())
	if err != nil {
		return err
	}
	tcpConn := conn.(*net.TCPConn)
	tcpConn.SetLinger(0)
	defer tcpConn.Close()

	if err := sendHandshake(conn, req); err != nil {
		return err
	}

	var buf = new(bytes.Buffer)
	if err := recvHandshake(conn, buf); err != nil {
		return err
	}
	buf.Reset()

	if err := sendHandshakeExt(conn, req); err != nil {
		return err
	}

	utMetadata, metadataSize, err := recvHandshakeExt(conn, buf)
	if err != nil {
		return err
	}

	var BLOCKSIZE int64 = 16384
	var pieceNum = metadataSize / BLOCKSIZE
	if metadataSize%BLOCKSIZE > 0 {
		pieceNum++
	}

	for i := 0; i < int(pieceNum); i++ {
		err = sendPieceRequst(conn, int(utMetadata), i)
		if err != nil {
			return err
		}
	}

	pieces := make([][]byte, int(pieceNum))
	for {
		buf.Reset()
		bid, _, err := readMessage(conn, buf)
		if err != nil {
			return err
		}

		if bid != 20 {
			buf.Reset()
			continue
		}

		msgType, pieceID, _, offset, err := parsePiece(buf.Bytes())
		if err != nil {
			return err
		}
		if msgType == REJECT {
			return errors.New("Peer reject")
		} else if msgType == DATA {
			piece, _ := ioutil.ReadAll(buf)
			pieces[pieceID] = piece[offset:]
			// log.Println(pieceID, int64(len(piece[offset:])))
			if int64(len(piece[offset:])) < BLOCKSIZE {
				break
			}
		}

	}
	p := bytes.Join(pieces, nil)
	metadata, err := NewMetadata(p)
	if err != nil {
		return err
	}

	c.handler(req, &metadata)
	return nil
}
