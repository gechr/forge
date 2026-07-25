package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Forge
	}{
		{
			name:  "empty defaults to github",
			input: "",
			want:  Forge{Name: ForgeGitHub, Host: HostGitHub, GitSuffix: true},
		},
		{
			name:  "github by name",
			input: "github",
			want:  Forge{Name: ForgeGitHub, Host: HostGitHub, GitSuffix: true},
		},
		{
			name:  "gitlab by name",
			input: "gitlab",
			want:  Forge{Name: ForgeGitLab, Host: HostGitLab, GitSuffix: true},
		},
		{
			name:  "sourcehut by name",
			input: "sourcehut",
			want:  Forge{Name: ForgeSourcehut, Host: HostSourcehut, GitSuffix: false},
		},
		{
			name:  "codeberg by name",
			input: "codeberg",
			want:  Forge{Name: ForgeCodeberg, Host: HostCodeberg, GitSuffix: true},
		},
		{
			name:  "bitbucket by name",
			input: "bitbucket",
			want:  Forge{Name: ForgeBitbucket, Host: HostBitbucket, GitSuffix: true},
		},
		{
			name:  "case insensitive",
			input: "GitHub",
			want:  Forge{Name: ForgeGitHub, Host: HostGitHub, GitSuffix: true},
		},
		{
			name:  "whitespace trimmed",
			input: "  gitlab  ",
			want:  Forge{Name: ForgeGitLab, Host: HostGitLab, GitSuffix: true},
		},
		{
			name:  "host lookup",
			input: "github.com",
			want:  Forge{Name: ForgeGitHub, Host: HostGitHub, GitSuffix: true},
		},
		{
			name:  "sourcehut host",
			input: "git.sr.ht",
			want:  Forge{Name: ForgeSourcehut, Host: HostSourcehut, GitSuffix: false},
		},
		{
			name:  "custom host",
			input: "git.example.com",
			want:  Forge{Name: "git.example.com", Host: "git.example.com", GitSuffix: true},
		},
		{
			name:  "azure devops host is self-hosted generic",
			input: "dev.azure.com",
			want:  Forge{Name: "dev.azure.com", Host: "dev.azure.com", GitSuffix: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Resolve(test.input)
			require.NoError(t, err)
			// Parse is a func field, which never compares equal; check it
			// separately and compare the rest structurally.
			require.NotNil(t, got.Parse)
			got.Parse = nil
			require.Equal(t, test.want, got)
		})
	}
}

func TestResolveRejectsUnknown(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"nope", "git hub", "foo bar", "azuredevops"} {
		_, err := Resolve(input)
		require.EqualError(
			t,
			err,
			`invalid forge "`+input+`": expected one of bitbucket, codeberg, github, gitlab, sourcehut, or a hostname`,
		)
	}
}

func TestRegister(t *testing.T) { //nolint:paralleltest // mutates the global registry
	custom := Forge{
		Name:      "internal",
		Host:      "gitea.internal.example",
		GitSuffix: true,
		Parse:     parseGitea,
	}
	Register(custom)
	t.Cleanup(func() {
		forgeMu.Lock()
		defer forgeMu.Unlock()
		delete(forgesByName, custom.Name)
		delete(forgesByHost, custom.Host)
	})

	got, err := Resolve("internal")
	require.NoError(t, err)
	require.Equal(t, "gitea.internal.example", got.Host)

	got, err = Resolve("gitea.internal.example")
	require.NoError(t, err)
	require.Equal(t, "internal", got.Name)

	// URL parsing now routes the registered host through the custom parser.
	ref, ok := ParseURL("https://gitea.internal.example/owner/repo/pulls/3")
	require.True(t, ok)
	require.Equal(t, "internal", ref.Forge)
	require.Equal(t, "3", ref.PullRequest)
}

func TestRegisterWithoutParser(t *testing.T) { //nolint:paralleltest // mutates the global registry
	custom := Forge{Name: "corp", Host: "git.corp.example", GitSuffix: true}
	Register(custom)
	t.Cleanup(func() {
		forgeMu.Lock()
		defer forgeMu.Unlock()
		delete(forgesByName, custom.Name)
		delete(forgesByHost, custom.Host)
	})

	// The registered name is claimed even though parsing stays generic.
	ref, ok := ParseURL("https://git.corp.example/owner/repo")
	require.True(t, ok)
	require.Equal(t, "corp", ref.Forge)
	require.Equal(t, "owner", ref.Owner)
	require.Equal(t, "repo", ref.Name)
}

func TestRegisterEvictsDisplacedEntries(
	t *testing.T,
) { //nolint:paralleltest // mutates the global registry
	first := Forge{Name: "corp", Host: "git.corp.example", GitSuffix: true}
	second := Forge{Name: "corp2", Host: "git.corp.example", GitSuffix: true}
	third := Forge{Name: "corp2", Host: "git2.corp.example", GitSuffix: true}
	Register(first)
	Register(second)
	t.Cleanup(func() {
		forgeMu.Lock()
		defer forgeMu.Unlock()
		delete(forgesByName, first.Name)
		delete(forgesByName, second.Name)
		delete(forgesByHost, first.Host)
		delete(forgesByHost, third.Host)
	})

	// Re-pointing the host evicted the first forge's name mapping, so name
	// and host lookups agree.
	_, err := Resolve("corp")
	require.Error(t, err)
	got, err := Resolve("git.corp.example")
	require.NoError(t, err)
	require.Equal(t, "corp2", got.Name)

	// Re-pointing the name evicts the old host mapping in turn.
	Register(third)
	ref, ok := ParseURL("https://git.corp.example/owner/repo")
	require.True(t, ok)
	require.Equal(t, "git.corp.example", ref.Forge)
}

func TestRegisterOverridesAzureDevOps(
	t *testing.T,
) { //nolint:paralleltest // mutates the global registry
	custom := Forge{Name: "ado", Host: HostAzureDevOps, GitSuffix: true, Parse: parseGeneric}
	Register(custom)
	t.Cleanup(func() {
		forgeMu.Lock()
		defer forgeMu.Unlock()
		delete(forgesByName, custom.Name)
		delete(forgesByHost, custom.Host)
	})

	// The registry takes precedence over the built-in Azure DevOps
	// special-case, so the generic parser accepts a plain owner/repo path
	// that parseAzureDevOps would reject.
	ref, ok := ParseURL("https://dev.azure.com/owner/repo")
	require.True(t, ok)
	require.Equal(t, "ado", ref.Forge)
	require.Equal(t, "owner", ref.Owner)
	require.Equal(t, "repo", ref.Name)
}
