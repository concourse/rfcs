# Summary

This RFC proposes a new step: `gate`.

# Motivation

Concourse pipeline lacks function of conditional workflow. Some use cases are:

1. A pipeline runs code coverage scan. If coverage is lower than 85%, then fail the
build.
 
2. A pipeline monitors GitLab merged MergeRequests to automatically generate releases, 
but if a MR is labeled with "no-release", then silently discard the MR.

For the first use case, there could be two solutions: 1) Add gating function to the
resource type of code coverage scan. This solution is not generic. With this direction,
all similar scan related resource types would have to add gating function; 2) Use a 
generic resource type of gating. This solution is better, but using a resource to do 
gating is kinda overhead.

For the second use case, gating resource seems to not fit. Because if a resource needs
to abort a build, only way is to fail the build. If a user just wants to silently abort
the build, then gating resource won't work. So that only solution is to make the GitLab
resource to support to filter by MR labels, but which will make GitLab heavier and heavier.

# Proposal

A native step `gate` may make things easier and better.

For the above use case 1, `gate` can be used like:

```yaml
- put: sonar-scan

- load_var: scan-result
  file: sonar-scan/result.json

- gate: code-coverage-gate
  condition: ((.:scan-result.coverage_rate)) < 0.85

```

For the above use case 2, `gate` can be used like:

```yaml
- get: mr

- load_var: mr_info
  file: mr/mr_metadata.json

- gate: no-release-gate
  condition: no-release in ((.:mr_info.labels))
  fail: false
```

Step `gate` takes a name for description of the gate, and takes the following
parameters:

* `condition` a boolean expression. when `condition` is true, abort the build.
* `fail` if `fail` is true, fail the build `condition` is true; otherwise abort the build
with succeeded result.

# Open Questions

How to define `condition`? 

Basically `condition` will be evaluated to a boolean value.
An expression may contains:

* integer comparison
* string comparison
* not logic
* check if an item is in a list

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
