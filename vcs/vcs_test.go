package vcs

import (
	"os"
	"path/filepath"
	"testing"

	xfilepath "github.com/gechr/x/filepath"
	"github.com/stretchr/testify/require"
)

// mkmarker creates dir with the given VCS marker. A file marker (e.g. a
// submodule's .git) is written as a file; otherwise it is a directory.
func mkmarker(t *testing.T, dir, marker string, asFile bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, marker)
	if asFile {
		require.NoError(t, os.WriteFile(path, []byte("gitdir: ../real\n"), 0o644))
		return
	}
	require.NoError(t, os.MkdirAll(path, 0o755))
}

func TestMarkers(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{".jj", ".git", ".hg", ".svn"}, Markers())

	// A fresh slice each call: a caller that sorts or appends to the result
	// must not reorder the table every later caller reads.
	Markers()[0] = "mutated"
	require.Equal(t, ".jj", Markers()[0])
}

func TestDetect(t *testing.T) {
	t.Parallel()

	t.Run("non-colocated jj", func(t *testing.T) {
		t.Parallel()

		// A non-colocated jj repo has no top-level .git; its git store
		// lives inside .jj/repo/store/git.
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, markerJJ, "repo", "store", "git"), 0o755))
		require.Equal(t, JJ, Detect(dir))
	})

	t.Run("git", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, markerGit), 0o755))
		require.Equal(t, Git, Detect(dir))
	})

	t.Run("colocated jj wins over git", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, markerJJ), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dir, markerGit), 0o755))
		require.Equal(t, JJ, Detect(dir))
	})

	t.Run("git worktree marker file", func(t *testing.T) {
		t.Parallel()

		// In a git worktree .git is a file, not a directory.
		dir := t.TempDir()
		require.NoError(
			t,
			os.WriteFile(filepath.Join(dir, markerGit), []byte("gitdir: /elsewhere\n"), 0o644),
		)
		require.Equal(t, Git, Detect(dir))
	})

	t.Run("dangling git symlink still counts", func(t *testing.T) {
		t.Parallel()

		// Lstat, not Stat: the directory was set up as a checkout even if the
		// marker's target has gone missing.
		dir := t.TempDir()
		require.NoError(t, os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, markerGit)))
		require.Equal(t, Git, Detect(dir))
	})

	t.Run("mercurial and subversion do not drive", func(t *testing.T) {
		t.Parallel()

		// Both delimit a repository for Resolver, but neither is something a
		// consumer of this module can drive, so Detect declines to name them.
		for _, marker := range []string{markerHg, markerSVN} {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, marker), 0o755))
			require.Empty(t, Detect(dir), marker)
		}
	})

	t.Run("neither", func(t *testing.T) {
		t.Parallel()

		require.Empty(t, Detect(t.TempDir()))
	})
}

// Joining a marker onto "" yields a relative ".git", which would be tested
// against wherever the process happens to be running. This chdirs into a real
// checkout first, so the unguarded version answers confidently and wrongly.
func TestDetectEmptyDirIgnoresProcessDirectory(t *testing.T) {
	repo := canonicalTempDir(t)
	require.NoError(t, os.Mkdir(filepath.Join(repo, markerJJ), 0o755))
	t.Chdir(repo)

	require.Empty(t, Detect(""))
}

func TestResolverRootVCS(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	git := filepath.Join(base, "git")
	colocated := filepath.Join(base, "coloc")
	mkmarker(t, git, markerGit, false)
	mkmarker(t, colocated, markerJJ, false)
	mkmarker(t, colocated, markerGit, false)
	require.NoError(t, os.MkdirAll(filepath.Join(git, "deep"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "loose"), 0o755))

	resolver := NewResolver()

	tests := []struct {
		name     string
		dir      string
		wantRoot string
		wantVCS  string
	}{
		{name: "git repo root", dir: git, wantRoot: git, wantVCS: Git},
		{
			name:     "git repo subdirectory",
			dir:      filepath.Join(git, "deep"),
			wantRoot: git,
			wantVCS:  Git,
		},
		{name: "colocated prefers jj", dir: colocated, wantRoot: colocated, wantVCS: JJ},
		{name: "outside any repo", dir: filepath.Join(base, "loose"), wantRoot: "", wantVCS: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, name := resolver.RootVCS(tt.dir)
			require.Equal(t, tt.wantRoot, root)
			require.Equal(t, tt.wantVCS, name)
		})
	}
}

