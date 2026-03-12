package builder

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibed-project/vibeD/internal/config"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

// WasmBuilder builds WebAssembly components by creating Kubernetes Jobs
// that run wash build + wash push. This compiles source code to wasm
// components and pushes them to an OCI registry.
type WasmBuilder struct {
	clientset    kubernetes.Interface
	namespace    string
	builderImage string
	registryURL  string
	insecure     bool
	pvcName      string
	storagePath  string
	timeout      time.Duration
	logger       *slog.Logger
}

// NewWasmBuilder creates a new WasmBuilder.
func NewWasmBuilder(
	clientset kubernetes.Interface,
	wasmCfg config.WasmBuilderConfig,
	registry config.RegistryConfig,
	namespace string,
	pvcName string,
	storagePath string,
	logger *slog.Logger,
) *WasmBuilder {
	builderImage := wasmCfg.Image
	if builderImage == "" {
		builderImage = "ghcr.io/vibed/wasm-builder:latest"
	}

	timeout := 10 * time.Minute
	if wasmCfg.Timeout != "" {
		if d, err := time.ParseDuration(wasmCfg.Timeout); err == nil {
			timeout = d
		}
	}

	return &WasmBuilder{
		clientset:    clientset,
		namespace:    namespace,
		builderImage: builderImage,
		registryURL:  registry.URL,
		insecure:     wasmCfg.Insecure,
		pvcName:      pvcName,
		storagePath:  storagePath,
		timeout:      timeout,
		logger:       logger,
	}
}

func (b *WasmBuilder) Build(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	b.logger.Info("building wasm component via wash build Job",
		"source", req.SourceDir,
		"image", req.ImageName,
		"language", req.Language,
	)

	// 1. Read source files for scaffolding
	files := make(map[string]string)
	entries, err := os.ReadDir(req.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading source directory %q: %w", req.SourceDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			files[e.Name()] = ""
		}
	}

	// 2. Generate wasm scaffold files (wasmcloud.toml, WIT, etc.)
	scaffoldFiles := GenerateWasmScaffold(req.Language, req.ImageName, files)
	for filename, content := range scaffoldFiles {
		path := filepath.Join(req.SourceDir, filename)
		// Skip if user already provided this file
		if _, statErr := os.Stat(path); statErr == nil {
			b.logger.Info("skipping scaffold file (user-provided)", "file", filename)
			continue
		}
		// Create parent directories for nested files (e.g., wit/world.wit)
		if dir := filepath.Dir(path); dir != req.SourceDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("creating scaffold directory: %w", err)
			}
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("writing scaffold file %q: %w", filename, err)
		}
	}

	// 3. Compute sub-path relative to PVC mount
	subPath := strings.TrimPrefix(req.SourceDir, b.storagePath+"/")

	// 4. Create a unique job name (use artifact ID from parent dir, not "src")
	shortID := filepath.Base(filepath.Dir(req.SourceDir))
	if len(shortID) > 16 {
		shortID = shortID[:16]
	}
	jobName := fmt.Sprintf("vibed-wasm-%s", shortID)

	// 5. Build the wash build + push command
	insecureFlag := ""
	if b.insecure {
		insecureFlag = "--insecure"
	}
	buildCmd := fmt.Sprintf(
		"cd /workspace && wash build && wash push %s %s /workspace/build/*.wasm",
		insecureFlag, req.ImageName,
	)

	// 6. Create K8s Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: b.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vibed",
				"vibed.dev/component":          "wasm-build",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            ptr.To(int32(0)),
			TTLSecondsAfterFinished: ptr.To(int32(120)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "wash",
							Image:           b.builderImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"sh", "-c"},
							Args:            []string{buildCmd},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "source",
									MountPath: "/workspace",
									SubPath:   subPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "source",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: b.pvcName,
								},
							},
						},
					},
				},
			},
		},
	}

	b.logger.Info("creating wasm build Job", "job", jobName, "namespace", b.namespace)
	_, err = b.clientset.BatchV1().Jobs(b.namespace).Create(ctx, job, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		b.logger.Warn("stale wasm build Job exists, deleting and retrying", "job", jobName)
		b.cleanup(ctx, jobName)
		time.Sleep(2 * time.Second)
		_, err = b.clientset.BatchV1().Jobs(b.namespace).Create(ctx, job, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("creating wasm build Job: %w", err)
	}

	// 7. Wait for Job completion
	err = b.waitForJob(ctx, jobName)
	if err != nil {
		logs := b.fetchJobLogs(ctx, jobName)
		b.cleanup(ctx, jobName)
		return nil, fmt.Errorf("wasm build failed: %w\nBuild logs:\n%s", err, logs)
	}

	b.logger.Info("wasm build completed", "image", req.ImageName)
	b.cleanup(ctx, jobName)

	return &BuildResult{
		ImageRef: req.ImageName,
	}, nil
}

func (b *WasmBuilder) waitForJob(_ context.Context, jobName string) error {
	// Use a detached context so MCP client disconnects don't kill the build.
	waitCtx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wasm build timed out after %v", b.timeout)
		case <-ticker.C:
			job, err := b.clientset.BatchV1().Jobs(b.namespace).Get(waitCtx, jobName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("checking Job status: %w", err)
			}

			if job.Status.Succeeded > 0 {
				return nil
			}
			if job.Status.Failed > 0 {
				return fmt.Errorf("wasm build Job failed")
			}
		}
	}
}

func (b *WasmBuilder) fetchJobLogs(_ context.Context, jobName string) string {
	logCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pods, err := b.clientset.CoreV1().Pods(b.namespace).List(logCtx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil || len(pods.Items) == 0 {
		return "(no build logs available)"
	}

	tailLines := int64(50)
	req := b.clientset.CoreV1().Pods(b.namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{
		TailLines: &tailLines,
	})
	stream, err := req.Stream(logCtx)
	if err != nil {
		return fmt.Sprintf("(failed to fetch logs: %v)", err)
	}
	defer stream.Close()

	var lines []string
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

func (b *WasmBuilder) cleanup(_ context.Context, jobName string) {
	cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	propagation := metav1.DeletePropagationBackground
	err := b.clientset.BatchV1().Jobs(b.namespace).Delete(cleanCtx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		b.logger.Warn("failed to cleanup wasm build Job", "job", jobName, "error", err)
	}
}
