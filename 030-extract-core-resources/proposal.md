# Summary

Concourse currently ships with the following "core" resource types:

* [`bosh-io-release`](https://github.com/concourse/bosh-io-release-resource)
* [`bosh-io-stemcell`](https://github.com/concourse/bosh-io-stemcell-resource)
* [`cf`](https://github.com/concourse/cf-resource)
* [`docker-image`](https://github.com/concourse/docker-image-resource)
* [`git`](https://github.com/concourse/git-resource)
* [`github-release`](https://github.com/concourse/github-release-resource)
* [`hg`](https://github.com/concourse/hg-resource)
* [`mock`](https://github.com/concourse/mock-resource)
* [`pool`](https://github.com/concourse/pool-resource)
* [`registry-image`](https://github.com/concourse/registry-image-resource)
* [`s3`](https://github.com/concourse/s3-resource)
* [`semver`](https://github.com/concourse/semver-resource)
* [`time`](https://github.com/concourse/time-resource)
* [`tracker`](https://github.com/concourse/tracker-resource)

This proposal is to remove them all from the `.tgz` distribution, leaving only the `registry-image` resource, as it will be necessary for fetching other resource types.

# Motivation

1. Packaging them all up with Concourse inflates the download size, especially as we standardize on `ubuntu`  as the base image instead of `alpine`.

1. Having commonly-used resource types bundled into the core set makes them feel "special/blessed" and may result in a delayed introduction to/understanding of `resource_types:` for new users.

1. A few of these resource types are awfully specific and not likely to be used by the majority of Concourse users:

    * `bosh-io-release`
    * `bosh-io-stemcell`
    * `cf`
    * `tracker`

# Proposal

Removing them is the easy part. The harder part is doing this without requiring everyone to update their pipelines.

We could do this by having an operator-specified config of resource types to automatically provide to each pipeline:

```yaml
---
resource_types:
- name: git
  type: registry-image
  source:
    repository: concourse/git-resource
    tag: 1
- name: time
  type: registry-image
  source:
    repository: concourse/time-resource
    tag: 1
```

This would be passed to `concourse web` on startup like so:

```sh
concourse web ... --core-resource-types resource-types.yml
```

This would have the effect of being merged into each pipeline's `resource_types:` (with the pipeline taking precedence).

Operators should be careful to provide a sane `tag:` value so that newly shipped versions do not break all pipelines. At the very least a major version should be specified. This is up to the operator's discretion, however, and will not be enforced.

# Open Questions

* This will result in a lot more polling against the Docker registry. Should we have a longer default check interval for resource types?
* Should Concourse come with a default set of operator-provided resource types?
  * If so, which ones?
* Are we ready to rely on the `registry-image` resource? See [concourse/concourse#3092](https://github.com/concourse/concourse/issues/3092).

# Answered Questions

* I don't think we should add team-level `resource_types:` configuration. That's too many layers to check for a config.

# New Implications

* We'll have to be careful not to break the latest version of each resource type, as all Concourse clusters will quickly pick up the new version (whether from `tag: latest` or `tag: 1`).