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
  condition: <a bool expression>

```

For the above use case 2, `gate` can be used like:

```yaml
- get: mr

- load_var: mr_info
  file: mr/mr_metadata.json

- gate: no-release-gate
  condition: <a bool expression>
  nofail: true
```

## Step definition

Step `gate` takes a name for description of the gate, and takes the following
parameters:

* `condition` a boolean expression. when `condition` is true, abort the build.
* `nofail` by default, if `condition` is true, the build will fail. If you just
want to silently quit the build (with succeeded result), then set `nofail` to 
true.
* `objects` a list of objects defined as:
  * `name` object name
  * `file` file name to load the object. The file should be in json or yaml format.

## Condition syntax

After some research, I found the package https://github.com/PaesslerAG/gval can
be used for expression evaluation. It supports basic evalutation syntax, and can
be extended to support more operations.

### Integer operations and comparisons

```yaml
- load_var: mr_info
  file: mr/data.json

- gate: some-gate
  condition: ( ( ((.:mr_info.error)) + ((.:mr_info.failure)) ) / 300 ) > 0.85
```

### String operations and comparisons

```yaml
- load_var: mr_info
  file: mr/data.json

- gate: some-gate
  condition: "((.:mr_info.result))" == "success"
```

```yaml
- load_var: mr_info
  file: mr/data.json

- gate: some-gate
  condition: prefix_with( "((.:mr_info.subject))", "WIP" )
```

### JSON object operations

```yaml
- gate: some-gate
  condition: "no-release" in mr_info.lables
  objects:
  - name: mr_info
    file: mr/data.json
```


# Open Questions

Any suggestions to condition syntax?

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
