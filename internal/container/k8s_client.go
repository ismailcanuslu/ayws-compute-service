package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var ErrK8sNotConfigured = errors.New("kubernetes yapılandırılmamış — kubeconfig veya in-cluster config gerekli")

// K8sService manages Kubernetes resources.
type K8sService struct {
	client    kubernetes.Interface
	namespace string
}

// NewK8sService creates a Kubernetes client using kubeconfig or in-cluster config.
func NewK8sService(kubeconfig, defaultNS string) (*K8sService, error) {
	var cfg *rest.Config
	var err error

	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// In-cluster (pod içinde çalışırken)
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("kubernetes config yüklenemedi: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes istemcisi oluşturulamadı: %w", err)
	}

	if defaultNS == "" {
		defaultNS = "default"
	}
	return &K8sService{client: client, namespace: defaultNS}, nil
}

// ListNamespaces returns all namespaces.
func (s *K8sService) ListNamespaces(ctx context.Context) ([]K8sNamespace, error) {
	list, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("namespace listesi alınamadı: %w", err)
	}
	result := make([]K8sNamespace, 0, len(list.Items))
	for _, ns := range list.Items {
		result = append(result, K8sNamespace{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
		})
	}
	return result, nil
}

// ListPods returns pods in a namespace.
func (s *K8sService) ListPods(ctx context.Context, ns string) ([]K8sPod, error) {
	if ns == "" {
		ns = s.namespace
	}
	list, err := s.client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("pod listesi alınamadı: %w", err)
	}
	result := make([]K8sPod, 0, len(list.Items))
	for _, p := range list.Items {
		age := time.Since(p.CreationTimestamp.Time).Round(time.Second).String()
		result = append(result, K8sPod{
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    string(p.Status.Phase),
			Node:      p.Spec.NodeName,
			IP:        p.Status.PodIP,
			Age:       age,
		})
	}
	return result, nil
}

// ListDeployments returns deployments in a namespace.
func (s *K8sService) ListDeployments(ctx context.Context, ns string) ([]K8sDeployment, error) {
	if ns == "" {
		ns = s.namespace
	}
	list, err := s.client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("deployment listesi alınamadı: %w", err)
	}
	result := make([]K8sDeployment, 0, len(list.Items))
	for _, d := range list.Items {
		result = append(result, mapDeployment(d))
	}
	return result, nil
}

// CreateDeployment creates a Kubernetes Deployment.
func (s *K8sService) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*K8sDeployment, error) {
	ns := req.Namespace
	if ns == "" {
		ns = s.namespace
	}
	replicas := req.Replicas
	if replicas == 0 {
		replicas = 1
	}

	// Env
	envVars := make([]corev1.EnvVar, 0, len(req.EnvVars))
	for k, v := range req.EnvVars {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	labels := req.Labels
	if labels == nil {
		labels = map[string]string{"app": req.Name}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  req.Name,
							Image: req.Image,
							Env:   envVars,
							Ports: []corev1.ContainerPort{{ContainerPort: req.Port}},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("64Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := s.client.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("deployment oluşturulamadı: %w", err)
	}
	d := mapDeployment(*created)
	return &d, nil
}

// ScaleDeployment updates the replica count.
func (s *K8sService) ScaleDeployment(ctx context.Context, ns, name string, replicas int32) (*K8sDeployment, error) {
	if ns == "" {
		ns = s.namespace
	}
	scale, err := s.client.AppsV1().Deployments(ns).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("scale alınamadı: %w", err)
	}
	scale.Spec.Replicas = replicas
	if _, err := s.client.AppsV1().Deployments(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{}); err != nil {
		return nil, fmt.Errorf("scale güncellenemedi: %w", err)
	}
	d, err := s.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	mapped := mapDeployment(*d)
	return &mapped, nil
}

// DeleteDeployment removes a deployment.
func (s *K8sService) DeleteDeployment(ctx context.Context, ns, name string) error {
	if ns == "" {
		ns = s.namespace
	}
	return s.client.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

func mapDeployment(d appsv1.Deployment) K8sDeployment {
	image := ""
	if len(d.Spec.Template.Spec.Containers) > 0 {
		image = d.Spec.Template.Spec.Containers[0].Image
	}
	replicas := int32(0)
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	}
	return K8sDeployment{
		Name:      d.Name,
		Namespace: d.Namespace,
		Image:     image,
		Replicas:  replicas,
		Ready:     d.Status.ReadyReplicas,
		Labels:    d.Labels,
		CreatedAt: d.CreationTimestamp.Time,
	}
}
