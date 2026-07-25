package forge

// Named forges recognized by Resolve and reported in Ref.Forge. Azure DevOps
// is deliberately not a Resolve name - its clone URLs embed an org/project
// path that owner/name construction cannot express - but dev.azure.com URLs
// are still recognized by ParseURL.
const (
	ForgeAzureDevOps = "azuredevops"
	ForgeBitbucket   = "bitbucket"
	ForgeCodeberg    = "codeberg"
	ForgeGitHub      = "github"
	ForgeGitLab      = "gitlab"
	ForgeSourcehut   = "sourcehut"
)

// Canonical hostnames of the built-in forges.
const (
	HostAzureDevOps = "dev.azure.com"
	HostBitbucket   = "bitbucket.org"
	HostCodeberg    = "codeberg.org"
	HostGitHub      = "github.com"
	HostGitLab      = "gitlab.com"
	HostSourcehut   = "git.sr.ht"
)

// Transports reported in Ref.Scheme. HTTP inputs are normalized to HTTPS.
const (
	SchemeGit   = "git"
	SchemeHTTPS = "https"
	SchemeSSH   = "ssh"
)

// VCS marker names returned by DetectVCS.
const (
	VCSGit = "git"
	VCSJJ  = "jj"
)

const (
	dotGit           = ".git"
	maxRepoNameBytes = 255 // common filesystem NAME_MAX for a single path component
	minRefSegments   = 4   // owner/repo/<kind>/<value>
	minRepoSegments  = 2   // owner/repo
	minSrcSegments   = 5   // owner/repo/src/<kind>/<value> (Gitea)
	pathSep          = "/" // URL path separator

	// URL path segments that introduce a ref and (optionally) a file path.
	segmentBlob    = "blob"
	segmentCommits = "commits"
	segmentTree    = "tree"
)
