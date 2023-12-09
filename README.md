[![Go Report Card](https://goreportcard.com/badge/github.com/anthdm/hollywood)](https://goreportcard.com/report/github.com/anthdm/hollywood)
![example workflow](https://github.com/anthdm/hollywood/actions/workflows/build.yml/badge.svg?branch=master)
<a href="https://discord.gg/gdwXmXYNTh">
	<img src="https://discordapp.com/api/guilds/1025692014903316490/widget.png?style=shield" alt="Discord Shield"/>
</a>

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
- fire & forget or request & response messaging, or both.
- High performance dRPC as the transport layer
- Optimized proto buffers without reflection
- lightweight and highly customizable
- cluster support [wip]

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

We recommend you start out by writing a few examples that run locally. Running locally is a bit simpler as the compiler
is able to figure out the types used. When running remotely, you'll need to provide protobuffer definitions for the
compiler.

## Hello world.

Let's go through a Hello world message. The complete example is available in the 
***[hello world](https://github.com/anthdm/hollywood/tree/master/examples/helloworld)*** folder. Let's start in main:
```go
	engine, err := actor.NewEngine()
```
This creates a new engine. The engine is the core of Hollywood. It's responsible for spawning actors, sending messages
and handling the lifecycle of actors. If Hollywood fails to create the engine it'll return an error. 

Next we'll need to create an actor. These are some times referred to as Receivers after the interface they must 
implement. Let's create a new actor that will print a message when it receives a message. 

```go
	pid := engine.Spawn(newHelloer, "hello")
```
This will cause the engine to spawn an actor with the ID "hello". The actor will be created by the provided 
function `newHelloer`. Ids must be unique. It will return a pointer to a PID. A PID is a process identifier. It's a unique identifier for the actor. Most of
the time you'll use the PID to send messages to the actor. Against remote systems you'll use the ID to send messages, 
but on local systems you'll mostly use the PID.

Let's look at the `newHelloer` function and the actor it returns. 

```go
	type helloer struct{}
	
	func newHelloer() actor.Receiver {
		return &helloer{}
	}
```

Simple enough. The `newHelloer` function returns a new actor. The actor is a struct that implements the actor.Receiver.
Lets look at the `Receive` method.

```go

    type message struct {}

	func (h *helloer) Receive(ctx *actor.Context) {
		switch msg := ctx.Message().(type) {
		case actor.Initialized:
			fmt.Println("helloer has initialized")
		case actor.Started:
			fmt.Println("helloer has started")
		case actor.Stopped:
			fmt.Println("helloer has stopped")
		case *message:
			fmt.Println("hello world", msg.data)
		}
	}
```

You can see we define a message struct. This is the message we'll send to the actor later. The Receive method
also handles a few other messages. These lifecycle messages are sent by the engine to the actor, you'll use these to 
initialize your actor

The engine passes an actor.Context to the `Receive` method. This context contains the message, the PID of the sender and
some other dependencies that you can use.

Now, lets send a message to the actor. We'll send a `message`, but you can send any type of message you want. The only
requirement is that the actor must be able to handle the message. For messages to be able to cross the wire 
they must be serializable. For protobuf to be able to serialize the message it must be a pointer. 
Local messages can be of any type.

Finally, lets send a message to the actor.

```go
	engine.Send(pid, "hello world!")
```

This will send a message to the actor. Hollywood will route the message to the correct actor. The actor will then print
a message to the console.

The **[examples](https://github.com/anthdm/hollywood/tree/master/examples)** folder is the best place to learn and
explore Hollywood further.


## Spawning receivers (actors) 

### With default configuration
```go
    e.Spawn(newFoo, "myactorname")
```

### With custom configuration
```go
    e.Spawn(newFoo, "myactorname",
		actor.WithMaxRestarts(4),
		actor.WithInboxSize(1024 * 2),
		actor.WithTags("bar", "1"),
	)
)
```
### As a stateless function 
Actors without state can be spawned as a function, because its quick and simple.
```go
e.SpawnFunc(func(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		fmt.Println("started")
		_ = msg
	}
}, "foo")
```

## Remote actors
Actors can communicate with each other over the network with the Remote package. 
This works the same as local actors but "over the wire". Hollywood supports serialization with protobuf.

***[Remote actor examples](https://github.com/anthdm/hollywood/tree/master/examples/remote)***

## Eventstream

The Eventstream is a powerful abstraction that allows you to build flexible and pluggable systems without dependencies. 

1. Subscribe any actor to a various list of system events
2. Broadcast your custom events to all subscribers 

Note that events that are not handled by any actor will be dropped. You should have an actor subscribed to the event 
stream in order to receive events. As a bare minimum, you'll want  to handle `DeadLetterEvent`. If Hollywood fails to 
deliver a message to an actor it will send a `DeadLetterEvent` to the event stream. 

Any event that fulfills the `actor.LogEvent` interface will be logged to the default logger, with the severity level, 
message and the attributes of the event set by the `actor.LogEvent` `log()` method.

You can find more in-depth information on how to use the Eventstream in your application in the Eventstream ***[examples](https://github.com/anthdm/hollywood/tree/master/examples/eventstream)***

### List of internal system events 
* `ActorStartedEvent`, an actor has started
* `ActorStoppedEvent`, an actor has stopped
* `DeadLetterEvent`, a message was not delivered to an actor
* `ActorRestartedEvent`, an actor has restarted after a crash/panic.

Have a look at the `actor/events.go` file for more information on the events.

## Customizing the Engine

We're using the function option pattern. All function options are in the actor package and start their name with
"EngineOpt". Currently, the only option is to provide a remote. This is done by

```go
	r := remote.New(remote.Config{ListenAddr: addr})
	engine, err := actor.NewEngine(actor.EngineOptRemote(r))
```
addr is a string with the format "host:port".

## Middleware

You can add custom middleware to your Receivers. This can be useful for storing metrics, saving and loading data for
your Receivers on `actor.Started` and `actor.Stopped`.

For examples on how to implement custom middleware, check out the middleware folder in the ***[examples](https://github.com/anthdm/hollywood/tree/master/examples/middleware)***

## Logging

Hollywood has some built in logging. It will use the default logger from the `log/slog` package. You can configure the
logger to your liking by setting the default logger using `slog.SetDefaultLogger()`. This will allow you to customize 
the log level, format and output. Please see the `slog` package for more information.

Note that some events might be logged to the default logger, such as `DeadLetterEvent` and `ActorStartedEvent` as these
events fulfill the `actor.LogEvent` interface. See the Eventstream section above for more information.

# Test

```
make test
```

# Community and discussions
Join our Discord community with over 2000 members for questions and a nice chat.
<br>
<a href="https://discord.gg/gdwXmXYNTh">
	<img src="https://discordapp.com/api/guilds/1025692014903316490/widget.png?style=banner2" alt="Discord Banner"/>
</a>

# Used in Production By

This project is currently used in production by the following organizations/projects:

- [Sensora IoT](https://sensora.io)

# License

Hollywood is licensed under the MIT licence.
