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
				dsm, err := nctx.Recv(false)
				if err != nil && (err.Error() != named_pipe_ipc.NoMessageMessage && err.Error() != named_pipe_ipc.PipeClosedMessage) {
					log.Fatal(err)
				}

				if dsm == nil {
					//log.Println("next recv...")
					continue
				}

				log.Println(dsm.Payload())
				//errMessage := named_pipe_ipc.Message("send to client")
				response := dsm.ResponsePayload(named_pipe_ipc.Message("send to client"))

				//_, err = nctx.Send(errMessage)
				_, err = nctx.Send(response)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(1 * time.Second)
			}
		}()

		err = nctx.Listen()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		log.Println("I am server")
		time.Sleep(60 * time.Second)
		if err := nctx.Close(); err != nil {
			log.Println(err)
		}
		break
	}
}
