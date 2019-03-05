# The Runtime Interface

## What is the Line Between Core and Runtime?

...

### Build Step Execution
The `exec` package is the cleanest place to discuss the process by which a scheduled build's plan is converted to executable steps, and handed off to the "runtime". The execution of steps involves the creation of containers and volumes by the "Runtime" (Currently most of this lives in the  `worker` package, but there's a higher-order abstraction for resources in the `resource` package, and the `db` package is also involved at different levels )

There are 3 builds step types which have a concrete execution which involves Containers and Volumes, while the other step types available are meta-steps:

#### `get`

[exec/get_step.go](https://github.com/concourse/concourse/blob/4cfe7e15f00de21d0aee7273867f75899cbeef20/atc/exec/get_step.go#L160-L196)

#### `task`
[exec/task_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/task_step.go#L185-L295)

#### `put`
[exec/put_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/put_step.go#L123-L166)

### Resource Checking

> But what about Resource checking! `radar` also creates containers!



## Runtime's State

### How much state can / should the runtime store in the DB?


### Are the Current "Runtime" DB Objects Too Garden Specific?
-

## Efficiency Gains for Different Runtime Engines

