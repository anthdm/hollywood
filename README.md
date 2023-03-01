[![Go Report Card](https://goreportcard.com/badge/github.com/anthdm/hollywood)](https://goreportcard.com/report/github.com/anthdm/hollywood)
![example workflow](https://github.com/anthdm/hollywood/actions/workflows/build.yml/badge.svg?branch=master)

# Blazingly fast, low latency actors for Golang
Hollywood is an actor engine build for speed and low-latency applications. Benchmarks coming soon...

## Features

- lock free LMAX based message queue for low latency messaging
- dRPC as the transport layer
- Optimized protoBuffers without reflection
- lightweight and highly customizable
- built and optimized for speed
- cluster support with Consul [coming soon...]

# Installation

```
go get github.com/anthdm/hollywood
```

# Quickstart

> The **[examples](https://github.com/anthdm/hollywood/tree/master/examples)** folder is the best place to learn and explore Hollywood.

```Go
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
	time.Sleep(time.Second * 1)
}
```

## Spawning receivers with configuration

```Go
e.Spawn(newFoo, "foo",
	actor.WithMaxRestarts(4),
	actor.WithInboxSize(999),
	actor.WithTags("bar", "1"),
)
```

## Listening to the Eventstream

```go
e := actor.NewEngine()

// Subscribe to a various list of event that are being broadcasted by
// the engine. But also published by you.
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

# PIDS

### Customize the PID separator.

```Go
actor.PIDSeparator = ">"
```

Will result in the following PID

```
// 127.0.0.1:3000>foo>bar>baz>1
```

# Test

```
make test
```
