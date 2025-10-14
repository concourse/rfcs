# Summary

The Concourse website is the source of all docs related to the project, however, it
is using a unique technology called booklit to power it. Booklit was created by vito,
the original creator / maintainer of Concourse as a way to produce documentation sites.

However, over time, the booklit repository has only seen minor changes, with the last change
being on 06 JUNE solely migrating the project to Dagger
Modules ([vito/booklit](https://github.com/vito/booklit/commits/master/)).

# Motivation

Currently, the static site generation rules of booklit are unique in separating out content from presentation from
logic leading to multiple files to maintain. Additionally, the format of the website, compared to other modern sites
(e.g [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/getting-started/)
or [Renovate](https://docs.renovatebot.com/)).

This leads to steeper learning curve to contribute docs on a project that already has a steeper implementation curve
compared to other CI/CD platforms.

Additionally, the project maintains four distinct code bases for serving up: Docs, Blog, and Resource Types (two
repositories). This is not only an annoyance to maintain, but it does not allow for native cross-linking between blogs
and docs when new features are released.

# Proposal

To mitigate these concerns and simplify management of Concourse documentation, the proposal is to migrate the following
repositories, over time, in a single repository using MkDocs and the theme Material for MkDocs.

This will allow for easier maintenance, more frequent updates, cross-linking of new functionality, and the opportunity
for more people to contribute missing functionality, such as the capabilities of the `fly cli`.

To further this conversation, a limited example has been created to demonstrate to the benefits of using MkDocs over
Booklit:

- Demo Site: https://aviator-labs.github.io/concourse-docs/docs/
- Demo Repo: https://github.com/Aviator-Labs/concourse-docs

# Open Questions

1. How do we generate the home page to download the Fly CLI and Concourse CLI?
    - Material for MkDocs has a custom home page, will probably just take a little engineering work
2. Are we concerned about breaking peoples bookmarks?

# Answered Questions

To be filled...

# New Implications

All Concourse sites would be under one domain and codebase. This will allow for easier updates as it relates to theming
such as adding the Cloud Foundry Foundation to the footer along with allowing more people to contribute new docs.
