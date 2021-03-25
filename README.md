# Concourse RFCs

The Concourse project uses the RFC (request for comments) process for
collaborating on substantial changes to Concourse. RFCs enable contributors to
collaborate during the design process, providing clarity and validation before
jumping to implementation.


## When this process is necessary

The RFC process is necessary for changes which have a substantial impact on end
users, operators, or contributors. "Substantial" is subjective, but it
generally includes:

* Changes to core workflow functionality (pipelines, tasks, new concepts).
* Changes to how Concourse is packaged, distributed, or configured.
* Changes with significant architectural implications (new runtimes, auth
  schemes, etc.).
* Changes which modify or introduce officially supported HTTP APIs.

An RFC is not necessary for changes which have narrow scope and don't leave
much to be discussed:

* Bug fixes and optimizations with no semantic change.
* Small features which only impact a narrow use case and affect users in an
  obvious way.

The RFC process aims to prevent wasted time and effort on substantial changes
that end up being sent back to the drawing board. If your change takes minimal
effort, or if you don't mind potentially scrapping it and starting over, feel
free to skip this process. Do note however that pull requests may be closed
with a polite request to submit an RFC.

If you're not sure whether to open an RFC for a change you'd like to propose,
feel free to [ask in `#dev`](https://discord.gg/MeRxXKW)!


## Submitting an RFC

1. Fork this repository.

1. Copy the `000-example` RFC template, naming it something like
   `000-my-proposal`.

1. Write your proposal in `000-my-proposal/proposal.md`.

    * Consult the [Concourse design principles](DESIGN_PRINCIPLES.md) to guide
      your design.

    * Include any dependent assets (examples, screenshots) under your RFC
      directory.

1. Submit a pull request. The pull request number determines the RFC number.

    * Keep the description light; your proposal should contain all relevant
      information. Feel free to link to any relevant GitHub issues, since that
      helps with networking.

1. Rename the proposal directory to match the pull request number, e.g.
   `123-my-proposal`.

   For convenience, update the PR description to link to the rendered proposal
   in the pull request body like so:

   ```
   [Rendered](https://github.com/{YOUR NAME}/rfcs/blob/{YOUR BRANCH}/123-my-proposal/proposal.md)
   ```

1. Feel free to leave your own comments and questions, using the pull request
   as a "captain's log" as you reason about the problem more and reach key
   decisions. Doing so helps build a clear public record for the
   decision-making process.

1. Feel free to collect feedback and votes (via GitHub reactions) for your own
   RFC by linking to it from issues or contexts where it is relevant.

   The [Concourse website](https://concourse-ci.org) lists the top open RFCs
   ranked by GitHub reactions in order to collect feedback from end users. The
   goal of listing them this way is to focus attention on the RFCs most
   relevant to the community, increasing clarity and accelerating them to
   resolution, making room for other RFCs in the list.

   Please be respectful of other RFC authors and avoid vote brigading;
   diversity of perspective is more important than simply having the most
   votes.

1. Amend your proposal in response to feedback by pushing more commits to your
   fork. Whenever possible, please make meaningful commits that summarize the
   changes and reasoning (rather than rebasing and force-pushing all the time).


## Reviewing RFCs

The [**core** team][core-team] assigns each RFC to one reviewer from the team.
The assignee is responsible for providing feedback and shepherding the RFC
through the [resolution process](#resolution). Feel free to ask your RFC's
assignee if you need help with any part of the RFC process or with the proposal
itself.

Concourse users and contributors are encouraged to review RFCs alongside
members of the core team. Feedback from many different perspectives is crucial
determining a proposal's efficacy, impact, and priority. Reviewing RFCs is also
good practice for [joining the core team][joining-a-team] someday!

Reviewers should focus on resolving open questions, surfacing risks and
drawbacks, and providing constructive critique of the overall approach. The
[Concourse design principles](DESIGN_PRINCIPLES.md) serve as a guiding hand to
determine the proposal's alignment with the Concourse philosophy.

Reviewers should leave questions and comments on individual lines via PR review
so that discussions may be threaded and marked as resolved. Leaving GitHub
reactions also helps to measure consensus without cluttering the comment thread
if you don't have much more to add.


### Resolution

When enough clarity and consensus has been reached the core team assignee will
grant the RFC one of the following labels:

* **resolution/merge**: the proposal will be merged; there are no outstanding
  objections, and implementation can begin as soon as the RFC is merged.
* **resolution/close**: the proposal will be closed.
* **resolution/postpone**: resolution will be deferred until a later time when
  the motivating factors may have changed.

These labels initiate a two-week quiet period, and any final feedback will be
sought by bumping the RFC to the top of the RFC table on the Concourse website.
No further changes should be made to the proposal during this period.

If there is a challenge to the resolution during the quiet period the label may
be removed at the discretion of the assignee, and the RFC process will continue
as before.


## Implementing an RFC

When an RFC is merged the core team assignee is responsible for opening an
issue on the [Concourse repository](https://github.com/concourse/concourse) in
order to keep track of its implementation. The issue can be lightweight and
just reference the RFC. The assignee must also add a link to the issue at the
top of the RFC's proposal document.

The [**maintainers** team][maintainers-team] is responsible for determining the
proposal's priority by adding a **priority/high**, **priority/medium**, or
**priority/low** label. Priority is an approximation by overall value and
desired timeline for implementation.

An RFC author is not necessarily responsible for its implementation, though
they may volunteer. If the maintainers have sufficient bandwidth they may place
it on their roadmap by prioritizing the issue in a GitHub project. Otherwise
the maintainers team will add a **help wanted** label to the issue.

In any case, contributors may volunteer to implement a proposal provided that
work has not already begun. If you would like to volunteer, please leave a
comment on the issue to let others know!

From there, the implementation process itself falls under the normal [Concourse
development process][contributing].


## License

All RFCs, and any accompanying code and example content, will fall under the
Apache v2 license present at the root of this repository.


[joining-a-team]: https://github.com/concourse/governance#joining-a-team
[core-team]: https://github.com/concourse/governance/blob/master/teams/core.yml
[maintainers-team]: https://github.com/concourse/governance/blob/master/teams/maintainers.yml
[contributing]: https://github.com/concourse/concourse/blob/master/CONTRIBUTING.md
