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
| `DetectVCS(dir)`                    | `"jj"` \| `"git"` \| `""` from checkout markers (jj wins if colocated) |

## Extraction rules

- `Branch` is only set when unambiguous: a trailing file path could belong to a slash-containing branch name, so `.../tree/main` sets it and `.../tree/main/src/foo.go` does not.
- Bitbucket's `src/<ref>` does not name its ref type: a 40-hex value is classified as `Commit`, anything else lands in `Rev`.
- Gitea's `src/branch|tag|commit/<value>` names its ref type explicitly and is extracted as such.
- Bare-host parsing requires a dot in the host, so `owner/repo` is shorthand, not a URL - and dotless hosts like `localhost` need an explicit scheme.
