# Reference
* Original RFC [#46](https://github.com/concourse/rfcs/pull/46)
* Original POC [#4818](https://github.com/concourse/concourse/pull/4818)

# Summary
This proposal outlines the beginnings of support for a `health` endpoint, which has a simple backend service which monitors crucial Concourse interfaces such as the database connectivity, the worker count (of healthy workers (*should be some threshold depending on the update strategy*)), the state of the webs (ATC/TSA) and others.

# Motivation
#### Currently, Concourse does not expose a dedicated, standardized health endpoint that external systems can query to determine the system’s overall health. This creates challenges in the following areas:

### 1. Monitoring & Alerting
Operators and platform teams often integrate Concourse with monitoring systems (e.g. Prometheus, Datadog, Kubernetes liveness/readiness probes). Without a clear health endpoint, they must rely on indirect signals (such as API responses, metrics, or manual checks), which can be unreliable or difficult to standardize.

### 2. Automation & Self-Healing
Modern infrastructure frequently depends on health endpoints for automated actions like restarting unhealthy pods, removing failing nodes from load balancers, or scaling workloads. The lack of a health endpoint makes such automation harder to implement for Concourse.

### 3. User Experience
When Concourse becomes partially degraded (e.g. workers are down, ATC is unresponsive, DB is lagging), it is not immediately obvious to users or operators. A health endpoint would provide a quick, single source of truth for identifying issues.

### 4. Consistency with Industry Standards
Most modern distributed systems (e.g. Kubernetes components, CI/CD systems, databases) expose health endpoints (commonly `/healthz`, `/readyz`, `/livez`). Introducing a similar endpoint in Concourse aligns it with best practices and user expectations.

## What will it bring?
By introducing a health endpoint, we make it easier to operate Concourse reliably in production environments, reduce the burden on operators, and enable better integration with external observability and orchestration systems.

# Proposal
## API Changes
What comes to mind is a simple **unauthenticated** HTTP endpoint (e.g. `/health`) that returns a JSON payload indicating the overall health status of the Concourse system. Could be something simple like:
```json
{
  "status": "healthy/unhealthy",
  "details": {
    "database": "healthy/unhealthy",
    "workers": "healthy/unhealthy",
  }
}
```

## Backend Service changes
A new service (e.g. `HealthChecker`) will be introduced to periodically check the health of critical components:
- **Database Connectivity**: Ensure the database is reachable and responsive - e.g. via a simple query, or checking logs for errors etc.
- **Worker Health**: Monitor the number of healthy workers and their responsiveness - we already know the desired workers, by introducing a simple threshold property (e.g. 80% of desired workers) we can determine if the system has enough registered workers to handle loads. The threshold can be calculated based on the update strategy (e.g. rolling updates might tolerate fewer workers temporarily, depending on the count of *in parallel/max in flight* configured).

## Alternatives
* There are solutions like [SLI runner](https://github.com/cirocosta/slirunner) that could potentially be leveraged for health checking in Concourse, but that requires SLA suites and additional configurations, which are much more granular, the proposition here is to have a simple, out-of-the-box health endpoint that can be used for basic high-end health checks, for the standard out-of-the-box Concourse. People can always build on top of that for more complex use cases.
* Extending the dataset of the `/info` endpoint to include a health json object is another alternative, but that endpoint is more about static information about the Concourse instance, rather than its dynamic health state.

# Open Questions
- I think it wouldn't require much changes to the existing infrastructure, but would it be better to have a dedicated microservice for this, or integrate it into the existing ATC service? From a general perspective and previous approaches/discussions, the idea is to have it within the ATC.
- Should we have a `degraded` state for the workers, where the number is bellow the expected, but not zero? I think in any case that could be further extended in the future, but as a start, a simple healthy/unhealthy state should suffice.

# Answered Questions
... TBD ...

# New Implications
I do not see (out of the box) negative implications of this change, rather it would improve the overall reliability and operability of Concourse in production environments.
