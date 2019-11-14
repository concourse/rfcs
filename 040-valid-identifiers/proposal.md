# Summary

Proposes a fairly limited set of valid characters for use in Concourse
identifiers:

* Team names
* Pipeline names
* Job names
* Step names
* [Var source][var-sources-rfc] names


# Motivation

Concourse is currently very permissive with identifiers. This is largely an
oversight, as validation was simply never implemented at the beginning.

Compounding on this oversight, users have configured pipelines with all sorts
of special characters, either in the pipeline name itself or in jobs,
resources, and other API objects defined within.

Allowing arbitrary symbols makes it difficult for Concourse to support semantic
notation in the CLI and elsewhere in the Concourse UX. For example, the `fly`
CLI uses `PIPELINE/JOB` notation for flags specifying a job, but this would be
ambiguous if your pipeline or job names allowed `/` in their name. For example,
`foo/bar/baz` could either be (`foo/bar`, `baz`) or (`foo`, `bar/baz`).

Allowing whitespace, capitalization, and mixed use of `_` and `-` also results
in inconsistent naming conventions between Concourse users. A 'deploy to prod'
job may be called any of the following:

* `deploy to prod`
* `deploy-to-prod`
* `deploy_to_prod`
* `Deploy to prod`
* `Deploy to Prod`

This variance is largely cosmetic and makes it difficult for a new Concourse
user on an existing team to predict the name of a given job.

Permitting so many different naming conventions makes the Concourse UX, which
is largely text-based, feel inconsistent between different projects with
different naming conventions.

Allowing spaces also makes it awkward to pass identifiers to the `fly` CLI, as
they would have to be explicitly quoted so they're not parsed as separate
arguments.


# Proposal

The success of tools like `go fmt` has shown that the engineers value global
consistency and reduction of petty debates over personal stylistic preferences.

In the spirit of consistency and simplicity, this proposal is to dramatically
reduce the allowed character set for Concourse identifiers.

The following characters will be permitted:

* Unicode letters.
* Decimal numbers.
* Hyphens (`-`), as the canonical word separator.
* Periods (`.`), in order to support domain names and version numbers.

All letters will be converted to lowercase. This is to enforce
case-insensitivity and to present a consistent UI/UX while still supporting
languages that don't have any casing (e.g. Japanese).

Notably, the underscore character (`_`) is forbidden. This is to further
enforce consistency in word separators.

All identifiers must start with a valid letter. Allowing digits or symbols at
the beginning would allow for a few confusing situations:

* Allowing `-` at the start would make the identifier look like a flag,
  confusing the `fly` CLI parser.
* Allowing `.` at the start would permit strange names like `.` and `..` which
  may look like paths.
* Allowing numbers at the start would make `123` a valid identifier, which
  would parse as a number in YAML instead of a string.

With Go's [`re2`](https://github.com/google/re2/wiki/Syntax) syntax, a valid
identifier would be matched by the following regular expression:

```re
^\p{L}[\p{L}\d\-.]*$
```

This scheme is very similar to the [restrictions on valid
hostnames][valid-hostnames], with the exception that any Unicode letter is
allowed instead of `a-z`. This similarity is incidental, but it's a convenient
comparison to draw as hostnames and Concourse identifiers have similar needs:
to be used in URLs, to be referenced from the commandline (`fly`), to be
referenced in configuration (pipelines), and to be case-insensitive.

## Renaming existing data

The following API resources can already be renamed manually:

* Pipelines can be renamed with `fly rename-pipeline`.
* Teams can be renamed with `fly rename-team`.
* Jobs can be renamed by updating their `name` and specifying the old name as
  [`old_name:`](https://concourse-ci.org/jobs.html#job-old-name). This will
  preserve their build history.
* Step names can be renamed without any migration necessary.

Resources cannot currently be renamed, so this may need to be supported first.

## Easing the transition

Enforcing rules about identifiers is easy. Doing this in a way that doesn't
alienate existing users and their practices is the hard part.

Requiring users to perform these manual steps in order to upgrade or
immediately after upgrading would likely slow down adoption. Being unable to
interact with mission-critical pipelines that have now-invalid identifiers
would be a major problem. Users should not be punished for upgrading.

To ease this pain, we can allow existing data to stay as-is, and only enforce
the identifier rules for newly created teams and pipelines.

Existing data will still be fully functional and writable (i.e. updated with
`fly set-pipeline`, `fly set-team`), and Concourse can emit warnings for any
invalid identifiers (including the pipeline/team name itself) instead of
erroring out completely.

After one year, we can turn these warnings into errors.


# Open Questions

## Are there any users who would become 'blocked' by this change?

Aside from strong personal preference, are there any Concourse users that would
be unable to upgrade given the new rules?

To put it another way: imagining Concourse had this strict naming convention
from the get-go, are there any users who would *not be able to use Concourse*
as a result?


# Answered Questions

* n/a


# New Implications

* n/a

[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
[valid-hostnames]: https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames
