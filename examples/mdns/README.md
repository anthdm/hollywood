# Remote Engine Discovery via mDNS
This example shows us how service discovery can be done via mDNS.

# About
mDNS or Multicast DNS can be used to discover services on the local network without the use of an authoritative DNS server.
This enables peer-to-peer discovery. It is important to note that many networks restrict the use of multicasting, which prevents mDNS from functioning.
Notably, multicast cannot be used in any sort of cloud, or shared infrastructure environments.
However it works well in most office, home, or private infrastructure environments.

# Quickstart

> The **[examples](https://github.com/anthdm/hollywood/tree/master/examples/mdns)** folder is the best place to explore Hollywood's Service Discovery

```
make build
```

# Flow
 0. When start engine with remote configuration, mDNS starting to announce itself and searches for other nodes
 1. If node founds new engine, discovery actor publishes a `DiscoveryEvent` via `actor.EventStream`
 2. (For demo purposes) An `chat` actor receives the discovery event and try to send a message to validate the flow.

# Execution 

```
./bin/arm/mdns -port 4001
    <omitted standart logs>
    INFO[0000] [REMOTE] server started                       listenAddr="127.0.0.1:4001"
    TRAC[0000] [EVENTSTREAM] subscribe                       id=1432518515 subs=1
    TRAC[0000] [PROCESS] started                             pid="127.0.0.1:4001/chat"
    TRAC[0000] [PROCESS] started                             pid="127.0.0.1:4001/mdns"
    INFO[0001] [DISCOVERY] remote discovered                 ID=engine_1682994946742073000 addrs="127.0.0.1:4002"
    TRAC[0001] [STREAM WRITER] connected                     remote="127.0.0.1:4002"
    TRAC[0001] [STREAM ROUTER] new stream route              pid="127.0.0.1:4001/stream/127.0.0.1:4002"
    INFO[0001] new message                                   fields.msg=hello
```
```
./bin/arm/mdns -port 4002
    <omitted standart logs>
    INFO[0000] [REMOTE] server started                       listenAddr="127.0.0.1:4002"
    TRAC[0000] [EVENTSTREAM] subscribe                       id=1432518515 subs=1
    TRAC[0000] [PROCESS] started                             pid="127.0.0.1:4002/chat"
    TRAC[0000] [PROCESS] started                             pid="127.0.0.1:4002/mdns"
    INFO[0000] [DISCOVERY] remote discovered                 ID=engine_1682994395132833000 addrs="127.0.0.1:4001"
    TRAC[0000] [INBOX] started                               pid="127.0.0.1:4002/stream/127.0.0.1:4001"
    TRAC[0000] [STREAM WRITER] connected                     remote="127.0.0.1:4001"
    TRAC[0000] [STREAM ROUTER] new stream route              pid="127.0.0.1:4002/stream/127.0.0.1:4001"
    INFO[0000] new message                                   fields.msg=hello
```

# References
- [mDNS](https://github.com/grandcat/zeroconf.git) library for golang

# License

Hollywood is licensed under the MIT licence.
