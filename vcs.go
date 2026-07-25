package forge

import "github.com/gechr/forge/vcs"

// DetectVCS inspects an existing checkout to decide which VCS drives it,
// returning VCSJJ, VCSGit, or "" when neither marker is present. A `.jj`
// directory takes precedence over `.git`, which covers both jj layouts:
// colocated repos have a top-level `.git` alongside `.jj` and jj should own
// the working copy, while non-colocated repos have no top-level `.git` at
// all (their git store lives inside `.jj/repo/store/git`).
//
// It delegates to [vcs.Detect]; reach for the vcs package directly when you
// also need [vcs.Resolver] to find the root a nested path belongs to.
func DetectVCS(dir string) string {
	return vcs.Detect(dir)
}
