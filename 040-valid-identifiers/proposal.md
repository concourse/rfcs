* Revision: [1.1](#revision-1.1)
* RFC PR: [concourse/rfcs#40](https://github.com/concourse/rfcs/pull/40)
* Concourse Issue: [concourse/concourse#5810](https://github.com/concourse/concourse/issues/5810)

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

Allowing whitespace and capitalization along with `_` and `-` results in
inconsistent naming conventions between Concourse users. A 'deploy to prod' job
may be called any of the following:

* `prod-deploy`
* `deploy to prod`
* `deploy-to-prod`
* `deploy_to_prod`
* `Deploy to prod`
* `deploy to Prod`
* `Deploy to Prod`

Permitting so many different naming conventions makes the Concourse UX, which
is largely text-based, feel inconsistent between different projects with
different naming conventions. This inconsistency may seem insignficant to users
who only use Concourse within a single team, but it will become more pronounced
if/when the Concourse project introduces a central place to share re-usable
pipeline templates and other configuration.

Allowing spaces also makes it awkward to pass identifiers to the `fly` CLI, as
they would have to be explicitly quoted so they're not parsed as separate
arguments.


# Proposal

The success of tools like `go fmt` has shown that the engineers value global
consistency and reduction of petty debates over personal stylistic preferences.

In the spirit of consistency and simplicity, this proposal is to dramatically
reduce the allowed character set for Concourse identifiers.

The following characters will be permitted:

* Non-uppercase Unicode letters (i.e. lowercase or letter with no uppercase).
* Decimal numbers.
* Hyphens (`-`) and underscores (`_`), as the canonical word separators.
* Periods (`.`), in order to support domain names and version numbers.

It's worth noting that both hyphens (`-`) and underscores (`_`) are allowed as
word separators. While this may lead to the kind of fragmentation this proposal
aims to prevent, allowing both is more pragmatic than forbidding either: `-` is
already commonplace, while `_` is more consistent with config params like
`resource_types` and is commonly used with other tools and naming conventions
(e.g. `x86_64`). The first iteration of this proposal did not allow the use of
underscore; see [Revision 1.1](#revision-1.1) for details.

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
^[\p{Ll}\p{Lt}\p{Lm}\p{Lo}\d][\p{Ll}\p{Lt}\p{Lm}\p{Lo}\d\-_.]*$
```

## Renaming existing data

All API resources can already be renamed manually:

* Pipelines can be renamed with `fly rename-pipeline`.
* Teams can be renamed with `fly rename-team`.
* Jobs can be renamed by updating their `name` and specifying the old name as
  [`old_name`][jobs-old-name]. This will preserve the job's build history.
* Resources can be renamed in the same fashion by setting
  [`old_name`][resources-old-name]. This will preserve the resource's state
  (i.e. disabled versions, pinning).
* Step names can be renamed without any migration necessary.

## Easing the transition

Enforcing rules about identifiers is easy. Doing this in a way that doesn't
alienate existing users and their practices is the hard part.

Requiring users to perform these manual steps in order to upgrade or
immediately after upgrading would likely slow down adoption. Being unable to
interact with mission-critical pipelines that have now-invalid identifiers
would be a major problem. Users should not be punished for upgrading.

To ease this pain, we can allow existing data to stay as-is, and only enforce
the identifier rules for newly created teams and pipelines. Additionally,
these validations can be implemented as warnings for a long period of time so
that users have time to adapt.

Existing data will still be fully functional and writable (i.e. updated with
`fly set-pipeline`, `fly set-team`), and Concourse can emit warnings for any
invalid identifiers (including the pipeline/team name itself) instead of
erroring out completely.

After one year, we can turn these warnings into errors.


# Open Questions

n/a


# Answered Questions

* **What about pipelines that use symbols as delimiters?**

  A common practice today is to configure many pipelines with a specific naming
  scheme, e.g.:

  * `dependency:github.com-username-repo-1`
  * `dependency:github.com-username-repo-2`

  Rather than cramming data into the pipeline name (and having to sanitize it),
  this should probably be resolved through use of [instanced
  pipelines](https://github.com/concourse/rfcs/pull/34).

  This would immediately improve the UX: pipeline names will be much shorter,
  related pipelines will be grouped together, the values no longer have to be
  sanitized, and the pipeline name is now just `dependency`, conforming to this
  RFC.

* **Are there any users who would become 'blocked' by this change?**

  Aside from strong personal preference, are there any Concourse users that would
  be unable to upgrade given the new rules?

  To put it another way: imagining Concourse had this strict naming convention
  from the get-go, are there any users who would *not be able to use Concourse*
  as a result?

  *(No one came forward with this RFC open for almost a year, so I guess that
  answers that.)*


# New Implications

* n/a


# Revisions

## Revision 1.1

In response to feedback in [concourse/concourse#6070][underscores-issue] this
RFC has been amended to allow the use of the underscore character (`_`) in
identifiers.


[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
[underscores-issue]: https://github.com/concourse/concourse/issues/6070
[jobs-old-name]: https://concourse-ci.org/jobs.html#schema.job.old_name
[resources-old-name]: https://concourse-ci.org/resources.html#schema.resource.old_name

## Pull Request [#9119](https://github.com/concourse/concourse/pull/9119)

Pull request [#9119](https://github.com/concourse/concourse/pull/9119) modified
the identifier regex to allow identifiers to start with numbers, but still
restrict an identifier from being only numbers.