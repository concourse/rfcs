# The Runtime Interface

## What is the Line Between Core and Runtime?

...

### Build Step Execution
The `exec` package is the cleanest place to discuss the process by which a scheduled build's plan is converted to executable steps, and handed off to the "runtime". The execution of steps involves the creation of containers and volumes by the "Runtime" (Currently most of this lives in the  `worker` package, but there's a higher-order abstraction for resources in the `resource` package, and the `db` package is also involved at different levels )

There are 3 builds step types which have a concrete execution involving Containers and Volumes, while the other step types available are meta-steps:

#### `get`

**Simplified:**

For a `get` step the Runtime requires the Resource Image (either a built-in resource RootFS living on the worker, or a cached image of a custom Resource Type) to create a container and run the `in` script. This will produce an Output volume (aka a Resource Cache Volume)

`( Resource Image ) -> ( Output )`

**Reality:**

[exec/get_step.go](https://github.com/concourse/concourse/blob/4cfe7e15f00de21d0aee7273867f75899cbeef20/atc/exec/get_step.go#L160-L196)

#### `task`

**Simplified:**

For a `task` step the Runtime requires Task Image (an `Output` produced by a previous step, or `image_resource`) to create a container and run the Task script, and any `Inputs` defined in the Task Config. Once the Task script successfully executes, there will be a set of `Output`s available to later steps.

`( Task Image + []Input ) -> ( []Output )`

**Reality:**

[exec/task_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/task_step.go#L185-L295)

#### `put`

**Simplified:**

For a `put` step the Runtime requires the Resource Image (either a built-in resource RootFS living on the worker, or a cached image of a custom Resource Type) to create a container and run the `out` script, and any `Inputs` defined on the `put` step. This will produce no `Output`s as all that is required is the version produced by the step, which is sent over Stdout.

`( Resource Image + []Input ) -> ()`

**Reality:**

[exec/put_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/put_step.go#L123-L166)


...


### Resource Checking

> But what about Resource checking! `radar` also creates containers!


## Runtime's State

### How much state can / should the runtime store in the DB?

### Are the Current "Runtime" DB Objects Too Garden Specific?


## Efficiency Gains for Different Runtime Engines

