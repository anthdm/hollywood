package actor

import "time"

const (
	defaultInboxSize   = 1024
	defaultMaxRestarts = 3
)

var defaultRestartDelay = 500 * time.Millisecond

type ReceiveFunc = func(*Context)

type MiddlewareFunc = func(ReceiveFunc) ReceiveFunc

type Opts struct {
	Producer     Producer
	Name         string
	Tags         []string
	MaxRestarts  int32
	RestartDelay time.Duration
	InboxSize    int
	Middleware   []MiddlewareFunc
}

type OptFunc func(*Opts)

// DefaultOpts returns default options from the given Producer.
func DefaultOpts(p Producer) Opts {
	return Opts{
		Producer:     p,
		MaxRestarts:  defaultMaxRestarts,
		InboxSize:    defaultInboxSize,
		RestartDelay: defaultRestartDelay,
		Middleware:   []MiddlewareFunc{},
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

func WithTags(tags ...string) OptFunc {
	return func(opts *Opts) {
		opts.Tags = tags
	}
}
