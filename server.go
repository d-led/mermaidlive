package mermaidlive

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/cskr/pubsub/v2"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	gm "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type Server struct {
	port           string
	server         *gin.Engine
	events         *pubsub.PubSub[string, Event]
	fsm            *AsyncFSM
	visitorTracker *VisitorTracker
	peerSource     *PeerSource
	uiFilesystem   http.FileSystem
}

func NewServerWithOptions(port string,
	events *pubsub.PubSub[string, Event],
	fs http.FileSystem,
	delay time.Duration) *Server {
	peerSource := NewFlyPeerSource(events)
	visitorTracker := NewVisitorTracker(events)
	server := &Server{
		port:           port,
		server:         configureGin(),
		events:         events,
		fsm:            NewCustomAsyncFSM(events, delay),
		visitorTracker: visitorTracker,
		peerSource:     peerSource,
		uiFilesystem:   fs,
	}
	server.configureRateLimiting()
	server.setupRoutes()
	return server
}

func (s *Server) Run(port string) {
	log.Printf("Server running at :%v", port)
	if myIp := getFlyPrivateIP(); myIp != "" {
		log.Printf("Private IP: %v", myIp)
	}
	log.Printf("Visit the UI at %s", s.getUIUrl())
	s.peerSource.Start()
	log.Println(s.server.Run(":" + port))
}

func (s *Server) configureRateLimiting() {
	limiterSpec := strings.TrimSpace(os.Getenv("RATE_LIMIT"))
	if limiterSpec == "" {
		log.Printf("No rate limiting configured")
	}
	rate, err := limiter.NewRateFromFormatted(limiterSpec)
	if err != nil {
		log.Printf("bad rate limit '%s': %v", limiterSpec, err)
		return
	}
	store := memory.NewStore()
	log.Printf("RATE_LIMIT: %s", limiterSpec)
	middleware := gm.NewMiddleware(limiter.New(store, rate))
	s.server.ForwardedByClientIP = true
	s.server.Use(middleware)
}

func (s *Server) setupRoutes() {
	s.server.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui")
	})
	s.server.StaticFS("/ui/", s.uiFilesystem)

	s.server.GET("/machine/state", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, s.fsm.CurrentState())
	})

	s.server.POST("/commands/:command", func(ctx *gin.Context) {
		command := ctx.Param("command")
		log.Println("command called:", command)
		switch command {
		case "start":
			s.fsm.StartWork()
			// to do: consider using HTTP 201 Created
			ctx.JSON(http.StatusOK, gin.H{})
			return
		case "abort":
			s.fsm.AbortWork()
			ctx.JSON(http.StatusOK, gin.H{})
			return
		default:
			msg := "unknown command: '" + command + "'"
			s.fsm.events.Pub(NewEventWithReason("CommandRejected", msg), Topic)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"result":  "rejected",
				"command": command,
				"reason":  msg,
			})
			return
		}
	})

	s.server.GET("/events", func(c *gin.Context) {
		c.Header("Connection", "Keep-Alive")
		c.Header("Keep-Alive", "timeout=10, max=1000")
		s.visitorTracker.Joined()
		defer s.visitorTracker.Left()

		ctx := c.Request.Context()
		closeNotify := c.Writer.CloseNotify()

		myEvents := s.events.Sub(Topic)
		defer s.events.Unsub(myEvents, Topic)

		streamOneEvent(c, NewSimpleEvent("StartedListening"))
		streamOneEvent(c, NewEventWithParam("ConnectedToRegion", getFlyRegion()))
		streamOneEvent(c, NewEventWithParam("Revision", versioninfo.Revision))
		streamOneEvent(c, NewEventWithParam("LastSeenState", s.fsm.CurrentState()))
		streamOneEvent(c, NewEventWithParam("ReplicasActive", 1 /*initial state*/))

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

func streamOneEvent(c *gin.Context, event any) {
	c.JSON(http.StatusOK, event)
	c.String(http.StatusOK, "\n")
	c.Writer.(http.Flusher).Flush()
}
