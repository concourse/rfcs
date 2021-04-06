# Concourse RFCs

The Concourse project uses the RFC (request for comments) process for
collaborating on substantial changes to Concourse. RFCs enable contributors to
collaborate during the design process, providing clarity and validation before
jumping to implementation.


## When the RFC process is necessary

The RFC process is necessary for changes which have a substantial impact on end
users, operators, or contributors. "Substantial" is subjective, but it
generally includes:

* Changes to core workflow functionality (pipelines, tasks, new concepts).
* Changes to how Concourse is packaged, distributed, or configured.
* Changes with significant architectural implications (new runtimes, auth
  schemes, etc.).
* Changes which modify or introduce officially supported interfaces (HTTP APIs, external integrations, etc).

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

1. Feel free to review your own RFC and leave comments and questions as you
   reason about the problem and reach key decisions. Doing so helps build a
   public record of the decision-making process.

1. The RFC will be assigned to a member of the [**core** team][core-team]. The
   assignee is responsible for providing feedback and eventually shepherding
   the RFC through the [resolution process](#resolution). Reach out to your
   RFC's assignee if you need any help with the RFC process.

1. Collect user feedback and votes (GitHub reactions) for your own RFC by
   linking to it in issues or contexts where it is relevant. Please be
   respectful of other RFC authors and avoid vote brigading; diversity of
   perspective is more important than having the most votes.

   The [Concourse website](https://concourse-ci.org) lists open RFCs ranked by
   GitHub reactions in order to increase exposure to end users. The goal of
   ranking them is to focus attention on the RFCs most relevant to the
   community, increasing clarity through user feedback and accelerating them to
   resolution.

1. Amend your proposal in response to feedback by pushing more commits to your
   fork. Whenever possible, please make meaningful commits that summarize the
   changes and reasoning (rather than rebasing and force-pushing all the time).


## Reviewing RFCs

Concourse users and contributors are encouraged to review RFCs alongside
members of the core team. Feedback from diverse perspectives is necessary for
determining a proposal's efficacy, impact, and priority. Reviewing RFCs is also
great practice for [joining the core team][joining-a-team] someday!

Reviewers should focus on resolving open questions, surfacing risks and
drawbacks, and providing constructive critique of the overall approach. The
[Concourse design principles](DESIGN_PRINCIPLES.md) serve as a guiding hand to
determine the proposal's alignment with the Concourse philosophy.

Reviewers should leave questions and comments on individual lines via PR review
so that discussions may be threaded and marked as resolved. Leaving GitHub
reactions also helps to measure consensus without cluttering the comment thread
if you don't have much more to add.


### Resolution

The review process should lead to consensus from three different perspectives:

* Members of the **core** team have determined whether the proposal fits with
  the Concourse design principles and whether the changes sufficiently improve
  the product.
* The **maintainers** have determined whether the proposal is worth
  maintaining, i.e. whether the benefits of the proposal outweigh any technical
  tradeoffs, or if it introduces an unsustainable maintenance burden.
* Enough community input has been provided to validate the need and efficacy of
  the proposal.

Once the review status stabilizes and clarity has been reached, the core team
assignee will grant the RFC one of the following labels:

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
issue on the [Concourse repository](https://github.com/concourse/concourse) to
keep track of its implementation. The issue can be lightweight and just
reference the RFC. The assignee must also add a link to the issue at the top of
the RFC's proposal document.

The [**maintainers** team][maintainers-team] is responsible for determining the
proposal's priority by adding a **priority/high**, **priority/medium**, or
**priority/low** label to the RFC's issue. Priority is an approximation of
overall value and desired timeline for implementation.

An RFC author is not necessarily responsible for its implementation, though
they may volunteer. If the maintainers have sufficient bandwidth they may place
it on their roadmap by prioritizing the issue in a GitHub project. Otherwise
the maintainers will add a **help wanted** label to the issue.

In any case, contributors may volunteer to implement a proposal provided that
work has not already begun. If you would like to volunteer, please leave a
comment on the issue to let others know!

From there, the implementation process falls under the normal [Concourse
development process][contributing].


## Revising an RFC

RFCs represent the planning phase. An RFC's proposal is not the source of truth
for the feature's documentation, and should not be revised to keep up with
later iterations after the initial proposal is implemented. A new RFC should be
proposed for subsequent changes instead.

If an RFC is merged and later changes are deemed necessary prior to final (i.e.
non-experimental) implementation, a follow-up PR may be submitted that updates
the proposal in-place. In this case the RFC author must include a MAJOR.MINOR
revision number in the proposal and maintain a brief summary of changes at the
bottom of the proposal.


## License

All RFCs, and any accompanying code and example content, will fall under the
Apache v2 license present at the root of this repository.


[joining-a-team]: https://github.com/concourse/governance#joining-a-team
[core-team]: https://github.com/concourse/governance/blob/master/teams/core.yml
[maintainers-team]: https://github.com/concourse/governance/blob/master/teams/maintainers.yml
[contributing]: https://github.com/concourse/concourse/blob/master/CONTRIBUTING.md
