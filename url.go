package forge

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	xstrings "github.com/gechr/x/strings"
)

// ParseURL attempts to parse a full repository URL. It is a probe rather
// than a validator - callers legitimately try it and fall through to
// shorthand parsing - so failure is reported as ok=false, not an error.
//
// Supported URL forms (per git-clone(1)):
//
//	ssh://[user@]host[:port]/path
//	[user@]host:path              (scp-like)
//	http[s]://host[:port]/path
//	git://host[:port]/path
//	host/path                     (bare hostname)
func ParseURL(raw string) (Ref, bool) {
	switch {
	case strings.HasPrefix(raw, "ssh://"),
		strings.HasPrefix(raw, "git://"),
		strings.HasPrefix(raw, "https://"),
		strings.HasPrefix(raw, "http://"):
		return parseSchemeURL(raw)
	case isSCPLike(raw):
		return parseSCPLike(raw)
	default:
		return parseBareHost(raw)
	}
}

// parseSchemeURL handles any scheme-based URL (https://, ssh://, git://).
func parseSchemeURL(raw string) (Ref, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Ref{}, false
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	path := strings.TrimPrefix(parsed.Path, pathSep)
	// Re-append the fragment so the #<pr> shorthand survives url.Parse;
	// parseGitHub cuts it back off the path.
	if parsed.Fragment != "" {
		path += "#" + parsed.Fragment
	}
	return dispatch(path, parsed.Scheme, host)
}

// isSCPLike returns true if raw looks like an scp-style remote
// ([user@]host:path). It must contain a colon after the host and
// must not look like an absolute path or scheme-based URL.
func isSCPLike(raw string) bool {
	colon := strings.IndexByte(raw, ':')
	// A colon at index 1 is a Windows drive path (C:\...), not a
	// single-letter remote host.
	if colon <= 1 {
		return false
	}
	// If there's a slash before the colon, it's not scp-like.
	slash := strings.IndexByte(raw, '/')
	return slash < 0 || colon < slash
}

// parseSCPLike handles [user@]host:path URLs (scp-style).
func parseSCPLike(raw string) (Ref, bool) {
	host, path, ok := strings.Cut(raw, ":")
	if !ok || path == "" {
		return Ref{}, false
	}
	path = strings.TrimPrefix(path, pathSep)
	// Strip optional user@ prefix from host.
	if _, after, ok := strings.Cut(host, "@"); ok {
		host = after
	}
	return dispatch(path, SchemeSSH, host)
}

// parseBareHost handles inputs like "github.com/owner/repo". The host must
// contain a dot so that "owner/repo" shorthand is not mistaken for a URL -
// which also means dotless hosts like "localhost/owner/repo" are not
// recognized here; use an explicit scheme for those.
func parseBareHost(raw string) (Ref, bool) {
	slash := strings.IndexByte(raw, '/')
	if slash <= 0 {
		return Ref{}, false
	}
	host := raw[:slash]
	if !strings.Contains(host, ".") {
		return Ref{}, false
	}
	path := raw[slash+1:]
	return dispatch(path, SchemeHTTPS, strings.ToLower(host))
}

// dispatch routes to the appropriate per-forge parser based on hostname and
// stamps the result with the forge name.
func dispatch(path, scheme, host string) (Ref, bool) {
	name, parse := host, parseGeneric
	if f, ok := forgeForHost(host); ok {
		// A registered forge always claims its name; a nil Parse keeps
		// the generic parser.
		name = f.Name
		if f.Parse != nil {
			parse = f.Parse
		}
	} else if host == HostAzureDevOps {
		// Azure DevOps is recognized by default but is not in the
		// registry: Resolve("dev.azure.com") yields a generic self-hosted
		// config because bare owner/name construction cannot express its
		// org/project clone paths. The registry lookup above runs first so
		// a consumer-registered forge for the host takes precedence.
		name, parse = ForgeAzureDevOps, parseAzureDevOps
	}
	ref, ok := parse(path, scheme, host)
	if !ok {
		return Ref{}, false
	}
	ref.Forge = name
	return ref, true
}

// splitPath splits a forge URL path into segments, trimming trailing slashes.
func splitPath(path string) []string {
	return strings.Split(strings.TrimSuffix(path, pathSep), pathSep)
}

// buildRef builds a Ref from the extracted components.
func buildRef(scheme, host, owner, name string, appendDotGit bool) (Ref, bool) {
	name = strings.TrimSuffix(name, dotGit)
	if owner == "" || name == "" {
		return Ref{}, false
	}
	if scheme == "http" {
		scheme = SchemeHTTPS
	}
	return Ref{
		CloneURL:      cloneURL(scheme, host, owner+pathSep+name, appendDotGit),
		ExplicitOwner: true,
		Host:          host,
		Name:          name,
		Owner:         owner,
		Scheme:        scheme,
	}, true
}

