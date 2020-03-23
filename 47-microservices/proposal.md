# Summary

Let's create a way to distribute the various procedures that the web node runs across multiple unix processes.

# Motivation

## A story of observability pain

Recently I was required to debug a Concourse cluster, in production, that was having essentially a total outage: The web nodes were no longer serving API requests. They were completely maxing out their memory consumption to the point that they failed their health checks and got restarted by monit (this was a bosh deployment).

I was on a long phone call with some stressed-out operators and everybody was somewhat-aimlessly brainstorming. At one point, somebody said

> it looks like there's a regular pattern of steps - the memory shoots up, stays at max for a minute or so, falls down to zero and then shoots up within about another minute. This doesn't look like a leak, but more like a resource-consumption bug in some kind of periodic asynchronous procedure.

She could definitely be right. In my heightened state (I was trying to fix a production issue in real-time over a Zoom call) I started rapidly reading through the codebase - what are all the periodic asynchronous procedures that the web nodes run?

I was looking through `atc/atccmd/command.go` and realized that the things I was looking for ("periodic asynchronous procedures") all fit more precisely into the category of `github.com/tedsuo/ifrit/grouper.Member` values.

https://github.com/concourse/concourse/blob/release/5.5.x/atc/atccmd/command.go#L796-L890

I started frantically tabulating each of these `grouper.Member`s and the intervals on which they ran, hoping to find something that might possibly cohere with my colleague's observation. Unfortunately it came to nothing, but the plot thickens.

Eventually we got some heap dumps using the pprof endpoint:

https://github.com/concourse/concourse/blob/release/5.5.x/atc/atccmd/command.go#L11

and then we discovered that the `ListAllJobs` endpoint was ultimately mostly responsible for the memory consumption, and we made a PR adding a config flag that to disable specifically that badly-behaved endpoint: https://github.com/concourse/concourse/pull/5340

## A request for non-invasive operator-configurable observability

How can we improve situations like this in the future? I would like the following acceptance criteria:

A way to report compute-resource consumption of Concourse by "member" (or smaller), such that
* data can be harvested in real-time
* operators can configure how they wish to group the members
* operators can opt-in or -out of each member's functionality, at a level of granularity that permits disabling a single endpoint.

Probably the existing mechanism of Linux processes and all of the existing technology for observing them should be leveraged to the greatest extent possible - we can always add our own advisory/application-level monitoring but it's probably always going to be chasing old incidents and won't be able to give adequate information about real-time production scenarios.

I can imagine a world where a common way to introduce Concourse features is by adding or removing "members" of this kind - frankly, `lidar`, the big feature in v5.6.0, was basically a drop-in replacement for the existing `radar` that performed resource checks.

I think there's a lot of value in placing control over these things in the hands of the operator, especially when operators may have strict change management controls that forbid them from modifying the actual software in production, but may be more lenient about making configuration changes.

# Proposal

Broadly, we need a solution that lets operators choose whether they want to run the various Concourse computations in `ifrit/grouper.Members` or actual separate unix processes -- these things are supposed to be nearly-interchangeable concepts anyway.

## More subcommands

Concretely, I want finer-grained `concourse` subcommands besides `quickstart`, `web` and `worker`. I want a bit more - if we look through the past at events where a new "member" was added, I want to know that there will always be a sub-command for it.

As an initial iteration, why not the following:
* `concourse ui` for the frontend, web handlers
* `concourse api` for the http and https servers
* `concourse backend` for the scheduler/gc/logcollector/syslogdrainer/etc
* `concourse gateway` for the TSA

## More flags

Another option, rather than aggressively creating these subcommands , we could add flags to enable/disable each one. Then operators could add a `DISABLE_*` env var when they suspected a component might be misbehaving, or they could set up deployment manifests where each member was running in a separate process/container by carefully toggling all the members but one at a time.

# Open Questions

I know, it's basically "microservices with a shared database", but that's pretty much what our architecture has already been anyway.

Maybe this approach could lead to each member maintaining its own persistence? Moving towards a (hopefully more effective) distributed persistence solution? If we knew that there was a strongish decoupling between the build event server and the rest of the system (like it was the only one that did reads on the `build_events` table), maybe this would open the possibility of configuring a different storage backend (like a blobstore or a [second postgres](https://github.com/concourse/concourse/issues/5306) or what have you)

# Answered Questions

# New Implications

Do we go all-in on microservices accoutrements? Service discovery? A message bus? the 'components' table is pretty much a service registry, as far as I understand the concept. the TSA is a decent choice for a service registry API server, since it's already expected that it will be publicly reachable for worker registration.
