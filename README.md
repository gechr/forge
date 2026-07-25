# forge

Parse repository references - URLs or shorthands - into their coordinates: host, owner, name, branch, tag, commit, pull request, file path. Pure and offline; no network calls anywhere.

Knows the URL shapes of GitHub, GitLab, Sourcehut, Codeberg/Gitea, Bitbucket, and Azure DevOps, and falls back to a generic `owner/repo` reading for self-hosted forges.

## Install

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
// Ref{Owner: "group/subgroup", Name: "repo"} - GitLab nests projects in groups
// arbitrarily deep, so every segment but the last is the owner.
// ParseURL is also a probe: ok reports whether the input was a URL at all.

ref.Slug()     // "owner/name"
ref.WebURL()   // "https://<host>/owner/name"
ref.SSHURL()   // "git@<host>:owner/name.git"
ref.HTTPSURL() // "https://<host>/owner/name.git"
```

### Accepted forms

- **URL** (per `git-clone(1)`): `https://`, `ssh://`, `git://`, scp-like `git@host:path`, bare host `github.com/owner/repo`
- **Shorthand**: `repo`, `owner/repo`, `owner/repo#42`, `owner/repo@rev`

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

Inspects an existing checkout from marker files alone - no subprocess, no repository opened. Driving a VCS is deliberately out of scope.

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

### Behaviour

- **Two questions, two marker sets.** `Detect` reports what *drives* a checkout - git and jj only, so `.hg` and `.svn` yield `""` rather than a name no consumer can act on. `Resolver` reports where a repository *begins*, and all four markers delimit one.
- **Markers are `Lstat`ed.** A `.git` file (submodule, linked worktree) counts, as does a dangling symlink.
- **Paths are canonicalised** before walking, so one repository has one root however the caller spells the way to it.
- **Pair `Root` with `Abs`**, not `filepath.Abs` - the latter leaves symlinks in place, so the path no longer sits under the root and `filepath.Rel` escapes it.
- **Prefer `RootVCS` to `Detect(r.RootDir(d))`** - `RootDir` yields `""` outside a repository, and `Detect("")` would inspect the *process* working directory.

## Extraction rules

- **`Branch`** is set only when unambiguous - a trailing file path could belong to a slash-containing branch name - so `.../tree/main` sets it and `.../tree/main/src/foo.go` does not.
- **Bitbucket** `src/<ref>` does not name its ref type: 40-hex lands in `Commit`, anything else in `Rev`.
- **Gitea** `src/branch|tag|commit/<value>` names its ref type explicitly, and is extracted as such.
- **Bare hosts** need a dot, so `owner/repo` is shorthand rather than a URL, and `localhost` needs an explicit scheme.
- **GitLab** paths are read whole - every segment but the last is the `Owner` - because groups nest arbitrarily deep. `/-/` introduces any non-project path, which is what makes that unambiguous; a legacy dash-less UI path (`/owner/repo/tree/main`) reads as a deeper project path instead.
- **Azure DevOps** keeps `org/project` in `Owner`, so the `Ref` is lossless. Its `_git` segment and separate SSH host cannot be expressed as `host/owner/name`, so `WebURL`, `HTTPSURL`, and `SSHURL` return `""` rather than a URL that has never existed - use `CloneURL`.
