package main

import (
	"encoding/json"
	"os"

	"github.com/concourse/concourse/atc"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var plans []atc.Plan = []atc.Plan{
	atc.Plan{
		Get: &atc.GetPlan{
			Name:     "booklit",
			Resource: "booklit",
			Type:     "git",
			Source: atc.Source{
				"uri": "https://github.com/vito/booklit",
			},
			Version: &atc.Version{
				"ref": "HEAD",
			},
		},
	},
	// atc.Plan{
	// 	Get: &atc.GetPlan{
	// 		Name:     "docs",
	// 		Resource: "docs",
	// 		Type:     "git",
	// 		Source: atc.Source{
	// 			"uri": "https://github.com/concourse/docs",
	// 		},
	// 		Version: &atc.Version{
	// 			"ref": "HEAD",
	// 		},
	// 	},
	// },
	atc.Plan{
		Task: &atc.TaskPlan{
			Name: "test",
			Config: &atc.TaskConfig{
				Platform: "linux",
				ImageResource: &atc.ImageResource{
					Type: "registry-image",
					Source: map[string]interface{}{
						"repository": "golang",
					},
				},

				Inputs: []atc.TaskInputConfig{
					atc.TaskInputConfig{
						Name: "booklit",
					},
				},

				Run: atc.TaskRunConfig{
					Path: "booklit/ci/test",
					Args: []string{},
				},

				Params: map[string]string{
					"COVERALLS_TOKEN": "",
					"GIT_BRANCH":      "master",
				},
			},
		},
	},
}

func main() {
	f, _ := os.Create("tekton-pipeline.yml")
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	internalBlobstore := setupBlobstore()
	encoder.Encode(internalBlobstore)

	pipeline := newPipeline()
	sequential := []string{}

	for i, plan := range plans {
		name := newName(0, i)
		var (
			task         tekton.Task
			pipelineTask *tekton.PipelineTask
		)

		if plan.Get != nil {
			task, pipelineTask = newGetStep(name, plan.Get.Name, plan.Get.Type, getRequest{
				Source:  plan.Get.Source,
				Version: *plan.Get.Version,
			})
		}

		if plan.Put != nil {
			task, pipelineTask = newPutStep(name, plan.Put.Name, plan.Put.Type, putRequest{
				Source: plan.Put.Source,
				Params: plan.Put.Params,
			})
		}

		if plan.Task != nil {
			image := plan.Task.ImageArtifactName
			if plan.Task.ImageArtifactName == "" && plan.Task.Config.ImageResource.Type == "registry-image" {
				if s, ok := plan.Task.Config.ImageResource.Source["repository"].(string); ok {
					image = s
				}
			}

			task, pipelineTask = newTaskStep(
				name,
				image,
				plan.Task.Config.Run.Path,
				plan.Task.Config.Run.Args,
				plan.Task.Privileged,
				plan.Task.Config.Params,
			)
		}

		if pipelineTask != nil {
			pipelineTask.RunAfter = sequential
			encoder.Encode(task)
			sequential = append(sequential, pipelineTask.Name)
			pipeline.Spec.Tasks = append(pipeline.Spec.Tasks, *pipelineTask)
		}
	}

	pipelineRun := newPipelineRun(pipeline)

	encoder.Encode(pipeline)

	encoder.Encode(pipelineRun)
}
