package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
	"time"
)

func main() {
	//tctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	//nctx, err := named_pipe_ipc.NewContext(tctx, "./", named_pipe_ipc.C)
	nctx, err := named_pipe_ipc.NewContext(context.Background(), "./", named_pipe_ipc.C)
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now().String()

	nctx.Send(named_pipe_ipc.Message("nihao-" + t))
	for {
		dsm, err := nctx.Recv(false)
		if err != nil && err.Error() != named_pipe_ipc.NoMessageMessage {
			log.Fatal(err)
		}

		if dsm == nil && err != nil && (err.Error() == named_pipe_ipc.NoMessageMessage && err.Error() == named_pipe_ipc.PipeClosedMessage) {
			time.Sleep(500 * time.Millisecond)
			log.Println("next recv...")
			continue
		}

		log.Println("from server", dsm.Payload())
		break
	}
}
