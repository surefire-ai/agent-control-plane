package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultWorkerImage = "ghcr.io/windosx/agent-control-plane-worker:latest"
)

var defaultWorkerCommand = []string{"/agent-control-plane-worker"}

type WorkerOptions struct {
	Client  client.Client
	Image   string
	Command []string
}

type WorkerRuntime struct {
	client  client.Client
	image   string
	command []string
}

func NewWorkerRuntime(options WorkerOptions) WorkerRuntime {
	image := strings.TrimSpace(options.Image)
	if image == "" {
		image = defaultWorkerImage
	}
	command := append([]string{}, options.Command...)
	if len(command) == 0 {
		command = append([]string{}, defaultWorkerCommand...)
	}
	return WorkerRuntime{
		client:  options.Client,
		image:   image,
		command: command,
	}
}

func (r WorkerRuntime) Execute(ctx context.Context, request Request) (Result, error) {
	jobName := jobNameForRun(request.Run)
	var job batchv1.Job
	key := types.NamespacedName{
		Namespace: request.Run.Namespace,
		Name:      jobName,
	}
	if err := r.client.Get(ctx, key, &job); err != nil {
		if apierrors.IsNotFound(err) {
			job := r.buildJob(request, jobName)
			if createErr := r.client.Create(ctx, &job); createErr != nil {
				return Result{}, createErr
			}
			return Result{}, ErrRuntimeInProgress
		}
		return Result{}, err
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return Result{}, fmt.Errorf("worker job %q failed: %s", job.Name, condition.Message)
		}
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return workerJobResult(request, job), nil
		}
	}

	if job.Status.Succeeded > 0 {
		return workerJobResult(request, job), nil
	}
	return Result{}, ErrRuntimeInProgress
}

func (r WorkerRuntime) buildJob(request Request, name string) batchv1.Job {
	backoffLimit := int32(0)
	ttlSecondsAfterFinished := int32(300)
	runAsUser := int64(65532)
	runAsGroup := int64(65532)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: request.Run.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "agent-control-plane-worker",
				"app.kubernetes.io/managed-by": "agent-control-plane",
				"windosx.com/agent":            request.Agent.Name,
				"windosx.com/agentrun":         request.Run.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: apiv1alpha1.GroupVersion.String(),
					Kind:       "AgentRun",
					Name:       request.Run.Name,
					UID:        request.Run.UID,
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       "agent-control-plane-worker",
						"app.kubernetes.io/managed-by": "agent-control-plane",
						"windosx.com/agentrun":         request.Run.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    &runAsUser,
						RunAsGroup:   &runAsGroup,
					},
					Containers: []corev1.Container{
						{
							Name:            "worker",
							Image:           r.image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         r.command,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Env: []corev1.EnvVar{
								{Name: "AGENT_NAME", Value: request.Agent.Name},
								{Name: "AGENT_RUN_NAME", Value: request.Run.Name},
								{Name: "AGENT_RUN_NAMESPACE", Value: request.Run.Namespace},
								{Name: "AGENT_REVISION", Value: request.Agent.Status.CompiledRevision},
							},
						},
					},
				},
			},
		},
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func workerJobResult(request Request, job batchv1.Job) Result {
	return Result{
		Output: apiv1alpha1.FreeformObject{
			"summary":          JSONValue(fmt.Sprintf("Worker job %s completed for %s.", job.Name, request.Agent.Name)),
			"hazards":          JSONValue([]interface{}{}),
			"overallRiskLevel": JSONValue("low"),
			"nextActions":      JSONValue([]string{"replace placeholder worker image with the real runtime"}),
			"confidence":       JSONValue(1.0),
			"needsHumanReview": JSONValue(false),
		},
		TraceRef: apiv1alpha1.FreeformObject{
			"provider": JSONValue("kubernetes-job"),
			"jobName":  JSONValue(job.Name),
		},
		Reason:  "WorkerJobSucceeded",
		Message: "worker job completed successfully",
	}
}

func jobNameForRun(run apiv1alpha1.AgentRun) string {
	hash := sha256.Sum256([]byte(run.Namespace + "/" + run.Name))
	suffix := hex.EncodeToString(hash[:])[:10]
	prefix := dnsLabelPrefix("agentrun-" + run.Name)
	maxPrefixLength := 63 - len(suffix) - 1
	if len(prefix) > maxPrefixLength {
		prefix = strings.TrimRight(prefix[:maxPrefixLength], "-")
	}
	if prefix == "" {
		prefix = "agentrun"
	}
	return prefix + "-" + suffix
}

func dnsLabelPrefix(value string) string {
	var builder strings.Builder
	lastWasDash := false
	for _, char := range strings.ToLower(value) {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if isAllowed {
			builder.WriteRune(char)
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			builder.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
