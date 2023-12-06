package actor

import (
	"log/slog"
	"time"
)

// Here the events are defined.

// eventLog is an interface that the various Events can choose to implement. If they do, the event stream
// will log these events to slog.
type eventLog interface {
	log() (slog.Level, string, []any)
}

// EventSub is the message that will be send to subscribe to the event stream.
type EventSub struct {
	pid *PID
}

// EventUnSub is the message that will be send to unsubscribe from the event stream.
type EventUnsub struct {
	pid *PID
}

// ActorStartedEvent is broadcasted over the EventStream each time
// a Receiver (Actor) is spawned and activated. This means, that at
// the point of receiving this event the Receiver (Actor) is ready
// to process messages.
type ActorStartedEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorStartedEvent) log() (slog.Level, string, []any) {
	return slog.LevelInfo, "Actor started", []any{"pid", e.PID.GetID()}
}

// ActorStoppedEvent is broadcasted over the EventStream each time
// a process is terminated.
type ActorStoppedEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorStoppedEvent) log() (slog.Level, string, []any) {
	return slog.LevelInfo, "Actor stopped", []any{"pid", e.PID.GetID()}
}

// ActorRestartedEvent is broadcasted when an actor crashes and gets restarted
type ActorRestartedEvent struct {
	PID        *PID
	Timestamp  time.Time
	Stacktrace []byte
	Reason     any
	Restarts   int32
}

func (e ActorRestartedEvent) log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor crashed and restarted",
		[]any{"pid", e.PID.GetID(), "stack", string(e.Stacktrace),
			"reason", e.Reason, "restarts", e.Restarts}
}

// ActorMaxRestartsExceededEvent gets created if an actor crashes too many times
type ActorMaxRestartsExceededEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorMaxRestartsExceededEvent) log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor crashed too many times", []any{"pid", e.PID.GetID()}
}

// ActorDuplicateIdEvent gets published if we try to register the same name twice.
// Todo: Make a test for this.
type ActorDuplicateIdEvent struct {
	PID *PID
}

func (e ActorDuplicateIdEvent) log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor name already claimed", []any{"pid", e.PID.GetID()}
}

type EngineRemoteMissingEvent struct {
	Target  *PID
	Sender  *PID
	Message any
}

func (e EngineRemoteMissingEvent) log() (slog.Level, string, []any) {
	return slog.LevelError, "Engine has no remote", []any{"sender", e.Target.GetID()}
}

type DeadLetterLoopEvent struct{}

func (e DeadLetterLoopEvent) log() (slog.Level, string, []any) {
	return slog.LevelError, "Deadletter loop detected", []any{}
}
