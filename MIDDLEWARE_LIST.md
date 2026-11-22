# Middleware list
The middlewares we use are:
The *Logger*, the *Router*, the *Limiter*, the *Authenticator*, the *Load Balancer*, and the *Proxy*.

The global information that we need from the gateway is:
*Redis*: Stores a redis client for the middlewares to use. 
*Logger*: Stores a log client that acts like a file descriptor that we can write to.
*Endpoints*: Stores a list of endpoints, with their constructed target list.
*Version Number*: A hash generated at the beginning to tell which iteration we are running.

The gateway also needs a WaitGroup semaphore. This is how we can tell if the gateway is safe to fully close. 
If we need to close this gateway iteration, we shut down the incoming interface, we wait until the WG is 0, then we close all
the resources. This is to ensure the resources do not close while in use. 

The data for the middlewares needed is the following:
### Limiter
The limiter may limit on *Global*, *Path*, and *Host* values. 
The limiter runs on these values if a configuration is built for them. Each of these limiters should 
represent their own seperate sub-objects called *Buckets*. 
In our project, limiters may reference either a *Redis Bucket* or a *Memory Bucket*.

If we use redis buckets, we need a *Version Numbering* and a *Redis Client*. 

### Router
The router relates *Paths* to *Endpoints*. Each endpoint has a *Target List* that may need a *Load Balancer* algorithm.
The router current has only 1 usable implementation, the *Trie Router*. 

The router should not own the endpoints, it should be stored in the gateway. It references these endpoints in its trie,
it does not store them there. It stores pointers into the endpoint slice to retrieve them. This is to make health-checking 
easier.

### Authenticator
The authenticator needs an *Endpoint* to be attached to know what private RSA key to use. 
The authenticator needs a relation between *Key IDs* and *Private RSA Keys*. It owns this relation.
It also needs to be able to check requests for correct authentication information.

The authenticator accesses endpoints, but needs no global information itself.

### Logger
The logger needs a *Logger*. It calls functions on this logger, but should not own it
to allow other middlewares to log. 

### Load Balancer
Relates all of the endpoints with a load-balancing algorithm package. We will use round-robin, so it stores
a counter for each endpoint which is the next target to hit. 

Needs the endpoint list, does not own it. 

### Proxy
The proxy needs a *Target* from a given endpoint. This target should be selected before it. It simply streams
a request to the target and then back to the host. 

