# Summary

Introduces P2P Volume Streaming, an enhancement of volume streaming that will
benefit clusters where workers are reachable from each other.

# Motivation

When Concourse launches a container on a worker, if some required volumes locate
on other workers, then Concourse will automatically copy the volumes to the worker.
This process is called volume streaming.

With the current Concourse implementation, volume streaming is done via ATC, in
other words, volume data is transferred from source worker to ATC first, then 
ATC transfers the data to dest worker. The design fits deployments where workers
don't see each other.

However, for deployments where workers can see each other, volume streaming can be
optimized by streaming volumes from a worker directly to the other. [A study](https://ops.tips/notes/concourse-workers-streaming-improvements/) 
has been done by [Ciro S. Costa](https://github.com/cirocosta) that shows how P2P volume 
streaming benefits.


# Proposal

## Workflow

The current volume streaming worker flow is:

```                           
source-baggageclaim       ATC               dest-baggageclaim
      |                    |                    |
      |   PUT stream-out   |                    |
      | <----------------- |                    |
                           |    PUT stream-in   |
                           | -----------------> |
```

The URL that ATC uses to access worker (baggageclaim) is not visible by workers. Thus
to allow source worker to access dest worker directly, ATC needs to ask dest worker for
its public IP. So we need to add a bc API `bc-p2p-url`, so that ATC can get dest worker's
baggageclaim url, then ATC invokes a new bc API `stream-p2p-out` to source worker, then 
source worker calls dest worker's bc API `stream-in` to ship volumes to dest worker.

```                           
source-baggageclaim       ATC               dest-baggageclaim
      |                    |                    |
      |                    |   GET bc-p2p-url   |
      |                    | -----------------> |
      | PUT stream-p2p-out |                    |
      | <----------------- |                    |
      |                                         |
      |               PUT stream-in             |
      | --------------------------------------> |
```

## Baggage-claim API changes

### 1. Add `GET /p2p-url`

This API takes no parameter and returns a HTTP URL that should be accessible from other
workers.

### 2. Add `PUT /stream-p2p-out?destUrl=<dest bc url>&encoding=<gzip/zstd>&path=<dest path>`

This API guides source worker to stream a volume directly to dest worker. It takes three
parameters:

* `destUrl`: dest baggage-claim's p2p url
* `encoding`: data compression method, `gzip` or `zstd`
* `path`: will be passed to `stream-in`

## Worker CLI option changes

* `p2p-interface-name-pattern`: using this pattern to find a network interface and use its IP address.
* `p2p-interface-family`: 4 or 6, meaning use IPv4 or IPv6 address.

## Web CLI option changes

A new cli flag `--enable-p2p-volume-streaming` should be added to opt-in the feature. 
By default, volume streaming will still go through ATC.


# Open Questions

1. About `baggageclaim-bind-ip`. To allow both ATC and other workers to connect, `baggageclaim-bind-ip`
has to be set to `0.0.0.0`. If workers are behind a firewall, which should not be a problem. However, if 
workers are on public cloud, is that a security concern? If yes, then we may consider using dedicate
bind-ip+port for p2p streaming.

2. As volume streaming is initiated from source worker, source workers knows size of the volume to stream
out, the other optimization is that, when streaming small files, like small text/json/yaml files, and so on,
transferring raw files might be cheaper than shell-executing `tar` command to compress the files then transferring.


# Answered Questions


# New Implications

