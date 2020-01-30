# Summary

This proposal outlines the beginnings of support for policy enforcement. OPA 
(Open Policy Agent) will be used as the first policy manager to integrate with.


# Motivation

Generally speaking, anything that exposes an HTTP API (whether an individual 
micro-service or an application as a whole) needs to control who can run those 
APIs and when. 

Concourse, as a CI platform, also needs to control who can do something and when,
which can be done by integrating with a policy manager. Before any action, send
a policy check request, with data that describes the action, to the policy manager,
and only continue if the policy manager replies a "pass".

Possible policies to apply could be:

* Steps, e.g. code security scan, by certain resource types must be done.
* Some team's tasks must be run with certain tag.
* Some docker image registries are not allowed.
* Some docker images are not not allowed.
* and so on ...


# Proposal

## Support multiple policy managers

Like how multiple credential managers are supported, Concourse should allow other
policy managers than OPA.

## Policy check points

* All API calls, for example `set-pipeline`, will also go through policy checks.
* Because Concourse supports task file, and task file is only fetched at runtime.
The action `set-pipeline` has no access to task files. To support policy check 
against tasks, a policy check should be invoked before a task run starts.

## OPA configuration

* CONCOURSE_OPA_URL - URL of OPA service including path

For example, OPA_URL is `http://opa.mycompany.com/v1/data/concourse/policy`, a OPA 
policy check request will look like:

```
POST http://opa.mycompany.com/v1/data/concourse/policy
{
   "input": {
      ...
   }
}
```

## Policy check input data

Policy check input data should include common metadata and action specific data. 
In principle, Concourse should send as much data as possible to the policy engine.

Common metadata include:

* `service`: should be always "concourse". This will allow Concourse specific 
policies to to configured in the policy engine.
* `cluster_name`: cluster name.
* `cluster_version`: Concourse version
* `action`: Action name. This follows same action names used in RBAC, plus an extra
action `RunTask`.
* `http_method`: HTTP method of the action, for `RunTask`, this field absents.
* `user`: username who invokes the action.
* `team`: team name.
* `pipeline`: pipeline name. Some action is not against a pipeline, then this field 
can be omitted.
* `data`: For API actions, `data` should carry data from the API call; for `RunTask`
action, data is the task configuration.

For example, a policy check request against `set_pipeline` looks like:

```json
{
  "input": {
    "service": "concourse",
    "cluster_name": "some-cluster",
    "cluster_version": "5.7.0",
    "user": "some-user",
    "team": "some-team",
    "pipeline": "some-pipeline",
    "action": "SetPipeline",
    "data": {
      "groups": [  ],
      "resource_types": [  ],
      "resources": [  ],
      "jobs": [  ]
    }
  }
}
```

## Policy check switches

If no policy manager is configured, then policy check is switched off.

When a policy manager is configured, Concourse intends to use "explicit" strategy
to decide which actions should run policy checks, meaning that, without defining
policy check filters, no action will run policy check.

Users may not want to run policy check against all actions. For example, Concourse
generate large amount of `ListAllPipelines`, and it makes not much sense to check
it.

It's assumable that users more tends to check policy against write actions than
read-only actions. As all actions expect `RunTask` are invoked from HTTP, we 
can provide a filter, `policy-check-filter-http-methods`, to specify HTTP 
methods via which actions are invoked. To skip read-only action for policy 
check, users may set `POST,PUT,DELETE` to the filter `policy-check-filter-http-
methods`, so that `GET` actions will not go through policy check.

User also may specifically want to or don't want to do run policy check against
certain actions. For example, a cluster will want to check policy against only
`SetPipeline`, or the other cluster don't want to check policy against `RunTask`,
for which two more filters, `policy-check-filter-action` and `policy-check-
filter-action-skip` are supported. If an action is defined in action list, then 
the action will always go through policy check, vice versa for action-skip list. 

In summary, where are three policy check filters:

* `policy-check-filter-http-methods`: defines HTTP methods of actions that
should go through policy checks. Default to empty list.
* `policy-check-filter-action`: defines action names that should always go 
through policy checks. Default to empty list.
* `policy-check-filter-action-skip`: defines action names that should never 
go through policy checks. Default to empty list.


# Open Questions

* What else policy points?

* What else policy engines than OPA folks are using?

* Does it make sense to share Audit's switches to control policy checks?



# Answered Questions

