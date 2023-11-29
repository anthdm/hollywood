[![Go Report Card](https://goreportcard.com/badge/github.com/anthdm/hollywood)](https://goreportcard.com/report/github.com/anthdm/hollywood)
![example workflow](https://github.com/anthdm/hollywood/actions/workflows/build.yml/badge.svg?branch=master)

# Blazingly fast, low latency actors for Golang

Hollywood is an ULTRA fast actor engine build for speed and low-latency applications. Think about game servers,
advertising brokers, trading engines, etc... It can handle **10 million messages in under 1 second**.

## What is the actor model?

The Actor Model is a computational model used to build highly concurrent and distributed systems. It was introduced by
Carl Hewitt in 1973 as a way to handle complex systems in a more scalable and fault-tolerant manner.

In the Actor Model, the basic building block is an actor, called receiver in Hollywood, which is an independent unit of
computation that communicates with other actors by exchanging messages. Each actor has its own state and behavior, and
can only communicate with other actors by sending messages. This message-passing paradigm allows for a highly
decentralized and fault-tolerant system, as actors can continue to operate independently even if other actors fail or
become unavailable.

Actors can be organized into hierarchies, with higher-level actors supervising and coordinating lower-level actors. This
allows for the creation of complex systems that can handle failures and errors in a graceful and predictable way.

By using the Actor Model in your application, you can build highly scalable and fault-tolerant systems that can handle a
large number of concurrent users and complex interactions.

## Features

- guaranteed message delivery on receiver failure (buffer mechanism)
- fire&forget or request&response messaging, or both.
- dRPC as the transport layer
- Optimized proto buffers without reflection
- lightweight and highly customizable
- cluster support [coming soon...]

# Benchmarks

```
make bench
```

```
[BENCH HOLLYWOOD LOCAL] processed 1_000_000 messages in 83.4437ms
[BENCH HOLLYWOOD LOCAL] processed 10_000_000 messages in 786.2787ms
[BENCH HOLLYWOOD LOCAL] processed 100_000_000 messages in 7.8718426s
```

# Installation

```
go get github.com/anthdm/hollywood/...
```

# Quickstart

> The **[examples](https://github.com/anthdm/hollywood/tree/master/examples)** folder is the best place to learn and
> explore Hollywood.

```go
type message struct {
	data string
}

type foo struct{}

func newFoo() actor.Receiver {
	return &foo{}
}

func (f *foo) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("foo has started")
	case *message:
		fmt.Println("foo has received", msg.data)
	}
}

func main() {
	engine := actor.NewEngine()
	pid := engine.Spawn(newFoo, "foo")
	engine.Send(pid, &message{data: "hello world!"})

	// Stop the actor, but let it process its messages first.
	engine.Poison(pid).Wait()
}
```

## Spawning receivers with configuration

```go
    e.Spawn(newFoo, "foo",
	actor.WithMaxRestarts(4),
	actor.WithInboxSize(1024 * 2),
	actor.WithTags("bar", "1"),
)
```

## Subscribing and publishing to the Eventstream

```go
e := actor.NewEngine()

// Subscribe to a various list of events that are being broadcast by
// the engine.
// The eventstream can also be used to publish custom events and notify all of the subscribers..
eventSub := e.EventStream.Subscribe(func(event any) {
	switch evt := event.(type) {
	case *actor.DeadLetterEvent:
		fmt.Printf("deadletter event to [%s] msg: %s\n", evt.Target, evt.Message)
	case *actor.ActivationEvent:
		fmt.Println("process is active", evt.PID)
	case *actor.TerminationEvent:
		fmt.Println("process terminated:", evt.PID)
		wg.Done()
	default:
		fmt.Println("received event", evt)
	}
})

// Cleanup subscription
defer e.EventStream.Unsubscribe(eventSub)

// Spawning receiver as a function
e.SpawnFunc(func(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		fmt.Println("started")
		_ = msg
	}
}, "foo")

time.Sleep(time.Second)
```

## Customizing the Engine

We're using the function option pattern. All function options are in the actor package and start their name with
"EngineOpt". So, setting a custom PID separator for the output looks like this:

```go
	e := actor.NewEngine(actor.EngineOptPidSeparator("→"))
```

After configuring the Engine with a custom PID Separator the string representation of PIDS will look like this:

```go
    pid := actor.NewPID("127.0.0.1:3000", "foo", "bar", "baz", "1")
    // 127.0.0.1:3000->foo->bar->baz->1
```

Note that you can also provide a custom logger to the engine. See the Logging section for more details.

## Custom middleware

You can add custom middleware to your Receivers. This can be usefull for storing metrics, saving and loading data for
your Receivers on `actor.Started` and `actor.Stopped`.

For examples on how to implement custom middleware, check out the middleware folder in the *
*[examples](https://github.com/anthdm/hollywood/tree/master/examples/middleware)**

## Logging

The default for Hollywood is, as any good library, not to log anything, but rather to rely on the application to
configure logging as it sees fit. However, as a convenience, Hollywood provides a simple logging package that
you can use to gain some insight into what is going on inside the library.

When you create a Hollywood engine, you can pass some configuration options. This gives you the opportunity to
have the log package create a suitable logger. The logger is based on the standard library's `log/slog` package.

If you want Hollywood to log with its defaults, it will provide structured logging with the loglevel being `ÌNFO`.
You'll then initialize the engine as such:

```go
    engine := actor.NewEngine(actor.EngineOptLogger(log.Default()))
```

If you want more control, say by having the loglevel be DEBUG and the output format be JSON, you can do so by

```go
	lh := log.NewHandler(os.Stdout, log.JsonFormat, slog.LevelDebug)
    engine := actor.NewEngine(actor.EngineOptLogger(log.NewLogger("[engine]", lh)))
```

This will have the engine itself log with the field "log", prepopulated with the value "[engine]" for the engine itself.
The various subsystems will change the log field to reflect their own name.

### Log levels

The log levels are, in order of severity:

* `slog.LevelDebug`
* `slog.LevelInfo`
* `slog.LevelWarn`
* `slog.LevelError`

### Log components.

The log field "log" will be populated with the name of the subsystem that is logging. The subsystems are:

* `[engine]`
* `[context`
* `[deadLetter]`
* `[eventStream]`
* `[registry]`
* `[stream_reader]`
* `[stream_writer]`
* `[stream_router]`

In addition, the logger will log with log=$ACTOR_NAME for any actor that has a name.

See the log package for more details about the implementation.

# Test

```
make test
```

# Used in Production By

This project is currently used in production by the following organizations/projects:

- [Sensora IoT](https://sensora.io)

# License

Hollywood is licensed under the MIT licence.
