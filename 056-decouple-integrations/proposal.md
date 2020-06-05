# Summary

Concourse has several types of integrations such as metrics and credential stores. For each type, there are several implementations such as Datadog, New Relic, Prometheus, etc.
This RFC proposes making those integrations `pluggable` to enhance the user experience.

# Motivation

Currently, an update to an existing Concourse integration or addition of a new integration requires a new release of Concourse. 
This results in additional time and complexity to make changes and extend Concourse 

For example, adding a new credentials store [Conjur](https://github.com/concourse/concourse/pull/4693) required getting a PR merged into the main Concourse codebase and a new release of Concourse with the changes.

Similarly, leveraging a [new batch processing](https://github.com/concourse/concourse/pull/4698) feature in the `New Relic API` required updating the main Concourse codebase and a new release of Concourse with the changes.


## Challenges
- greater concensus requirements for adding Concourse integrations
- longer turn around for enhancements and bug fixes to Concourse integrations
- leverage the large and vibrant Concourse community
- prevent the Concourse team from being a bottleneck especially when we aren't experts in a particular specific integration (eg. `Conjur`)

# Proposal

Define interfaces and externalize the integrations such that adding new integrations or updating existing integrations doesn't require a new version of Concourse.

One way to achieve this would be to use [gRPC](https://grpc.io/) where each integration type would have a `service` defining the interface. 
The interface would define the set of methods including their arguments and responses. 
This would be preferred over defining a `json/http` style API due to
- code generation for clients
- transport optimized
- simpler management 

This would allow users to use a new integration or update an existing integration by downloading a new version of the integration or bulding it locally and allowing Concourse to connect to it via socket connection. 
This would also allow integrations to be written in other languages.

There are many examples of successfully doing this in the larger community such as the [containerd project](https://www.slideshare.net/Docker/driving-containerd-oprations-with-grpc). 

# Open Questions

How do we manage changes to the interface ?
How does this work impact standardizing on [opentelemetry](https://github.com/concourse/concourse/pull/5104) for metrics ?


# Answered Questions

# New Implications

- Defining interfaces for integration types that we wish to externalize
- Defining the method by which users may leverage new integrations or different versions of integrations than what Concourse was shipped with ( This would be very similar to how we manage built-in resources )

