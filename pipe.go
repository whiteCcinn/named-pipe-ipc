package named_pipe_ipc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	uuid2 "github.com/satori/go.uuid"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	defaultUmask             = 0600
	defaultDelim             = '\n'
	defaultNamedPipeForRead  = "golang.pipe.1.r"
	defaultNamedPipeForWrite = "golang.pipe.1.w"
)

const (
	protoNormalType   byte = '0'
	protoResponseType byte = '1'
	protoRetranType   byte = '2'
	protoFlag              = "named-pipe-ipc"
)

var defaultOption = &options{
	defaultDelim,
	defaultNamedPipeForRead,
	defaultNamedPipeForWrite,
}

type options struct {
	delim             byte
	namedPipeForRead  string
	namedPipeForWrite string
}

type Option interface {
	apply(*options)
}

type OptionsFunc func(o *options)

func (f OptionsFunc) apply(o *options) {
	f(o)
}

func WithNamedPipeForRead(name string) Option {
	return OptionsFunc(func(o *options) {
		o.namedPipeForRead = name
	})
}

func WithNamedPipeForWrite(name string) Option {
	return OptionsFunc(func(o *options) {
		o.namedPipeForWrite = name
	})
}

func WithDelim(delim byte) Option {
	return OptionsFunc(func(o *options) {
		o.delim = delim
	})
}

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

type Context struct {
	out  chan Message
	role RoleType

	delim byte
	rPipe *os.File
	wPipe *os.File
	br    *bufio.Reader
	bw    *bufio.Writer

	context           context.Context
	chroot            string
	namedPipeForRead  string
	namedPipeForWrite string

	clientID uuid2.UUID
}

