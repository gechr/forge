package forge

import (
	"errors"
	"fmt"
	"strings"

	xstrings "github.com/gechr/x/strings"
)

// Option configures Parse and ParseShorthand.
type Option func(*options)

type options struct {
	defaultHost  string
	defaultOwner string
}

// WithDefaultOwner sets the owner assumed when a shorthand reference omits
// one ("repo" rather than "owner/repo").
func WithDefaultOwner(owner string) Option {
	return func(o *options) { o.defaultOwner = owner }
}

// WithDefaultHost sets the Host recorded on shorthand results, so a
// GitLab-first consumer can flip the default away from GitHub. It does not
// affect URL parsing, and when unset shorthand results carry an empty Host.
func WithDefaultHost(host string) Option {
	return func(o *options) { o.defaultHost = host }
}

// Parse resolves a repository reference: a full URL in any form ParseURL
// accepts, or failing that a shorthand in any form ParseShorthand accepts.
// Destination suffixes (e.g. clone's "=<dir>") are not part of the grammar -
// strip them before calling.
func Parse(input string, opts ...Option) (Ref, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return Ref{}, errors.New("repository reference cannot be empty")
	}
	if ref, ok := ParseURL(s); ok {
		return ref, nil
	}
	return ParseShorthand(s, opts...)
}

// ParseShorthand parses a shorthand repository reference:
//
//	repo
//	owner/repo
//	owner/repo#42          (pull request)
//	owner/repo@main        (branch or tag - see Ref.Rev)
//	owner/repo@v1.2.3
//	owner/repo@<sha>       (40-hex commit)
func ParseShorthand(input string, opts ...Option) (Ref, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	text := strings.TrimSpace(input)
	if text == "" {
		return Ref{}, errors.New("repository reference cannot be empty")
	}

	// @rev is cut from the whole reference (not the name) so that
	// slash-containing branch names like feat/x survive the owner split. A
	// leading @ is an owner sigil (e.g. GitHub's @me), never a rev.
	var rev string
	if at := strings.LastIndex(text, "@"); at > 0 {
		text, rev = text[:at], text[at+1:]
		// A "#" inside the rev means the PR reference came after the rev
		// ("repo@main#42"); reject it rather than silently treating it as
		// part of the rev. The valid ordering is "repo#42@main".
		if rev == "" || strings.Contains(rev, "#") {
			return Ref{}, fmt.Errorf("invalid repository %q", input)
		}
	}

	owner := o.defaultOwner
	name := text
	explicitOwner := false
	if before, after, ok := strings.Cut(text, pathSep); ok {
		if before == "" || after == "" || strings.Contains(after, pathSep) {
			return Ref{}, fmt.Errorf("invalid repository %q", input)
		}
		owner = before
		name = after
		explicitOwner = true
	}

	var pr string
	if namePart, prPart, ok := strings.Cut(name, "#"); ok {
		name = namePart
		pr = prPart
	}

	if !IsValidRepoName(name) {
		return Ref{}, fmt.Errorf("invalid repository %q", input)
	}

	ref := Ref{
		ExplicitOwner: explicitOwner,
		Host:          o.defaultHost,
		Name:          strings.TrimSuffix(name, dotGit),
		Owner:         owner,
		PullRequest:   pr,
	}
	// A 40-hex rev can only be a commit; anything else is ambiguous between
	// a branch and a tag, so it lands in Rev for the consumer to resolve
	// against the remote.
	if xstrings.IsGitCommit(rev) {
		ref.Commit = rev
	} else {
		ref.Rev = rev
	}
	return ref, nil
}