// The trap RootVCS exists to close: outside any repository it must report
// nothing, not whatever VCS governs the directory the process was started in.
func TestResolverRootVCSIgnoresProcessDirectory(t *testing.T) {
	base := canonicalTempDir(t)
	repo := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	mkmarker(t, repo, markerJJ, false)
	require.NoError(t, os.MkdirAll(outside, 0o755))
	t.Chdir(repo)

	root, name := NewResolver().RootVCS(outside)
	require.Empty(t, root)
	require.Empty(t, name)
}

func TestResolverRoot(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	git := filepath.Join(base, "git")
	jj := filepath.Join(base, "jj")           // pure jj working copy, no .git
	hg := filepath.Join(base, "hg")           // not drivable, but still a root
	colocated := filepath.Join(base, "coloc") // .jj + .git in one dir
	sub := filepath.Join(git, "vendor", "tool")
	mkmarker(t, git, markerGit, false)
	mkmarker(t, jj, markerJJ, false)
	mkmarker(t, hg, markerHg, false)
	mkmarker(t, colocated, markerJJ, false)
	mkmarker(t, colocated, markerGit, false)
	mkmarker(t, sub, markerGit, true) // submodule: .git is a file

	resolver := NewResolver()

	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "git repo", file: filepath.Join(git, "Dockerfile"), want: git},
		{name: "nested file", file: filepath.Join(git, "deep", "x.yaml"), want: git},
		{name: "pure jj repo", file: filepath.Join(jj, "Dockerfile"), want: jj},
		{name: "mercurial repo", file: filepath.Join(hg, "Dockerfile"), want: hg},
		{name: "jj-colocated repo", file: filepath.Join(colocated, "Dockerfile"), want: colocated},
		{name: "nearest repo wins (submodule)", file: filepath.Join(sub, "go.mod"), want: sub},
		{name: "outside any repo", file: filepath.Join(base, "loose.txt"), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, resolver.Root(tt.file))
		})
	}
}

func TestResolverRootDir(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	repo := filepath.Join(base, "repo")
	nested := filepath.Join(repo, "deep", "pkg")
	mkmarker(t, repo, markerGit, false)
	require.NoError(t, os.MkdirAll(nested, 0o755))

	resolver := NewResolver()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		// Unlike Root, the directory itself is the search start, not a file whose
		// parent is searched: the repo root resolves to itself, not its parent.
		{name: "repo root resolves to itself", dir: repo, want: repo},
		{name: "nested dir resolves to root", dir: nested, want: repo},
		{name: "dir outside any repo", dir: base, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, resolver.RootDir(tt.dir))
		})
	}
}

func TestResolverDistinctRootsDoNotClash(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	repoA := filepath.Join(base, "a")
	repoB := filepath.Join(base, "b")
	mkmarker(t, repoA, markerGit, false)
	mkmarker(t, repoB, markerGit, false)

	resolver := NewResolver()

	// The same per-repository identifier in two repos namespaces to different
	// roots, so prefixing yields distinct keys - no clash.
	keyA := resolver.Root(filepath.Join(repoA, "Dockerfile")) + "\x00nginx-version"
	keyB := resolver.Root(filepath.Join(repoB, "Dockerfile")) + "\x00nginx-version"
	require.NotEqual(t, keyA, keyB)
}

// A resolver created after a working-directory change anchors relative paths
// on that directory: the cwd is captured per resolver, not per process, so a
// caller that changed directory before constructing one is never poisoned by
// an earlier capture.
// It also does not matter which spelling of the working directory os.Getwd
// hands back - it returns $PWD when that names the same directory, which under
// t.Chdir is the unresolved one - because the captured value is canonicalised
// like any other path.
func TestResolverRelativePaths(t *testing.T) {
	repo := canonicalTempDir(t)
	mkmarker(t, repo, markerGit, false)
	t.Chdir(repo)

	resolver := NewResolver()
	require.Equal(t, repo, resolver.RootDir("."))
	require.Equal(t, repo, resolver.RootDir(filepath.Join("deep", "nested")))
	require.Equal(t, repo, resolver.Root(filepath.Join("deep", "x.yaml")))
}

