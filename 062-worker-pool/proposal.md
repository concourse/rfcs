# Summary

This proposal outlines the support for worker pool. A worker pool is a group
of Concourse workers, that can be associated with one or multiple teams.

# Motivation

One or multiple Concourse workers can be associated with a team by adding 
`--team <team name>` option when launching `concourse worker`. Then all 
pipeline containers of the team will be dispatched to the workers. The 
relationship between workers and teams are N-to-1:

```
  Workers (0..n) <----------------> (0..1) Team
```

This is a useful mechanism to isolate some teams' workload or to provide
dedicate workers to some teams.

In practice, some big projects may split their pipelines to multiple small
teams. When a project that has multiple teams requests dedicate workers,
it is hard to give. Because the project actually wants some workers sharing
with all of its teams, but Concourse only support workers per team.

For example, a big project of my company has 100 teams. By evaluating
workload of those teams, the project requests 20 dedicate workers. However, due
to the restriction of 1 worker serving at most 1 team, we have to deploy at
least 100 workers (1 worker for a team) to the project. Moreover, each team 
having a single worker is unreliable, if the worker is down, then all builds of
the team will fail. If we deploy 2 workers for each team for redundancy, then
we need to deploy 200 workers. Comparing 200 workers we need to deploy with 20
workers they request, it is a huge waste.

# Proposal

With introducing a new concept "worker pool", we build N-N relationship between
workers and teams:

```
  Workers (0..n) <----> (0..1) Worker Pool (0..n) <------> (0..n) Teams
```

* A worker pool may contain 0 or multiple workers.
* A worker can be added to at most 1 worker pool.
* A worker may also not belong to any worker pool.
* If a worker is added to a worker pool, then it cannot be associated with a
team directly.
* If a worker is not added to any worker pool, then it can still be associated
with a team directly for backward-compatibility.
* A worker pool may be assign to 0 or multiple teams.
* A team can be associated with 0 or multiple worker pools. When a team is 
associated with multiple worker pools, candidate workers will be union of
those pools.

The logic of choosing a worker for a step will look like:

```go
   team := step.TeamName()
   var candidateWorkers = map[string][]Workers {
      "primary":  []Workers{},
      "fallback": []Workers{},
   }
   if team.WorkerPools() != nil {
      for _, pool := range team.WorkerPools() {
         candidateWorkers["primary"] = append(candidateWorkers["primary"], pool.Workers())
      }
      candidateWorkers["fallback"] = AllSharedWorkers()
   } else if team.Workers() != nil {
      candidateWorkers["primary"] = append(candidateWorkers["primary"], team.Workers())
   } else {
      candidateWorkers["primary"] = append(candidateWorkers["primary"], AllSharedWorkers())
   }
   // chooseWorker should choose a worker from primary workers first, if not found then
   // try fallback workers.
   chosenWorker, err := chooseWorker(candidateWorkers, step.Tags)
```

## Tag

There is no impact to tags.

When launching `concourse worker`, `--tag` option can still be used to mark the worker
with some tags. As the above code demonstrates, worker pool only impacts candidate workers,
but the algorithm of choose a worker from candidate workers.

## fly

### To list all worker pools:

```
$ fly -t <target> worker-pools
```

### To create a worker pool:

```
$ fly -t <target> set-worker-pool --name <pool name> [--public-worker-key=<public ssh key file>]
```

When creating a worker pool, it allows to optionally specify a ssh public key 
file. If a public key is configured with a pool, then workers trying to join
the pool must use the corresponding private key.

### To destroy a worker pool:

```
$ fly -t <target> destroy-worker-pool --name <pool name>
```

A worker pool should only be able to be deleted if there is no team associated 
with it, and no worker in the pool. 

## To associate a team with a worker pool

```
$ fly -t <target> attach-worker-pool --name <pool-name> --team <team name> [--team <team name2> ...]
```

When associating a team with a worker pool, it should verify that the team
currently does not associate with any other worker pool.

## To dissociate a team from a worker pool

```
$ fly -t <target> detach-worker-pool --name <pool-name> --team <team name> [--team <team name2> ...]
```

## RBAC

Only admin users have permission to create worker pools, list all worker pools, and 
associate teams to worker pools.

## Database

1. Add a new table `worker_pools`. This table stores info of worker pools.

```sql
  CREATE TABLE worker_pools (
      id integer NOT NULL,
      name text NOT NULL,
      public_worker_key text,
  );
```

2. Add a column `worker_pool_id` to table `workers`.

```sql
  ALTER TABLE "workers" ADD COLUMN "worker_pool_id" integer;
```

3. Add a new table `worker_pool_teams`. This table stores relationship between
worker pools and teams.

```sql
  CREATE TABLE worker_pool_teams (
      worker_pool_id integer NOT NULL,
      team_id integer NOT NULL,
  );
```

_Note: omitting indexes and constraints in RFC._

## Worker process

An optional new option `--pool=<pool name>` should be added to `concourse worker`. 

The options `--pool=<pool name>` and `--team=<team name>` should be mutually exclusive.

## Web UI

No UI change needed.

# Open Questions

1. Do we really need `--public-worker-key` option for `fly set-worker-pool`? This
option provides a way to add a public worker key, which allows adding worker public
keys without restarting ATC. However, the problem is, there is no way to ensure
the specified public key only working for specified pool. Because public keys are
used during SSL handshake, a bunch of known-authorized keys are given to `ssh.ServerConfig`,
but we don't know which public key matched after handshake. The ATC option
`--tsa-team-authorized-keys` has the same problem. So should we just add a separate
command like `fly set-worker-public-key`? If yes, which could be a separate story 
than this RFC.


# Answered Questions

> If there were any major concerns that have already (or eventually, through
> the RFC process) reached consensus, it can still help to include them along
> with their resolution, if it's otherwise unclear.
>
> This can be especially useful for RFCs that have taken a long time and there
> were some subtle yet important details to get right.
>
> This may very well be empty if the proposal is simple enough.


# New Implications

> What is the impact of this change, outside of the change itself? How might it
> change peoples' workflows today, good or bad?
