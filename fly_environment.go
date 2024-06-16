package mermaidlive

import (
	"fmt"
	"os"
)

func getFlyRegion() string {
	if region, ok := os.LookupEnv("FLY_REGION"); ok {
		return region
	}
	return "local"
}

func getFlyPeersDomain() string {
	if appName, ok := os.LookupEnv("FLY_APP_NAME"); ok {
		return appName + ".internal"
	}
	return ""
}

func getCounterFilename() string {
	if counterFilename, ok := os.LookupEnv("COUNTER_FILENAME"); ok {
		return counterFilename
	}
	return "local.gcounter"
}

func GetReplicasEvent(count int) Event {
	return NewEventWithParam("ReplicasActive",
		fmt.Sprintf("%d (you are on '%s')", count, getPublicReplicaId()))
}

func getPublicReplicaId() string {
	if id, ok := os.LookupEnv("FLY_MACHINE_ID"); ok && len(id) > 5 {
		// do not show the full machine ID
		return id[len(id)-5:]
	}
	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}
	return "local"
}
