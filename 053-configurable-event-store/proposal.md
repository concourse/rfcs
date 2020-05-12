# Summary

This RFC proposes a way to support pluggable external stores for build events
and to provide a general mechanism for processing build events.


# Motivation

The first (and primary) motivation for this change is to allow operators to
configure external data store for build events. They may want to do this for a
number of reasons:

1. Builds can generate an enormous amount of build events - storing everything
   in the same database as all of the other core-Concourse information means
   that, over-time, the `build_events` tables will tend to dominate (in many
   cases). For instance, in our (fairly small) Concourse deployment at
   `ci.concourse-ci.org`, build events comprise ~70% of the total ~13GB used.
   While storage is cheap, I can imagine this could result in general sluggishness
   in larger Concourse deployments as indexes get large. While much of this can be
   alleviated by more aggressive build log reaping, that can introduce performance
   problems of its own, since deleting lots of build events can be taxing on
   Postgres. I'm sure it's fairly obvious at this point that I really don't know
   much about running Postgres/Concourse at scale, so let me know if what I'm
   saying here doesn't hold true.
1. Postgres wasn't designed for handling huge amounts of text data. Again, I
   don't have much evidence/expertise to back this up, but having many large
   rows in a `build_events` table strikes me as sub-optimal (compared to a system
   designed for handling large amounts of text).
1. Storing build events in a system designed for handling text data (like
   Elasticsearch, for instance) allows you to do some pretty cool querying that
   isn't feasible in Postgres.
1. Gaining observability into build events isn't great. The syslog drainer
   functionality mostly addresses this usecase, but isn't ideal IMO - it runs
   on an interval (rather than continuously), and if you syslog forward build
   events to an external store, you can choose to either store the build events in
   two places, or reap the build events from Postgres (meaning you won't be able
   to view the logs after).

Some discussion around this motivation can be found in this discussion: concourse/concourse#5306.

The second motivation is to introduce a general mechanism for processing build
events. I can foresee having a configurable data transformation pipeline for
build events that modifies events before storing them. I have a few ideas about
some event processing that could be useful:

1. Secret redaction - while we already have this feature, I think it's
   implementation could be simplified by thinking of it as a data
   transformation step (i.e. it receives log events and emits new log events with
   secrets redacted), rather than trying to redact secrets on the fly.
1. Server-side ANSI processing, à la concourse/concourse#3879

