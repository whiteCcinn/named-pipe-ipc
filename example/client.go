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
