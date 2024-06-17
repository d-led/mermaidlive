package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/cskr/pubsub/v2"
	"github.com/d-led/mermaidlive"
)

var transpileOnly *bool
var port *string
var countdownDelayString *string

// limits the amount of connected clients
const pubSubChannelCapacity = 1024
const defaultCountdownDelay = 800 * time.Millisecond

func main() {
	migrate()

	flag.Parse()

	if *transpileOnly {
		mermaidlive.Refresh()
		log.Println("exiting")
		return
	}

	eventPublisher := pubsub.New[string, mermaidlive.Event](pubSubChannelCapacity /* to do: unbounded mailbox*/)

	if !mermaidlive.DoEmbed {
		mermaidlive.Refresh()
		watcher := mermaidlive.StartWatching(eventPublisher)
		defer watcher.Close()
	}

	if !versioninfo.DirtyBuild {
		log.Println("Revision:", versioninfo.Revision)
	}

	countdownDelay := getCountdownDelay()

	server := mermaidlive.NewServerWithOptions(
		*port,
		eventPublisher,
		mermaidlive.GetFS(),
		countdownDelay,
	)
	server.Run(*port)
}

func init() {
	transpileOnly = flag.Bool("transpile", false, "transpile only and exit")
	port = flag.String("port", "8080", "port to run on")
	if portFromEnv, ok := os.LookupEnv("PORT"); ok {
		log.Println("Overriding the PORT via the environment variable")
		*port = portFromEnv
	}
	countdownDelayString = flag.String("delay", "800ms", "countdown delay")
}

func getCountdownDelay() time.Duration {
	d, err := time.ParseDuration(*countdownDelayString)
	if err != nil {
		log.Printf("provided countdown delay ignored: '%s', using default: '%v'", *countdownDelayString, defaultCountdownDelay)
		return defaultCountdownDelay
	}
	log.Printf("countdown delay: %v", d)
	return d
}

func migrate() {
	oldFilename := os.Getenv("OLD_COUNTER_FILENAME")
	newFilename := os.Getenv("COUNTER_FILENAME")
	newDir := os.Getenv("COUNTER_DIRECTORY")
	if oldFilename == "" || newFilename == "" || newDir == "" {
		log.Println("nothing to migrate!")
		return
	}
	err := os.MkdirAll(newDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = copyFile(oldFilename, newFilename)
	if err != nil {
		log.Println("original counter file already copied")
		return
	}
	log.Println("copied counter file")
	err = os.Remove(oldFilename)
	if err != nil {
		panic(err)
	}
	log.Println("migrations complete")
}

func copyFile(src, dst string) error {
	text, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, text, 0644)
}
