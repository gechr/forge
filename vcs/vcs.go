// Package vcs inspects an existing checkout: which version control system
// drives it, and where the repository containing a given path begins. Both
// answers come from marker files alone - no subprocess is spawned and no
// repository is opened - so this package is as pure and offline as the
// reference parsing in the parent package.
//
// Driving a VCS is deliberately out of scope. Running git or jj belongs to the
// consumer, whose needs vary too widely (progress reporting, dry-run
// rendering, signing flags) for a shared primitive to serve without either
// crippling one caller or dragging os/exec into this module.
package vcs

import (
	"os"
	"path/filepath"
	"sync"

	xfilepath "github.com/gechr/x/filepath"
	xslices "github.com/gechr/x/slices"
)

// The version control systems this package recognizes.
const (
	Git = "git"
	Hg  = "hg"
	JJ  = "jj"
	SVN = "svn"
)

// The marker each recognized VCS writes at its repository root.
const (
	markerGit = ".git"
	markerHg  = ".hg"
	markerJJ  = ".jj"
	markerSVN = ".svn"
)

// vcsMarkers pairs each recognized VCS with the marker naming its repository
// root, in the order Detect consults them. A marker may be a directory (the
// usual case) or a file (.git in a submodule or linked worktree), so presence
// is what counts, not kind.
//
// drives separates the two questions this package answers. Every marker
// delimits a repository, so Resolver honours all four; only git and jj can
// drive a working copy, so Detect reports only those. Widening Detect would
// hand a caller a VCS name it has no driver for.
type vcsMarker struct {
	name   string
	marker string
	drives bool
}

var vcsMarkers = []vcsMarker{
	// jj precedes git so a colocated repo - .jj alongside a top-level .git -
	// resolves to jj, which owns the working copy. A non-colocated jj repo has
	// no top-level .git at all; its git store lives in .jj/repo/store/git.
	{name: JJ, marker: markerJJ, drives: true},
	{name: Git, marker: markerGit, drives: true},
	{name: Hg, marker: markerHg},
	{name: SVN, marker: markerSVN},
}

// Markers returns the marker names that delimit a repository root, as a fresh
// slice. A caller pruning a directory walk builds its skip set from this
// rather than repeating the literals and letting the two drift.
func Markers() []string {
	markers := make([]string, 0, len(vcsMarkers))
	for _, m := range vcsMarkers {
		markers = append(markers, m.marker)
	}
	return markers
}

// Detect reports which VCS drives the checkout rooted at dir - JJ, Git, or ""
// when neither marker is present. Only dir itself is examined; use a Resolver
// to find the root a nested path belongs to.
//
// Mercurial and Subversion checkouts report "", not Hg or SVN: Detect answers
// "what should I drive this with", and this module's consumers drive only git
// and jj.
//
// An empty dir reports "" rather than inspecting the process working directory,
// which is what joining a marker onto "" would otherwise do. That case is not
// hypothetical: it is exactly what pairing Detect with a Resolver produces for a
// path in no repository, and the wrong answer would depend on where the process
// happened to be started. Prefer [Resolver.RootVCS], which pairs them safely.
func Detect(dir string) string {
	if dir == "" {
		return ""
	}
	for _, m := range vcsMarkers {
		if m.drives && hasMarker(dir, m.marker) {
			return m.name
		}
	}
	return ""
}

// Resolver maps a path to the root of the repository it belongs to, caching
// each directory it resolves so an ancestor is checked at most once. The root
// is the namespace under which a per-repository identifier is unique, which is
// what makes the same name in two checkouts not a clash. Safe for concurrent
// use.
//
// Paths are canonicalised before they are walked, so one repository has one
// root however a caller spells the way to it - relative or absolute, through a
// symlink or through the real path. Without that a single repository would
// yield two namespaces and two identifiers that should collide would not, which
// is a failure that reports nothing.
type Resolver struct {
	cwd string

	mu    sync.Mutex
	cache map[string]string
	canon map[string]string
}

// NewResolver returns an empty resolver. It captures the working directory
// once - callers hand it relative paths, and filepath.Abs would otherwise
// re-issue the getwd syscall for every lookup of a value that never changes
// during a run.
func NewResolver() *Resolver {
	cwd, _ := os.Getwd()
	return &Resolver{
		cwd:   cwd,
		cache: make(map[string]string),
		canon: make(map[string]string),
	}
}

