package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRefSlug(t *testing.T) {
	t.Parallel()

	require.Equal(t, "owner/repo", Ref{Owner: "owner", Name: "repo"}.Slug())
}

func TestRefWebURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  Ref
		want string
	}{
		{
			name: "empty host defaults to github",
			ref:  Ref{Owner: "owner", Name: "repo"},
			want: "https://github.com/owner/repo",
		},
		{
			name: "explicit host",
			ref:  Ref{Host: HostGitLab, Owner: "owner", Name: "repo"},
			want: "https://gitlab.com/owner/repo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, test.ref.WebURL())
		})
	}
}

func TestRefCloneURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       Ref
		wantHTTPS string
		wantSSH   string
	}{
		{
			name:      "github default",
			ref:       Ref{Owner: "owner", Name: "repo"},
			wantHTTPS: "https://github.com/owner/repo.git",
			wantSSH:   "git@github.com:owner/repo.git",
		},
		{
			name:      "sourcehut has no git suffix",
			ref:       Ref{Host: HostSourcehut, Owner: "~owner", Name: "repo"},
			wantHTTPS: "https://git.sr.ht/~owner/repo",
			wantSSH:   "git@git.sr.ht:~owner/repo",
		},
		{
			name:      "unknown host defaults to git suffix",
			ref:       Ref{Host: "git.example.com", Owner: "owner", Name: "repo"},
			wantHTTPS: "https://git.example.com/owner/repo.git",
			wantSSH:   "git@git.example.com:owner/repo.git",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.wantHTTPS, test.ref.HTTPSURL())
			require.Equal(t, test.wantSSH, test.ref.SSHURL())
		})
	}
}

// Azure DevOps interposes a _git segment in its web and clone paths and uses a
// different host entirely for SSH, so no arrangement of host, owner, and name
// reproduces them. The builders report that by returning "" rather than naming
// a page that has never existed; CloneURL still carries the real path.
func TestRefURLBuildersRefuseAzureDevOps(t *testing.T) {
	t.Parallel()

	ref, ok := ParseURL("https://dev.azure.com/org/project/_git/repo")
	require.True(t, ok)

	require.Equal(t, "org/project", ref.Owner, "the project segment stays in the Ref")
	require.Equal(t, "org/project/repo", ref.Slug())
	require.Equal(t, "https://dev.azure.com/org/project/_git/repo", ref.CloneURL)

	require.Empty(t, ref.WebURL())
	require.Empty(t, ref.HTTPSURL())
	require.Empty(t, ref.SSHURL())
}

// A Ref built by hand from coordinates is refused on the same grounds: the
// guard keys on the host, not on how the Ref was produced.
func TestRefURLBuildersRefuseAzureDevOpsFromCoordinates(t *testing.T) {
	t.Parallel()

	ref := Ref{Host: HostAzureDevOps, Owner: "org/project", Name: "repo"}
	require.Empty(t, ref.WebURL())
	require.Empty(t, ref.HTTPSURL())
	require.Empty(t, ref.SSHURL())
}
