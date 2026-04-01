package operations

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vibed-project/vibeD/internal/config"
	"github.com/vibed-project/vibeD/pkg/api"
)

func TestLoadFilesFromRepo_RespectsSubdirAndGitignore(t *testing.T) {
	repoDir := newTestGitRepo(t)
	writeRepoFile(t, repoDir, ".gitignore", "ignored.txt\n")
	writeRepoFile(t, repoDir, "README.md", "# root\n")
	writeRepoFile(t, repoDir, "app/index.html", "<h1>hello</h1>\n")
	writeRepoFile(t, repoDir, "app/style.css", "body { color: red; }\n")
	writeRepoFile(t, repoDir, "ignored.txt", "ignored\n")
	gitRun(t, repoDir, "add", ".")
	gitRun(t, repoDir, "commit", "-m", "initial")

	files, err := loadFilesFromRepo(context.Background(), config.LimitsConfig{
		MaxFileCount:     10,
		MaxTotalFileSize: 1024,
	}, DeployArtifactFromRepoRequest{
		RepoURL: repoDir,
		Path:    "app",
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"index.html": "<h1>hello</h1>\n",
		"style.css":  "body { color: red; }\n",
	}, files)
}

func TestLoadFilesFromRepo_CheckoutCommit(t *testing.T) {
	repoDir := newTestGitRepo(t)
	writeRepoFile(t, repoDir, "index.html", "<h1>v1</h1>\n")
	gitRun(t, repoDir, "add", "index.html")
	gitRun(t, repoDir, "commit", "-m", "v1")
	commitV1 := strings.TrimSpace(gitOutput(t, repoDir, "rev-parse", "HEAD"))

	writeRepoFile(t, repoDir, "index.html", "<h1>v2</h1>\n")
	gitRun(t, repoDir, "commit", "-am", "v2")

	files, err := loadFilesFromRepo(context.Background(), config.LimitsConfig{
		MaxFileCount:     10,
		MaxTotalFileSize: 1024,
	}, DeployArtifactFromRepoRequest{
		RepoURL: repoDir,
		Commit:  commitV1,
	})
	require.NoError(t, err)
	assert.Equal(t, "<h1>v1</h1>\n", files["index.html"])
}

func TestLoadFilesFromRepo_EnforcesFileLimits(t *testing.T) {
	repoDir := newTestGitRepo(t)
	writeRepoFile(t, repoDir, "index.html", "<h1>too big</h1>\n")
	gitRun(t, repoDir, "add", "index.html")
	gitRun(t, repoDir, "commit", "-m", "initial")

	_, err := loadFilesFromRepo(context.Background(), config.LimitsConfig{
		MaxFileCount:     10,
		MaxTotalFileSize: 4,
	}, DeployArtifactFromRepoRequest{
		RepoURL: repoDir,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, `invalid input for "files"`)
}

func TestRepoCloneURL_WithAuthToken(t *testing.T) {
	cloneURL, err := repoCloneURL("https://github.com/example/project", "token-123")
	require.NoError(t, err)
	assert.Equal(t, "https://x-access-token:token-123@github.com/example/project", cloneURL)
}

func TestRepoCloneURL_RejectsAuthTokenForNonHTTPS(t *testing.T) {
	_, err := repoCloneURL("http://example.com/repo.git", "token-123")
	require.Error(t, err)
	assert.IsType(t, &api.ErrInvalidInput{}, err)
	assert.ErrorContains(t, err, "auth_token is only supported with https repo URLs")
}

func TestNormalizeRepoPathRejectsTraversal(t *testing.T) {
	_, err := normalizeRepoPath("../secrets")
	require.Error(t, err)
	assert.IsType(t, &api.ErrInvalidInput{}, err)
}

func newTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.name", "vibeD tests")
	gitRun(t, dir, "config", "user.email", "tests@example.com")
	return dir
}

func writeRepoFile(t *testing.T, repoDir, relativePath, content string) {
	t.Helper()
	absPath := filepath.Join(repoDir, relativePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, output)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, output)
	return string(output)
}
