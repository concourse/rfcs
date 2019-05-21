# `artifact_spaces` examples

In this scenario, there are three top-level definitions for resources:

* `spaces:` lists the resources for discovering config fragments
* `artifacts:` is equivalent to today's `resources:`
* `artifact_spaces:` defines spatial artifacts, by specifying `across` as one of:
  * `[fragments]` to statically define config fragments
  * `space-name` to dynamically determine config fragments

In addition, there is a new `span` step which is similar to `get` except it points to an entry under `artifact_spaces:`. This is to make it clear that the job runs against spaces.

## Pros

* It's easy to notice that a job runs against multiple spaces, by just looking for `span:`.
* There are never any 'partially-defined' resources.

## Cons

* It's not easy to notice how many combinations there are for a given job.