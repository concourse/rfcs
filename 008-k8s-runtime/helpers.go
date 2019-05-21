package main

import (
	"encoding/json"
	"fmt"
	"log"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const blobstoreName = "concourse-internal-blobstore"

func newName(jobID, stepID int) string {
	return fmt.Sprintf("job-%d-step-%d", jobID, stepID)
}

func newObjMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: name,
	}
}

func newTypeMeta(kind string) metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       kind,
	}
}
func newParam(name, value string) tekton.Param {
	return tekton.Param{
		Name:  name,
		Value: value,
	}
}

func newSecretParam(fieldName, secretKey, secretName string) tekton.SecretParam {
	return tekton.SecretParam{
		FieldName:  fieldName,
		SecretKey:  secretKey,
		SecretName: secretName,
	}
}

func newContainer(name, image, cmd string, args []string, env map[string]string) corev1.Container {
	envs := []corev1.EnvVar{}

	for k, v := range env {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return corev1.Container{
		Name:    name,
		Image:   image,
		Command: []string{cmd},
		Args:    args,
		Env:     envs,
	}
}

func setupBlobstore() tekton.PipelineResource {
	return tekton.PipelineResource{
		ObjectMeta: newObjMeta(blobstoreName),
		TypeMeta:   newTypeMeta("PipelineResource"),
		Spec: tekton.PipelineResourceSpec{
			Type: tekton.PipelineResourceTypeStorage,
			Params: []tekton.Param{
				newParam("type", "gcs"),
				newParam("location", "gs://k8s-runtime-blobstore/blobs"),
				newParam("dir", "yeah"),
			},
			SecretParams: []tekton.SecretParam{
				newSecretParam("GOOGLE_APPLICATION_CREDENTIALS", "service-account-key.json", "service-account"),
			},
		},
	}
}

func blobstoreTaskResource() []tekton.TaskResource {
	return []tekton.TaskResource{
		tekton.TaskResource{
			Name: blobstoreName,
			Type: tekton.PipelineResourceTypeStorage,
		},
	}
}

func newGetStep(stepName, resourceName, resourceType string, resourceInput getRequest) (tekton.Task, *tekton.PipelineTask) {
	inputJSON, err := json.Marshal(resourceInput)
	if err != nil {
		log.Fatal("can't parse resource input json")
	}

	task := tekton.Task{
		ObjectMeta: newObjMeta(stepName),
		TypeMeta:   newTypeMeta("Task"),
		Spec: tekton.TaskSpec{
			Outputs: &tekton.Outputs{
				Resources: blobstoreTaskResource(),
			},
			Steps: []corev1.Container{
				newContainer(
					"run-resource-get",
					fmt.Sprintf("concourse/%s-resource:ubuntu", resourceType),
					"bash",
					[]string{
						"-c",
						fmt.Sprintf(
							`
# since it does gsutils cp -r /workspace/output/%%s/*, if we exclude the "blobs" dir, it will
# flatten the directory structure
output=/workspace/output/%s/%s
mkdir -p $output
/opt/resource/in $output <<EOF
%s
EOF
ls -lR /workspace/output
							`,
							blobstoreName, resourceName, string(inputJSON)),
					},
					nil,
				),
			},
		},
	}

	pipelineTask := &tekton.PipelineTask{
		Name: stepName,
		TaskRef: tekton.TaskRef{
			Name: stepName,
		},
		Resources: &tekton.PipelineTaskResources{
			Outputs: []tekton.PipelineTaskOutputResource{
				tekton.PipelineTaskOutputResource{
					Name:     blobstoreName,
					Resource: blobstoreName,
				},
			},
		},
	}

	return task, pipelineTask
}

// wip
func newPutStep(stepName, resourceName, resourceType string, resourceOutput putRequest) (tekton.Task, *tekton.PipelineTask) {
	outputJSON, err := json.Marshal(resourceOutput)
	if err != nil {
		log.Fatal("can't parse resource output json")
	}

	task := tekton.Task{
		ObjectMeta: newObjMeta(stepName),
		TypeMeta:   newTypeMeta("Task"),
		Spec: tekton.TaskSpec{
			Outputs: &tekton.Outputs{
				Resources: blobstoreTaskResource(),
			},
			Steps: []corev1.Container{
				newContainer(
					"run-resource-put",
					fmt.Sprintf("concourse/%s-resource:ubuntu", resourceType),
					"bash",
					[]string{
						"-c",
						fmt.Sprintf(
							`
output=/workspace/%s
/opt/resource/out $output <<EOF
%s
EOF
							`,
							resourceName, string(outputJSON)),
					},
					nil,
				),
			},
		},
	}

	pipelineTask := &tekton.PipelineTask{
		Name: stepName,
		TaskRef: tekton.TaskRef{
			Name: stepName,
		},
	}

	return task, pipelineTask
}

func newTaskStep(stepName, image, cmd string, args []string, privalged bool, env map[string]string) (tekton.Task, *tekton.PipelineTask) {
	task := tekton.Task{
		ObjectMeta: newObjMeta(stepName),
		TypeMeta:   newTypeMeta("Task"),
		Spec: tekton.TaskSpec{
			Inputs: &tekton.Inputs{
				Resources: []tekton.TaskResource{
					tekton.TaskResource{
						Name:       blobstoreName,
						Type:       tekton.PipelineResourceTypeStorage,
						TargetPath: ".",
					},
				},
			},
			Steps: []corev1.Container{
				// gsutils cp has to be passed a "-P" flag in order to preserve file permissions
				// since tekton doesn't provide that flag to gsutils, we lose file permissions (PR?)
				// for now we'll just recursively make everything executable......
				newContainer("make-everything-executable",
					"alpine",
					"chmod",
					[]string{"-R", "+x", "."},
					map[string]string{}),
				newContainer("run-job-step", image, cmd, args, env),
			},
		},
	}

	pipelineTask := &tekton.PipelineTask{
		Name: stepName,
		TaskRef: tekton.TaskRef{
			Name: stepName,
		},
		Resources: &tekton.PipelineTaskResources{
			Inputs: []tekton.PipelineTaskInputResource{
				tekton.PipelineTaskInputResource{
					Name:     blobstoreName,
					Resource: blobstoreName,
				},
			},
		},
	}

	return task, pipelineTask
}

func newPipeline() tekton.Pipeline {
	return tekton.Pipeline{
		ObjectMeta: newObjMeta("pipeline"),
		TypeMeta:   newTypeMeta("Pipeline"),
		Spec: tekton.PipelineSpec{
			Resources: []tekton.PipelineDeclaredResource{
				tekton.PipelineDeclaredResource{
					Name: blobstoreName,
					Type: tekton.PipelineResourceTypeStorage,
				},
			},
			Tasks: []tekton.PipelineTask{},
		},
	}
}

func newPipelineRun(pipeline tekton.Pipeline) tekton.PipelineRun {
	return tekton.PipelineRun{
		ObjectMeta: newObjMeta("pipeline-instance"),
		TypeMeta:   newTypeMeta("PipelineRun"),
		Spec: tekton.PipelineRunSpec{
			PipelineRef: tekton.PipelineRef{
				Name: pipeline.ObjectMeta.Name,
			},
			Trigger: tekton.PipelineTrigger{
				Type: "manual",
			},
			Resources: []tekton.PipelineResourceBinding{
				tekton.PipelineResourceBinding{
					Name: blobstoreName,
					ResourceRef: tekton.PipelineResourceRef{
						Name: blobstoreName,
					},
				},
			},
		},
	}
}