Additionally, we could introduce behaviour like continuous syslog forwarding
and migrating completed builds to cold storage fairly easily. This is explored
more in [`EventStore` Composition](#event-store-composition).


# Proposal

To support having custom event stores, introduce an `EventStore` interface:

```go
type Key interface{}

type EventStore interface {
    Setup() error
    Close() error

    Initialize(build db.Build) error
    Finalize(build db.Build) error

    Put(build db.Build, events []atc.Event) error
    Get(build db.Build, requested int, cursor *Key) ([]event.Envelope, error)

    Delete(builds []db.Build) error
    DeletePipeline(pipeline db.Pipeline) error
    DeleteTeam(team db.Team) error

    UnmarshalKey(data []byte, key *Key) error
}
```

Let's unpack each function:

* `Setup()` is used to, well, perform any required setup. It'll be called once
  when Concourse starts up. For an `Elasticsearch` implementation of
  `EventStore`, for instance, `Setup()` would likely initialize a connection to
  the Elasticsearch cluster. For a `Postgres` implementation, `Setup()` will
  check if the `build_events` table exists, and if not, create it (note: this
  was previously done by DB migrations - more on this in [Migrations](#migrations))

* `Close()` is for cleaning up any left over resources. For instance, if a
  connection was opened to an Elasticsearch cluster in `Setup`, `Close` should
  gracefully terminate the connection. However, it should not necessarily be the
  inverse of `Setup` - e.g. for `Postgres`, `Setup` creates the `build_events`
  table, but `Close` should not drop it.

* `Initialize(build db.Build)` will be triggered when a build is first created, and
  before any build events are `Put` into the store. For the `Postgres`
  implementation, this will create a sequence like `build_event_id_seq_x` for the
  current build. It will also create either the table `pipeline_build_events_x`
  or `team_build_events_x`, depending on whether it's a one-off build or not.
  This represents a behaviour change from what we have now, where this table is
  created when the team/pipeline is first created, but it shouldn't really have
  much impact.

* `Finalize(build db.Build)` will be triggered when a build is `Finish`ed. No
  build events may be `Put` into the store after this is called. For
  `Postgres`, this involves dropping the `build_event_id_seq_x` sequence.

* `Put(build db.Build, events []atc.Event)` is called whenever
  `build.SaveEvent(event)` is called in order to write build events to the
  external store. The reason for accepting a list of `events` (despite
  `db.Build.SaveEvent` only ever passing in a single event) is because if we
  want to migrate build events to another `EventStore`, it would be super slow
  making a huge number of small `Put` calls when we can do one batch `Put` per
  Build.

* `Get(build db.Build, requested int, cursor *Key)` fetches events from the
  `EventStore`, starting from an initial `Key` (which is excluded from the
  result set - e.g.  if `cursor` points to the 10th `Key`, the 11th event will be
  the next one fetched). `Key` can be any type, and is implementation-specific.
  For `Postgres`, this will likely be something like `uint`, but other stores
  (e.g. DynamoDB) that don't provide easy support for auto-incrementing IDs may
  choose a different type of `Key`. Regardless of the type of `Key`, if the goal
  is to fetch all events, `cursor` should initially point to `nil` (but the
  pointer itself must not be `nil`). For instance, you can iterate over all
  events for a completed build like so:

  ```go
  var cursor Key
  batchSize := 1000
  for {
      events, err := eventStore.Get(build, batchSize, &cursor)
      if err != nil {
          panic(err)
      }
      // process events here
      if len(events) < batchSize {
          // there are no more events (see note below about `requested`)
          break
      }
  }
  ```

  However, when possible, consumers of these build events should prefer to use
  `db.Build.Events()`, which queries the `EventStore` and subscribes to a
  notification bus to allow tracking in-progress builds (more on this in
  [Notifications](#notifications)).

  `Get` also takes in `requested` number of events to fetch. This is a soft
  limit. To explain what I mean by this, suppose there are `n` events available
  in the `EventStore` (after the `cursor`):
  * If `n < requested`, `Get(...)` should return all `n` events
  * If `n >= requested`, `Get(...)` should return at least `requested` events,
    but may return more than `requested` if convenient (i.e. it may return
    `requested <= x <= n`). If the backend store doesn't provide an easy way to
    read exactly `requested` elements, and the `EventStore` reads in a chunk of
    data that results in more than `requested` events, there's no use in throwing
    the extra events away.

* `Delete(builds []db.Builds)` deletes the build events for a list of builds.
  This is used by the build log collector.

* `DeletePipeline(pipeline db.Pipeline)` deletes all of the builds for a given
  pipeline. While this could be done by doing something like:

  ```go
  builds, _ := pipeline.Builds()
  eventStore.Delete(builds)
  ```

  ...this doesn't work so well for the Postgres implementation. Currently, when
  we delete a Pipeline from the database, `TRIGGER`s will `DROP` the
  `pipeline_build_events_x` table for the appropriate pipeline. This results in
  much more performant build event deletions, since dropping a table is faster
  than deleting many thousands/millions of rows. I'd be happy to hear any clever
  ideas anyone has on how to just have a single `Delete` method that acts on
  builds while achieving the same table-dropping ability for Postgres.

* `DeleteTeam(team db.Team)` deletes all builds for a given team. Same
  reasoning as `DeletePipeline()`.

* `UnmarshalKey(data []byte, key *Key)` works like `UnmarshalJSON` - give it
  some bytes and tell it where you want to put the result. This is used e.g.
  for the `/api/v1/builds/build_id}/events` endpoint where you can pass in a
  `Last-Event-ID` header - we would call `UnmarshalKey` on this raw value, and
  the `EventStore` would parse it appropriately. For instance, for Postgres, we
  would expect `data` to look like a number in ASCII, so we could parse it
  appropriately.

  This is only necessary if we care about the `Last-Event-ID` header from what
  I can tell (it doesn't appear that any internal code requires starting events
  from a certain point - hence no need for a `MarshalKey` method). I left as an
  open question whether we want to keep `Last-Event-ID`.


## <a name="event-store-composition">`EventStore` Composition</a>

In addition to enabling Concourse to interact with external stores besides
Postgres, we can construct `EventStore` implementations that compose other
`EventStores` to do some interesting things. One application is data
transformations. I'll take the example of secret redaction:

```go
type SecretRedactingEventStore struct {
    EventStore
    // Some cred vars tracking state
    ...
}

func (s *SecretRedactingEventStore) Initialize(build db.Build) error {
    s.initializeCredVarsTrackingForBuild(build)
    return s.EventStore.Initialize(build)
}

func (s *SecretRedactingEventStore) Put(build db.Build, events []atc.Event) error {
    redactedEvents := make([]atc.Event, len(events)
    for i, evt := range events {
        redactedEvents[i] = s.redactSecrets(build, evt)
    }
    return s.EventStore.Put(build, redactedEvent)
}
```

Another application is continuous syslog forwarding:

```go
type SyslogForwardingEventStore struct {
    EventStore
    // Some syslog forwarding state
}

func (s *SyslogForwardingEventStore) Put(build db.Build, events []atc.Event) error {
    if err := s.syslogForwardEvents(build, events); err != nil {
        return err
    }
    return s.EventStore.Put(build, event)
}
```

How about migrating build events to cold-storage after they're completed:

```go
type ColdStorage interface {
    Create(filename string) (io.WriteCloser, error)
    Open(filename string) (io.ReadCloser, error)
}

type ColdStorageEventStore struct {
    EventStore
    ColdStorage ColdStorage
}

func (c *ColdStorageEventStore) Finalize(build Build) error {
    if err := c.EventStore.Finalize(build); err != nil {
        return err
    }
    // perhaps run this in the background
    return c.migrateToColdStorage(build)
}

func (c *ColdStorageEventStore) migrateToColdStorage(build Build) error {
    file, err := c.ColdStorage.Open(fmt.Sprintf("build_%d", build.ID()))
    if err != nil {
        return err
    }
    if err != nil {
        file.Close()
        return err
    }
    enc := json.NewEncoder(file)
    batchSize := 1000
    var cursor Key
    for { 
        events, err := c.EventStore.Get(build, batchSize, &cursor)
        if err != nil {
            file.Close()
            return err
        }
        for _, evt := range events {
            if err = enc.Encode(event); err != nil {
                file.Close()
                return err
            }
        }
        if len(events) < batchSize {
            break
        }
    }
    if err = file.Close(); err != nil {
        return err
    }
    if err = c.EventStore.Delete(build); err != nil {
        return err
    }
    return nil
}

func (c *ColdStorageEventStore) Get(build Build, requested int, cursor *Key) ([]event.Envelope, error) {
    if build.IsCompleted() {
        // Note that we're ignoring `requested` here: if we read from cold storage,
        // we have to download the whole file, so it might as well return all of the events
        events, err := c.readFromColdStorage(build)
        if err == nil {
            return events, nil
        }
        // if fail to read from cold storage, assume it's because we failed to create
        // the file in cold storage initially. fallback to the primary event store
    }
    return c.EventStore.Get(build, requested, from)
}

func (c *ColdStorageEventStore) readFromColdStorage(build Build) ([]event.Envelope, error) {
    file, err := c.ColdStorage.Open(fmt.Sprintf("build_%d", build.ID()))
    if err != nil {
        return nil, err
    }
    defer file.Close()
    dec := json.NewDecoder(file)
    var events []event.Envelope
    for {
        var evt event.Envelope
        if err := dec.Decode(&evt); err == io.EOF {
            break
        } else if err != nil {
            return nil, err
        }
        events = append(events, evt)
    }
    return events, nil
}
```

The point I'm trying to illustrate is that the `EventStore` hooks allow you to
do some pretty cool things when you compose `EventStores`. If you can think of
any hooks that are missing that could support other useful functionality, I'm
all ears!

## <a name="notifications">`Notifications`</a>

As I mentioned before, while you can iterate over a completed build's events by
looping over `EventStore.Get`, this doesn't work so well for builds that are
still producing events, since (in general) there's no way to know when new
events come in (besides polling). The solution that I came up with is to use
the Postgres notification bus to notify `db.Build.Events()` of new build
event(s). It feels a bit weird to rely on Postgres for this even when using an
external store that shouldn't need to touch Postgres - I left an open question
about whether people feel this is a bad idea, and another question about
whether there are use cases where `EventStores` would benefit from using their
own native pub/sub system (e.g. MongoDB has a `collection.watch()`
functionality).

Note that `EventStore` implementations don't need to think about notifications.
This all happens at the DB layer:

```go
func (b *build) SaveEvent(event atc.Event) error {
    // use the `EventStore` to save
    err := b.eventStore.Put(b, event)
    if err != nil {
        return err
    }
    // ...but notify the main Postgres DB's notification bus.
    return b.conn.Bus().Notify(buildEventsNotificationChannel(b.ID()))
}
```

```go
func (b *build) Events(from uint) (EventSource, error) {
    notifier, err := newConditionNotifier(b.conn.Bus(), buildEventsNotificationChannel(b.ID()), func() (bool, error) {
        return true, nil
    })
    if err != nil {
        return nil, err
    }

    return newBuildEventSource(
        b,
        b.conn,
        b.eventStore, // fetch events from the EventStore using EventStore.Get(...)
        notifier,     // but subscribe to the Postgres notification bus to know when new events arrive
        from,
    ), nil
}
```

I've toyed with the idea of having an optional interface that `EventStore`
implementations can implement that would overwrite this default...something
like:

```go
type EventNotificationBus interface {
    Notify(build Build) error
    Listen(build Build) (<-chan struct{}, error)
}
```

But I'm not sure if there's much value in that.


## <a name="migrations">Migrations</a>

Currently, we have migrations to set up the `build_events` table, as well as
`TRIGGER`s for creating/deleting `pipeline_build_events_x` and
`team_build_events_x` tables when pipelines and teams are created/deleted.

I propose that we comment out these up migrations (so future deployments don't
get these tables/triggers), and leave any database initialization (or whatever
appropriate backend setup) to `BuildEvent.Setup()`. This means `Setup` will
have to be "smart" (i.e. have to handle the cases where the initialization has
already been applied). It also makes it difficult to make changes to the
schema, unlike with migrations - each `EventStore` (that stores data in a
structured way - this may just end up being Postgres) will need to basically
implement its own form of migrations if we need to change the schema. My guess
is that we won't be changing the schema too often, though (given that we
haven't changed `build_events` since the initial migration).


## Configuration

One thing I haven't really explored is how operators would configure the
backend store. I think it could be similar to how Credential managers are
configured - there are groups of independent configuration options for each
possible `EventStore`. If certain flags are configured for a given backend, use
that backend.


# Open Questions

* Does `Key` need to be `interface{}`, or can it be more specific like `uint`?
  For backends that don't support auto-incrementing, but still need some
  ordering (if the timestamp isn't sufficient), we **could** create a Postgres
  sequence for every backend (external to the `EventStore`), and change the
  signature to `Put(build db.Build, eventID uint, event atc.Event)`

* Do we care about the `Last-Event-ID` header on the
  `/api/v1/builds/{build_id}/events` endpoint? I don't think we use it anywhere
  internally, but I guess we may want to keep it for backwards compatibility?
  It's not exposed by `fly` or `go-concourse`, either.

* Would any event stores benefit from bringing their own notification bus
  implementation, rather than relying on Postgres' notification bus? Does using
  the Postgres notification bus defeat the purpose of having an external store in
  any way (e.g. could this be a bottleneck)?

* Users won't be able to access build events that exist in the main Postgres
  database if they switch to a new backend store. How can we make the
  transition easier? One idea I have is to have a `FallbackEventStore` that
  composes two `EventStores` - it will always `Put` to the primary one, but if
  `Get` fails or returns no events from the primary `EventStore`, try the
  fallback `EventStore`.


# Answered Questions


# New Implications

* By default, there will be no change - Concourse will default to using a
  Postgres `EventStore` using the same main Postgres database.
* Operators will be able to configure a different store
* If they configure a new store, they won't be able to access any existing
  build events that are currently stored in Postgres without migrating them
  over (how should we recommend doing this?)
