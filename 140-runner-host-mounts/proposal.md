# Summary

This proposal introduces a mechanism through which tasks can request files, directories, or
block devices to be mounted from the host the worker is on to the task's container.

The worker must be configured to allow tasks to mount a given location, which we will make clear
is a last-resort option when other mechanisms are unsuitable.

# Motivation

When performing tasks that require access to a device on the host, such as a GPU, it is required to
make the 'files' that the kernel creates available in a task's container.

For instance, to use an AMD GPU within a container, the container needs access to these paths from the host:

  - `/dev/kfd` - GPU compute interface
  - `/dev/dri` - folder containing an interface for each GPU.

Currently, it is not possible to run these workloads using Concourse.

# Proposal

## Worker configuration option `--allowed-host-mounts`

Add a new configuration option to the worker command, `--allowed-host-mounts`:
The value should be a regular expression. Any host path that the expression fully matches will be *allowed* to be
mounted into a task's container if it is requested.

This option can be specified only once, but can use the regex OR operator `|` to match separate paths as required.

## Task configuration option `host_mounts`

Add a new configuration key (`host_mounts`) to the schema for `task` steps, which can be used to request paths from the host to be
mounted into that task's container.

Formally:

  - `host_mounts` is a list of `host-mount-config`s, defaulting to `[]`
  - A `host-mount-config` is either:
    - An object, with the following keys:
      - `host` (required, non-empty): The desired path to mount from the host
      - `container` (optional): The mount's location in the container. Defaults to the same as `host`
      - More options may be added down the line, as required.
    - A string, of format:
      - `host:container`, where `host` and `container` are strings not containing `:`, and have meanings as defined above.
      - or, `host`, leaving `container` to default to `host` as above
      - This is just a shorthand syntax, similar to the one Docker & Docker Compose uses.

## New behaviour when spawning task containers

If the task's `host_mounts` list is empty, then behaviour is unchanged.

If not, then each element in the list must be validated:

  - The `host` key must be fully matched by the `--allowed-host-mounts` regex.
    - If the `--allowed-host-mounts` option was not specified, validation fails automatically.
  - The path referred to by the `host` key must exist from the worker's perspective.
    - Permissions, writability, etc, are not checked.

If any of the elements do not validate, the task fails with an error message explaining why.

If validation has succeeded, each requested path is mounted into the task's container, using the container runtime's native feature for doing so.

# Open Questions

# Answered Questions

# New Implications

Adding this feature allows users to run new, increasingly common, types of workloads using Concourse.
Whilst it could also be used to introduce worker state, which we consider an antipattern, the intended use and limitations
of this feature should discourage users from doing so unnecessarily.

# Workers in docker

If the worker is itself running in a docker container, the mount will need to also be specified there. This will need to be made clear to users of the feature.

This will need to be made clear in documentation.

# Permissions errors

Mounting directories from the host opens users up to weird permissions errors, which are often counterintuitive. For instance, group membership is not inherited from the worker user, so if the mounted directory is not world-accessible and its user/group ID doesn't match with what's specified in the container, users will get permission errors.

This will need to be made clear in documentation, and should be dealt with by operators.
