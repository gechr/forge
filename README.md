# forge

A pure, offline Go library that parses repository references - full URLs or shorthands - into their coordinates: host, owner, name, branch, tag, commit, pull request, and file path. It knows the URL shapes of GitHub, GitLab, Sourcehut, Codeberg/Gitea, Bitbucket, and Azure DevOps, and falls back to a generic `owner/repo` reading for self-hosted forges. No network calls anywhere.

## Installation

```text
go get github.com/gechr/forge
```

## Usage

```go
ref, err := forge.Parse("https://github.com/gechr/forge/pull/42")
// Ref{Host: "github.com", Owner: "gechr", Name: "forge", PullRequest: "42",
//     Forge: "github", Scheme: "https", CloneURL: "https://github.com/gechr/forge.git", ...}

ref, err = forge.Parse("forge#42", forge.WithDefaultOwner("gechr"))
// Ref{Owner: "gechr", Name: "forge", PullRequest: "42"}

ref, err = forge.Parse("gechr/forge@v1.2.3")
// Ref{Owner: "gechr", Name: "forge", Rev: "v1.2.3"} - branch or tag; the
// consumer resolves Rev against the remote. A 40-hex rev lands in Commit.

ref, ok := forge.ParseURL("git@gitlab.com:group/subgroup/repo.git")
// ParseURL is a probe: ok reports whether the input was a URL at all.

ref.Slug()     // "owner/name"
ref.WebURL()   // "https://<host>/owner/name"
ref.SSHURL()   // "git@<host>:owner/name.git"
ref.HTTPSURL() // "https://<host>/owner/name.git"
```

Accepted URL forms (per `git-clone(1)`): `https://`, `ssh://`, `git://`, scp-like (`git@host:path`), and bare host (`github.com/owner/repo`). Shorthand forms: `repo`, `owner/repo`, `owner/repo#42`, `owner/repo@rev`.

## API

| Symbol                              | Description                                                            |
| ----------------------------------- | ---------------------------------------------------------------------- |
| `Parse(input, opts...)`             | Full grammar: URL or shorthand                                         |
| `ParseURL(raw)`                     | URL forms only; `(Ref, bool)` probe                                    |
| `ParseShorthand(s, opts...)`        | `repo` \| `owner/repo` \| `owner/repo#42` \| `owner/repo@rev`          |
| `WithDefaultOwner(owner)`           | Owner assumed when shorthand omits one                                 |
| `WithDefaultHost(host)`             | Host recorded on shorthand results                                     |
| `Resolve(value)`                    | Forge by name (`github`, ...) or hostname; `""` defaults to GitHub     |
| `Register(forge)`                   | Add a self-hosted forge with its own parser                            |
| `ExpandPRList(spec)`                | `"1,2,5-7"` → `[1 2 5 6 7]`                                            |
| `IsValidRepoName(name)`             | Plausible repository name check                                        |
| `DetectVCS(dir)`                    | Alias for `vcs.Detect`, below                                          |

## Checkout inspection - `forge/vcs`

`github.com/gechr/forge/vcs` answers questions about an existing checkout from marker files alone - no subprocess, no repository opened - so it is as pure and offline as the parsing above. Driving a VCS is deliberately out of scope.

```go
which := vcs.Detect(dir)   // "jj" | "git" | "" - what drives this checkout

r := vcs.NewResolver()
root := r.Root("src/main.go")     // repository the file belongs to, or ""
root, which = r.RootVCS(dir)      // both at once, for a nested path

rel, err := filepath.Rel(r.Root(p), r.Abs(p))  // path within its repository
```

| Symbol                   | Description                                                               |
| ------------------------ | ------------------------------------------------------------------------- |
| `Detect(dir)`            | `"jj"` \| `"git"` \| `""` from `dir`'s own markers (jj wins if colocated) |
| `NewResolver()`          | Resolver caching each directory it walks; safe for concurrent use         |
| `(*Resolver).Root(p)`    | Repository containing the *file* `p`, or `""`                             |
| `(*Resolver).RootDir(d)` | Repository containing `d`, searching `d` itself first                     |
| `(*Resolver).RootVCS(d)` | `(root, vcs)` for `d` - the safe pairing of `RootDir` and `Detect`        |
| `(*Resolver).Abs(p)`     | `p` absolute and symlink-resolved, as the resolver keys on it             |
| `Markers()`              | `.jj`, `.git`, `.hg`, `.svn` - build walk-pruning sets from this          |

`Detect` and `Resolver` answer different questions, so they consider different markers. `Detect` reports what a caller should *drive* the checkout with, and only git and jj qualify - a Mercurial or Subversion checkout yields `""` rather than a name no consumer has a driver for. `Resolver` reports where a repository *begins*, and all four markers delimit one.

Markers are tested with `Lstat`, so a `.git` file (submodule, linked worktree) counts, as does a dangling symlink: the directory was set up as a checkout either way.

A `Resolver` canonicalises paths before walking them, so one repository has one root however a caller spells the way to it - relative or absolute, through a symlink or not. Two consequences worth knowing:

- Pair `Root` with `Abs`, not `filepath.Abs`. The latter does not resolve symlinks, so a path reaching a repository through one does not sit under the root it resolved to, and `filepath.Rel` yields a `"../"`-escaping result instead of a path within the repository.
- Prefer `RootVCS` over `Detect(r.RootDir(d))`. `RootDir` yields `""` outside a repository, and joining a marker onto `""` would inspect the *process* working directory - reporting whatever VCS governs wherever the program was started.

## Extraction rules

- `Branch` is only set when unambiguous: a trailing file path could belong to a slash-containing branch name, so `.../tree/main` sets it and `.../tree/main/src/foo.go` does not.
- Bitbucket's `src/<ref>` does not name its ref type: a 40-hex value is classified as `Commit`, anything else lands in `Rev`.
- Gitea's `src/branch|tag|commit/<value>` names its ref type explicitly and is extracted as such.
- Bare-host parsing requires a dot in the host, so `owner/repo` is shorthand, not a URL - and dotless hosts like `localhost` need an explicit scheme.
