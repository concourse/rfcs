# Concourse Design Principles

Concourse's goal is to solve automation, once and for all, without becoming part of the problem.


## Expressive by being precise

Concourse should provide concepts that build a strong mental model for the user's project that remains intuitive as their automation grows.

Concepts should precisely outline their motivation and intended workflows. Friction and complexity resulting from the imprecise application of a concept should be a cue to introduce new ideas. ([Example](https://blog.concourse-ci.org/reinventing-resource-types/))


## Versatile by being universal

Concourse should be able to do a lot with a little. New concepts should only be introduced if their intended workflow cannot be precisely expressed in terms of existing concepts.

Concepts should not be highly specialized for one domain or introduce tight coupling to specific technologies. Users should be able to relate to every concept, and their automation should be insulated from the constant churn of the tech industry.


## Safe by being destructible

Concourse should prevent [anti-patterns](https://github.com/concourse/concourse/wiki/Anti-Patterns) and the accumulation of technical debt. Concourse's concepts should make good practices feel intuitive and bad practices feel uncomfortable.

Automation should portable and reproducible, able to be recovered with minimal disruption to the path to production. All mission-critical state should be kept external to Concourse, leaving Concourse to just be a connector-of-dots.
