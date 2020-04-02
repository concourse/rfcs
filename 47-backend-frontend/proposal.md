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

Broadly, we need a solution that lets operators choose whether they want to run the various Concourse computations in `ifrit/grouper.Members` or actual separate unix processes -- these things are supposed to be nearly-interchangeable concepts anyway. We could provide this configurability at many levels of granularity, so let's focus this RFC on a simple case.

## Frontend and Backend

Concretely, I want finer-grained `concourse` subcommands besides `quickstart`, `web` and `worker`. Let's add two more:

* `concourse frontend` for the http and https servers serving the API and web UI
* `concourse backend` for the scheduler/gc/logcollector/syslogdrainer/etc

Ideally each of these subcommands should really only be concerned with its own configuration - i.e. there's no need to configure a gc interval on the frontend. However, they will have some overlapping configuration (like the database) that will need to be duplicated.

Furthermore, `concourse web` should continue to behave the same as it does now. 

This would coarsely solve the problem posed in the above narrative - we could have quickly diagnosed that the memory consumption was coming from the `frontend` process.

# Open Questions

# Answered Questions

# New Implications
