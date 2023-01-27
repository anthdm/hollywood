# Hollywood

# Installation

```
go get github.com/anthdm/hollywood
```

The examples folder is the best place to learn how to work with Hollywood.

# Quickstart

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

# Contributing

TODO
