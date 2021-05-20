package named_pipe_ipc

import (
	"bufio"
	"context"
	"io"
	"os"
	"syscall"
	"time"
)

const (
	defaultUmask             = 0600
	defaultNamedPipeForRead  = "golang.pipe.1.r"
	defaultNamedPipeForWrite = "golang.pipe.1.w"
)

var defaultOption = &options{
	defaultNamedPipeForRead,
	defaultNamedPipeForWrite,
}

type options struct {
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

type Message []byte

func (M Message) String() string {
	return string(M)
}

func (M Message) Byte() []byte {
	return M
}

type Context struct {
	out  chan Message
	role RoleType

	rPipe *os.File
	wPipe *os.File
	br    *bufio.Reader
	bw    *bufio.Writer

	context           context.Context
	chroot            string
	namedPipeForRead  string
	namedPipeForWrite string
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

// OpenPipeFile
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
func OpenPipeFile(nctx *Context) (err error) {
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

	for _, o := range opts {
		o.apply(defaultOption)
	}

	nctx := &Context{
		role:              role,
		chroot:            chroot,
		namedPipeForRead:  defaultOption.namedPipeForRead,
		namedPipeForWrite: defaultOption.namedPipeForWrite,
	}

	if nctx.role == C {
		nctx.namedPipeForWrite = defaultOption.namedPipeForRead
		nctx.namedPipeForRead = defaultOption.namedPipeForWrite
	}

	nctx.context = ctx
	nctx.out = make(chan Message, 10)

	err := createFifo(nctx)
	if err != nil {
		return nil, err
	}

	err = OpenPipeFile(nctx)
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
func (nctx *Context) Recv(block bool, delim byte) (Message, error) {
	if nctx.role == S {
		if !block {
			if len(nctx.out) == 0 {
				time.Sleep(1 * time.Millisecond)
				return nil, NoMessage{}
			}
		}
		return <-nctx.out, nil
	} else {
		bf, err := nctx.br.ReadBytes(delim)
		if err != nil && err != io.EOF {
			return nil, err
		}
		return bf, nil
	}
}

// Listen Message
func (nctx *Context) Listen(delim byte) error {
	var err error
	var bf Message
	for err == nil {
		select {
		case <-nctx.context.Done():
			close(nctx.out)
			return nil
		default:
		}

		bf, err = nctx.br.ReadBytes(delim)
		if err != nil && err != io.EOF {
			return err
		}

		nctx.out <- bf
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
