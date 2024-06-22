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

func getCounterDirectory() string {
	if counterDirectory, ok := os.LookupEnv("COUNTER_DIRECTORY"); ok {
		return counterDirectory
	}
	return "."
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

func getPrivateReplicaId() string {
	if id, ok := os.LookupEnv("FLY_MACHINE_ID"); ok && len(id) > 5 {
		return id
	}
	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}
	return "local"
}
