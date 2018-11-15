# scan queue

Let's increase the symmetry between the radar and scheduler packages.

Scheduling consists of
1. creating pending builds (BuildScheduler)
1. starting builds (BuildStarter)

On the other hand, scanning consists of
1. scanning/checking resources synchronously on a schedule or on demand
   (resourceScanner.scan)

If we took the two-step approach used in build scheduling, we could schedule
scans and have them run asynchronously. Ultimately this would buy us a similar
decoupling of the API from resource checking, because the "on-demand" version
of resource checking (fly check-resource/check-resource-type or webhooks) would
use the same interface as the "periodic" version (pipelineSyncer). So we
would see components like

1. ScanScheduler
1. ScanStarter
1. engine.execEngine learns how to createScan
1. engine.dbEngine learns how to createScan
1. engine.execScan
1. engine.dbScan
1. db.Scan

At this point it seems we don't need to dictate the implementation of how these
components communicate (i.e. a `scans` table paralleling the `builds` table vs
a notification bus paralleling the `buildEventsChannel` construct), as just
separating these two steps and using the database as a communication channel
should be enough to achieve the desired decoupling.

## pros

* decoupling of API from scanning
* fewer code paths related to scanning
* allows for "rate-limiting"/"backpressure" on resource checks
* allows for aborting resource checks
* removes duplication in resource checks
* allows for resource check history, and resource check error history

## cons

* two steps instead of one does introduce more complexity
* we would need to decide if the check handler would poll for a result in the
  database or if requests to the check endpoint would be async and anyone that
  requires a response would need to poll a separate endpoint.
* there seem to be some considerations around locking and the `mustComplete`
  flag that currently exists on the `scan` function.
