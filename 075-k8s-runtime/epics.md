# Epic: Connect to Kubernetes Worker

Acceptance Criteria:
- `fly workers` displays the Kubernetes worker
- Worker remains healthy; worker does not stall

## Register Kubernetes Worker

Given a Kubernetes cluster and an externally deployed Concourse web node
Concourse web should be able to register the Kubernetes cluster as a worker

## Ensure Kubernetes Worker stays Running

Given a Concourse web node has a Kubernetes worker
The Kubernetes worker stays in the `running` state.

## Fly workers displays the Kubernetes Worker

Given a Concourse web node has registered a Kubernetes worker
`fly workers` should display the Kubernetes worker

---

# Epic: Hello World Task - Part 1

Acceptance Criteria:
All iterations of this pipeline can be ran successfully on a Kubernetes worker:
- https://github.com/concourse/rfcs/blob/075-k8s-runtime/075-k8s-runtime/proposal.md (Dev use-case #1)

## Run a task on a Kubernetes Worker

Given a user sets a `hello-world` pipeline
the task completes successfully

## Exit status of task is captured

Given a user sets a `hello-world` pipeline
the task's exit status is saved in the database

## Run a task with task params

Given a user sets a `hello-world` pipeline
that uses task params
the task should use the params
the task completes successfully

## Run a task with Container limits

Given a user sets a `hello-world` pipeline
that uses task container limits
the task should test the container limits
the task completes successfully

## Run a privileged task

Given a user sets a `hello-world` pipeline
that has a privileged task
the task should verify that it's in a privileged container
the task completes successfully

## Abort a task

Given a user sets a `hello-world` pipeline
and the task runs for a long period of time
a user should be able to abort the task successfully

## Capture task output

Given a user sets a `hello-world` pipeline
the stdout and stderr should be saved to the database
