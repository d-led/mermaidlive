package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/cskr/pubsub/v2"
	"github.com/gin-gonic/gin"
)

const uiSrc = "ui-src"
const dist = "dist"

var doEmbed = false
var transpileOnly *bool
var port *string

//go:embed dist/*
var embeddedDist embed.FS

func main() {
	flag.Parse()

	if *transpileOnly {
		refresh()
		log.Println("exiting")
		return
	}

	eventPublisher := pubsub.New[string, Event](1 /* to do: unbounded mailbox*/)

	if !doEmbed {
		refresh()
		watcher := startWatching(eventPublisher)
		defer watcher.Close()
	}

	server := NewServerWithOptions(*port, eventPublisher, getFS())
	server.Run()
}

func getFS() http.FileSystem {
	if doEmbed {
		return getEmbeddedFS()
	}
	return getLocalFS()
}

func getLocalFS() http.FileSystem {
	return http.FS(os.DirFS(dist))
}

func getEmbeddedFS() http.FileSystem {
	sub, err := fs.Sub(embeddedDist, dist)

	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

func timestampEvent() gin.H {
	return gin.H{
		"timestamp": now(),
	}
}

func streamOneEvent(c *gin.Context, event any) {
	c.JSON(http.StatusOK, event)
	c.String(http.StatusOK, "\n")
	c.Writer.(http.Flusher).Flush()
}

func init() {
	transpileOnly = flag.Bool("transpile", false, "transpile only and exit")
	port = flag.String("port", "8080", "port to run on")
	if portFromEnv, ok := os.LookupEnv("PORT"); ok {
		*port = portFromEnv
	}
}
