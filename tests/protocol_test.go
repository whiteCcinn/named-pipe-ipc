package tests

import (
	"bytes"
	"encoding/binary"
	uuid2 "github.com/satori/go.uuid"
	"testing"
	"time"
)

const (
	protoNormalType   byte = '0'
	protoResponseType byte = '1'
	protoRetranType   byte = '2'
	protoFlag              = "named-pipe-ipc"
)

/**
protocol:
	8byte - 14byte - 1byte - 16byte - 8byte - string
	byteLength - flag - type - uuid  - ttl - content
*/

type Message []byte

func (M Message) String() string {
	return string(M)
}

func (M Message) Byte() []byte {
	return M
}

func (M Message) segmentPackageLengthLen() int {
	return 8
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

func (M Message) segmentPackageLength() int64 {
	return int64(binary.BigEndian.Uint64(M[0:M.segmentPackageLengthLen()]))
}

func (M Message) segmentFlag() (flag []byte) {
	flag = M[M.segmentPackageLengthLen() : M.segmentPackageLengthLen()+M.segmentFlagLen()].Byte()

	return flag
}

func (M Message) segmentType() (t byte) {
	t = M[M.segmentPackageLengthLen()+M.segmentFlagLen() : M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()].Byte()[0]

	return t
}

func (M Message) segmentUUID() (uuid uuid2.UUID, err error) {
	uuid, err = uuid2.FromBytes(M[M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen() : M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()])
	return
}

func (M Message) segmentTTL() (ttl int64) {
	timestamp := M[M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen() : M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen()]
	ttl = int64(binary.BigEndian.Uint64(timestamp))

	return
}

func (M Message) segmentPayload() Message {
	return M[M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen():]
}

func (M Message) Payload() Message {
	return M.segmentPayload()
}

func (M Message) isLegal() bool {
	return bytes.Equal(M.segmentFlag(), []byte(protoFlag))
}

func (M Message) isRetran() bool {
	return M[M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()-1] == protoRetranType
}

func (M Message) changeRetran() {
	M[M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()-1] = protoRetranType
}

func (M Message) ResponsePayload(message Message) Message {
	ma := make([]byte, 0)
	ma = append(ma, M[M.segmentPackageLengthLen():M.segmentPackageLengthLen()+M.segmentFlagLen()+M.segmentTypeLen()+M.segmentUUIDLen()+M.segmentTTLLen()]...)
	ma[M.segmentFlagLen()+M.segmentTypeLen()-1] = protoResponseType
	ma = append(ma, message.Byte()...)
	packageLengthBuf := make([]byte, 8)
	// package-buf's length + delim's length
	// 8 + 1s
	binary.BigEndian.PutUint64(packageLengthBuf, uint64(len(ma)+8+1))
	m := append(make([]byte, 0), packageLengthBuf...)
	m = append(m, ma...)

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

	packageLengthBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(packageLengthBuf, uint64(len(buf)+8))

	protocol := make([]byte, 0, len(packageLengthBuf)+len(buf))
	protocol = append(protocol, packageLengthBuf...)
	protocol = append(protocol, buf...)

	recv := Message(protocol)

	if int64(len(protocol)) != recv.segmentPackageLength() {
		t.Error("package-length not equal")
	}

	if string(recv.segmentFlag()) != protoFlag {
		t.Error("flag not equal")
	}

	if recv.segmentType() != protoNormalType {
		t.Error("type not equal")
	}

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

	newMessage := recv.ResponsePayload(Message("hello world"))

	if newMessage.Payload().String() != "hello world" {
		t.Error("new payload not equal")
	}

	newMessage.changeRetran()
	if !newMessage.isRetran() {
		t.Error("new payload is not retran")
	}

	tmpUUID2, _ := newMessage.segmentUUID()
	if tmpUUid != tmpUUID2 {
		t.Error("uuid2 not equal")
	}
}
