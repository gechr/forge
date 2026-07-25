package forge

// Ref is the resolved coordinates of a repository reference.
type Ref struct {
	// Branch is the branch name from URLs like .../tree/<branch> (GitHub,
	// GitLab, Sourcehut) or .../src/branch/<branch> (Gitea). It is only
	// set when the branch is unambiguous: with a trailing file path a
	// slash-containing branch name cannot be told apart from the path, so
	// forms like .../tree/<branch>/<path> leave Branch empty.
	Branch string
	// CloneURL is the clone URL derived from the reference, preserving the
	// transport the reference used. Empty for shorthand references.
	CloneURL string
	// Commit is a full 40-hex commit hash from URLs like .../commit/<sha>
	// or .../src/commit/<sha>, or from @<sha> shorthand.
	Commit string
	// ExplicitOwner reports whether the reference named its owner, as
	// opposed to inheriting WithDefaultOwner.
	ExplicitOwner bool
	// FilePath is the sub-path from URLs like .../blob/<ref>/<path>,
	// .../tree/<ref>/<path>, or .../src/<ref>/<path>, assuming a
	// single-segment ref.
	FilePath string
	// Forge is the forge that recognized the reference: a registered name
	// (github, gitlab, ...) or the bare hostname for unrecognized hosts.
	// Empty for shorthand references.
	Forge string
	// Host is the canonical hostname, lowercased with any "www." prefix
	// stripped. Empty for shorthand references unless WithDefaultHost is
	// given.
	Host string
	// Name is the repository name with any ".git" suffix stripped.
	Name string
	// Owner is the user, organization, or (for GitLab) nested group path
	// that owns the repository.
	Owner string
	// PullRequest is the pull/merge request number from URLs like
	// .../pull/<n> or the #<n> shorthand. It is kept as a string because
	// parsing does not validate it - consumers decide how strict to be.
	PullRequest string
	// Rev is an ambiguous revision from @<rev> shorthand or from URLs like
	// Bitbucket's .../src/<ref>: it may name a branch or a tag, and only
	// the remote can tell, so Branch and Tag are left empty. A 40-hex rev
	// is classified as Commit instead.
	Rev string
	// Scheme is the transport the reference used: https, ssh, or git.
	// Empty for shorthand references.
	Scheme string
	// Tag is the tag name from URLs like .../releases/tag/<tag> or
	// .../src/tag/<tag>.
	Tag string
}

// Slug returns "owner/name".
func (r Ref) Slug() string {
	return r.Owner + pathSep + r.Name
}

// WebURL returns the browsable web URL for the repository, or "" when the
// forge's URLs cannot be built from host/owner/name - see [Ref.constructable].
// An empty Host defaults to GitHub.
func (r Ref) WebURL() string {
	if !r.constructable() {
		return ""
	}
	return "https://" + r.hostOrDefault() + pathSep + r.Slug()
}

// HTTPSURL returns the HTTPS clone URL for the repository, or "" when it cannot
// be built - see [Ref.constructable]. Prefer CloneURL when the Ref came from
// ParseURL: it preserves the exact path the reference carried.
func (r Ref) HTTPSURL() string {
	if !r.constructable() {
		return ""
	}
	host := r.hostOrDefault()
	return cloneURL(SchemeHTTPS, host, r.Slug(), gitSuffixFor(host))
}

// SSHURL returns the SSH clone URL for the repository, or "" when it cannot be
// built - see [Ref.constructable].
func (r Ref) SSHURL() string {
	if !r.constructable() {
		return ""
	}
	host := r.hostOrDefault()
	return cloneURL(SchemeSSH, host, r.Slug(), gitSuffixFor(host))
}

// constructable reports whether this forge's URLs follow the host/owner/name
// shape the URL builders assume.
//
// Azure DevOps does not: its web and clone paths interpose a _git segment
// (dev.azure.com/org/project/_git/repo) and its SSH form is different again
// (ssh.dev.azure.com/v3/org/project/repo). Owner carries org/project so the Ref
// itself is lossless, but no arrangement of host, owner, and name reproduces
// those URLs. Returning "" says so; returning a URL assembled anyway would name
// a page that has never existed, and a caller cannot tell that from a good one.
// Use CloneURL, which ParseURL fills in from the path actually given.
func (r Ref) constructable() bool {
	return r.hostOrDefault() != HostAzureDevOps
}

func (r Ref) hostOrDefault() string {
	if r.Host == "" {
		return HostGitHub
	}
	return r.Host
}
