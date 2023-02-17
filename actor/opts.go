package actor

const (
	defaultInboxSize   = 100
	defaultMaxRestarts = 3
)

type ReceiveFunc = func(*Context)

type Opts struct {
	Producer    Producer
	Name        string
	Tags        []string
	MaxRestarts int32
	InboxSize   int
	Middleware  func(ReceiveFunc) ReceiveFunc
}

type OptFunc func(*Opts)

// DefaultOpts returns default options from the given Producer.
func DefaultOpts(p Producer) Opts {
	return Opts{
		Producer:    p,
		MaxRestarts: defaultMaxRestarts,
		InboxSize:   defaultInboxSize,
	}
}

func WithMiddleware(rf func(ReceiveFunc) ReceiveFunc) OptFunc {
	return func(opts *Opts) {
		opts.Middleware = rf
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
