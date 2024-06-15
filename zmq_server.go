package mermaidlive

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-zeromq/zmq4"
)

type ZmqServer struct {
	bindAddr string
	socket   zmq4.Socket
}

func NewZmqServer(bindAddr string) *ZmqServer {
	socket := zmq4.NewPull(context.Background())
	return &ZmqServer{
		bindAddr: bindAddr,
		socket:   socket,
	}
}

func (s *ZmqServer) Start() {
	err := s.socket.Listen(s.bindAddr)
	if err != nil {
		log.Fatalf("Could not start listening at %s: %v", s.bindAddr, err)
	}
	for {
		msg, err := s.socket.Recv()
		if err != nil {
			fmt.Printf("Could not receive at %s: %v", s.bindAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Printf("Received %s\n", string(msg.Bytes()))
	}
}
