package main

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
}

func NewAsyncFSM(events *pubsub.PubSub[string, Event]) *AsyncFSM {
	return NewCustomAsyncFSM(events, 800*time.Millisecond)
}

func NewCustomAsyncFSM(events *pubsub.PubSub[string, Event], delay time.Duration) *AsyncFSM {
	return &AsyncFSM{
		ctx:    context.Background(),
		cancel: noOp(), /*no-op*/
		events: events,
		delay:  delay,
	}
}

func (fsm *AsyncFSM) StartWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount != 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "cannot start: machine busy"), topic)
			return
		}
		fsm.ctx, fsm.cancel = context.WithCancel(context.Background())
		fsm.events.Pub(NewSimpleEvent("WorkStarted"), topic)
		fsm.currentCount = 10
		go fsm.tick()
	})
	log.Println("StartWork finished")
}

func (fsm *AsyncFSM) AbortWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount == 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "cannot abort: machine not busy"), topic)
			return
		}
		fsm.cancel()
		fsm.events.Pub(NewSimpleEvent("WorkAbortRequested"), topic)
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
				fsm.events.Pub(NewSimpleEvent("WorkAborted"), topic)
			}()
			fsm.currentCount = 0
			return
		default:
			// not canceled yet
		}
		fsm.events.Pub(NewEventWithParam("Tick", fsm.currentCount), topic)
		fsm.currentCount--
		if fsm.currentCount == 0 {
			go func() {
				time.Sleep(fsm.delay)
				fsm.events.Pub(NewSimpleEvent("WorkDone"), topic)
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
	var currentCount uint8
	phony.Block(fsm, func() {
		currentCount = fsm.currentCount
	})
	return currentCount == 0
}

func noOp() context.CancelFunc {
	return func() {}
}
