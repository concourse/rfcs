# Concourse Design Principles

Concourse's goal is to solve automation, once and for all, without becoming part of the problem.


## Expressive by being precise

Concourse should provide concepts that build a strong mental model for the user's project, and this model should remain intuitive as the user's automation requirements grow.

Concepts should precisely outline their motivation and intended workflows. Friction and complexity resulting from the imprecise application of a concept should be a cue to introduce new ideas. ([Example](https://blog.concourse-ci.org/reinventing-resource-types/))


## Versatile by being universal

Concourse should be able to do a lot with a little. New concepts should only be introduced if their intended workflow cannot be precisely expressed in terms of existing concepts.

Concepts should not be highly specialized for one domain or introduce tight coupling to specific technologies. Users should be able to relate to every concept, and their automation should be insulated from the constant churn of the tech industry.


## Safe by being destructible

Concourse should prevent [anti-patterns](https://github.com/concourse/concourse/wiki/Anti-Patterns) and the accumulation of technical debt. Concourse's concepts should make good practices feel intuitive and bad practices feel uncomfortable.

Automation should be portable and reproducible. Recovery from disaster scenarios (e.g. total cluster loss) should be possible with minimal disruption to the path to production. Each piece of mission-critical state should be kept in an external source of truth, allowing Concourse to be an ephemeral choreographer.
