package mermaidlive

import (
	"net"
)

type UDPClient struct{}

func NewUDPClient() *UDPClient {
	return &UDPClient{}
}

func (c *UDPClient) Send(addr string, msg []byte) error {
	udpAddr, err := net.ResolveUDPAddr("udp6", addr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}

	_, err = conn.Write(msg)
	if err != nil {
		return err
	}
	return nil
}
