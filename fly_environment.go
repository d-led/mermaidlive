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

func getFlyPrivateIP() string {
	return os.Getenv("FLY_PRIVATE_IP")
}

func getCounterFilename() string {
	if counterFilename, ok := os.LookupEnv("COUNTER_FILENAME"); ok {
		return counterFilename
	}
	return "local.gcounter"
}

func GetReplicasEvent(count int) Event {
	return NewEventWithParam("ReplicasActive",
		fmt.Sprintf("%d (you are on '%s')", count, getFlyPublicReplicaId()))
}

func getFlyPublicReplicaId() string {
	if id, ok := os.LookupEnv("FLY_MACHINE_ID"); ok && len(id) > 5 {
		// do not show the full machine ID
		return id[len(id)-5:]
	}
	return "local"
}
