# named_pipe_ipc
🚀 With the aid of a named pipe to achieve fast communication with the process

## Features

- named-pipe-ipc(Not limited to parent-child processes)
- full-duplex communication

## Installation

```shell
go get github.com/whiteCcinn/named-pipe-ipc
```

## Usage

### server

```go
package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
	"time"
)

func main() {
	// use pipe-IPC
	nctx, err := named_pipe_ipc.NewContext(context.Background(), "./", named_pipe_ipc.S)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		go func() {
			for {
				msg, err := nctx.Recv(true)
				if err != nil {
					log.Fatal(err)
				}

				log.Println("from clint", msg)

				_, err = nctx.Send(named_pipe_ipc.Message("send to client\n"))
				if err != nil {
					log.Fatal(err)
				}
			}
		}()

		err = nctx.Listen()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		log.Println("I am server")
		time.Sleep(5 * time.Second)
	}
}

```

### client

```go
package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
)

func main() {
	nctx, err := named_pipe_ipc.NewContext(context.Background(), "./", named_pipe_ipc.C)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = nctx.Send(named_pipe_ipc.Message("nihao"))
	msg, err := nctx.Recv(true)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("from server", msg)
}
```

## More Example
- [example](https://github.com/whiteCcinn/named-pipe-ipc/tree/main/example)

## Example Log

```go
root@0140ee5d78cf:/www/example# go run server.go
2021/05/20 07:47:05 I am server
2021/05/20 07:47:10 I am server
2021/05/20 07:47:12 from clint nihao

2021/05/20 07:47:12 pass
2021/05/20 07:47:15 I am server

# other tty
root@0140ee5d78cf:/www/example# go run client.go
2021/05/20 07:47:12 from server send to client
```

## Projects using

- [whiteCcinn/daemon: Go supervisor daemon module, similar to the Erlang | python's supervisor, assist you in better monitor your business processes 🚀](https://github.com/whiteCcinn/daemon)