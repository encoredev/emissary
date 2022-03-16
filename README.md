<div align="center">
  <a href="https://encore.dev" alt="encore"><img width="189px" src="https://encore.dev/assets/img/logo.svg"></a>
  <h3><a href="https://encore.dev">Encore – The Backend Development Engine</a></h3>
</div>

# Emissary

When you deploy an [Encore application to your own cloud](https://encore.dev/docs/deploy/own-cloud) Emissary is deployed
alongside. It provides the Encore platform a way to access resources with your clouds private network in an authenticated
manner.

It's primary use right now is to allow the Encore platform a way to tunnel to your 
[SQL database](https://encore.dev/docs/develop/databases) to perform the database migrations to your application and to
allow [access to the database from the Encore CLI](https://encore.dev/docs/develop/databases#connecting-to-databases)
                                                                                                                          

## How it works

Emissary is split into two modules:
- [the `server` module](./server) which is the binary we deploy to your cloud.
- [the `library` module (this folder)](.) which serves as the client library and is used by our platform to establish connections.
                                                                                                                                       
The library provides a `emissary.Dialer` which provides a `Dial` and a `DialContext` method. These can be provided to most
Go networking enabled code as an underlying `Dial` function. When that other Go code tries to open a socket, it will use
the `emissary.Dialer` to establish the socket, which in-turn will be transparently routed through the Emissary server.

The Emissary server upon receiving the request, with authenticate the request using a shared secret key. Once
authenticated, the dialer can then request the server forwards the connection onto one of a predefined list of
allowed remote resources within the private network.

At the bottom most level, Emissary is simply a Socks 5 client/server however we wrap the socks 5 connection in a
transport layer, which allows us to run Emissary in various locations to adapt to requirements in the target cloud.
                                                                     
```
                      ┌─────────────────────────────────────────────────────────────────────────┐
                      │Go `net.Conn`                                                            │
┌────────────────┐    │  ┌────────────────┐     ┌─────────────────────┐     ┌────────────────┐  │    ┌────────────────┐
│                │    │  │                │     │Transport Layer      │     │                │  │    │                │
│ Go             │    │  │ Emissary       │     │ ┌─────────────────┐ │     │ Emissary       │  │    │ Target         │
│                ├───►│  │                ├────►│ │Socks 5 Protocol │ ├────►│                │  ├───►│ Remote         │
│ Client         │    │  │ Dialer         │     │ └─────────────────┘ │     │ Server         │  │    │ Resource       │
│                │    │  │                │     │                     │     │                │  │    │                │
└────────────────┘    │  └────────────────┘     └─────────────────────┘     └────────────────┘  │    └────────────────┘
                      │                                                                         │
                      └─────────────────────────────────────────────────────────────────────────┘
```
                
Currently, the only supported transport layer is a `websocket`; this transport layer allows Emissary to run behind HTTP
aware load balancers or in environments which only allow HTTP traffic in. In such environments we expect TLS termination
to have occurred at the edge before the code executes such as AWS Lambda functions.

To learn how to configure an Emissary server see [example.env](./server/example.env). The server will load the configuration
from either environmental variables or an `.env` file located within the working directory.
