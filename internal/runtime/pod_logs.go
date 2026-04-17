package runtime

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const workerContainerName = "worker"

type PodLogs struct {
	PodName       string
	ContainerName string
	Text          string
}

type PodLogReader interface {
	ReadJobPodLogs(ctx context.Context, namespace string, jobName string) (PodLogs, error)
}

type KubernetesPodLogReader struct {
	Clientset kubernetes.Interface
}

func (r KubernetesPodLogReader) ReadJobPodLogs(ctx context.Context, namespace string, jobName string) (PodLogs, error) {
	if r.Clientset == nil {
		return PodLogs{}, fmt.Errorf("kubernetes clientset is required")
	}
	pods, err := r.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{"job-name": jobName}.String(),
	})
	if err != nil {
		return PodLogs{}, err
	}
	if len(pods.Items) == 0 {
		return PodLogs{}, fmt.Errorf("worker job %q has no Pods", jobName)
	}
	pod := pods.Items[0]
	request := r.Clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: workerContainerName,
	})
	stream, err := request.Stream(ctx)
	if err != nil {
		return PodLogs{}, err
	}
	defer stream.Close()

	raw, err := io.ReadAll(stream)
	if err != nil {
		return PodLogs{}, err
	}
	return PodLogs{
		PodName:       pod.Name,
		ContainerName: workerContainerName,
		Text:          string(raw),
	}, nil
}
