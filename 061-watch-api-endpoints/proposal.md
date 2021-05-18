# Summary

Allow watching several API endpoints for changes to avoid needing to perform frequent polling in the UI.


# Motivation

* Some API endpoints (most notoriously `ListAllJobs`) can be very taxing on both the database and the ATCs on large deployments. This is exacerbated by the fact that the UI is continually polling several of these endpoints for changes (currently every 5 seconds in most cases).
* In addition to being computationally expensive, the polling approach can consume a lot of network bandwidth. With several thousand jobs in the cluster, `ListAllJobs` alone can easily hit e.g. 10MB per request - if the ATC/DB is able to keep up with the 5 second poll interval, that equates to 1GB of network traffic every ~10 minutes (for just keeping the dashboard open!). In many cases, the actual relevant change in the jobs data will be very small comparatively (I would speculate in the low-MBs range).
* Changes can be reflected more quickly in the UI, resulting in a more snappy UX. This is the least compelling motivation IMO, but still has merit.


# Proposal

At a high level, I propose making use of Postgres' notification bus ([`NOTIFY`](https://www.postgresql.org/docs/9.5/sql-notify.html) and [`LISTEN`](https://www.postgresql.org/docs/9.5/sql-listen.html)) for detecting and propagating changes to the API handlers, and [server sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) for sending these changes from the API handler to the clients (e.g. UI). Let's get a bit more concrete:

## API

By default, hitting the `ListAllJobs` endpoint will not start watching - it will behave as it currently does (i.e. return all of the visible jobs as JSON). To opt-in to watching, clients should set the `Accept` header to `text/event-stream`.

I propose the following interface(s) to be consumed by the API Handlers:

```go
type EventType string
const (
    Put    EventType = "PUT"
    Delete EventType = "DELETE"
)

type DashboardJobEvent struct {
	ID   int
	Type EventType
	Job  *atc.DashboardJob
}


type ListAllJobsWatcher interface {
    WatchListAllJobs(ctx context.Context, access accessor.Access) (<-chan []DashboardJobEvent, error)
}

// and other ...Watcher interfaces for different endpoints
```

`WatchListAllJobs` will send along the returned `chan` whenever a job (that is visible to the user according to `access`) is inserted, updated, or deleted. In the case of a `Delete`, the corresponding `*atc.DashboardJob` will be `nil` - otherwise, it will contain the job to be sent directly to the client.

`WatchListAllJobs` returns an error to support a few cases:

1. Watch endpoints are disabled (enabling watch endpoints will be feature flagged, at least to start out with)
1. We could set limits on the total number of watchers - if adding one more watcher would exceed this limit, we'd return an error

The API Handler will first send an event of the form:

```
id: 0
name: initial
data: [
  // all current jobs (what we'd return without watching)
]
```

and will subsequently send events of the form:

```json
id: 123
name: patch
data: [
  {
    "id": 1,
    "eventType": "PUT",
    "job": {
      // job 1
    }
  },
  {
    "id": 2,
    "eventType": "DELETE"
  },
]
```

Note that I'm using `ListAllJobs` as an example, but a similar pattern can be followed for other endpoints.

## Database

We currently use Postgres' notification bus for communication between components of the ATC by manually creating notifications using `NOTIFY`. These notifications are simply triggers in that they carry no payload - the notifications just act to say "something on my end has changed, now do your thing".

I envision using the same notification bus with a few differences in how we make use of it:

1. Rather than manually creating notifications, add a trigger to **each relevant Postgres table** that looks something like:
   ```sql
   CREATE TRIGGER some_table_notify AFTER INSERT OR UPDATE OR DELETE ON some_table
   FOR EACH ROW EXECUTE PROCEDURE notify_trigger(
     // ... NOTIFY the channel with appropriate payload
   );
   ```

   * This will prevent needing to add a `NOTIFY` to every place where the data could change by handling it at the source of truth.
   * Determining **each relevant Postgres table** isn't necessarily something we should automate. For instance, we *could* watch all tables that we `JOIN` against in the `ListAllJobs` query for all types of changes, but that would be very noisy and would result in duplicate updates

1. Our notifications will contain a payload consisting of the key of the entity being modified as well as the event type, e.g.
   ```json
   {
     "event": "CREATE" | "UPDATE" | "DELETE",
     "id": 123
   }
   ```

   The payloads do not contain the data of the modification - only the primary/foreign key(s) of interest

   For tables that multiple queries `JOIN` on, the `id` could contain multiple values, e.g.
   ```json
   {
     "event": "CREATE" | "UPDATE" | "DELETE",
     "id": {
       "job_id": 123,
       "pipeline_id": 456
     }
   }
   ```

   and the `...Watcher` implementations can consume the id that's relevant to them.

1. Currently, our internal notification bus does not queue notifications up - if we receive a new notification, but the consumer hasn't processed it yet, we ignore the new notification. This made sense for our prior use cases (we only want to signal that something has changed), but doesn't when we want to know about every change that happens. You can find a potential implementation for this new behaviour here: https://github.com/concourse/concourse/pull/5802/commits/3686fed7d936b09d8179753c506d81fbcba95017

We'd then have an implementation of the previously mentioned `ListAllJobsWatcher` interface that makes use of the notification bus:

```go
type Watcher struct {
    conn db.Conn
    ...
}

func NewWatcher(conn db.Conn, ...) (*Watcher, error) {
    watcher := &DBWatcher{conn: conn}
    err := watcher.listenForListAllJobs()
    if err != nil {
        return nil, err
    }
    ... // other endpoints
    return watcher
}

func (w *Watcher) listenForListAllJobs() error {
    jobNotifs, err := watcher.conn.Bus().Listen("jobs")
    if err != nil {
        return err
    }
    pipelineNotifs, err := watcher.conn.Bus().Listen("pipelines")
    if err != nil {
        return err
    }
    // ... subscribe to all other tables of interest
    go func() {
        for {
            var notif db.Notification
            select {
            case notif = <-jobNotifs:
            case notif = <-pipelineNotifs:
            ...
            }
            // parse notification
            // read data from DB if appropriate (i.e. UPDATE or INSERT and there are any subscribers)
            // notify all subscribers that have access to this job (i.e. are on the same team, or the pipeline is visible)
        }
    }()
}
```

### Access Control

Since all notifications for a given table are going through the same Postgres channel, we'll obviously need to be careful about not serving events to users that don't have access to the entities that are being modified. For instance, if a user from `teamA` is watching `ListAllJobs`, we should not send them updates to a job residing in `teamB`. For this, the `...Watcher` interfaces take in an `accessor.Access` that includes enough information to make access control decisions on a per-subscriber basis.

# Open Questions

* What are the performance implications of Postgres' LISTEN/NOTIFY, or of having TRIGGERS for every modification in a table?
* Are there any limits on LISTEN/NOTIFY that might hinder this approach at scale?
* Should there be a timeout on the request?
* What's the best way to determine the list of "relevant tables" to watch?


# Answered Questions


# New Implications

* The API will still support the endpoints without watching, which should have the same behaviour as we have now
* Watch endpoints can also be feature flagged, and the UI will fall back to polling