// parseGitHub extracts owner/repo from GitHub paths, with PR support.
//
// Recognized patterns:
//
//	owner/repo
//	owner/repo/pull/42
//	owner/repo#42
//	owner/repo/tree/main
//	owner/repo/blob/main/README.md
//	owner/repo/commits/main
//	owner/repo/releases/tag/v1.2.3
//	owner/repo/commit/83c74cc3e85aeaa4b63de7dc529909791de67206
func parseGitHub(path, scheme, host string) (Ref, bool) {
	clean := strings.TrimSuffix(path, pathSep)

	// Extract fragment-based PR shorthand (owner/repo#42).
	var fragment string
	if before, after, ok := strings.Cut(clean, "#"); ok {
		clean = before
		fragment = after
	}

	segments := splitPath(clean)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}

	var pr string
	if len(segments) >= minRefSegments && segments[2] == "pull" {
		pr = segments[3]
	}
	if pr == "" && fragment != "" {
		pr = fragment
	}

	ref, ok := buildRef(scheme, host, segments[0], segments[1], true)
	if !ok {
		return Ref{}, false
	}
	if len(segments) >= minRefSegments && segments[2] == "releases" && segments[3] == "tag" {
		// url.Parse decodes %2F into Path before splitPath splits on it,
		// so a tag like release%2Fv1.0.0 arrives as multiple segments;
		// this Join deliberately reassembles it.
		ref.Tag = strings.Join(segments[4:], pathSep)
	}
	if len(segments) >= minRefSegments && segments[2] == "commit" &&
		xstrings.IsGitCommit(segments[3]) {
		ref.Commit = segments[3]
	}
	// Branch is only set for the bare 4-segment form: with a file path
	// present there is no way to tell where a slash-containing branch name
	// ends and the path begins.
	if len(segments) == minRefSegments &&
		(segments[2] == segmentTree || segments[2] == segmentCommits) {
		ref.Branch = segments[3]
	}
	if len(segments) > minRefSegments &&
		(segments[2] == segmentTree || segments[2] == segmentBlob) {
		ref.FilePath = strings.Join(segments[4:], pathSep)
	}
	ref.PullRequest = pr
	return ref, true
}

// parseGitLab extracts owner/repo from GitLab paths, supporting nested groups
// via the /-/ separator.
//
// Recognized patterns:
//
//	owner/repo
//	owner/repo/-/tree/main
//	owner/repo/-/blob/main/README.md
//	group/subgroup/repo/-/merge_requests/5
func parseGitLab(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}

	var owner, name string
	var rest []string
	if dashIdx := slices.Index(segments, "-"); dashIdx >= minRepoSegments {
		// Everything before /-/ is the project path.
		owner = strings.Join(segments[:dashIdx-1], pathSep)
		name = segments[dashIdx-1]
		rest = segments[dashIdx+1:]
	} else {
		// No /-/ separator: standard 2-segment.
		owner = segments[0]
		name = segments[1]
	}

	ref, ok := buildRef(scheme, host, owner, name, true)
	if !ok {
		return Ref{}, false
	}
	if len(rest) > 1 {
		switch rest[0] {
		case "merge_requests":
			ref.PullRequest = rest[1]
		case segmentTree:
			// Branch only when unambiguous: a trailing file path could
			// belong to a slash-containing branch name.
			if filePath := rest[2:]; len(filePath) > 0 {
				ref.FilePath = strings.Join(filePath, pathSep)
			} else {
				ref.Branch = rest[1]
			}
		case segmentBlob:
			if filePath := rest[2:]; len(filePath) > 0 {
				ref.FilePath = strings.Join(filePath, pathSep)
			}
		}
	}
	return ref, true
}

