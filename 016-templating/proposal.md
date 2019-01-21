# Summary

@vito suggested we raise this as an RFC some time before Christmas (on Discord).   Now that we've had time to reflect on it, here is our proposal.  Looking at the other RFC's/issues, it may be complementary to Issue #7 

We love Concourse and use it extensively, but as your pipelines grow they become increasingly more complicated.  The cognitive load becomes overwhelming at times, and changes become increasingly more difficult to identify and track down.  There is a general need for a way of splitting out parts of a pipeline, and also allowing for some form of templating to accommodate a simpler cognitive mode.  This RFC proposes one possible set of solutions to templating.  By adding the functionality proposed here to `fly`, we believe that concourse can have genuinely powerful templating function.

UAV is a POC app that performs the tasks outlined in the proposal, and can be found @ https://github.com/finbourne/uav ; the golang version is in the `golang` directory.

# Motivation


Refs:

* https://github.com/concourse/concourse/issues/1994
* https://github.com/concourse/concourse/issues/595
* https://discuss.concourse-ci.org/t/proposal-poc-pipeline-templating/694


Some of the issues we see with the current handling of pipelines:

1. There is a disparity between what pipeline has currently been deployed, and what the git repository contains for pipeline utilities.
2. When deploying to different environments, a lot of code needs to be repeated, even when trying to utilise best practices with out-of-band tasks (i.e. *task templates*)
3. Out-of-band tasks (*task templates*) have no automatic support for determining whether or not all of the arguments/required parameters have been set. The only way of knowing is by deploying a pipeline, and seeing if it runs successfully.

For the purposes of our proposal, #1 is out of scope.  


# Proposal

To tackle the following:

1. Pipeline composition
2. Pipeline templating (aka DRY pipelines)
3. Task template validation

We propose the following:

1. Addition of a new top level construct in pipeline yaml; **merge**

   * In the context of the pipeline this can be recursively used.

   * It takes an array of objects consisting of `template` and `args`
     `template` is a path to another pipeline
     `args` is a map of key/value pairs; think `params` to a task

   * Merges have the following effects:

     * `jobs`; append
     * `groups`; append
     * `resources`; duplicate names are checked for equality.  Merge failure if not equal
     * `resourceTypes`; duplicate names are checked for equality.  Merge failure if  not equal

   * Example:

     ```yaml
     merge: 
     - template: pipelines/test.tpl
       args:
         env: qa
         repo_master: github
     ```

