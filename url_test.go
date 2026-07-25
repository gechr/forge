package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Ref
	}{
		// GitHub
		{
			name:  "github https",
			input: "https://github.com/owner/repo",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https trailing slash",
			input: "https://github.com/owner/repo/",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https .git suffix",
			input: "https://github.com/owner/repo.git",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https commits page",
			input: "https://github.com/jurplel/InstantSpaceSwitcher/commits/main/",
			want: Ref{
				Branch:        "main",
				CloneURL:      "https://github.com/jurplel/InstantSpaceSwitcher.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "InstantSpaceSwitcher",
				Owner:         "jurplel",
				Scheme:        "https",
			},
		},
		{
			name:  "github https tree path",
			input: "https://github.com/owner/repo/tree/develop/src/foo.go",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "src/foo.go",
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github branch tree",
			input: "https://github.com/owner/repo/tree/develop",
			want: Ref{
				Branch:        "develop",
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github branch commits",
			input: "https://github.com/owner/repo/commits/develop",
			want: Ref{
				Branch:        "develop",
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https blob path",
			input: "https://github.com/owner/repo/blob/main/README.md",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "README.md",
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https nested blob path",
			input: "https://github.com/owner/repo/blob/main/src/deep/file.go",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "src/deep/file.go",
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https actions",
			input: "https://github.com/owner/repo/actions/runs/12345",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https pull request",
			input: "https://github.com/owner/repo/pull/42",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "42",
				Scheme:        "https",
			},
		},
		{
			name:  "github release tag",
			input: "https://github.com/owner/repo/releases/tag/v1.2.3",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
				Tag:           "v1.2.3",
			},
		},
		{
			name:  "github release tag containing slash",
			input: "https://github.com/owner/repo/releases/tag/release%2Fv1.0.0",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
				Tag:           "release/v1.0.0",
			},
		},
		{
			name:  "github commit",
			input: "https://github.com/owner/repo/commit/83c74cc3e85aeaa4b63de7dc529909791de67206",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				Commit:        "83c74cc3e85aeaa4b63de7dc529909791de67206",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https www prefix",
			input: "https://www.github.com/owner/repo",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github http",
			input: "http://github.com/owner/repo",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github ssh",
			input: "git@github.com:owner/repo.git",
			want: Ref{
				CloneURL:      "git@github.com:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},
		{
			name:  "github ssh scheme",
			input: "ssh://git@github.com/owner/repo.git",
			want: Ref{
				CloneURL:      "git@github.com:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},
		{
			name:  "github git scheme",
			input: "git://github.com/owner/repo",
			want: Ref{
				CloneURL:      "git://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "git",
			},
		},
		{
			name:  "github bare host",
			input: "github.com/owner/repo",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "github https with PR fragment",
			input: "https://github.com/owner/repo#21",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "21",
				Scheme:        "https",
			},
		},
		{
			name:  "github bare host with PR fragment",
			input: "github.com/owner/repo#21",
			want: Ref{
				CloneURL:      "https://github.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "github",
				Host:          "github.com",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "21",
				Scheme:        "https",
			},
		},

		// GitLab
		{
			name:  "gitlab https",
			input: "https://gitlab.com/owner/repo",
			want: Ref{
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab https tree via dash separator",
			input: "https://gitlab.com/owner/repo/-/tree/main",
			want: Ref{
				Branch:        "main",
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab tree with file path leaves branch empty",
			input: "https://gitlab.com/owner/repo/-/tree/main/src/deep",
			want: Ref{
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "src/deep",
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab nested group with dash separator",
			input: "https://gitlab.com/group/subgroup/repo/-/tree/main",
			want: Ref{
				Branch:        "main",
				CloneURL:      "https://gitlab.com/group/subgroup/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "group/subgroup",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab merge request",
			input: "https://gitlab.com/owner/repo/-/merge_requests/5",
			want: Ref{
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "5",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab nested group merge request",
			input: "https://gitlab.com/group/subgroup/repo/-/merge_requests/5",
			want: Ref{
				CloneURL:      "https://gitlab.com/group/subgroup/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "group/subgroup",
				PullRequest:   "5",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab blob file path",
			input: "https://gitlab.com/owner/repo/-/blob/main/src/main.go",
			want: Ref{
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "src/main.go",
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "gitlab ssh",
			input: "git@gitlab.com:owner/repo.git",
			want: Ref{
				CloneURL:      "git@gitlab.com:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},
		{
			name:  "gitlab bare host",
			input: "gitlab.com/owner/repo",
			want: Ref{
				CloneURL:      "https://gitlab.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "gitlab",
				Host:          "gitlab.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},

		// Codeberg / Forgejo
		{
			name:  "codeberg https",
			input: "https://codeberg.org/owner/repo",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg src path leaves branch empty",
			input: "https://codeberg.org/owner/repo/src/branch/main/file.go",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "file.go",
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg src branch",
			input: "https://codeberg.org/owner/repo/src/branch/develop",
			want: Ref{
				Branch:        "develop",
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg src tag",
			input: "https://codeberg.org/owner/repo/src/tag/v1.2.3",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
				Tag:           "v1.2.3",
			},
		},
		{
			name:  "codeberg src commit",
			input: "https://codeberg.org/owner/repo/src/commit/83c74cc3e85aeaa4b63de7dc529909791de67206",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				Commit:        "83c74cc3e85aeaa4b63de7dc529909791de67206",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg src commit with file path",
			input: "https://codeberg.org/owner/repo/src/commit/83c74cc3e85aeaa4b63de7dc529909791de67206/file.go",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				Commit:        "83c74cc3e85aeaa4b63de7dc529909791de67206",
				ExplicitOwner: true,
				FilePath:      "file.go",
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg pull request",
			input: "https://codeberg.org/owner/repo/pulls/7",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "7",
				Scheme:        "https",
			},
		},
		{
			// A value that is not a full 40-hex hash is not a commit.
			name:  "codeberg src commit non-hex",
			input: "https://codeberg.org/owner/repo/src/commit/notahex",
			want: Ref{
				CloneURL:      "https://codeberg.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "codeberg ssh",
			input: "git@codeberg.org:owner/repo.git",
			want: Ref{
				CloneURL:      "git@codeberg.org:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "codeberg",
				Host:          "codeberg.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},

		// Bitbucket
		{
			name:  "bitbucket https",
			input: "https://bitbucket.org/owner/repo",
			want: Ref{
				CloneURL:      "https://bitbucket.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "bitbucket src path leaves rev empty",
			input: "https://bitbucket.org/owner/repo/src/main/README.md",
			want: Ref{
				CloneURL:      "https://bitbucket.org/owner/repo.git",
				ExplicitOwner: true,
				FilePath:      "README.md",
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			// Bitbucket's src/<ref> does not name its ref type, so a
			// non-commit value stays ambiguous.
			name:  "bitbucket src rev",
			input: "https://bitbucket.org/owner/repo/src/main",
			want: Ref{
				CloneURL:      "https://bitbucket.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				Rev:           "main",
				Scheme:        "https",
			},
		},
		{
			name:  "bitbucket src commit",
			input: "https://bitbucket.org/owner/repo/src/83c74cc3e85aeaa4b63de7dc529909791de67206",
			want: Ref{
				CloneURL:      "https://bitbucket.org/owner/repo.git",
				Commit:        "83c74cc3e85aeaa4b63de7dc529909791de67206",
				ExplicitOwner: true,
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "bitbucket pull request",
			input: "https://bitbucket.org/owner/repo/pull-requests/9",
			want: Ref{
				CloneURL:      "https://bitbucket.org/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				PullRequest:   "9",
				Scheme:        "https",
			},
		},
		{
			name:  "bitbucket ssh",
			input: "git@bitbucket.org:owner/repo.git",
			want: Ref{
				CloneURL:      "git@bitbucket.org:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "bitbucket",
				Host:          "bitbucket.org",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},

		// Sourcehut
		{
			name:  "sourcehut https",
			input: "https://git.sr.ht/~sircmpwn/scdoc",
			want: Ref{
				CloneURL:      "https://git.sr.ht/~sircmpwn/scdoc",
				ExplicitOwner: true,
				Forge:         "sourcehut",
				Host:          "git.sr.ht",
				Name:          "scdoc",
				Owner:         "~sircmpwn",
				Scheme:        "https",
			},
		},
		{
			name:  "sourcehut tree branch",
			input: "https://git.sr.ht/~sircmpwn/scdoc/tree/main",
			want: Ref{
				Branch:        "main",
				CloneURL:      "https://git.sr.ht/~sircmpwn/scdoc",
				ExplicitOwner: true,
				Forge:         "sourcehut",
				Host:          "git.sr.ht",
				Name:          "scdoc",
				Owner:         "~sircmpwn",
				Scheme:        "https",
			},
		},
		{
			name:  "sourcehut tree with file path leaves branch empty",
			input: "https://git.sr.ht/~sircmpwn/scdoc/tree/main/item/scdoc.c",
			want: Ref{
				CloneURL:      "https://git.sr.ht/~sircmpwn/scdoc",
				ExplicitOwner: true,
				Forge:         "sourcehut",
				Host:          "git.sr.ht",
				Name:          "scdoc",
				Owner:         "~sircmpwn",
				Scheme:        "https",
			},
		},
		{
			name:  "sourcehut https log path",
			input: "https://git.sr.ht/~sircmpwn/scdoc/log/main",
			want: Ref{
				CloneURL:      "https://git.sr.ht/~sircmpwn/scdoc",
				ExplicitOwner: true,
				Forge:         "sourcehut",
				Host:          "git.sr.ht",
				Name:          "scdoc",
				Owner:         "~sircmpwn",
				Scheme:        "https",
			},
		},
		{
			name:  "sourcehut ssh",
			input: "git@git.sr.ht:~sircmpwn/scdoc",
			want: Ref{
				CloneURL:      "git@git.sr.ht:~sircmpwn/scdoc",
				ExplicitOwner: true,
				Forge:         "sourcehut",
				Host:          "git.sr.ht",
				Name:          "scdoc",
				Owner:         "~sircmpwn",
				Scheme:        "ssh",
			},
		},

		// Azure DevOps
		{
			name:  "azure devops https",
			input: "https://dev.azure.com/org/project/_git/repo",
			want: Ref{
				CloneURL:      "https://dev.azure.com/org/project/_git/repo",
				ExplicitOwner: true,
				Forge:         "azuredevops",
				Host:          "dev.azure.com",
				Name:          "repo",
				Owner:         "org",
				Scheme:        "https",
			},
		},

		// Generic / unknown forges (self-hosted)
		{
			name:  "self-hosted gitea https",
			input: "https://git.example.com/owner/repo",
			want: Ref{
				CloneURL:      "https://git.example.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "git.example.com",
				Host:          "git.example.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "self-hosted with ui path",
			input: "https://git.example.com/owner/repo/commits/main",
			want: Ref{
				CloneURL:      "https://git.example.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "git.example.com",
				Host:          "git.example.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
		{
			name:  "unknown forge ssh",
			input: "git@git.example.com:owner/repo.git",
			want: Ref{
				CloneURL:      "git@git.example.com:owner/repo.git",
				ExplicitOwner: true,
				Forge:         "git.example.com",
				Host:          "git.example.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "ssh",
			},
		},
		{
			name:  "unknown forge bare host",
			input: "git.example.com/owner/repo",
			want: Ref{
				CloneURL:      "https://git.example.com/owner/repo.git",
				ExplicitOwner: true,
				Forge:         "git.example.com",
				Host:          "git.example.com",
				Name:          "repo",
				Owner:         "owner",
				Scheme:        "https",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseURL(test.input)
			require.True(t, ok, "ParseURL(%q) returned false", test.input)
			require.Equal(t, test.want, got)
		})
	}
}

func TestParseURLRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "plain word", input: "repo"},
		{name: "shorthand", input: "owner/repo"},
		{name: "empty", input: ""},
		{name: "host only", input: "https://github.com/"},
		{name: "host owner only", input: "https://github.com/owner"},
		{name: "azure no _git", input: "https://dev.azure.com/org/project/repo"},
		{name: "windows drive path backslash", input: `C:\path\to\repo`},
		{name: "windows drive path slash", input: "C:/path/to/repo"},
		{name: "dotless host", input: "localhost/owner/repo"},
		{name: "unparseable scheme url", input: "https://%zz"},
		{name: "scp-like empty path", input: "git.example.com:"},
		{name: "generic host owner only", input: "https://git.example.com/owner"},
		{name: "sourcehut owner only", input: "https://git.sr.ht/~owner"},
		{name: "github empty name", input: "https://github.com/owner/.git"},
		{name: "gitlab empty owner", input: "https://gitlab.com//repo"},
		{name: "codeberg empty owner", input: "https://codeberg.org//repo"},
		{name: "bitbucket empty owner", input: "https://bitbucket.org//repo"},
		{name: "azure empty org", input: "https://dev.azure.com//project/_git/repo"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, ok := ParseURL(test.input)
			require.False(t, ok, "ParseURL(%q) should return false", test.input)
		})
	}
}
