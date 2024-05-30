package main

import (
	"context"
	"log"
	"time"

	"github.com/Arceliar/phony"
)

type AsyncFSM struct {
	phony.Inbox
	ctx          context.Context
	cancel       context.CancelFunc
	events       EventPublisher
	currentCount uint8
}

func NewAsyncFSM(events EventPublisher) *AsyncFSM {
	return &AsyncFSM{
		ctx:    context.Background(),
		cancel: noOp(), /*no-op*/
		events: events,
	}
}

func (fsm *AsyncFSM) StartWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount != 0 {
			fsm.events.Publish(NewEventWithReason("RequestIgnored", "cannot start: machine busy"))
			return
		}
		fsm.ctx, fsm.cancel = context.WithCancel(context.Background())
		fsm.events.Publish(NewSimpleEvent("WorkStarted"))
		fsm.currentCount = 10
		go fsm.tick()
	})
	log.Println("StartWork finished")
}

func (fsm *AsyncFSM) AbortWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount == 0 {
			fsm.events.Publish(NewEventWithReason("RequestIgnored", "cannot abort: machine not busy"))
			return
		}
		fsm.cancel()
		fsm.events.Publish(NewSimpleEvent("WorkAbortRequested"))
	})
	log.Println("AbortWork finished")
}

func (fsm *AsyncFSM) tick() {
	fsm.Act(fsm, func() {
		// check if aborted
		select {
		case <-fsm.ctx.Done():
			go func() {
				time.Sleep(800 * time.Millisecond)
				fsm.events.Publish(NewSimpleEvent("WorkAborted"))
			}()
			fsm.currentCount = 0
			return
		default:
			// not canceled yet
		}
		fsm.events.Publish(NewEventWithParam("Tick", fsm.currentCount))
		fsm.currentCount--
		if fsm.currentCount == 0 {
			go func() {
				time.Sleep(800 * time.Millisecond)
				fsm.events.Publish(NewSimpleEvent("WorkDone"))
			}()
			return
		}
		go func() {
			time.Sleep(800 * time.Millisecond)
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
