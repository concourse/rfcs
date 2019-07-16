# Summary

This proposal outlines a relatively straightforward 'archiving' operation which can be used to soft-delete a pipeline while preserving its data for later perusal.

# Proposal

Pipelines can be archived by a user through the `fly` CLI:

```sh
$ fly -t ci archive-pipeline -p pipeline-name
pipeline 'pipeline-name' archived
```

Archived pipelines are permanently paused - no resource checking or job scheduling is performed. They should consume no scheduling resources - only disk space.

Archived pipelines can be viewed in the web UI, grouped into a separate section, hidden by default.

Archived pipelines free up their name for use by later pipelines. It follows that there can be multiple archived pipelines with the same name by archiving `foo`, configuring a new `foo`, and then archiving it. One way to handle this is to have archived pipelines navigated by ID, with their name being only cosmetic.

# Open Questions

## Can a pipeline be 'un-archived'?

This would be nice to support, but we'd have to make sure it's safe.

For example, un-archiving a pipeline could result in a name collision if a new pipeline has been configured with the same name. We would have to handle that specially.

Un-archiving would also imply that we kept all of the config intact, which we wouldn't normally need to do if the pipeline isn't going to be running anymore. In fact we may want to clear out that data to prevent credentials from being stored indefinitely, in the event that the user isn't using a credential manager.
