package tests

import (
	"bytes"
	"encoding/binary"
	uuid2 "github.com/satori/go.uuid"
	"testing"
	"time"
)

const (
	protoNormalType byte = '0'
	protoResponseType byte = '1'
	protoFlag            = "named-pipe-ipc"
)

/**
protocol:
	14byte 1byte - 16byte - 8byte - string
	flag - type - uuid  - ttl - content
*/

type Message []byte

func (M Message) String() string {
	return string(M)
}

func (M Message) Byte() []byte {
	return M
}

func (M Message) segmentTypeLen() int {
	return 1
}

func (M Message) segmentUUIDLen() int {
	return 16
}

func (M Message) segmentFlagLen() int {
	return len(protoFlag)
}

func (M Message) segmentTTLLen() int {
	return 8
}

func (M Message) segmentFlag() (flag []byte) {
	flag = M[0:M.segmentFlagLen()].Byte()

	return flag
}

func (M Message) segmentType() (t byte) {
	t = M[M.segmentFlagLen():M.segmentTypeLen()].Byte()[0]

	return t
}

func (M Message) segmentUUID() (uuid uuid2.UUID, err error) {
	uuid, err = uuid2.FromBytes(M[M.segmentFlagLen()+M.segmentTypeLen() : M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()])
	return
}

func (M Message) segmentTTL() (ttl int64) {
	timestamp := M[M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen() : M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen()]
	ttl = int64(binary.BigEndian.Uint64(timestamp))

	return
}

func (M Message) segmentPayload() Message {
	return M[M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen():]
}

func (M Message) Payload() Message {
	return M.segmentPayload()
}

func (M Message) isLegal() bool {
	return bytes.Equal(M.segmentFlag(), []byte(protoFlag))
}

func (M Message) ResponsePayload(message Message) Message {
	M[M.segmentFlagLen()] = protoResponseType
	m := M[0 : M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen()]
	m = append(m, message.Byte()...)
	return m
}

func TestProtocol(t *testing.T) {
	var delim byte = '\r'
	message := Message("caiwenhui, 你好啊")
	uuid := uuid2.NewV4()
	buf := make([]byte, 0, 0)
	// type
	buf = append(buf, []byte(protoFlag)...)
	buf = append(buf, protoNormalType)
	// uuid
	buf = append(buf, uuid.Bytes()...)
	// ttl
	ttl := time.Now().Unix()
	timeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBuf, uint64(ttl))
	buf = append(buf, timeBuf...)
	// content
	buf = append(buf, append(message, delim).Byte()...)

	//fmt.Println(buf)

	recv := Message(buf)
	tmpUUid, _ := recv.segmentUUID()
	if uuid != tmpUUid {
		t.Error("uuid not equal")
	}

	if ttl != recv.segmentTTL() {
		t.Error("ttl not equal")
	}

	if string(append(message.Byte(), delim)) != recv.segmentPayload().String() {
		t.Error("payload not equal")
	}
}
