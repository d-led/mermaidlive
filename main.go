package main

import (
	"context"
	"embed"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cskr/pubsub/v2"
	"github.com/gin-gonic/gin"
)

const uiSrc = "ui-src"
const dist = "dist"

var doEmbed = false
var transpileOnly *bool

//go:embed dist/*
var embeddedDist embed.FS

func main() {
	flag.Parse()

	refresh()

	if *transpileOnly {
		log.Println("exiting")
		return
	}
	if !doEmbed {
		watcher := startWatching()
		defer watcher.Close()
	}
	r := gin.Default()
	fs := getFS()
	r.StaticFS("/ui/", fs)

	eventPublisher := pubsub.New[string, Event](1 /* to do: unlimited mailbox*/)

	fsm := NewAsyncFSM(eventPublisher)

	r.POST("/commands/:command", func(ctx *gin.Context) {
		command := ctx.Param("command")
		log.Println("command called:", command)
		switch command {
		case "waiting":
			fsm.events.Pub(NewEventWithReason("CommandRejected", "wait or cancel, please"), topic)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"result":  "rejected",
				"command": command,
				"reason":  "wait or cancel, please",
			})
			return
		case "working":
			fsm.StartWork()
			ctx.JSON(http.StatusOK, gin.H{})
			return
		case "aborting":
			fsm.CancelWork()
			ctx.JSON(http.StatusOK, gin.H{})
			return
		default:
			msg := "unknown command: '" + command + "'"
			fsm.events.Pub(NewEventWithReason("CommandRejected", msg), topic)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"result":  "rejected",
				"command": command,
				"reason":  msg,
			})
			return
		}
	})

	r.GET("/events", func(c *gin.Context) {
		ctx := c.Request.Context()
		tickContext, stopTicking := context.WithCancel(ctx)
		closeNotify := c.Writer.CloseNotify()
		ticks := make(chan gin.H, 1)
		myEvents := eventPublisher.Sub(topic)
		defer eventPublisher.Unsub(myEvents, topic)

		// ticks are private channels and goroutines per connection
		go tick(tickContext, ticks)

		// callback returns false on end of processing
		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				log.Printf("client disconnected")
				stopTicking()
				return false

			case <-closeNotify:
				log.Printf("client closed the connection")
				stopTicking()
				return false

			case event := <-myEvents:
				c.JSON(http.StatusOK, event)
				c.String(http.StatusOK, "\n")
				c.Writer.(http.Flusher).Flush()
				return true

			case tick := <-ticks:
				c.JSON(http.StatusOK, tick)
				c.String(http.StatusOK, "\n")
				c.Writer.(http.Flusher).Flush()
				return true
			}
		})
	})

	log.Printf("http://localhost:8080/ui")

	r.Run()
}

func tick(ctx context.Context, ticks chan gin.H) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// fall-through
		}
		tick := gin.H{
			"timestamp": now(),
		}
		ticks <- tick
		time.Sleep(1 * time.Second)
	}
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
