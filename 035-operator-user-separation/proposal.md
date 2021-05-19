# tl;dr

- Concourse to separate between the operator role and the user role by having `operator`
  team as the default and then creating teams as needed.

# Motivation

- When running Concourse at a scale, or even when running a small environment, there is
  always a need for having a separation between the operator which is the person that
  controls the Concourse cluster and the user which is the one that uses Concourse without
  caring a lot about operational problems.

- The separation between the 2 roles started to appear when the concept of
  [RBAC](https://concourse-ci.org/user-roles.html) was implemented as of Concourse 5.0
  that opened the doors towards different types of roles. 

- Also, when working on the super-admin concept
  ([concourse/concourse#4192](https://github.com/concourse/concourse/issues/4192)), We
  found that a lot of the features of the superadmin only made sense in the light of
  having a separate team for the super-admin persona.

# Proposal

- Renaming the `main` team into the `operator` team.

- On startup, Concourse would start with either the `operator` team only or both the
  `operator` and the `main` team.

- When on the `operator` team, the user would have *super privileges*, yet this team
  should not accept pipelines, or workers. The purpose of this team is to only manage the
  Concourse Cluster and not to create pipelines.

- **Super Privileges:** these are cross-team abilities, for instance viewing all the
  pipelines, teams, running `set-team` ... as well as any commands that would be more
  operation specific. Also, the web ui should also show everything and also pipeline
  controls should be available for operators. We have covered most of the super features
  in [concourse/concourse#4195](https://github.com/concourse/concourse/issues/4192).

- Depending on the type of the logged-in user (operator, user), fly would show different
  help menus and would accept different commands. This also allows for more fly commands
  that are only operator specific. Maybe another cli that is more operator specific.

- As a phase 2 of that, sub-operator teams can be created with privileged access on only a
  specific set of teams, *orgs*.

- **Orgs:** An Org is a set of teams that are managed by the same operator, I currently
  don't have a lot of ideas about what would be added to an "Org" but seems to me like an
  abstraction that can be used later on for different features, like org workers for
  example.

- for the time being, `set-team` would only create normal teams since there only one
  operator team. Next, a parameter can be added to set the team as operator or normal.
  
- One more wild thought, the operators' landing page can be different from the normal
  dashboard, having more operator specific monitoring.

# Open Questions

- If accepted, do we want to startup Concourse with `operator` and `main` teams or only
  `operator` and allow operators to create teams as needed?
- Would this be too much in case of a small Concourse deployment?
- How to run Concourse as an operator and a user at the same time?