2. [Pipelines are golang text/templates](https://golang.org/pkg/text/template/) with [Masterminds/sprig template extensions](https://github.com/Masterminds/sprig) and a couple of additional functions for working with yaml and json as well as an `include` function that works very similarly to the `template` function, but allows for piping into `indent`; remarkably similar to [helm](https://helm.sh/) templates, which is where the inspiration (and some code) came from.

   * All pipelines are golang text/templates

   * Following the existing example; couple the pipeline with `pipelines/test.tpl`

     ```yaml
     jobs:
     - name: deploy-{{ .env }}
       serial: true
       plan:
       - get: repo
       - task: task1
         config:
           platform: linux
         
           image_resource:
             type: docker-image
             source:
               repository: test/docker-container
           run:
             path: /bin/bash
             args: 
             - -cel
             - |
               cd repo
               echo Hello {{ .env }}!
     merge:
     - template: resources/repo.tpl
       args:
         repo_master: {{ .repo_master }}
     
     ```

     and `resources/repo.tpl`

     ```yaml
     resources:
     - name: repo
       type: git
       source:
         uri: git@{{ .repo_master }}.com:concourse/concourse.git
         branch: master
         private_key: ((github.privatekey))
     
     ```

     These would be merged into the following pipeline.

     ```yaml
     jobs:
     - name: deploy-qa
       serial: true
       plan:
       - get: repo
       - task: task1
         config:
           platform: linux
         
           image_resource:
             type: docker-image
             source:
               repository: test/docker-container
           run:
             path: /bin/bash
             args: 
             - -cel
             - |
               cd repo
               echo Hello qa!
     resources:
     - name: repo
       type: git
       source:
         uri: git@github.com:concourse/concourse.git
         branch: master
         private_key: ((github.privatekey))
     
     ```

3. Remove task templates
   We no longer believe this is necessary if the #1 and #2 are implemented.  Any re-use can be achieved either with named templates, or just by having the file integrated in the normal way into the decomposed pipeline.
   Identical functionality can be achieved by referencing the code of a task in a separate file.  So where you would have had:
   `partial-pipeline.yaml`

   ```yaml
   ...
   - task: my-file-task
     file: build/tasks/test1.yaml
   ...
   ```

   and:
   `build/tasks/test1.yaml`

   ```yaml
   ---
   platform: linux
       
   image_resource:
     type: docker-image
     source:
       repository: test/docker-container
   run:
     path: /bin/bash
     args: 
     - -cel
     - |
       echo I am testing!
   ```

   Instead you would have:

   `partial-pipeline.yaml`

   ```yaml
   ...
   - task: my-task
     config:
       platform: linux
       
       image_resource:
         type: docker-image
         source:
           repository: test/docker-container
       inputs:
       - name: pipeline-stuff
       
       run:
         path: build/tasks/task1.sh
   ...
   ```

   and:

   `task1.sh`

   ```bash
   #!/usr/env -S bash -cel
   echo I am testing!
   ```

   Cases can be made for both approaches, and removing the current functionality should not be top priority, and could be dropped from the proposal without affecting it.

4. Removal of the current double curly brace variables.  These would interfere with the golang template syntax.  Not impossible to work around, but it would make the pipelines look significantly less elegant.
   *How to work around this if not removed* example:

   ```yaml
   jobs:
   - name: deploy-{{ .env }}
     serial: true
     plan:
     - get: repo
     - task: task1
       config:
         platform: linux
       
         image_resource:
           type: docker-image
           source:
             repository: test/docker-container
         run:
           path: /bin/bash
           args: 
           - -cel
           - |
             cd repo
             echo {{ "{{my_oldie_style_secret_that_says_hello}}" }} {{ .env }}!
   ```


The running example is extremely simple, but it's composability and the power of golang templates makes for an extremely powerful and compelling tool.

We originally proposed UAV as a separate tool that did part of this job; pipeline merging with some variable substitution.  We have since re-written UAV as a stand-alone app in golang, with all the above functionality, and would propose that the vision outlined here be adopted by Concourse, specifically into `fly`.  We also encourage others to examine and comment on UAV as it stands.  We are currently migrating all our pipelines to use it, and have come up with the following set of practices:

1. The following directory structure, with the main pipeline at the root.  This pipeline to only consist of plain yaml.

   ```
   |+ pipelines/
    |- job1.tpl
    |- job2.tpl
   |+ resources/
    |- res1.tpl
    |- res2.tpl
   |+ templates/
    |- helpers.tpl
   |- pipeline.yaml
   ```

   This is by no means dogmatic.  It just helped organise one large file into many smaller ones.

2. Keep resource types with each resource that requires it; or merge it in from another file, but always reference it.

3. Don't get too funky with the golang text/templates.  Being too DRY here can completely obscure intent and make it difficult for others; including your future self; to understand the flow.

4. Start off by simply splitting out a `job` with it's `resources` and `groups`.  Grow the templating over time, but be very prudent and scarce with it.  It has great power, but it doesn't all need to be used at once.

5. Given that each partial pipeline is a valid pipeline; they don't have to be as long as together they constitute one; it would be simple to test jobs in isolation from other jobs, simply by using the partial pipeline under test in it's own pipeline.  i.e. `uav` the partial pipeline to `my-new-pipeline.yaml` and run `fly set-pipeline` on `my-new-pipeline.yaml` calling the pipeline by a new name.



## Pros/Cons

We see the following benefits and problems:

* **PRO:** even if only the `merge` capability were added to `fly`, this would constitute a composability win.  Pipelines could be split and it would be significantly easier to understand each part of the larger landscape.  This does not make for DRY pipelines, but does reduce the cognitive load significantly.
* **PRO:** golang templating is not new, is quite powerful, and is used in a number of other comparable projects; `helm` for instance.  This leverages existing knowledge where a concourse user/admin/maintainer has previously used tools that also leverage golang templates.
* **CON:** not everyone will have used golang templates.  It is something else to come to grips with when learning concourse, and does not feel concourse native.  That was a deliberate choice, but comes with its downside.


# Open Questions

One thing that is not currently covered by UAV is resource and resource-type pruning.  For simplicity, it would be desirable to leave out `if`ing resources and resource types, and instead just remove unused ones at the point that the pipeline is output.  In it's current form, `uav` cannot achieve this easily.  However as part of `fly` it would be nearly trivial to add a post-process that does exactly this.   We'd like to propose adding an additional flag that achieves this objective `--strip-unused` . Anyone wishing to produce strictly compliant pipelines can still do so, but anyone wanting a simpler set of templates also has the option to add this argument.


# Answered Questions


# New Implications

* A new top level construct `merge`.
* The addition of some very thorough templating to `fly`.
* The possible removal of task file templates.