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
	log.Printf("http://localhost:8080/ui")

	r.GET("/events", func(c *gin.Context) {
		ctx := c.Request.Context()
		tickContext, stopTicking := context.WithCancel(ctx)
		closeNotify := c.Writer.CloseNotify()
		ticks := make(chan gin.H, 1)

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

			case tick := <-ticks:
				c.JSON(http.StatusOK, tick)
				c.String(http.StatusOK, "\n")
				c.Writer.(http.Flusher).Flush()
				return true
			}
		})
	})

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
			"timestamp": time.Now().Format(time.RFC3339Nano),
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
