package actor

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInboxSendAndProcess(t *testing.T) {
	inbox := NewInbox(10)
	processedMessages := make(chan Envelope, 10)
	mockProc := MockProcessor{
		processFunc: func(envelopes []Envelope) {
			for _, e := range envelopes {
				processedMessages <- e
			}
		},
	}
	inbox.Start(mockProc)
	msg := Envelope{}
	inbox.Send(msg)
	select {
	case <-processedMessages: // Message processed
	case <-time.After(time.Millisecond):
		t.Errorf("Message was not processed in time")
	}

	inbox.Stop()
}

func TestInboxSendAndProcessMany(t *testing.T) {
	for i := 0; i < 100000; i++ {
		inbox := NewInbox(10)
		processedMessages := make(chan Envelope, 10)
		mockProc := MockProcessor{
			processFunc: func(envelopes []Envelope) {
				for _, e := range envelopes {
					processedMessages <- e
				}
			},
		}
		inbox.Start(mockProc)
		msg := Envelope{}
		inbox.Send(msg)

		timer := time.NewTimer(time.Second)
		select {
		case <-processedMessages: // Message processed
		case <-timer.C:
			t.Errorf("Message was not processed in time")
		}
		timer.Stop()

		inbox.Stop()
	}
}

type MockProcessor struct {
	processFunc func([]Envelope)
}

func (m MockProcessor) Start() {}
func (m MockProcessor) PID() *PID {
	return nil
}
func (m MockProcessor) Send(*PID, any, *PID) {}
func (m MockProcessor) Invoke(envelopes []Envelope) {
	m.processFunc(envelopes)
}
func (m MockProcessor) Shutdown() {}

func TestInboxStop(t *testing.T) {
	inbox := NewInbox(10)
	done := make(chan struct{})
	mockProc := MockProcessor{
		processFunc: func(envelopes []Envelope) {
			inbox.Stop()
			done <- struct{}{}
		},
	}
	inbox.Start(mockProc)
	inbox.Send(Envelope{})
	<-done
	require.True(t, atomic.LoadInt32(&inbox.procStatus) == stopped)
}