// One repository has one root however the caller spells the way to it. Reaching
// it through a symlink must not mint a second namespace: two identifiers that
// should collide would then not, and a missed collision reports nothing.
func TestResolverSymlinkedPathsShareOneRoot(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	realRepo := filepath.Join(base, "realrepo")
	link := filepath.Join(base, "linkrepo")
	mkmarker(t, realRepo, markerGit, false)
	require.NoError(t, os.MkdirAll(filepath.Join(realRepo, "deep"), 0o755))
	require.NoError(t, os.Symlink(realRepo, link))

	resolver := NewResolver()

	viaReal := resolver.Root(filepath.Join(realRepo, "deep", "a.txt"))
	viaLink := resolver.Root(filepath.Join(link, "deep", "a.txt"))
	require.Equal(t, realRepo, viaReal)
	require.Equal(t, viaReal, viaLink)

	// A missing tail more than one component deep must still canonicalise -
	// resolving only the immediate parent would leave this under linkrepo.
	require.Equal(t, realRepo, resolver.Root(filepath.Join(link, "no", "such", "dir", "a.txt")))
}

// The sharp case, and the one a single resolver can hit on its own: Getwd
// resolves symlinks, so a relative path is anchored on a canonical cwd. An
// absolute path left unresolved would land in a different namespace than the
// identical relative one, within one resolver.
func TestResolverRelativeAndAbsoluteAgree(t *testing.T) {
	base := canonicalTempDir(t)
	realRepo := filepath.Join(base, "realrepo")
	link := filepath.Join(base, "linkrepo")
	mkmarker(t, realRepo, markerGit, false)
	require.NoError(t, os.MkdirAll(filepath.Join(realRepo, "deep"), 0o755))
	require.NoError(t, os.Symlink(realRepo, link))
	t.Chdir(realRepo)

	resolver := NewResolver()
	viaRel := resolver.Root(filepath.Join("deep", "a.txt"))
	viaAbs := resolver.Root(filepath.Join(link, "deep", "a.txt"))
	require.Equal(t, realRepo, viaRel)
	require.Equal(t, viaRel, viaAbs)
}

// Abs and Root are two halves of one contract: a caller relating a file to its
// repository does filepath.Rel(Root(p), Abs(p)), which is only meaningful while
// the root contains the path. filepath.Abs does not resolve symlinks, so using
// it here escapes the root - the trap Abs exists to close.
func TestResolverAbsStaysWithinRoot(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	realRepo := filepath.Join(base, "realrepo")
	link := filepath.Join(base, "linkrepo")
	mkmarker(t, realRepo, markerGit, false)
	require.NoError(t, os.MkdirAll(filepath.Join(realRepo, "deep"), 0o755))
	require.NoError(t, os.Symlink(realRepo, link))

	resolver := NewResolver()
	file := filepath.Join(link, "deep", "a.txt")
	root := resolver.Root(file)

	require.True(t, xfilepath.IsWithin(root, resolver.Abs(file)))
	rel, err := filepath.Rel(root, resolver.Abs(file))
	require.NoError(t, err)
	require.Equal(t, filepath.Join("deep", "a.txt"), rel)

	// The contrast that motivates Abs: filepath.Abs leaves the symlink in
	// place, so the same file no longer sits under the root it resolved to.
	naive, err := filepath.Abs(file)
	require.NoError(t, err)
	require.False(t, xfilepath.IsWithin(root, naive))
}

// canonicalTempDir returns a temporary directory with symlinks resolved, so a
// test asserting on canonical roots is not defeated by the platform's own
// symlinked temp path (macOS /var -> /private/var).
func canonicalTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

// The cache is keyed on the directory walked, so a second lookup under the same
// repository reuses the first one's answer rather than re-statting ancestors.
func TestResolverCachesAncestors(t *testing.T) {
	t.Parallel()

	base := canonicalTempDir(t)
	repo := filepath.Join(base, "repo")
	mkmarker(t, repo, markerGit, false)

	resolver := NewResolver()
	require.Equal(t, repo, resolver.Root(filepath.Join(repo, "deep", "a.txt")))

	// Removing the marker cannot change the answer for a directory already
	// resolved - proving the second lookup came from the cache, not the disk.
	require.NoError(t, os.Remove(filepath.Join(repo, markerGit)))
	require.Equal(t, repo, resolver.Root(filepath.Join(repo, "deep", "b.txt")))
}
