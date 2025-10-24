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

# Answered Questions

1. Are we concerned about breaking peoples bookmarks?
   1. Yes since we're concerned about existing links in issues, comments, etc.
   2. Done by setting up redirect maps like the following:
      ```yaml
      docs.html.md: docs/index.md
      fly.html.md: docs/fly.md
      fly.md: docs/fly.md
      ``` 
   3. Both html.md and .md need to be specified since concourse docs can either be at:
      1. https://concourse-ci.org/task-inputs-outputs-example.html or,
      2. https://concourse-ci.org/task-inputs-outputs-example

# New Implications

All Concourse sites would be under one domain and codebase. This will allow for easier updates as it relates to theming
such as adding the Cloud Foundry Foundation to the footer along with allowing more people to contribute new docs.
