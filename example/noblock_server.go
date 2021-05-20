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
				msg, err := nctx.Recv(false, '\n')
				if err != nil && err.Error() != named_pipe_ipc.NoMessageMessage {
					log.Fatal(err)
				}

				if msg == nil {
					log.Println("next recv...")
					continue
				}

				log.Println("from clint", msg)

				_, err = nctx.Send(named_pipe_ipc.Message("send to client\n"))
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(5 * time.Second)
			}
		}()

		err = nctx.Listen('\n')
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		log.Println("I am server")
		time.Sleep(5 * time.Second)
	}
}
