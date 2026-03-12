package deployer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/pkg/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
)

var applicationGVR = schema.GroupVersionResource{
	Group:    "core.oam.dev",
	Version:  "v1beta1",
	Resource: "applications",
}

// WasmCloudDeployer deploys artifacts as wasmCloud OAM Applications
// via the wasmcloud-operator's Kubernetes CRDs.
type WasmCloudDeployer struct {
	dynamicClient dynamic.Interface
	k8sClientset  kubernetes.Interface
	namespace     string
	logger        *slog.Logger
}

// NewWasmCloudDeployer creates a new WasmCloudDeployer.
func NewWasmCloudDeployer(
	dynClient dynamic.Interface,
	k8sClientset kubernetes.Interface,
	cfg config.DeploymentConfig,
	logger *slog.Logger,
) *WasmCloudDeployer {
	return &WasmCloudDeployer{
		dynamicClient: dynClient,
		k8sClientset:  k8sClientset,
		namespace:     cfg.Namespace,
		logger:        logger,
	}
}

func (d *WasmCloudDeployer) Deploy(ctx context.Context, artifact *api.Artifact) (*DeployResult, error) {
	app := d.buildApplication(artifact)

	d.logger.Info("creating wasmCloud OAM Application",
		"name", artifact.Name,
		"namespace", d.namespace,
	)

	_, err := d.dynamicClient.Resource(applicationGVR).Namespace(d.namespace).Create(
		ctx, app, metav1.CreateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("creating OAM Application: %w", err)
	}

	// wasmCloud apps get a URL via the operator-created service
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local", artifact.Name, d.namespace)

	return &DeployResult{URL: url}, nil
}

func (d *WasmCloudDeployer) Update(ctx context.Context, artifact *api.Artifact) (*DeployResult, error) {
	app := d.buildApplication(artifact)

	_, err := d.dynamicClient.Resource(applicationGVR).Namespace(d.namespace).Update(
		ctx, app, metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("updating OAM Application: %w", err)
	}

	url := fmt.Sprintf("http://%s.%s.svc.cluster.local", artifact.Name, d.namespace)
	return &DeployResult{URL: url}, nil
}

func (d *WasmCloudDeployer) Delete(ctx context.Context, artifact *api.Artifact) error {
	d.logger.Info("deleting wasmCloud OAM Application", "name", artifact.Name)
	return d.dynamicClient.Resource(applicationGVR).Namespace(d.namespace).Delete(
		ctx, artifact.Name, metav1.DeleteOptions{},
	)
}

func (d *WasmCloudDeployer) GetURL(_ context.Context, artifact *api.Artifact) (string, error) {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", artifact.Name, d.namespace), nil
}

func (d *WasmCloudDeployer) GetLogs(ctx context.Context, artifact *api.Artifact, lines int) ([]string, error) {
	tailLines := int64(lines)
	pods, err := d.k8sClientset.CoreV1().Pods(d.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", artifact.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return []string{"(no pods found for wasmCloud application)"}, nil
	}

	pod := pods.Items[0]
	req := d.k8sClientset.CoreV1().Pods(d.namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		TailLines: &tailLines,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("streaming logs: %w", err)
	}
	defer stream.Close()

	var logLines []string
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}
	return logLines, scanner.Err()
}

// buildApplication creates an OAM Application manifest for wasmCloud.
func (d *WasmCloudDeployer) buildApplication(artifact *api.Artifact) *unstructured.Unstructured {
	port := artifact.Port
	if port == 0 {
		port = 8080
	}

	// Build OAM Application spec
	spec := map[string]interface{}{
		"components": []interface{}{
			map[string]interface{}{
				"name": artifact.Name,
				"type": "component",
				"properties": map[string]interface{}{
					"image": artifact.ImageRef,
				},
				"traits": []interface{}{
					map[string]interface{}{
						"type": "spreadscaler",
						"properties": map[string]interface{}{
							"instances": 1,
						},
					},
					map[string]interface{}{
						"type": "httpserver",
						"properties": map[string]interface{}{
							"port": port,
						},
					},
				},
			},
		},
	}

	labels := map[string]interface{}{
		"app.kubernetes.io/managed-by": "vibed",
		"vibed.dev/artifact-id":        artifact.ID,
	}

	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "core.oam.dev/v1beta1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      artifact.Name,
				"namespace": d.namespace,
				"labels":    labels,
				"annotations": map[string]interface{}{
					"version": "v0.0.1",
				},
			},
			"spec": spec,
		},
	}

	// Ensure the app is valid JSON by marshaling/unmarshaling
	data, _ := json.Marshal(app.Object)
	json.Unmarshal(data, &app.Object)

	return app
}
