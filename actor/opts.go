package actor

import (
	"context"
	"time"
)

const (
	defaultInboxSize   = 1024
	defaultMaxRestarts = 3
	defaultRetries     = 2
	defaultMaxRetries  = 2
)

var defaultRestartDelay = 500 * time.Millisecond

type ReceiveFunc = func(*Context)

type MiddlewareFunc = func(ReceiveFunc) ReceiveFunc

type Opts struct {
	Producer     Producer
	Kind         string
	ID           string
	MaxRestarts  int32
	Retries      int32
	MaxRetries   int32
	RestartDelay time.Duration
	InboxSize    int
	Middleware   []MiddlewareFunc
	Context      context.Context
}

type OptFunc func(*Opts)

// DefaultOpts returns default options from the given Producer.
func DefaultOpts(p Producer) Opts {
	return Opts{
		Context:      context.Background(),
		Producer:     p,
		MaxRestarts:  defaultMaxRestarts,
		Retries:      defaultRetries,
		MaxRetries:   defaultMaxRetries,
		InboxSize:    defaultInboxSize,
		RestartDelay: defaultRestartDelay,
		Middleware:   []MiddlewareFunc{},
	}
}

func WithContext(ctx context.Context) OptFunc {
	return func(opts *Opts) {
		opts.Context = ctx
	}
}

func WithMiddleware(mw ...MiddlewareFunc) OptFunc {
	return func(opts *Opts) {
		opts.Middleware = append(opts.Middleware, mw...)
	}
}

func WithRestartDelay(d time.Duration) OptFunc {
	return func(opts *Opts) {
		opts.RestartDelay = d
	}
}

func WithInboxSize(size int) OptFunc {
	return func(opts *Opts) {
		opts.InboxSize = size
	}
}

func WithMaxRestarts(n int) OptFunc {
	return func(opts *Opts) {
		opts.MaxRestarts = int32(n)
	}
}

func WithRetries(n int) OptFunc {
	return func(opts *Opts) {
		opts.Retries = int32(n)
	}
}

func WithMaxRetries(n int) OptFunc {
	return func(opts *Opts) {
		opts.MaxRetries = int32(n)
	}
}

func WithID(id string) OptFunc {
	return func(opts *Opts) {
		opts.ID = id
	}
}
