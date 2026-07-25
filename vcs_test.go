package forge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectVCS(t *testing.T) {
	t.Parallel()

	t.Run("non-colocated jj", func(t *testing.T) {
		t.Parallel()

		// A non-colocated jj repo has no top-level .git; its git store
		// lives inside .jj/repo/store/git.
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jj", "repo", "store", "git"), 0o755))
		require.Equal(t, VCSJJ, DetectVCS(dir))
	})

	t.Run("git", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
		require.Equal(t, VCSGit, DetectVCS(dir))
	})

	t.Run("colocated jj wins over git", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".jj"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
		require.Equal(t, VCSJJ, DetectVCS(dir))
	})

	t.Run("git worktree marker file", func(t *testing.T) {
		t.Parallel()

		// In a git worktree .git is a file, not a directory.
		dir := t.TempDir()
		require.NoError(
			t,
			os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /elsewhere\n"), 0o644),
		)
		require.Equal(t, VCSGit, DetectVCS(dir))
	})

	t.Run("neither", func(t *testing.T) {
		t.Parallel()

		require.Empty(t, DetectVCS(t.TempDir()))
	})
}
