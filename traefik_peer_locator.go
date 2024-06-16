package mermaidlive

type TraefikPeerLocator struct {
	traetraefikServicesUrl string
}

func NewTraefikPeerLocator(traetraefikServicesUrl string) *TraefikPeerLocator {
	return &TraefikPeerLocator{
		traetraefikServicesUrl: traetraefikServicesUrl,
	}
}

func (*TraefikPeerLocator) GetPeers() ([]string, int, error) {
	return []string{}, 1 /*this one*/, nil
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
