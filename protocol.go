package metadata

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/IncSW/go-bencode"
	"io"
	"net"
	"time"
)

const (
	// REQUEST  type
	REQUEST = 0
	// DATA type
	DATA = 1
	// REJECT type
	REJECT = 2
)

var (
	readTimeout  = 2 * time.Second
	writeTimeout = 1 * time.Second
)

func writePacket(conn net.Conn, data []byte) error {
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	n, err := conn.Write(data)
	if err != nil {
		return err
	}

	if n != len(data) {
		return errors.New("write error")
	}

	// log.Println("send =>", data)
	return nil
}

func readPacket(conn net.Conn, buf *bytes.Buffer, size int64) error {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	readLen, err := io.CopyN(buf, conn, size)
	if err != nil {
		return err
	}

	if readLen != size {
		return errors.New("readPacket error")
	}

	// log.Println("recv <=", buf.Bytes())
	return nil
}

func readMessage(conn net.Conn, buf *bytes.Buffer) (bid int, eid int, err error) {
	err = readPacket(conn, buf, 4)
	if err != nil {
		return
	}

	msgLen := int64(binary.BigEndian.Uint32(buf.Next(4)))
	if msgLen == 0 {
		return 0, 0, errors.New("read message error")
	}

	err = readPacket(conn, buf, msgLen)
	if err != nil {
		return
	}

	b, err := buf.ReadByte()
	if err != nil {
		return
	}

	bid = int(b)

	e, err := buf.ReadByte()
	if err != nil {
		return
	}

	eid = int(e)
	return
}

func writePacketExt(conn net.Conn, data []byte, bid, eid int) error {
	length := len(data) + 2
	packet := make([]byte, 4+length)
	binary.BigEndian.PutUint32(packet[:4], uint32(length))
	packet[4] = byte(bid)
	packet[5] = byte(eid)
	copy(packet[6:], data)

	return writePacket(conn, packet)
}

func sendHandshake(conn net.Conn, query *metadataQuery) error {
	packet := make([]byte, 68)
	packet[0] = 19
	copy(packet[1:20], []byte("BitTorrent protocol"))
	packet[25] = 0x10

	infohash, _ := hex.DecodeString(query.HashInfo)
	peerID, _ := hex.DecodeString(query.PeerID)

	copy(packet[28:48], infohash)
	copy(packet[48:68], peerID)

	return writePacket(conn, packet)
}

func recvHandshake(conn net.Conn, buf *bytes.Buffer) error {
	return readPacket(conn, buf, 68)
}

func sendHandshakeExt(conn net.Conn, query *metadataQuery) error {
	d := map[string]interface{}{
		"m": map[string]interface{}{
			"ut_metadata": 0,
		},
	}

	var (
		bid = 20
		eid int
	)
	out, _ := bencode.Marshal(d)
	return writePacketExt(conn, out, bid, eid)
}

func sendPieceRequst(conn net.Conn, utMetadata int, pieceID int) error {
	out, _ := bencode.Marshal(map[string]interface{}{
		"msg_type": REQUEST,
		"piece":    pieceID,
	})

	var (
		bid = 20
		eid = utMetadata
	)

	return writePacketExt(conn, out, bid, eid)
}

func recvHandshakeExt(conn net.Conn, buf *bytes.Buffer) (utMetadata int64, metadataSize int64, err error) {
	_, _, err = readMessage(conn, buf)
	if err != nil {
		return
	}

	if buf.Len() == 0 {
		err = errors.New("read error")
		return
	}

	i, err := bencode.Unmarshal(buf.Bytes())
	if err != nil {
		return
	}

	metadataSize, ok := i.(map[string]interface{})["metadata_size"].(int64)
	if !ok {
		err = errors.New("no metadata_size")
		return
	}
	utMetadata = i.(map[string]interface{})["m"].(map[string]interface{})["ut_metadata"].(int64)
	return
}

func parsePiece(data []byte) (msgType, pieceID, totalSize, offset int, err error) {
	i, err := bencode.Unmarshal(data)
	if err != nil {
		return
	}

	m := i.(map[string]interface{})
	msgType = int(m["msg_type"].(int64))
	pieceID = int(m["piece"].(int64))
	// totalSize = int(m["total_size"].(int64))

	// TODO: FIX
	b, _ := bencode.Marshal(i)
	offset = len(b)
	return
}
