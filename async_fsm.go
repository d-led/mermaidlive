package mermaidlive

import (
	"context"
	"log"
	"time"

	"github.com/Arceliar/phony"
	"github.com/cskr/pubsub/v2"
)

type AsyncFSM struct {
	phony.Inbox
	ctx          context.Context
	cancel       context.CancelFunc
	events       *pubsub.PubSub[string, Event]
	delay        time.Duration
	currentCount uint8
	currentState string
}

func NewAsyncFSM(events *pubsub.PubSub[string, Event]) *AsyncFSM {
	return NewCustomAsyncFSM(events, 800*time.Millisecond)
}

func NewCustomAsyncFSM(events *pubsub.PubSub[string, Event], delay time.Duration) *AsyncFSM {
	return &AsyncFSM{
		ctx:          context.Background(),
		cancel:       noOp(), /*no-op*/
		events:       events,
		delay:        delay,
		currentState: "waiting",
	}
}

func (fsm *AsyncFSM) StartWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount != 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "cannot start: machine busy"), Topic)
			return
		}
		fsm.ctx, fsm.cancel = context.WithCancel(context.Background())
		fsm.currentState = "working"
		fsm.events.Pub(NewSimpleEvent("WorkStarted"), Topic)
		fsm.currentCount = 10
		go fsm.tick()
	})
	log.Println("StartWork finished")
}

func (fsm *AsyncFSM) AbortWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount == 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "cannot abort: machine not busy"), Topic)
			return
		}
		fsm.cancel()
		fsm.currentState = "aborting"
		fsm.events.Pub(NewSimpleEvent("WorkAbortRequested"), Topic)
	})
	log.Println("AbortWork finished")
}

func (fsm *AsyncFSM) tick() {
	fsm.Act(fsm, func() {
		// check if aborted
		select {
		case <-fsm.ctx.Done():
			go func() {
				time.Sleep(fsm.delay)
				fsm.currentState = "waiting"
				fsm.events.Pub(NewSimpleEvent("WorkAborted"), Topic)
			}()
			fsm.currentCount = 0
			return
		default:
			// not canceled yet
		}
		fsm.events.Pub(NewEventWithParam("Tick", fsm.currentCount), Topic)
		fsm.currentCount--
		if fsm.currentCount == 0 {
			go func() {
				time.Sleep(fsm.delay)
				fsm.currentState = "waiting"
				fsm.events.Pub(NewSimpleEvent("WorkDone"), Topic)
			}()
			return
		}
		go func() {
			time.Sleep(fsm.delay)
			fsm.tick()
		}()
	})
}

// sync queries - not to be used from within actor behaviors (methods)
func (fsm *AsyncFSM) IsWaiting() bool {
	return fsm.getCurrentCount() == 0
}

func (fsm *AsyncFSM) CurrentState() string {
	var res string
	phony.Block(fsm, func() {
		res = fsm.currentState
	})
	return res
}

func (fsm *AsyncFSM) getCurrentCount() uint8 {
	var currentCount uint8
	phony.Block(fsm, func() {
		currentCount = fsm.currentCount
	})
	return currentCount
}

func noOp() context.CancelFunc {
	return func() {}
}
