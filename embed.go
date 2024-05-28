//go:build embed
// +build embed

package main

import "log"

func init() {
	log.Println("using embedded resources")
	doEmbed = true
}
