package mermaidlive

import (
	"fmt"
	"os"

	"github.com/evanw/esbuild/pkg/api"
)

const Topic = "events"
const InternalTopic = "internal-events"
const NewConnectionsCounter = "newconnections"
const StartedConnectionsCounter = "started-connections"
const ClosedConnectionsCounter = "closed-connections"
const VisitorJoinedEvent = "VisitorJoined"
const VisitorLeftEvent = "VisitorLeft"
const VisitorsActiveEvent = "VisitorsActive"
const TotalVisitorsEvent = "TotalVisitors"
const TotalClusterVisitorsActiveEvent = "TotalClusterVisitorsActive"
const SourceReplicaIdKey = "Source-Replica-Id"

type PeerLocator interface {
	GetPeers() ([]string, int, error)
}

var DoEmbed = false

func crashOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func handleErrors(errors []api.Message) {
	for _, msg := range errors {
		if msg.Location != nil {
			fmt.Printf(
				"%s:%v:%v: %s\n",
				msg.Location.File,
				msg.Location.Line,
				msg.Location.Column,
				msg.Text,
			)
		} else {
			fmt.Println(msg)
		}
	}

	if len(errors) > 0 {
		os.Exit(1)
	}
}
