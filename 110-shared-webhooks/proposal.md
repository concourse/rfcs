# Summary

Decouple webhooks from individual resources, allowing Concourse to process a webhook payload and queue a `check` for the approriate resource(s) (and eventually [var sources]).


# Motivation

Concourse's primary method of detecting new resource versions is through periodically running `check`. The default polling frequency for each resource is 1 minute.

Polling can lead to a lot of pressure on the other end of the `check` requests. This can lead to rate limiting, or worse, your CI/CD stack DDoSing your own internal services.

To reduce polling pressure it is typical to configure webhooks, by configuring a `webhook_token` on a resource and either setting a higher `check_every` or configuring `--resource-with-webhook-checking-interval` cluster-wide. This token value is then given to the service, as part of a per-resource URL.

This is a tedious operation and has severe limitations, as some external services (e.g. GitHub) have a limit on how many webhooks can be configured on a repo/org. This means you can only configure webhooks for up to 20 resources (minus whatever other hooks you have), which is a painful limitation when there's a common resource used in many pipelines.

Additionally, especially when operating a large Concourse deployment spanning many different teams, it's difficult to get users to configure webhooks since it requires per-resource configuration (both in and out of Concourse). It would be much simpler to define webhooks globally and have them automatically apply to matching resources.

It is also infeasible to use webhooks with dynamic groups of [instanced pipelines]. For instance, if you set an instanced pipeline for each GitHub pull request, you would need to somehow update your webhook configuration within GitHub to point to the correct resources.

Relevant issues/discussions:
* https://github.com/concourse/concourse/issues/2230
* https://github.com/concourse/concourse/discussions/6869#discussioncomment-949051


# Proposal

I propose adding a new webhook entity that can be managed via `fly`:

```sh
$ fly set-webhook --name github --type github --token abc123def456 (--team my-team | --global)
url: https://ci.concourse-ci.org/api/v1/teams/my-team/webhooks/github?token=abc123def456
```

...and a `webhooks` field to be added to the [prototype info response] that tells Concourse what webhook payload(s) should trigger a check for the resource. For instance, the resource:

```yaml
resources:
- name: my-repo
  type: git # note: this is the git prototype, not resource_type
  source:
    uri: git@github.com:concourse/concourse.git
    branch: master
```

may produce the following `info` response:

```json
{
  "interface_version": "1.0",
  "icon": "mdi:github-circle",
  "messages": ["check"],
  "webhooks": [{
    "type": "github",
    "payload_filter": {
      "repository": {"full_name": "concourse/concourse"},
      "ref": "refs/heads/master"
    }
  }]
}
```

