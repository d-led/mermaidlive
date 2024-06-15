package mermaidlive

import "os"

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
