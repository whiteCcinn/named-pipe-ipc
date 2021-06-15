package main

import (
	"context"
	named_pipe_ipc "github.com/whiteCcinn/named-pipe-ipc"
	"log"
	"time"
)

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	// use pipe-IPC
	nctx, err := named_pipe_ipc.NewContext(ctx, "./", named_pipe_ipc.S)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		go func() {
			for {
				msg, err := nctx.Recv(true)
				if err != nil && err.Error() != named_pipe_ipc.PipeClosedMessage {
					log.Fatal(err)
				}

				log.Println("from clint", msg)

				//time.Sleep(4 * time.Second)

				_, err = nctx.Send(named_pipe_ipc.Message("send to client"))
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
		time.Sleep(1 * time.Second)
		if err := nctx.Close(); err != nil {
			log.Println(err)
		}
		break
	}
}
