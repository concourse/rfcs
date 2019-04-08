# Spatial Resources

This proposal uses the [generalized resource interface](../024-generalized-resources/proposal.md) to show how the interface would be implemented and interpreted by Concourse to support "spatial resources", which model change over *space*, not time (as with [artifact resources](../024-artifact-resources/proposal.md)).

## Motivation

* Support multi-branch workflows: [concourse/concourse#1172](https://github.com/concourse/concourse/issues/1172)

* Begin phasing out `version: every` in by reframing the problem as 'pipeline per commit': [concourse/concourse#736](https://github.com/concourse/concourse/issues/736)

## Proposal

> Describe your proposal.
>
> Things that can help: clearly defining terms, providing example content,
> pseudocode, etc.
>
> Feel free to mention key implementation concerns.

## Open Questions

> Raise any concerns here for things you aren't sure about yet.

## Answered Questions

> If there were any major concerns that have already (or eventually, through
> the RFC process) reached consensus, it can still help to include them along
> with their resolution, if it's otherwise unclear.
>
> This can be especially useful for RFCs that have taken a long time and there
> were some subtle yet important details to get right.
>
> This may very well be empty if the proposal is simple enough.

## New Implications

> What is the impact of this change, outside of the change itself? How might it
> change peoples' workflows today, good or bad?
