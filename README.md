# Concourse RFCs

A process for collaborating on substantial changes to Concourse.


## Providing feedback on an RFC

Everyone is encouraged to provide feedback on RFCs!

RFC review should be focused on bringing clarity to the proposal, resolving
open questions and surfacing risks and drawbacks. The [Concourse design
principles](DESIGN_PRINCIPLES.md) should be used to guide the solution and to
determine whether the proposal should be accepted.

Questions and comments should be left via PR review on individual lines so that
they can be marked as resolved. Top-level comments and pull-request reviews are
OK for overarching commentary, but in general line-wise comments are preferred.

Using GitHub reactions not only on the proposal itself but on other feedback
comments is encouraged as a way to measure consensus without cluttering the
comment thread.


## What should be proposed as an RFC?

RFCs should be opened for changes that have a substantial impact on Concourse
users or contributors. RFCs enable the community to collaborate during the
architecture and feature design process, providing clarity and validation
before jumping to implementation.

An RFC may not be necessary for changes that are narrow enough in scope and
impact. If you feel that this is the case, you can jump straight to submitting
a PR. Do note however that pull requests may be closed with a polite request to
submit an RFC if we find that it needs broader feedback.

If you're not sure whether to open an RFC for a change you'd like to propose,
feel free to [ask in `#dev`](https://discord.gg/MeRxXKW).


## Submitting an RFC

1. Fork this repository.
1. Copy the `000-example` RFC template, naming it something like
   `000-my-proposal`.
1. Write your RFC in `proposal.md` under your RFC directory.
    * It may help to consult the [Concourse design principles](DESIGN_PRINCIPLES.md) to guide your design.
    * The following sub-sections are common:
      * **Summary**: A brief summary (up to 1 paragraph) of the proposal.
      * **Motivation**: What issues will this proposal resolve? What patterns
        will it improve?
      * **Glossary**: A list of important terms introduced by or used within
        the proposal.
      * **Open Questions**: A list of unresolved concerns with the proposal.
        Take special care to think about any risks, side effects, or drawbacks
        to your proposal ahead-of-time. These have to be assessed at some
        point!
1. Submit a pull request. Your proposal may include any dependent assets
   (example content, screenshots) under its RFC directory. For convenience,
   link to the rendered proposal in the pull request body like so:

   ```
   [Rendered](https://github.com/{YOUR NAME}/rfcs/blob/{YOUR BRANCH}/123-my-proposal/proposal.md)
   ```

   Keep the description light; the focus should be on the proposal.
   Feel free to reference any relevant GitHub issues, since that helps with
   context-building.
1. Update the proposal directory name to match the pull request number, e.g.
   `123-my-proposal`.
1. Each RFC will be assigned to at least one reviewer. Feel free to reach out
   to them if you need help on any part of the process or with the proposal
   itself.
1. As the RFC author, feel free to leave your own comments/feedback, using the
   pull request as a "captain's log" as you think about the problem more and
   reach key decisions. The point of all this is to have a clear public record
   for the decision-making process.
1. Amendments to the RFC should be made by pushing more commits to the RFC
   branch. **Please do not rebase and force-push over your own commits.**
   Instead, try to make meaningful commits that summarize their changes.


### Collecting feedback

The top 10 open RFCs, ranked by total GitHub reactions, will be listed on the
[Concourse website](https://concourse-ci.org). This will obviously bias even
more reactions towards the top RFCs, but the hope is to focus-fire feedback on
the most impactful RFCs to bring more clarity and accelerate them to
resolution, making room for other RFCs in the list.

If you have submitted a new RFC, feel free to advocate for it and collect votes
by linking to it from issues or contexts where you feel it is relevant, but
please play fair; don't just vote brigade your way to the list.


### Resolution

Once consensus builds and things slow down, the RFC will be granted with one of
the following labels:

* `resolution/merge`: there are no outstanding objections to the RFC and
  implementation can begin as soon as the RFC is merged.
* `resolution/postpone`: there are no outstanding objections to the RFC, but we
  have decided to defer its implementation until some time in the future, and
  until then it's better to leave the proposal unmerged in case things change
  by the time we can get to implementation.
* `resolution/close`: we have decided not to accept the RFC, and have no plans
  for implementation.

These labels mark the beginning of the final phase of the RFC. During this
point, any additional feedback will be sought out by bumping it to the top of
the RFC list at the Concourse website.

There will then be a two-week quiet period on the RFC. If during this time
there is a challenge to the resolution, the label will be removed and the RFC
process will continue. Ideally there are no changes to the RFC during this
period (all typos should be resolved by now, and wording should be fairly
clear).

Depending on the resolution, the following will happen to the RFC pull request:

* `resolution/merge`: the PR will be merged!
* `resolution/postpone`: the PR will be closed and stamped with a `postponed`
  label. At some point in the future the pull request may be re-opened.
* `resolution/close`: the PR will be closed with no additional label.


## Implementing an RFC

Once an RFC is accepted, an associated issue will be opened on the [`concourse`
repository](https://github.com/concourse/concourse) repository, referencing the
RFC's pull request. This issue can be created by the RFC author or assignee.

By the time an RFC is merged, we should have a pretty good idea of who's going
to implement it. This may or may not be the same person that submitted the RFC.
Large-scale proposals for example may be picked up by the core Concourse team
instead (but obviously that'd be something we agree on prior to merging).

The implementation process itself falls under the normal [Concourse development
process](https://github.com/concourse/concourse/blob/master/CONTRIBUTING.md).


## License

All RFCs, and any accompanying code and example content, will fall under the
Apache v2 license present at the root of this repository.
