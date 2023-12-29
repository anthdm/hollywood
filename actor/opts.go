package actor

import (
	"time"
)

const (
	defaultInboxSize   = 1024
	defaultMaxRestarts = 3
)

// defaultRestartDelay is a constant representing the default time delay before restarting a process.
// It's set to 500 milliseconds.
var defaultRestartDelay = 500 * time.Millisecond

// ReceiveFunc is a type definition for a function that takes a pointer to a Context object.
// This type is typically used to define how a message or event is processed in the system.
type ReceiveFunc = func(*Context)

// MiddlewareFunc is a type definition for a function that takes a ReceiveFunc and returns a ReceiveFunc.
// This pattern is used for creating middleware functions that can modify or extend the behavior of
// message processing functions.
type MiddlewareFunc = func(ReceiveFunc) ReceiveFunc

// Opts struct defines various options for a particular component or process in the system.
type Opts struct {
	Producer     Producer         // Producer is an interface or struct that represents a message producer.
	Kind         string           // Kind is a string indicating the type or category of the component.
	ID           string           // ID is a unique identifier for the component.
	MaxRestarts  int32            // MaxRestarts specifies the maximum number of restarts allowed for the component.
	RestartDelay time.Duration    // RestartDelay sets the delay duration between restarts.
	InboxSize    int              // InboxSize determines the size of the inbox or queue for incoming messages.
	Middleware   []MiddlewareFunc // Middleware is a slice of MiddlewareFunc, allowing for middleware to be applied.
}

// OptFunc is a function type that takes a pointer to Opts and allows for modification of its fields.
type OptFunc func(*Opts)

// DefaultOpts is a function that returns default options for a given Producer.
// It initializes the Opts struct with default values.
func DefaultOpts(p Producer) Opts {
	return Opts{
		Producer:     p,                   // Sets the Producer
		MaxRestarts:  defaultMaxRestarts,  // Sets the maximum number of restarts to a default value
		InboxSize:    defaultInboxSize,    // Sets the inbox size to a default value
		RestartDelay: defaultRestartDelay, // Sets the restart delay to a default value
		Middleware:   []MiddlewareFunc{},  // Initializes an empty slice for MiddlewareFunc
	}
}

// WithMiddleware is a function that returns an OptFunc.
// It appends one or more MiddlewareFunc to the Middleware slice in the Opts.
func WithMiddleware(mw ...MiddlewareFunc) OptFunc {
	return func(opts *Opts) {
		opts.Middleware = append(opts.Middleware, mw...) // Append the provided middleware to the existing slice
	}
}

// WithRestartDelay returns an OptFunc that sets the RestartDelay in Opts.
func WithRestartDelay(d time.Duration) OptFunc {
	return func(opts *Opts) {
		opts.RestartDelay = d // Sets the restart delay to the provided duration
	}
}

// WithInboxSize returns an OptFunc that sets the InboxSize in Opts.
func WithInboxSize(size int) OptFunc {
	return func(opts *Opts) {
		opts.InboxSize = size // Sets the inbox size to the provided value
	}
}

// WithMaxRestarts returns an OptFunc that sets the MaxRestarts in Opts.
func WithMaxRestarts(n int) OptFunc {
	return func(opts *Opts) {
		opts.MaxRestarts = int32(n) // Sets the maximum number of restarts to the provided value
	}
}

// WithID returns an OptFunc that sets the ID in Opts.
func WithID(id string) OptFunc {
	return func(opts *Opts) {
		opts.ID = id // Sets the ID to the provided string
	}
}
