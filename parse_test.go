package forge

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testDefaultOwner = "default-owner"

func TestParseShorthand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  Ref
	}{
		{
			input: "repo",
			want:  Ref{Owner: testDefaultOwner, Name: "repo"},
		},
		{
			input: "repo.git",
			want:  Ref{Owner: testDefaultOwner, Name: "repo"},
		},
		{
			input: "owner/repo",
			want:  Ref{ExplicitOwner: true, Owner: "owner", Name: "repo"},
		},
		{
			input: "repo#42",
			want:  Ref{Owner: testDefaultOwner, Name: "repo", PullRequest: "42"},
		},
		{
			input: "owner/repo#21",
			want:  Ref{ExplicitOwner: true, Owner: "owner", Name: "repo", PullRequest: "21"},
		},
		{
			input: "@me/repo",
			want:  Ref{ExplicitOwner: true, Owner: "@me", Name: "repo"},
		},
		{
			input: "repo@main",
			want:  Ref{Owner: testDefaultOwner, Name: "repo", Rev: "main"},
		},
		{
			input: "owner/repo@main",
			want:  Ref{ExplicitOwner: true, Owner: "owner", Name: "repo", Rev: "main"},
		},
		{
			input: "owner/repo@v1.2.3",
			want:  Ref{ExplicitOwner: true, Owner: "owner", Name: "repo", Rev: "v1.2.3"},
		},
		{
			input: "owner/repo@feat/nested-branch",
			want: Ref{
				ExplicitOwner: true,
				Owner:         "owner",
				Name:          "repo",
				Rev:           "feat/nested-branch",
			},
		},
		{
			input: "owner/repo@83c74cc3e85aeaa4b63de7dc529909791de67206",
			want: Ref{
				Commit:        "83c74cc3e85aeaa4b63de7dc529909791de67206",
				ExplicitOwner: true,
				Owner:         "owner",
				Name:          "repo",
			},
		},
		{
			// A short hex prefix is not a full commit hash, so it stays
			// ambiguous.
			input: "owner/repo@83c74cc",
			want:  Ref{ExplicitOwner: true, Owner: "owner", Name: "repo", Rev: "83c74cc"},
		},
		{
			input: "owner/repo#42@main",
			want: Ref{
				ExplicitOwner: true,
				Owner:         "owner",
				Name:          "repo",
				PullRequest:   "42",
				Rev:           "main",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, err := ParseShorthand(test.input, WithDefaultOwner(testDefaultOwner))
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestParseShorthandDefaultHost(t *testing.T) {
	t.Parallel()

	got, err := ParseShorthand("owner/repo", WithDefaultHost(HostGitLab))
	require.NoError(t, err)
	require.Equal(
		t,
		Ref{ExplicitOwner: true, Host: HostGitLab, Owner: "owner", Name: "repo"},
		got,
	)
}

func TestParseShorthandRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty",
			input:   "",
			wantErr: "repository reference cannot be empty",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: "repository reference cannot be empty",
		},
		{
			name:    "missing name",
			input:   "owner/",
			wantErr: `invalid repository "owner/"`,
		},
		{
			name:    "missing owner",
			input:   "/repo",
			wantErr: `invalid repository "/repo"`,
		},
		{
			name:    "too many segments",
			input:   "a/b/c",
			wantErr: `invalid repository "a/b/c"`,
		},
		{
			name:    "empty rev",
			input:   "owner/repo@",
			wantErr: `invalid repository "owner/repo@"`,
		},
		{
			name:    "PR reference after rev",
			input:   "owner/repo@main#42",
			wantErr: `invalid repository "owner/repo@main#42"`,
		},
		{
			name:    "invalid name characters",
			input:   "owner/re po",
			wantErr: `invalid repository "owner/re po"`,
		},
		{
			name:  "overlong name",
			input: "owner/" + strings.Repeat("a", maxRepoNameBytes+1),
			wantErr: fmt.Sprintf(
				"invalid repository %q",
				"owner/"+strings.Repeat("a", maxRepoNameBytes+1),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseShorthand(test.input, WithDefaultOwner(testDefaultOwner))
			require.EqualError(t, err, test.wantErr)
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("url", func(t *testing.T) {
		t.Parallel()

		got, err := Parse("https://github.com/owner/repo/pull/42")
		require.NoError(t, err)
		require.Equal(t, Ref{
			CloneURL:      "https://github.com/owner/repo.git",
			ExplicitOwner: true,
			Forge:         ForgeGitHub,
			Host:          HostGitHub,
			Name:          "repo",
			Owner:         "owner",
			PullRequest:   "42",
			Scheme:        SchemeHTTPS,
		}, got)
	})

	t.Run("scp-like url", func(t *testing.T) {
		t.Parallel()

		got, err := Parse("git@github.com:owner/repo.git")
		require.NoError(t, err)
		require.Equal(t, "git@github.com:owner/repo.git", got.CloneURL)
	})

	t.Run("falls through to shorthand", func(t *testing.T) {
		t.Parallel()

		got, err := Parse("repo#7", WithDefaultOwner(testDefaultOwner))
		require.NoError(t, err)
		require.Equal(t, Ref{Owner: testDefaultOwner, Name: "repo", PullRequest: "7"}, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, err := Parse("  ")
		require.EqualError(t, err, "repository reference cannot be empty")
	})
}
