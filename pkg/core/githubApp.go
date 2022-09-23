package core

// GithubApp hanldes operation related to github apps
type GithubApp interface {
	// GetInstallation returns installation data (organization name and installation token)
	GetInstallation(installationID string) (string, *Token, error)
	// GetInstallationToken returns installation token for particular installation ID(Org)
	GetInstallationToken(installationID string) (string, *Token, error)
}
