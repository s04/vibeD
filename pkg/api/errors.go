package api

import "fmt"

// ErrNotFound indicates the requested artifact does not exist.
type ErrNotFound struct {
	ArtifactID string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("artifact %q not found", e.ArtifactID)
}

// ErrAlreadyExists indicates an artifact with the given name already exists.
type ErrAlreadyExists struct {
	Name string
}

func (e *ErrAlreadyExists) Error() string {
	return fmt.Sprintf("artifact with name %q already exists", e.Name)
}

// ErrTargetUnavailable indicates the requested deployment target is not available.
type ErrTargetUnavailable struct {
	Target DeploymentTarget
}

func (e *ErrTargetUnavailable) Error() string {
	return fmt.Sprintf("deployment target %q is not available in this cluster", e.Target)
}

// ErrBuildFailed indicates the container image build failed.
type ErrBuildFailed struct {
	Reason string
}

func (e *ErrBuildFailed) Error() string {
	return fmt.Sprintf("build failed: %s", e.Reason)
}

// ErrDeployFailed indicates the deployment to the cluster failed.
type ErrDeployFailed struct {
	Reason string
}

func (e *ErrDeployFailed) Error() string {
	return fmt.Sprintf("deployment failed: %s", e.Reason)
}

// ErrInvalidInput indicates invalid input parameters.
type ErrInvalidInput struct {
	Field   string
	Message string
}

func (e *ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input for %q: %s", e.Field, e.Message)
}
