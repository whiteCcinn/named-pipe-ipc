package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
	"time"
)

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	//ctx := context.Background()
	nctx, err := named_pipe_ipc.NewContext(ctx, "./", named_pipe_ipc.C)
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