func createFifo(nctx *Context) (err error) {
	if ex, err := Exists(nctx.namedPipeForReadFullPath()); err != nil {
		return err
	} else {
		if !ex {
			err = syscall.Mkfifo(nctx.namedPipeForReadFullPath(), defaultUmask)
			if err != nil {
				return err
			}
		}
	}
	if ex, err := Exists(nctx.namedPipeForWriteFullPath()); err != nil {
		return err
	} else {
		if !ex {
			err = syscall.Mkfifo(nctx.namedPipeForWriteFullPath(), defaultUmask)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// openPipeFile
//
// Why use os.RDWR? not os.RD_ONLY or os.WD_ONLY ?
//
// When the FIFO is turned on, the non-blocking flag (O_NONBLOCK) has the following effects:
// i. O_NONBLOCK is not specified (i.e. open has no bit or O_NONBLOCK).
// 	2. When a FIFO is opened in read-only mode, the FIFO is blocked until a process opens the FIFO for writing
// 	3. When the FIFO is opened in write-only mode, it is blocked until a process opens the FIFO for reads.
// 	4. When the FIFO is opened in read-only, write-only mode, it blocks. When the read function is called to read data from the FIFO, the read function also blocks.
// 	4, Call the write function to write data to the FIFO, and write will block when the buffer is full.
// 	5, communication process if the writing process first quit, then call the read function to read data from the FIFO does not block; If the writing process starts again, the read function is called to read data from the FIFO.
// 	6. During the communication process, when the reader process exits and the writer process writes data to the named pipe, the writer process will also exit (receiving SIGPIPE signal).
// If no process has opened a FIFO for write, read - only open succeeds, and open is not blocked.
// ii. Specify O_NONBLOCK(that is, open bit or O_NONBLOCK)
// 	1. If no process has opened a FIFO for read, writing only open will return -1.
// 	2. Named pipes do not block when reading data.
//  3. During the communication process, when the reader process exits and the writer process writes data to the named pipe, the writer process will also exit (receiving SIGPIPE signal).
func openPipeFile(nctx *Context) (err error) {
	nctx.rPipe, err = os.OpenFile(nctx.namedPipeForReadFullPath(), os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return err
	}
	nctx.wPipe, err = os.OpenFile(nctx.namedPipeForWriteFullPath(), os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	nctx.br = bufio.NewReader(nctx.rPipe)
	nctx.bw = bufio.NewWriter(nctx.wPipe)

	return nil
}

func NewContext(ctx context.Context, chroot string, role RoleType, opts ...Option) (*Context, error) {
	if !IsDir(chroot) {
		return nil, NotDirectory{}
	}

	if !strings.HasSuffix(chroot, "/") {
		chroot += "/"
	}

	for _, o := range opts {
		o.apply(defaultOption)
	}

	nctx := &Context{
		role:              role,
		chroot:            chroot,
		delim:             defaultOption.delim,
		namedPipeForRead:  defaultOption.namedPipeForRead,
		namedPipeForWrite: defaultOption.namedPipeForWrite,
	}

	if nctx.role == C {
		nctx.namedPipeForWrite = defaultOption.namedPipeForRead
		nctx.namedPipeForRead = defaultOption.namedPipeForWrite

		for {
			nctx.clientID = uuid2.NewV4()
			if !bytes.Contains(nctx.clientID.Bytes(), []byte{nctx.delim}) {
				break
			}
		}
	}

	nctx.context = ctx
	nctx.out = make(chan Message, 10)

	if nctx.role == S {
		err := createFifo(nctx)
		if err != nil {
			return nil, err
		}
	}

	err := openPipeFile(nctx)
	if err != nil {
		return nil, err
	}

	return nctx, nil
}

func (nctx *Context) namedPipeForReadFullPath() string {
	return nctx.chroot + nctx.namedPipeForRead
}

func (nctx *Context) namedPipeForWriteFullPath() string {
	return nctx.chroot + nctx.namedPipeForWrite
}

func (nctx *Context) NamedPipeForRead() string {
	return nctx.namedPipeForRead
}

func (nctx *Context) NamedPipeForWrite() string {
	return nctx.namedPipeForWrite
}

func (nctx *Context) Chroot() string {
	return nctx.chroot
}

// Send Message
//
// This API should work best with Write, but since most people are web developers
// the send()/ recv() combination is more acceptable
func (nctx *Context) Send(message Message) (int, error) {
	if nctx.role == S {
		if !message.isLegal() {
			return 0, new(MessageNotLegal)
		}
		return nctx.directlySend(append(message, nctx.delim))
	}
	buf := make([]byte, 0, 0)
	// flag
	buf = append(buf, []byte(protoFlag)...)
	// type
	buf = append(buf, protoNormalType)
	// uuid
	buf = append(buf, nctx.clientID.Bytes()...)
	// ttl 30 second
	ttl := time.Now().Unix() + 10
	timeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBuf, uint64(ttl))
	buf = append(buf, timeBuf...)
	// content
	buf = append(buf, append(message, nctx.delim).Byte()...)
	// package length
	packageLengthBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(packageLengthBuf, uint64(len(buf)+8))

	protocol := make([]byte, 0, len(packageLengthBuf)+len(buf))
	protocol = append(protocol, packageLengthBuf...)
	protocol = append(protocol, buf...)

	nn, err := nctx.bw.Write(protocol)
	if err != nil {
		return 0, nil
	}
	err = nctx.bw.Flush()
	if err != nil {
		return 0, err
	}

	return nn, nil
}

func (nctx *Context) directlySend(message Message) (int, error) {
	nn, err := nctx.bw.Write(message)
	if err != nil {
		return 0, nil
	}
	err = nctx.bw.Flush()
	if err != nil {
		return 0, err
	}

	return nn, nil
}

// Recv Message
//
// This API should work best with Read, but since most people are web developers
// the send()/ recv() combination is more acceptable
func (nctx *Context) Recv(block bool) (Message, error) {
	if nctx.role == S {
		if !block {
			if len(nctx.out) == 0 {
				time.Sleep(1 * time.Millisecond)
				return nil, NoMessage{}
			}
		}

		for {
			select {
			case <-nctx.context.Done():
				return nil, nctx.context.Err()
			case msg := <-nctx.out:
				if msg == nil {
					return msg, Closed{}
				}

				if msg.isRetran() {
					_, err := nctx.Send(msg)
					if err != nil {
						return nil, err
					}
					continue
				}
				return msg, nil
			}
		}
	} else {
		var (
			bf  Message
			err error
		)

		ok := make(chan bool, 1)

		go func() {
		ResetReadBytes:
			var bfs = make([]Message, 0)
			var expectLength int64 = 0
		ReadBytes:
			bf, err = nctx.br.ReadBytes(nctx.delim)
			if err != nil && err != io.EOF {
				if pe, ok := err.(*os.PathError); ok {
					if pe.Err == os.ErrClosed {
						bf = nil
						err = Closed{}
						return
					}
				}

				bf = nil
				return
			}

			if bf != nil {
				if expectLength == 0 {
					expectLength = bf.segmentPackageLength()
					bfs = append(bfs, bf)
				} else {
					bfs = append(bfs, bf)
				}

				var sum int64 = 0
				for _, tbf := range bfs {
					sum += int64(len(tbf.Byte()))
				}

				if sum != expectLength {
					if sum > expectLength {
						goto ResetReadBytes
					}
					goto ReadBytes
				}

				var buf Message
				for _, tbf := range bfs {
					buf = append(buf, tbf.Byte()...)
				}

				message := buf
				uuid, err := message.segmentUUID()
				if err != nil {
					return
				}
				if message.segmentTTL() < time.Now().Unix() {
					// drop message
					goto ReadBytes
				}

				if uuid != nctx.clientID {
					// resend message to server
					message.changeRetran()
					_, err = nctx.directlySend(message)
					if err != nil {
						return
					}
					goto ReadBytes
				}

				// read not include nctx.delim
				bf = buf[:len(buf)-1]
				ok <- true
			}
		}()

		for {
			select {
			case <-nctx.context.Done():
				err = nctx.close()
				return nil, HybridError{nctx.context.Err(), err}
			case <-ok:
				return bf, nil
			}
		}
	}
}

// Listen Message
func (nctx *Context) Listen() error {
	var err error
	var bf Message
	var bfs = make([]Message, 0)
	var expectLength int64 = 0
	for err == nil {
		select {
		case <-nctx.context.Done():
			close(nctx.out)
			return nil
		default:
			if nctx.context.Err() != nil {
				close(nctx.out)
				return nctx.context.Err()
			}
		}

		bf, err = nctx.br.ReadBytes(nctx.delim)
		if err != nil && err != io.EOF {
			if pe, ok := err.(*os.PathError); ok {
				if pe.Err == os.ErrClosed {
					return nil
				}
			}

			return err
		}

		if bf != nil {
			if expectLength == 0 {
				expectLength = bf.segmentPackageLength()
				bfs = append(bfs, bf)
			} else {
				bfs = append(bfs, bf)
			}

			var sum int64 = 0
			for _, tbf := range bfs {
				sum += int64(len(tbf.Byte()))
			}

			if sum != expectLength {
				continue
			}

			var buf Message
			for _, tbf := range bfs {
				buf = append(buf, tbf.Byte()...)
			}

			// read not include nctx.delim
			buf = buf[:len(buf)-1]
			nctx.out <- buf

			expectLength = 0
			bfs = make([]Message, 0)
		}
	}

	return nil
}

func (nctx *Context) Close() error {
	if err := nctx.close(); err != nil {
		return err
	}

	if err := nctx.removeFiFo(); err != nil {
		if pe, ok := err.(*os.PathError); ok {
			if pe.Err != os.ErrClosed {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (nctx *Context) close() error {
	if nctx.rPipe != nil {
		if err := nctx.rPipe.Close(); err != nil {
			if pe, ok := err.(*os.PathError); ok {
				if pe.Err != os.ErrClosed {
					return err
				}
			} else {
				return err
			}
		}
	}

	if nctx.wPipe != nil {
		if err := nctx.wPipe.Close(); err != nil {
			if pe, ok := err.(*os.PathError); ok {
				if pe.Err != os.ErrClosed {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

func (nctx *Context) removeFiFo() error {
	if IsFile(nctx.namedPipeForWriteFullPath()) {
		err := os.Remove(nctx.namedPipeForWriteFullPath())
		if err != nil {
			return err
		}
	}

	if IsFile(nctx.namedPipeForReadFullPath()) {
		err := os.Remove(nctx.namedPipeForReadFullPath())
		if err != nil {
			return err
		}
	}

	return nil
}

func Exists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			if s.Mode() != os.ModeNamedPipe {
				return true, AlreadyExistButNotNamedPipe{}
			} else {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func IsFile(path string) bool {
	return !IsDir(path)
}
