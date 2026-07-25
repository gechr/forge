package forge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidRepoName(t *testing.T) {
	t.Parallel()

	valid := []string{
		"repo",
		"my-repo",
		"my_repo",
		"my.repo",
		"Repo123",
		".github",
		strings.Repeat("a", maxRepoNameBytes),
	}
	for _, name := range valid {
		require.True(t, IsValidRepoName(name), "IsValidRepoName(%q) should be true", name)
	}

	invalid := []string{
		"",
		".",
		"..",
		"re po",
		"re/po",
		"re@po",
		"répo",
		strings.Repeat("a", maxRepoNameBytes+1),
	}
	for _, name := range invalid {
		require.False(t, IsValidRepoName(name), "IsValidRepoName(%q) should be false", name)
	}
}
