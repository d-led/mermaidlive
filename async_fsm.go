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
	currentCount uint8
}

func NewAsyncFSM(events *pubsub.PubSub[string, Event]) *AsyncFSM {
	return &AsyncFSM{
		ctx:    context.Background(),
		cancel: noOp(), /*no-op*/
		events: events,
	}
}

func (fsm *AsyncFSM) StartWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount != 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "machine busy"), topic)
			return
		}
		fsm.ctx, fsm.cancel = context.WithCancel(context.Background())
		fsm.events.Pub(NewSimpleEvent("WorkStarted"), topic)
		fsm.currentCount = 10
		go fsm.tick()
	})
	log.Println("StartWork finished")
}

func (fsm *AsyncFSM) CancelWork() {
	fsm.Act(fsm, func() {
		if fsm.currentCount == 0 {
			fsm.events.Pub(NewEventWithReason("RequestIgnored", "machine not busy"), topic)
			return
		}
		fsm.cancel()
		fsm.events.Pub(NewSimpleEvent("WorkCancellationRequested"), topic)
	})
	log.Println("CancelWork finished")
}

func (fsm *AsyncFSM) tick() {
	fsm.Act(fsm, func() {
		// check if canceled
		select {
		case <-fsm.ctx.Done():
			fsm.events.Pub(NewSimpleEvent("WorkCanceled"), topic)
			fsm.currentCount = 0
			return
		default:
			// not canceled yet
		}
		fsm.events.Pub(NewEventWithParam("Tick", fsm.currentCount), topic)
		fsm.currentCount--
		if fsm.currentCount == 0 {
			fsm.events.Pub(NewSimpleEvent("WorkDone"), topic)
			return
		}
		go func() {
			time.Sleep(800 * time.Millisecond)
			fsm.tick()
		}()
	})
}

func noOp() context.CancelFunc {
	return func() {}
}
