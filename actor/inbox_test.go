package actor

import (
	"sync"
	"testing"
	"time"
)

func TestInboxSendAndProcess(t *testing.T) {
	inbox := NewInbox(10)
	processedMessages := make(chan Envelope, 10)
	mockProc := MockProcesser{
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

type MockProcesser struct {
	processFunc func([]Envelope)
}

func (m MockProcesser) Start() {}
func (m MockProcesser) PID() *PID {
	return nil
}
func (m MockProcesser) Send(*PID, any, *PID) {}
func (m MockProcesser) Invoke(envelopes []Envelope) {
	m.processFunc(envelopes)
}
func (m MockProcesser) Shutdown(_ *sync.WaitGroup) {}
