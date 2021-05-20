package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
	"time"
)

func main() {
	nctx, err := named_pipe_ipc.NewContext(context.Background(), "./", named_pipe_ipc.C)
	if err != nil {
		log.Fatal(err)
	}

	nctx.Send(named_pipe_ipc.Message("nihao\n"))
	for {
		msg, err := nctx.Recv(false, '\n')
		if err != nil && err.Error() != named_pipe_ipc.NoMessageMessage {
			log.Fatal(err)
		}

		if msg == nil && err != nil && err.Error() == named_pipe_ipc.NoMessageMessage {
			time.Sleep(500 * time.Millisecond)
			log.Println("next recv...")
			continue
		}

		log.Println("from server", msg)
		break
	}
}
