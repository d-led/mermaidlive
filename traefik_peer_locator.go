package mermaidlive

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type TraefikPeerLocator struct {
	traetraefikServicesUrl string
	client                 *resty.Client
}

func NewTraefikPeerLocator(traetraefikServicesUrl string) *TraefikPeerLocator {
	client := resty.New().
		SetTimeout(15 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &TraefikPeerLocator{
		traetraefikServicesUrl: traetraefikServicesUrl,
		client:                 client,
	}
}

func (l *TraefikPeerLocator) GetPeers() ([]string, int, error) {
	myIp, err := getMyIPv4()
	if err != nil {
		return nil, 1 /*this one*/, err
	}
	replicas := TraefikReplicas{}
	resp, err := l.client.R().
		SetResult(&replicas).
		Get(l.traetraefikServicesUrl)

	if err != nil {
		return nil, 1 /*this one*/, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, 1 /*this one*/, fmt.Errorf("trafik returned [%d]: %s", resp.StatusCode(), resp.String())
	}

	res := []string{}
	for server, status := range replicas.ServerStatus {
		ip, err := getIPOf(server)
		if err != nil {
			log.Printf("Could not extract IP from '%s'", server)
		}
		if strings.ToLower(status) != "up" {
			// no need to talk to replicas that are not up
			continue
		}
		if ip != myIp {
			res = append(res, ip)
		}
	}

	return res, len(replicas.LoadBalancer.Servers), nil
}

func (l *TraefikPeerLocator) GetMyIP() string {
	myIP, err := getMyIPv4()
	if err != nil {
		log.Printf("Could not get my ip: %v", err)
		return ""
	}
	return myIP
}

func getIPOf(replicaUrl string) (string, error) {
	u, err := url.Parse(replicaUrl)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", err
	}
	ip, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return ip, nil
}

func getMyIPv4() (string, error) {
	myHostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	ips, err := net.LookupIP(myHostname)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}
	return "", errors.New("could not find my IPv4")
}

// via https://app.quicktype.io/
type TraefikReplicas struct {
	LoadBalancer LoadBalancer `json:"loadBalancer"`
	Status       string       `json:"status"`
	UsedBy       []string     `json:"usedBy"`
	ServerStatus ServerStatus `json:"serverStatus"`
	Name         string       `json:"name"`
	Provider     string       `json:"provider"`
	Type         string       `json:"type"`
}

type LoadBalancer struct {
	Servers            []TraefikServerResponse `json:"servers"`
	PassHostHeader     bool                    `json:"passHostHeader"`
	ResponseForwarding ResponseForwarding      `json:"responseForwarding"`
}

type ResponseForwarding struct {
	FlushInterval string `json:"flushInterval"`
}

type TraefikServerResponse struct {
	URL string `json:"url"`
}

type ServerStatus map[string]string
