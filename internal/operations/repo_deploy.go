package operations

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/internal/orchestrator"
	"github.com/vibed-project/vibeD/pkg/api"
)

var gitCommandContext = exec.CommandContext

func DeployArtifactFromRepo(ctx context.Context, orch *orchestrator.Orchestrator, limits config.LimitsConfig, req DeployArtifactFromRepoRequest) (*orchestrator.DeployResult, error) {
	files, err := loadFilesFromRepo(ctx, limits, req)
	if err != nil {
		return nil, err
	}

	return DeployArtifact(ctx, orch, limits, DeployArtifactRequest{
		Name:       req.Name,
		Files:      files,
		Language:   req.Language,
		Target:     req.Target,
		EnvVars:    req.EnvVars,
		SecretRefs: req.SecretRefs,
		Port:       req.Port,
	})
}

func loadFilesFromRepo(ctx context.Context, limits config.LimitsConfig, req DeployArtifactFromRepoRequest) (map[string]string, error) {
	if req.RepoURL == "" {
		return nil, &api.ErrInvalidInput{Field: "repo_url", Message: "repo_url is required"}
	}

	subdir, err := normalizeRepoPath(req.Path)
	if err != nil {
		return nil, err
	}

	cloneURL, err := repoCloneURL(req.RepoURL, req.AuthToken)
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "vibed-repo-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cloneArgs := []string{"clone", "--depth", "1"}
	if req.Branch != "" {
		cloneArgs = append(cloneArgs, "--branch", req.Branch)
	}
	cloneArgs = append(cloneArgs, cloneURL, tempDir)
	if err := runGit(ctx, "", cloneArgs...); err != nil {
		return nil, &api.ErrInvalidInput{Field: "repo_url", Message: fmt.Sprintf("failed to clone repository: %v", err)}
	}

	if req.Commit != "" {
		if err := runGit(ctx, tempDir, "fetch", "--depth", "1", "origin", req.Commit); err != nil {
			return nil, &api.ErrInvalidInput{Field: "commit", Message: fmt.Sprintf("failed to fetch commit %q: %v", req.Commit, err)}
		}
		if err := runGit(ctx, tempDir, "checkout", "--detach", "FETCH_HEAD"); err != nil {
			return nil, &api.ErrInvalidInput{Field: "commit", Message: fmt.Sprintf("failed to check out commit %q: %v", req.Commit, err)}
		}
	}

	root := tempDir
	if subdir != "" {
		root = filepath.Join(tempDir, filepath.FromSlash(subdir))
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, &api.ErrInvalidInput{Field: "path", Message: fmt.Sprintf("path %q does not exist in repo", req.Path)}
			}
			return nil, fmt.Errorf("stat path %q: %w", req.Path, err)
		}
		if !info.IsDir() {
			return nil, &api.ErrInvalidInput{Field: "path", Message: "path must point to a directory"}
		}
	}

	paths, err := trackedRepoFiles(ctx, tempDir)
	if err != nil {
		return nil, &api.ErrInvalidInput{Field: "repo_url", Message: fmt.Sprintf("failed to list repository files: %v", err)}
	}

	files := make(map[string]string)
	for _, repoPath := range paths {
		if subdir != "" {
			if repoPath == subdir {
				continue
			}
			prefix := subdir + "/"
			if !strings.HasPrefix(repoPath, prefix) {
				continue
			}
			repoPath = strings.TrimPrefix(repoPath, prefix)
		}

		absPath := filepath.Join(root, filepath.FromSlash(repoPath))
		info, err := os.Lstat(absPath)
		if err != nil {
			return nil, fmt.Errorf("stat file %q: %w", repoPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, &api.ErrInvalidInput{Field: "repo_url", Message: fmt.Sprintf("symlinked files are not supported: %s", repoPath)}
		}
		if !info.Mode().IsRegular() {
			continue
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", repoPath, err)
		}
		files[repoPath] = string(content)
	}

	if len(files) == 0 {
		field := "repo_url"
		message := "repository does not contain any deployable files"
		if req.Path != "" {
			field = "path"
			message = fmt.Sprintf("path %q does not contain any deployable files", req.Path)
		}
		return nil, &api.ErrInvalidInput{Field: field, Message: message}
	}

	if err := ValidateFileLimits(files, limits); err != nil {
		return nil, err
	}

	return files, nil
}

func repoCloneURL(repoURL, authToken string) (string, error) {
	if authToken == "" {
		return repoURL, nil
	}

	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", &api.ErrInvalidInput{Field: "repo_url", Message: "auth_token requires an absolute https repo_url"}
	}
	if u.Scheme != "https" {
		return "", &api.ErrInvalidInput{Field: "repo_url", Message: "auth_token is only supported with https repo URLs"}
	}

	u.User = url.UserPassword("x-access-token", authToken)
	return u.String(), nil
}

func normalizeRepoPath(path string) (string, error) {
	if path == "" || path == "." {
		return "", nil
	}

	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == "" {
		return "", nil
	}
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.HasPrefix(clean, "/") {
		return "", &api.ErrInvalidInput{Field: "path", Message: "path must be a relative directory inside the repository"}
	}

	return clean, nil
}

func trackedRepoFiles(ctx context.Context, repoDir string) ([]string, error) {
	cmd := gitCommandContext(ctx, "git", "-C", repoDir, "-c", "core.quotepath=false", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list repository files: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	raw := strings.Split(stdout.String(), "\x00")
	files := make([]string, 0, len(raw))
	for _, path := range raw {
		if path == "" {
			continue
		}
		files = append(files, filepath.ToSlash(path))
	}
	return files, nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmdArgs := append([]string{}, args...)
	if dir != "" {
		cmdArgs = append([]string{"-C", dir}, cmdArgs...)
	}

	cmd := gitCommandContext(ctx, "git", cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}
