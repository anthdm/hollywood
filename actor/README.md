# Developers guide for the actor package.

This provides an overview of the abstractions and concepts used in the actor package.

## Engine

The Hollywood engine is the core of the actor model. It is responsible for spawning actors, sending messages
to actors and stopping actors. The engine is also responsible for the lifecycle of the actors.

## Receiver / Actor

The receiver is the interface that all actors must implement. It is the interface that the engine uses to 
communicate with the actor. We sometimes refer to actors as receivers.

### Child & Parent

An actor can have a parent. If an actor is spawned by another actor,
the spawning actor is the parent of the spawned actor. This is typically used to implement supervision or to facilitate
logical routing of messages.

## Message

The basis of communication between actors is the message. A message can be of any type. If the message needs to
cross the wire, it must be serializable, so a protobuf description must be provided.

## Context

The context is a struct that is passed to all user-supplied actors. It should contain all the dependencies
that the actor needs to do its work. The context is also used to send messages to other actors.

## Request

A request is a message that is sent to an actor synchronously. The request will block until the actor has
processed the message and returned a response.

Note, that some performance overhead of sending requests. Inside Hollywood a single-use actor is spawned to 
handle the request. This actor is then stopped when the response is received.

Since requests are synchronous, they can deadlock if the actor that is processing the request sends a request
to the actor that sent the original request. So be careful when using requests.

## Remoter

The Remoter interface is an interface that is used to break up a circular dependency between the engine and
the remote. The engine needs to be able to send messages to the remote, but the remote also needs to be able
to send messages to the engine. The Remoter interface is used to break this circular dependency.

## Repeater

A repeater is started in the Engine when you'll like to send a message to an actor at a regular interval. 

## Event & Event Stream

Since Hollywood is asynchronous, a lot of what would typically be returned as errors are instead broadcasted 
as events. Each Engine has an Event Stream that can be used to listen for events. The most important event is
likely the DeadLetter event. This event is broadcasted when a message is sent to an actor that doesn't exist or cannot
be reached.

See `events.go` for a list of events.

## Inbox

Each actor has an inbox. The inbox is implemented as a ring buffer, the size of which is configurable when you spawn 
an actor. Note that when you run out of capacity in the inbox, the inbox will impose backpressure on the sender. This
means that the sender will block until there is capacity in the inbox. So sizing the inbox is important.

## Tag

Each actor can have an arbitrary number of tags. Tags are used to route messages to actors. You can send broadcast a message
to all actors with a specific tag

## Registry

The Engine keeps a list of all local actors in a registry, which is a map
of actor names to actor references. The registry is used to route messages to actors.

## Scheduler

TODO: Describe the scheduler

## Envelope

When a message is sent to an actor is is wrapped in an envelope. The envelope contains the message, the sender and the
receiver. The envelope is used to route the message to the correct actor.

## Process

A process is an abstraction over the actor. Todo: Describe the process.

## Processer

Todo: Not really sure.

## PID

A PID is a process identifier. It is used to identify a process. An actor is a PID.

## Middleware

Middleware is used to intercept messages before they are sent to the actor. This can be used to implement
logging, metrics, tracing, etc.

## PoisonPill

When an actor needs to be shutdown, a PoisonPill message is sent to the actor, which will shut down the actor.
