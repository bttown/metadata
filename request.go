package metadata

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"time"
)

// Request is the metadata request.
type Request struct {
	IP       string
	Port     int
	HashInfo string
	PeerID   string
}

// RemoteAddr returns net address of the querying peer.
func (req *Request) RemoteAddr() string {
	return fmt.Sprintf("%s:%d", req.IP, req.Port)
}

type metadataQuery struct {
	*Request

	metadata *Metadata
	err      error
	done     chan struct{}

	then   Then
	reject Reject

	dialTimeout time.Duration
}

func newMetadataQuery(req *Request, then Then, reject Reject) *metadataQuery {
	return &metadataQuery{
		Request:     req,
		then:        then,
		reject:      reject,
		dialTimeout: 10 * time.Second,
		done:        make(chan struct{}),
	}
}

func (query *metadataQuery) wait() {
	<-query.done
}

func (query *metadataQuery) start(collect *Collector) {
	query.do()
	select {
	case <-collect.closeQ:
	case collect.reply <- struct{}{}:
	}

	if query.err != nil {
		query.reject(query.Request, query.err)
	} else {
		query.then(query.Request, query.metadata)
	}

	close(query.done)
}

func (query *metadataQuery) do() {
	defer func() {
		v := recover()
		if v != nil {
			buf := make([]byte, 1024)
			n := runtime.Stack(buf, false)
			log.Println("[panic]", v, string(buf[:n]))
			query.err = errors.New("query panic")
		}
	}()

	conn, err := net.DialTimeout("tcp", query.RemoteAddr(), query.dialTimeout)
	if err != nil {
		query.err = err
		return
	}
	tcpConn := conn.(*net.TCPConn)
	tcpConn.SetLinger(0)
	defer tcpConn.Close()

	if err := sendHandshake(conn, query); err != nil {
		query.err = err
		return
	}

	var buf = new(bytes.Buffer)
	if err := recvHandshake(conn, buf); err != nil {
		query.err = err
		return
	}
	buf.Reset()

	if err := sendHandshakeExt(conn, query); err != nil {
		query.err = err
		return
	}

	utMetadata, metadataSize, err := recvHandshakeExt(conn, buf)
	if err != nil {
		query.err = err
		return
	}

	var BLOCKSIZE int64 = 16384
	var pieceNum = metadataSize / BLOCKSIZE
	if metadataSize%BLOCKSIZE > 0 {
		pieceNum++
	}

	if pieceNum > 10000 {
		query.err = ErrTooMuchPieces
		return
	}

	for i := 0; i < int(pieceNum); i++ {
		err = sendPieceRequst(conn, int(utMetadata), i)
		if err != nil {
			query.err = err
			return
		}
	}

	var getPiecesTimeout = time.After(20 * time.Second)

	pieces := make([][]byte, int(pieceNum))
getPieces:
	for {
		select {
		case <-getPiecesTimeout:
			query.err = ErrGetPiecesTimeout
			return
		default:

			buf.Reset()
			bid, _, err := readMessage(conn, buf)
			if err != nil {
				query.err = err
				return
			}

			if bid != 20 {
				buf.Reset()
				continue
			}

			msgType, pieceID, _, offset, err := parsePiece(buf.Bytes())
			if err != nil {
				query.err = err
				return
			}
			if msgType == REJECT {
				query.err = ErrRejectByPeer
				return
			} else if msgType == DATA {
				piece, _ := ioutil.ReadAll(buf)
				pieces[pieceID] = piece[offset:]
				// log.Println(pieceID, int64(len(piece[offset:])))
				if int64(len(piece[offset:])) < BLOCKSIZE {
					break getPieces
				}
			}

		}
	}
	p := bytes.Join(pieces, nil)
	metadata, err := newMetadata(p)
	if err != nil {
		query.err = err
		return
	}

	query.metadata = &metadata
	query.err = nil
	return
}
