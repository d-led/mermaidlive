package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/cskr/pubsub/v2"
	"github.com/gin-gonic/gin"
)

type Server struct {
	port    string
	server  *gin.Engine
	events  *pubsub.PubSub[string, Event]
	fsm     *AsyncFSM
	visitor *VisitorTracker
	fs      http.FileSystem
}

func NewServerWithOptions(port string,
	events *pubsub.PubSub[string, Event],
	fs http.FileSystem) *Server {
	server := &Server{
		port:    port,
		server:  configureGin(),
		events:  events,
		fsm:     NewAsyncFSM(events),
		visitor: NewVisitorTracker(events),
		fs:      fs,
	}
	server.setupRoutes()
	return server
}

func (s *Server) Run() {
	log.Printf("Server running at :%v", *port)
	log.Printf("Visit the UI at %s", s.getUIUrl())
	log.Println(s.server.Run(":" + *port))
}

func (s *Server) setupRoutes() {
	s.server.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui")
	})
	s.server.StaticFS("/ui/", s.fs)

	s.server.POST("/commands/:command", func(ctx *gin.Context) {
		command := ctx.Param("command")
		log.Println("command called:", command)
		switch command {
		case "start":
			s.fsm.StartWork()
			ctx.JSON(http.StatusOK, gin.H{})
			return
		case "abort":
			s.fsm.AbortWork()
			ctx.JSON(http.StatusOK, gin.H{})
			return
		default:
			msg := "unknown command: '" + command + "'"
			s.fsm.events.Pub(NewEventWithReason("CommandRejected", msg), topic)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"result":  "rejected",
				"command": command,
				"reason":  msg,
			})
			return
		}
	})

	s.server.GET("/events", func(c *gin.Context) {
		s.visitor.Joined()
		defer s.visitor.Left()

		ctx := c.Request.Context()
		closeNotify := c.Writer.CloseNotify()

		myEvents := s.events.Sub(topic)
		defer s.events.Unsub(myEvents, topic)

		streamOneEvent(c, NewEventWithParam("ConnectedToRegion", getRegion()))
		streamOneEvent(c, NewEventWithParam("Revision", versioninfo.Revision))
		streamOneEvent(c, NewSimpleEvent("StartedListening"))

		// callback returns false on end of processing
		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				log.Printf("client disconnected")
				return false

			case <-closeNotify:
				log.Printf("client closed the connection")
				return false

			case event := <-myEvents:
				streamOneEvent(c, event)

				return true
			}
		})
	})
}

func (s *Server) getUIUrl() string {
	baseUrl := "http://localhost"
	return fmt.Sprintf("%v:%v/ui", baseUrl, s.port)
}

func configureGin() *gin.Engine {
	return gin.Default()
}

func getRegion() string {
	if region, ok := os.LookupEnv("FLY_REGION"); ok {
		return region
	}
	return "local"
}