// parseGitea extracts owner/repo from Gitea/Forgejo paths (e.g. Codeberg).
//
// Recognized patterns:
//
//	owner/repo
//	owner/repo/pulls/42
//	owner/repo/src/branch/main
//	owner/repo/src/tag/v1.2.3
//	owner/repo/src/commit/83c74cc3e85aeaa4b63de7dc529909791de67206
//	owner/repo/src/branch/main/file.go
func parseGitea(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}
	ref, ok := buildRef(scheme, host, segments[0], segments[1], true)
	if !ok {
		return Ref{}, false
	}
	switch {
	case len(segments) >= minRefSegments && segments[2] == "pulls":
		ref.PullRequest = segments[3]
	case len(segments) >= minSrcSegments && segments[2] == "src":
		// src/<kind>/<value>[/<path>] - the kind names which ref type the
		// value is, so unlike other forges there is nothing to guess.
		kind, value := segments[3], segments[4]
		filePath := segments[minSrcSegments:]
		if len(filePath) > 0 {
			ref.FilePath = strings.Join(filePath, pathSep)
		}
		switch kind {
		case "commit":
			if xstrings.IsGitCommit(value) {
				ref.Commit = value
			}
		case "branch":
			// Branch/tag only when unambiguous: a trailing file path
			// could belong to a slash-containing ref name.
			if len(filePath) == 0 {
				ref.Branch = value
			}
		case "tag":
			if len(filePath) == 0 {
				ref.Tag = value
			}
		}
	}
	return ref, true
}

// parseBitbucket extracts owner/repo from Bitbucket paths.
//
// Recognized patterns:
//
//	owner/repo
//	owner/repo/pull-requests/42
//	owner/repo/src/main
//	owner/repo/src/main/README.md
func parseBitbucket(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}
	ref, ok := buildRef(scheme, host, segments[0], segments[1], true)
	if !ok {
		return Ref{}, false
	}
	switch {
	case len(segments) >= minRefSegments && segments[2] == "pull-requests":
		ref.PullRequest = segments[3]
	case len(segments) >= minRefSegments && segments[2] == "src":
		// src/<ref> is ambiguous between branch, tag, and commit: only a
		// full 40-hex commit is self-identifying; anything else lands in
		// Rev for the consumer to resolve against the remote.
		value := segments[3]
		filePath := segments[minRefSegments:]
		if len(filePath) > 0 {
			ref.FilePath = strings.Join(filePath, pathSep)
		}
		switch {
		case xstrings.IsGitCommit(value):
			ref.Commit = value
		case len(filePath) == 0:
			ref.Rev = value
		}
	}
	return ref, true
}

// parseSourcehut extracts ~owner/repo from Sourcehut paths.
//
// Recognized patterns:
//
//	~owner/repo
//	~owner/repo/tree/main
//	~owner/repo/log/main
func parseSourcehut(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}
	ref, ok := buildRef(scheme, host, segments[0], segments[1], false)
	if !ok {
		return Ref{}, false
	}
	// Branch only when unambiguous: a trailing file path could belong to a
	// slash-containing branch name.
	if len(segments) == minRefSegments && segments[2] == segmentTree {
		ref.Branch = segments[3]
	}
	return ref, true
}

// parseAzureDevOps extracts org/repo from Azure DevOps paths. The _git
// segment is required: URLs without it (e.g. project overview pages) do not
// identify a repository.
//
// Recognized pattern:
//
//	org/project/_git/repo
func parseAzureDevOps(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)

	gitIdx := slices.Index(segments, "_git")
	if gitIdx < 1 || gitIdx+1 >= len(segments) {
		return Ref{}, false
	}

	owner := segments[0]
	name := segments[gitIdx+1]
	if owner == "" || name == "" {
		return Ref{}, false
	}

	if scheme == "http" {
		scheme = SchemeHTTPS
	}

	// Azure DevOps clone URLs include the full org/project/_git/repo path.
	repoPath := strings.Join(segments[:gitIdx+2], pathSep)

	return Ref{
		CloneURL:      cloneURL(scheme, host, repoPath, false),
		ExplicitOwner: true,
		Host:          host,
		Name:          name,
		Owner:         owner,
		Scheme:        scheme,
	}, true
}

// parseGeneric handles any forge with standard owner/repo paths.
// It takes the first two path segments as owner and repo, discarding
// any trailing UI path segments.
func parseGeneric(path, scheme, host string) (Ref, bool) {
	segments := splitPath(path)
	if len(segments) < minRepoSegments {
		return Ref{}, false
	}
	return buildRef(scheme, host, segments[0], segments[1], true)
}

// cloneURL constructs a clone URL for the given scheme, host, and path.
func cloneURL(scheme, host, repoPath string, appendDotGit bool) string {
	suffix := ""
	if appendDotGit {
		suffix = dotGit
	}
	switch scheme {
	case SchemeSSH:
		return fmt.Sprintf("git@%s:%s%s", host, repoPath, suffix)
	case SchemeGit:
		return fmt.Sprintf("git://%s/%s%s", host, repoPath, suffix)
	default:
		return fmt.Sprintf("https://%s/%s%s", host, repoPath, suffix)
	}
}
