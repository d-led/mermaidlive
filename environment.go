package mermaidlive

import "os"

func getRegion() string {
	if region, ok := os.LookupEnv("FLY_REGION"); ok {
		return region
	}
	return "local"
}

func getPeersDomain() string {
	if appName, ok := os.LookupEnv("FLY_APP_NAME"); ok {
		return appName + ".internal"
	}
	return ""
}
