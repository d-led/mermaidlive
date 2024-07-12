package mermaidlive

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/cskr/pubsub/v2"
	"github.com/d-led/zmqcluster"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	gm "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type Server struct {
	port                 string
	server               *gin.Engine
	events               *pubsub.PubSub[string, Event]
	fsm                  *AsyncFSM
	visitorTracker       *VisitorTracker
	peerSource           *Cluster
	uiFilesystem         http.FileSystem
	serverContext        context.Context
	activeConnections    sync.WaitGroup
	clusterEventObserver *PersistentClusterObserver
}

func NewServerWithOptions(port string,
	events *pubsub.PubSub[string, Event],
	fs http.FileSystem,
	delay time.Duration) *Server {
	myIp := ChoosePeerLocator().GetMyIP()
	clusterEventObserver := NewPersistentClusterObserver(
		GetCounterIdentity(),
		myIp,
		events,
	)
	cluster := zmqcluster.NewZmqCluster(GetCounterIdentity(), getFlyZmqBindAddr())
	log.Printf("My IP: %s", myIp)
	cluster.SetMyIP(ChoosePeerLocator().GetMyIP())
	peerSource := NewCluster(events, clusterEventObserver, cluster)
	visitorTracker := NewVisitorTracker(events)
	server := &Server{
		port:                 port,
		server:               configureGin(),
		events:               events,
		fsm:                  NewCustomAsyncFSM(events, delay),
		visitorTracker:       visitorTracker,
		peerSource:           peerSource,
		uiFilesystem:         fs,
		clusterEventObserver: clusterEventObserver,
	}
	server.configureRateLimiting()
	server.setupRoutes()
	server.setupSignalHandler()
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
		return
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
		sourceReplicaId := strings.Join(ctx.Request.Header[http.CanonicalHeaderKey(SourceReplicaIdKey)], "")
		log.Println("command called:", command)
		myReplicaId := getPublicReplicaId()
		if sourceReplicaId != myReplicaId {
			log.Printf("Command and Event Stream Replica ID mismatch: %s != %s", sourceReplicaId, myReplicaId)
		}
		ctx.Header(SourceReplicaIdKey, myReplicaId)
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
		s.activeConnections.Add(1)
		defer s.visitorTracker.Left()
		defer s.activeConnections.Done()

		ctx := c.Request.Context()
		closeNotify := c.Writer.CloseNotify()

		myEvents := s.events.Sub(Topic)
		defer s.events.Unsub(myEvents, Topic)

		streamOneEvent(c, NewSimpleEvent("StartedListening"))
		streamOneEvent(c, NewEventWithParam("ConnectedToRegion", getFlyRegion()))
		streamOneEvent(c, NewEventWithParam("Revision", versioninfo.Revision))
		streamOneEvent(c, NewEventWithParam("LastSeenState", s.fsm.CurrentState()))
		streamOneEvent(c, GetReplicasEvent(1))
		streamOneEvent(c, NewEventWithParam("ConnectedToReplica", getPublicReplicaId()))

		// callback returns false on end of processing
		c.Stream(func(w io.Writer) bool {
			select {
			case <-s.serverContext.Done():
				log.Printf("closing the connection: server shutting down")
				return false

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

	if ClusterObservabilityEnabled {
		s.setupClusterObservabilityRoutes()
	}
}

func (s *Server) setupClusterObservabilityRoutes() {
	clusterGroup := s.server.Group("cluster")
	// httpie> http -S http://localhost:8080/cluster/events
	clusterGroup.GET("/events", func(c *gin.Context) {
		c.Header("Connection", "Keep-Alive")
		c.Header("Keep-Alive", "timeout=10, max=1000")

		ctx := c.Request.Context()
		closeNotify := c.Writer.CloseNotify()

		myEvents := s.events.Sub(ClusterMessageTopic)
		defer s.events.Unsub(myEvents, ClusterMessageTopic)

		streamOneEvent(c, NewSimpleEvent("StartedListening"))
		streamOneEvent(c, NewEventWithParam("ConnectedToRegion", getFlyRegion()))
		streamOneEvent(c, GetReplicasEvent(1))
		streamOneEvent(c, NewEventWithParam("ConnectedToReplica", getPublicReplicaId()))

		// callback returns false on end of processing
		c.Stream(func(w io.Writer) bool {
			select {
			case <-s.serverContext.Done():
				log.Printf("closing the connection: server shutting down")
				return false

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

func (s *Server) setupSignalHandler() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	serverContext, triggerShutdown := context.WithCancel(context.Background())
	s.serverContext = serverContext
	go func() {
		sig := <-signals
		log.Printf("Received signal: %v, gracefully shutting down the server", sig)
		triggerShutdown()
	}()
}

func (s *Server) WaitToDrainConnections() {
	// wait for global context to be cancelled
	<-s.serverContext.Done()
	// now wait for all connections to stop
	log.Println("Waiting to close all connections...")

	// but with a safe timeout
	done := make(chan bool, 1)
	go func() {
		s.activeConnections.Wait()
		done <- true
	}()
	select {
	case <-done:
		log.Println("All connections drained safely")
	case <-time.After(time.Second):
		log.Println("Did not close all connections on time. Forcing exit...")
	}

	// allow counters to propagate (opportunistically)
	time.Sleep(100 * time.Millisecond)
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

func init() {
	if os.Getenv("MML_CLUSTER_OBSERVABILITY_ENABLED") == "true" {
		log.Println("Cluster observability routes enabled")
		ClusterObservabilityEnabled = true
	}
}
