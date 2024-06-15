package mermaidlive

import (
	"errors"
	"log"
	"net"
)

const maxPacketSize = 10000

type UDPServer struct {
	bindAddr string
}

func NewUDPServer(bindAddr string) *UDPServer {
	return &UDPServer{
		bindAddr: bindAddr,
	}
}

func (s *UDPServer) Start() {
	udp, err := net.ListenPacket("udp", s.bindAddr)
	if err != nil {
		log.Fatalf("can't listen on %s/udp: %v", s.bindAddr, err)
	}
	s.handleUDP(udp)
}

func (s *UDPServer) handleUDP(c net.PacketConn) {
	packet := make([]byte, maxPacketSize)

	for {
		n, addr, err := c.ReadFrom(packet)
		if n > 0 {
			log.Printf("received '%s' from %s", string(packet[:n]), addr.String())
		}
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Printf("stopped listening to UDP packets")
				return
			}

			log.Printf("error reading on %s/udp: %v", s.bindAddr, err)
			continue
		}
	}
}
