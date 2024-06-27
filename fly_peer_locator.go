package mermaidlive

import (
	"fmt"
	"net"
	"os"
)

type FlyPeerLocator struct {
	flyDiscoveryDomainName string
}

func NewFlyPeerLocator(flyDiscoveryDomainName string) *FlyPeerLocator {
	return &FlyPeerLocator{
		flyDiscoveryDomainName: flyDiscoveryDomainName,
	}
}

func (l *FlyPeerLocator) GetPeers() ([]string, int, error) {
	addrs, err := net.LookupHost(l.flyDiscoveryDomainName)
	if err != nil {
		return nil, 1 /*this one*/, fmt.Errorf("DNS resolution error: %v", err)
	}
	myIp := getFlyPrivateIP()
	peers := []string{}
	for _, peer := range addrs {
		if peer != myIp {
			peers = append(peers, peer)
		}
	}
	return peers, len(addrs), nil
}

func (l *FlyPeerLocator) GetMyIP() string {
	return getFlyPrivateIP()
}

func getFlyPrivateIP() string {
	return os.Getenv("FLY_PRIVATE_IP")
}
