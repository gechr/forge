package forge

// IsValidRepoName reports whether name is a plausible repository name:
// ASCII letters, digits, '-', '_', and '.', at most 255 bytes (the common
// filesystem NAME_MAX for a single path component), and not "." or "..".
func IsValidRepoName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if len(name) > maxRepoNameBytes {
		return false
	}
	for _, r := range name {
		switch {
		case isLower(r), isUpper(r), isDigit(r):
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
