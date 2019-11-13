# Summary

Proposes a fairly limited set of valid characters for use in Concourse identifiers:

* Team names
* Pipeline names
* Job names
* Step names
* [Var source][var-sources-rfc] names

# Motivation

Concourse does not currently perform much validation for identifiers that can
be set through the API or in pipeline configs. As a result, users are able to
name pipelines and jobs with all sorts of special characters.

Allowing arbitrary symbols makes it difficult for Concourse to support semantic
notation in the CLI and elsewhere in the Concourse UX. For example, the `fly`
CLI uses `PIPELINE/JOB` syntax, but this becomes ambiguous if your pipeline or
job names have a `/` present in their name. For example, `foo/bar/baz` could
either be (`foo/bar`, `baz`) or (`foo`, `bar/baz`).

Allowing whitespace, capitalization, and mixed use of `_` and `-` also results
in inconsistent naming conventions between Concourse users. A 'deploy to prod'
job may be called any of the following:

* `deploy to prod`
* `deploy-to-prod`
* `deploy_to_prod`
* `Deploy to prod`
* `Deploy to Prod`

This variance is largely cosmetic and only makes it difficult for a new
Concourse user on an existing team to predict the name of a given job. It also
results in an inconsistent UX when using Concourse.

# Proposal

This proposal is to dramatically reduce the allowed character set for Concourse
identifiers, in the spirit of simplicity and consistency across Concourse
usage:

* Any lowercase Unicode letter.
* Any Unicode number.
* `-`

With Go's [`re2`](https://github.com/google/re2/wiki/Syntax) syntax this would
be represented by the following regular expression:

```re
[\p{Ll}\p{Nd}\-]+
```

## Migrating existing data

TODO

# Open Questions

* Do we want to support a 'display name'?


# Answered Questions

* n/a


# New Implications

* n/a

[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
