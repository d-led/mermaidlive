package mermaidlive

import (
	"context"
	"fmt"

	"github.com/go-zeromq/zmq4"
)

type ZmqClient struct {
	addr   string
	socket zmq4.Socket
}

func NewZmqClient(addr string) *ZmqClient {
	socket := zmq4.NewPush(context.Background())
	return &ZmqClient{
		addr:   addr,
		socket: socket,
	}
}

func (c *ZmqClient) Connect() error {
	err := c.socket.Dial(c.addr)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %v", c.addr, err)
	}
	return nil
}

func (c *ZmqClient) Send(msg []byte) error {
	return c.socket.Send(zmq4.NewMsg(msg))
}

func (c *ZmqClient) Close() {
	c.socket.Close()
}
