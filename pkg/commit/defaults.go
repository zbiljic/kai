package commit

const (
	DefaultMaxLength = 72
)

/**
 * References:
 * Commitlint:
 * https://github.com/conventional-changelog/commitlint/blob/18fbed7ea86ac0ec9d5449b4979b762ec4305a92/%40commitlint/config-conventional/index.js#L40-L100
 *
 * Conventional Changelog:
 * https://github.com/conventional-changelog/conventional-changelog/blob/d0e5d5926c8addba74bc962553dd8bcfba90e228/packages/conventional-changelog-conventionalcommits/writer-opts.js#L182-L193
 */
var ConventionalCommitTypes = map[string]string{
	"build":    "Changes that affect the build system or external dependencies",
	"chore":    "Other changes that don't modify src or test files",
	"ci":       "Changes to our CI configuration files and scripts",
	"docs":     "Documentation only changes",
	"feat":     "A new feature",
	"fix":      "A bug fix",
	"perf":     "A code change that improves performance",
	"refactor": "A code change that neither fixes a bug nor adds a feature",
	"revert":   "Reverts a previous commit",
	"style":    "Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)",
	"test":     "Adding missing tests or correcting existing tests",
}
