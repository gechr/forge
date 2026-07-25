package forge

import (
	"path/filepath"

	xos "github.com/gechr/x/os"
)

// DetectVCS inspects an existing checkout to decide which VCS drives it,
// returning VCSJJ, VCSGit, or "" when neither marker is present. A `.jj`
// directory takes precedence over `.git`, which covers both jj layouts:
// colocated repos have a top-level `.git` alongside `.jj` and jj should own
// the working copy, while non-colocated repos have no top-level `.git` at
// all (their git store lives inside `.jj/repo/store/git`).
func DetectVCS(dir string) string {
	if ok, _ := xos.Exists(filepath.Join(dir, ".jj")); ok {
		return VCSJJ
	}
	if ok, _ := xos.Exists(filepath.Join(dir, dotGit)); ok {
		return VCSGit
	}
	return ""
}
