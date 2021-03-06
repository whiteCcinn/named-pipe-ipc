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
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	// use pipe-IPC
	nctx, err := named_pipe_ipc.NewContext(ctx, "./", named_pipe_ipc.S)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		go func() {
			for {
				dsm, err := nctx.Recv(true)
				if err != nil && err.Error() != named_pipe_ipc.PipeClosedMessage {
					log.Fatal(err)
				}

				log.Println("from clint", dsm.Payload())

				_, err = nctx.Send(dsm.ResponsePayload(named_pipe_ipc.Message("send to client")))
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
		if err := nctx.Close(); err != nil {
			log.Println(err)
		}
		break
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

	_, err = nctx.Send(named_pipe_ipc.Message("nihao"))
	if err != nil {
		log.Fatal(err)
	}
	msg, err := nctx.Recv(true)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("from server", msg)
}
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

## Stress Test

```
## server
go run example/noblock_server.go

# other window
i="0";while [ $i -lt 10 ]; do nohup go run example/noblock_client.go > output.$i 2>&1 &;i=$[$i+1];done

# see the out.* content
```

## Projects using

- [whiteCcinn/daemon: Go supervisor daemon module, similar to the Erlang | python's supervisor, assist you in better monitor your business processes 🚀](https://github.com/whiteCcinn/daemon)
