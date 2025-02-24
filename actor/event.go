package actor

import (
	"log/slog"
	"time"
)

// Here the events are defined.

// EventLogger is an interface that the various Events can choose to implement. If they do, the event stream
// will log these events to slog.
type EventLogger interface {
	Log() (slog.Level, string, []any)
}

// ActorStartedEvent is broadcasted over the eventStream each time
// a Receiver (Actor) is spawned and activated. This means, that at
// the point of receiving this event the Receiver (Actor) is ready
// to process messages.
type ActorStartedEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorStartedEvent) Log() (slog.Level, string, []any) {
	return slog.LevelDebug, "Actor started", []any{"pid", e.PID}
}

// ActorInitializedEvent is broadcasted over the eventStream before an actor
// received and processed its started event.
type ActorInitializedEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorInitializedEvent) Log() (slog.Level, string, []any) {
	return slog.LevelDebug, "Actor initialized", []any{"pid", e.PID}
}

// ActorStoppedEvent is broadcasted over the eventStream each time
// a process is terminated.
type ActorStoppedEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorStoppedEvent) Log() (slog.Level, string, []any) {
	return slog.LevelDebug, "Actor stopped", []any{"pid", e.PID}
}

// ActorRestartedEvent is broadcasted when an actor crashes and gets restarted
type ActorRestartedEvent struct {
	PID        *PID
	Timestamp  time.Time
	Stacktrace []byte
	Reason     any
	Restarts   int32
}

func (e ActorRestartedEvent) Log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor crashed and restarted",
		[]any{"pid", e.PID.GetID(), "stack", string(e.Stacktrace),
			"reason", e.Reason, "restarts", e.Restarts}
}

// ActorMaxRestartsExceededEvent gets created if an actor crashes too many times
type ActorMaxRestartsExceededEvent struct {
	PID       *PID
	Timestamp time.Time
}

func (e ActorMaxRestartsExceededEvent) Log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor crashed too many times", []any{"pid", e.PID.GetID()}
}

// ActorUnprocessableMessageEvent gets published if an actor is unable to process the message after retries
type ActorUnprocessableMessageEvent struct {
	PID       *PID
	Timestamp time.Time
	Message   any
}

func (e ActorUnprocessableMessageEvent) Log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor unable to process message", []any{"pid", e.PID.GetID()}
}

// ActorDuplicateIdEvent gets published if we try to register the same name twice.
type ActorDuplicateIdEvent struct {
	PID *PID
}

func (e ActorDuplicateIdEvent) Log() (slog.Level, string, []any) {
	return slog.LevelError, "Actor name already claimed", []any{"pid", e.PID.GetID()}
}

// EngineRemoteMissingEvent gets published if we try to send a message to a remote actor but the remote
// system is not available.
type EngineRemoteMissingEvent struct {
	Target  *PID
	Sender  *PID
	Message any
}

func (e EngineRemoteMissingEvent) Log() (slog.Level, string, []any) {
	return slog.LevelError, "Engine has no remote", []any{"sender", e.Target.GetID()}
}

// RemoteUnreachableEvent gets published when trying to send a message to
// an remote that is not reachable. The event will be published after we
// retry to dial it N times.
type RemoteUnreachableEvent struct {
	// The listen address of the remote we are trying to dial.
	ListenAddr string
}

// DeadLetterEvent is delivered to the deadletter actor when a message can't be delivered to it's recipient
type DeadLetterEvent struct {
	Target  *PID
	Message any
	Sender  *PID
}
