# Summary

A public HTTP endpoint that gives basic information about the health of Concourse
 cluster.


# Motivation

There are some scenarios in which a Concourse cluster is part of bigger setup for
automation. In such scenario other parts of the system rely on Concourse not
only for CI/CD, but for other type of automations as well. In order to have the
whole process working the system also monitors the availability of its parts and
 takes measures if some of its parts are not operational (for example executes
some predefined steps - sends informing mails, triggers alerts, executes
medication steps, etc). In such cases it is important for the system to be able
to determine the state of every part of it. So it would be nice if the parts of
this system have a common way to return their health/availability status.

Currently there is no easy way for external (monitoring) system to understand
if a Concourse cluster is live and operational. It would be nice if Concourse
also gives information for its health so in (the rare) case it is not healthy
the external system can react.


# Proposal

Concourse can expose a public HTTP endpoint called "health" endpoint that gives
basic information for its health. Similar to the approach with the existing
"info" endpoint the "health" endpoint can be found at <concourse_url>/health.
It can return a JSON object with very basic information about the health of its
parts:
{
db: <status>  
<node_id> : <status>
<node_id> : <status>
...
}


# Open Questions

For which parts the information should be present in the JSON object - for all
the VMs (DB, web and worker nodes), or only for the DB and all worker nodes?
This question arises because a web node will serve the request - so at least one
 web node should be healthy enough to return the response (if there is no healthy
web node - the caller would receive an error anyway).

What the status should contain (level of details) - only "OK" and "NOK" - or more
detailed information about the state of the specific workers or web nodes?

Should there be a specific property to configure caching responses interval
(caching responses might help in the prevention of DoS attacks)?


# Answered Questions


# New Implications

This change doesn't aim to change the general workflow of the users, i.e.
creating/updating pipelines and executing jobs. It only aims to ease the
monitoring of the Concourse cluster in order to better integrate it into bigger
systems.
