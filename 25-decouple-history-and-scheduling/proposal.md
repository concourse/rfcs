# Summary

Some of the database design around builds could increase in conceptual
complexity,  in service of the system's scalability.

# Motivation

The core observation is contained in
http://www.github.com/concourse/concourse/issues/3756 -- but it feels like the
builds table can, in some pretty common large-scale use cases, cause the
database to consume excessive storage.

We notice two distinct flows that access the build history:
1. Scheduling, where you need to know its inputs and outputs.
2. Displaying the build history, where you just need the name, time and status.
   For displaying logs, you also need to know the plan and events.

Use case 1. requires indices because it's a performance-critical operation.
Those indices are useless for the other case, and keeping them _probably_ takes
a bunch of extra space.

In a loose sense, once build data is old enough, you will never do anything
data-intensive (complex queries that might require an index) with it. There
might be some performance benefit in collapsing old build data into a single
row per job, or something similar.

# Proposal

Perhaps we could introduce a new notion to describe old/archived builds that
still have all their data, but don't affect automatic scheduling.

Perhaps old builds should live in some kind of "compressed" or "denormalized"
table: the stuff you need to remember for displaying history is what the build's
name was, when it ran and what its status was. If you have to do this for a lot
of builds, maybe there's some way to condense it down, even potentially to a
single row per job?

If we were to remove the input/output data from old builds there, there could be
some interaction with the http://www.github.com/concourse/concourse/issues/413
(now-forthcoming) feature -- if
the old build was reaped, do we need to still persist some information about
inputs/outputs in order to allow it to be re-triggered?

# Open Questions

We haven't actually validated that the index is the cause of the extra storage
consumption -- http://www.github.com/concourse/concourse/issues/3756 should be
completed to establish that.