When a webhook comes in to the webhook endpoint, the endpoint handler will find all of the resources that satisfy the payload (and [webhook type](#webhook-types)) and queue a new check for each of them. Resources that have been checked via a webhook will have their [`check` frequency reduced](#reducing-check-frequency) significantly.

Webhooks can either be global or scoped to a team. Configuring a webhook globally allows Concourse operators to define a single webhook that can e.g. be configured as an organization-level GitHub webhook, meaning you can handle many repositories with a single webhook.

The webhook endpoint for a global webhook is `/api/v1/webhooks/some-webhook`. The webhook endpoint for a team scoped webhook would be `/api/v1/teams/some-team/webhooks/some-webhook`. 


## Info Response

Prototypes can optionally specify a `webhooks` configuration in the [prototype info response] - an array of JSON objects containing:

* `type`: the [type](#webhook-types) of the webhook
* `payload_filter`: a JSON object used to filter the incoming webhook payloads. Webhook payloads are filtered by [containment] (i.e. a payload satisfies this checkable entity if it "contains" the `payload_filter`)

If a webhook payload comes in satisfying *any* of the `webhooks` (i.e. it is for a webhook of the same `type`, and the payload contains the `payload_filter`), a `check` will be created.

For instance, the following example filters on any open or close events for merge requests to the GitLab repository `git@example.com:mike/diaspora.git`:

```json
{
  "interface_version": "1.0",
  "icon": "mdi:github-circle",
  "messages": ["check"],
  "webhooks": [
    {
      "type": "gitlab",
      "payload_filter": {
        "object_kind": "merge_request",
        "repository": {"url": "git@example.com:mike/diaspora.git"},
        "object_attributes": {"action": "open"}
      }
    },
    {
      "type": "gitlab",
      "payload_filter": {
        "object_kind": "merge_request",
        "repository": {"url": "git@example.com:mike/diaspora.git"},
        "object_attributes": {"action": "close"}
      }
    }
  ]
}
```


## Webhook Types

Every webhook needs to be configured with a `type` that defines the external service that will be publishing to it. This serves two purposes:

1. Namespace webhooks to the type of external service, since just using the `payload_filter` may be ambiguous (if two external services use the same payload field names)
2. Support services that require the webhook endpoint to behave in a certain way. For instance, Slack requires the webhook endpoint to respond to a [`url_verification` request](https://api.slack.com/events/url_verification) when the webhook is first configured. Some external services may also provide special methods of proving that the webhook payload originates from a valid source (e.g. providing a special header that the endpoint must validate). Custom `types` may be implemented to support this behaviour.

Note that the implementation of these webhook `types` must be built in to Concourse. Different types *may* accept different configuration options in `fly set-webhook`

However, many external services don't require any special behaviour. To avoid requiring handling of every external service in Concourse, the `--type` field is just an arbitrary string. If the `type` is one of the explicitly implemented types (e.g. `slack` to support the `url_verification` request flow), the webhook endpoint will adopt that behaviour - otherwise, the type will solely be used to disambiguate webhook payloads.

When the type is not explicitly handled, Concourse will still need to validate the identity of the caller, but we can typically just use tokens as we do now (see [`resource.webhook_token`]).


## Reducing `check` Frequency

In order to reduce the polling pressure of resources that are backed by a webhook, we can keep track of when the last webhook check for a resource config completed.

When we try to queue a check in Lidar, we check if has been checked by a webhook. If so, we could set the default check interval to something much larger (e.g. `24h`). The intent of configuring a high interval, rather than disabling the polling entirely, is to be resilient to missed events.


# Open Questions

### Can we avoid needing to implement `types` within Concourse?

One thought here is to allow [prototypes] to define a webhook handler (rather than requiring each prototype to define its `webhooks` configuration in the [prototype info response]). Webhooks would still be linked to a type, but this type would reference a prototype (rather than an arbitrary string that is occasionally special cased). When a webhook payload comes in, we could invoke the webhook message on the linked prototype.

This has the benefit of being much more flexible and decoupling Concourse from the specific types of webhooks.

However, it comes at the cost of needing to run a container for each webhook payload, which counter-acts many of the benefits of reducing check frequency in the first place.

Given that there probably aren't *that* many external services that mandate special behaviour, I suspect implementing `types` within Concourse is the way to go.


### Is filtering based on webhook payload [containment] flexible enough?

Will we ever be in a situation where we need something more flexible than specifying a subset of the fields of the webhook payload?

Perhaps we may want to filter by something other than strict equality of the fields (e.g. substring search).

### Are there any use-cases for [`resource.webhook_token`] under this proposal?

### Is there a risk of `types` becoming fragmented?

Since `types` are arbitrary, is it likely that different prototypes would use different types for the same external service? This could make it difficult to define a single webhook to handle this service.


# Answered Questions


# New Implications

* A Concourse operator can reduce check frequency across the cluster by adding a single webhook (without requiring changes to any pipelines!)
* Webhooks for instanced pipelines are now much easier to implement, allowing proper webhook support with multi-branch workflows



[var sources]: https://github.com/concourse/rfcs/pull/39
[instanced pipelines]: https://github.com/concourse/rfcs/pull/34
[`resource.webhook_token`]: https://concourse-ci.org/resources.html#schema.resource.webhook_token
[prototypes]: https://github.com/concourse/rfcs/pull/37
[prototype info response]: https://github.com/concourse/rfcs/blob/master/037-prototypes/proposal.md#prototype-info
[containment]: https://www.postgresql.org/docs/9.6/datatype-json.html#JSON-CONTAINMENT