// Root returns the absolute path of the repository the file at path belongs to
// - the nearest ancestor directory holding any marker in Markers - or "" when
// the file is not inside a repository. The file itself need not exist.
func (r *Resolver) Root(path string) string {
	return r.RootDir(filepath.Dir(r.join(path)))
}

// RootDir returns the absolute path of the repository containing the directory
// dir - the nearest ancestor, dir itself included, holding a marker - or ""
// when dir is not inside a repository. Unlike Root, dir is the search start
// rather than a file whose parent is searched, so a repository root resolves to
// itself.
func (r *Resolver) RootDir(dir string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resolve(r.canonical(r.join(dir)))
}

// RootVCS returns the repository containing dir and which VCS drives it, or
// ("", "") when dir is in no repository.
//
// It exists because the obvious pairing is a trap: RootDir yields "" outside a
// repository, and joining a marker onto "" would inspect the process working
// directory instead, reporting whatever VCS happens to govern where the program
// was started. Detect guards that case for exactly this reason, so the pairing
// here is a single expression - but a caller writing it by hand has to know
// that, and three consumers should not each have to rediscover it.
//
// The two answers can disagree about which VCS "owns" a colocated checkout:
// Detect prefers jj because it drives the working copy. A caller wanting a
// different policy - git because it needs no jj binary, say - has the root here
// and can decide for itself.
func (r *Resolver) RootVCS(dir string) (string, string) {
	root := r.RootDir(dir)
	return root, Detect(root)
}

// Abs returns path as this resolver keys on it: absolute, cleaned, and with
// symlinks resolved.
//
// Use it in place of [filepath.Abs] when relating a file to its repository, so
// that the two agree:
//
//	rel, err := filepath.Rel(r.Root(p), r.Abs(p))
//
// [filepath.Abs] does not resolve symlinks, so a path reaching a repository
// through one does not sit under the root it resolves to and Rel yields a
// "../"-escaping result instead of a path within the repository.
func (r *Resolver) Abs(path string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.canonical(r.join(path))
}

// join returns path made absolute against the captured working directory, or
// path unchanged when that is not possible. Symlinks are left alone; see
// canonical.
func (r *Resolver) join(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if r.cwd != "" {
		return filepath.Join(r.cwd, path)
	}
	if a, err := filepath.Abs(path); err == nil {
		return a
	}
	return path
}

// canonical resolves the symlinks in an absolute path, memoised per path so a
// scan resolves each directory once. The caller holds r.mu.
//
// Resolving fails on a path that does not exist, and Root is legitimately asked
// about files that do not, so a missing path is resolved down to its deepest
// existing ancestor and the missing tail re-appended. The recursion is what the
// memo pays for: those ancestors are shared by every path beneath them.
// [xfilepath.ResolveLenient] is not a substitute - it retries only the
// immediate parent, so two or more missing components leave a symlinked path
// unresolved and the repository with two namespaces again.
func (r *Resolver) canonical(path string) string {
	if cached, ok := r.canon[path]; ok {
		return cached
	}

	resolved, err := xfilepath.Resolve(path)
	if err != nil {
		if parent := filepath.Dir(path); parent != path {
			resolved = filepath.Join(r.canonical(parent), filepath.Base(path))
		} else {
			resolved = path
		}
	}

	r.canon[path] = resolved
	return resolved
}

// resolve walks up from dir to find the repository root. The caller holds r.mu;
// recursion is memoised so each directory is statted once across all lookups.
func (r *Resolver) resolve(dir string) string {
	if cached, ok := r.cache[dir]; ok {
		return cached
	}

	var root string
	switch {
	case hasAnyMarker(dir):
		root = dir
	default:
		if parent := filepath.Dir(dir); parent != dir {
			root = r.resolve(parent)
		}
	}

	r.cache[dir] = root
	return root
}

// hasAnyMarker reports whether dir holds any recognized VCS marker.
func hasAnyMarker(dir string) bool {
	return xslices.AnyFunc(vcsMarkers, func(m vcsMarker) bool {
		return hasMarker(dir, m.marker)
	})
}

// hasMarker reports whether dir holds the given marker, as a file or a
// directory. It does not follow symlinks: a marker is a name the VCS owns, and
// a dangling .git symlink still means the directory was set up as a checkout.
func hasMarker(dir, marker string) bool {
	_, err := os.Lstat(filepath.Join(dir, marker))
	return err == nil
}
