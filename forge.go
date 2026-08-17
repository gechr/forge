// Package forge parses repository references into their coordinates. A
// reference can be a full URL in any of the forms git-clone(1) accepts
// (https, ssh, scp-like, git, bare hostname) or a shorthand (repo,
// owner/repo, owner/repo#42, owner/repo/pull/42, owner/repo@rev), and parsing
// yields the host, owner, name, and any branch, tag, commit, pull request, or
// file path the reference carries. The URL shapes of GitHub, GitLab,
// Sourcehut, Codeberg, Bitbucket, and Azure DevOps are known; anything else
// falls back to a generic owner/repo reading, so self-hosted forges work out
// of the box. Everything here is pure and offline - no network calls;
// resolving what an ambiguous reference points at (e.g. Ref.Rev) is the
// caller's job.
package forge

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
)

// Forge describes how to recognize and construct URLs for a single forge.
type Forge struct {
	// GitSuffix reports whether the forge's clone URLs end in ".git".
	// Sourcehut is the only built-in forge whose URLs do not.
	GitSuffix bool
	// Host is the forge's canonical hostname.
	Host string
	// Name is the identifier Resolve accepts and Ref.Forge reports.
	Name string
	// Parse extracts a Ref from a URL path already split from its scheme
	// and host. It is a plain field rather than an interface method so that
	// registering a self-hosted forge stays a one-liner.
	Parse func(path, scheme, host string) (Ref, bool)
}

// forgeMu guards forgesByName and forgesByHost, so Register is safe to call
// concurrently with parsing.
var forgeMu sync.RWMutex

var forgesByName = map[string]Forge{
	ForgeBitbucket: {
		Name:      ForgeBitbucket,
		Host:      HostBitbucket,
		GitSuffix: true,
		Parse:     parseBitbucket,
	},
	ForgeCodeberg: {Name: ForgeCodeberg, Host: HostCodeberg, GitSuffix: true, Parse: parseGitea},
	ForgeGitHub:   {Name: ForgeGitHub, Host: HostGitHub, GitSuffix: true, Parse: parseGitHub},
	ForgeGitLab:   {Name: ForgeGitLab, Host: HostGitLab, GitSuffix: true, Parse: parseGitLab},
	ForgeSourcehut: {
		Name:      ForgeSourcehut,
		Host:      HostSourcehut,
		GitSuffix: false,
		Parse:     parseSourcehut,
	},
}

var forgesByHost = func() map[string]Forge {
	byHost := make(map[string]Forge, len(forgesByName))
	for _, f := range forgesByName {
		byHost[f.Host] = f
	}
	return byHost
}()

// Resolve resolves a forge identifier - a registered name or a hostname,
// case-insensitively and with surrounding whitespace ignored - to its Forge.
// An empty value defaults to GitHub, and any unregistered value containing a
// dot is accepted as a self-hosted host with generic owner/repo parsing.
func Resolve(value string) (Forge, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	forgeMu.RLock()
	defer forgeMu.RUnlock()
	if v == "" {
		return forgesByName[ForgeGitHub], nil
	}
	if f, ok := forgesByName[v]; ok {
		return f, nil
	}
	if f, ok := forgesByHost[v]; ok {
		return f, nil
	}
	if strings.Contains(v, ".") {
		return Forge{Name: v, Host: v, GitSuffix: true, Parse: parseGeneric}, nil
	}
	return Forge{}, fmt.Errorf(
		"invalid forge %q: expected one of %s, or a hostname",
		value,
		strings.Join(slices.Sorted(maps.Keys(forgesByName)), ", "),
	)
}

// Register adds a forge to the registry consulted by Resolve and ParseURL,
// replacing any existing entry with the same name or host. It lets consumers
// wire up self-hosted forges (e.g. an internal Gitea) whose URL shapes differ
// from the generic owner/repo fallback.
func Register(f Forge) {
	forgeMu.Lock()
	defer forgeMu.Unlock()
	// Evict any entry the new forge displaces so that name and host lookups
	// never disagree about the same forge.
	if old, ok := forgesByName[f.Name]; ok {
		delete(forgesByHost, old.Host)
	}
	if old, ok := forgesByHost[f.Host]; ok {
		delete(forgesByName, old.Name)
	}
	forgesByName[f.Name] = f
	forgesByHost[f.Host] = f
}

// forgeForHost returns the registered forge for a hostname, if any.
func forgeForHost(host string) (Forge, bool) {
	forgeMu.RLock()
	defer forgeMu.RUnlock()
	f, ok := forgesByHost[host]
	return f, ok
}

// gitSuffixFor reports whether clone URLs for host end in ".git". Unknown
// hosts default to true; Sourcehut is the only built-in exception.
func gitSuffixFor(host string) bool {
	if f, ok := forgeForHost(host); ok {
		return f.GitSuffix
	}
	return true
}
