package core

import (
	"time"
)

type RepoSecretItem struct {
	Value   string    `json:"value"`
	Updated time.Time `json:"updated_at"`
}

// Vault defines operation for working with vault store
type Vault interface {
	// CreateSecret create the Vault secret.
	CreateSecret(path string, values map[string]interface{}) error
	// CreateNamespace create the Vault namespace.
	CreateNamespace(namespacePath string) error
	// ListNamespace list the Vault namespace details.
	ListNamespace(namespacePath string) error
	// ListAllNamespaces list all namespaces in Vault.
	ListAllNamespaces(namespacePath string) ([]interface{}, error)
	// GetTokenPath returns the OAuth token path.
	GetTokenPath(gitProvider SCMDriver, mask, id string) string
	// GetInstallationTokenPath returns the Installation token path.
	GetInstallationTokenPath(gitProvider SCMDriver, k8namespace, orgName string) string
	// GetSecretPath returns the repo secret path.
	GetSecretPath(gitProvider SCMDriver, k8namespace, orgName, mask, id string) string
	// GetSecretMetadataPath returns the repo secrets metadata path.
	GetSecretMetadataPath(gitProvider SCMDriver, k8namespace, orgName, mask, id string) string
	// ReadSecret returns the secret in given path.
	ReadSecret(path string) (map[string]interface{}, error)
	// CreateRole create the k8s auth rile in Vault.
	CreateRole(roleName, k8namespace, serviceAccountName string) error
	// DeleteRole delete the k8s role in Vault.
	DeleteRole(roleName string) error
	// DestroySecret destroy the secret and metadata in given path.
	DestroySecret(path string) error
	// HasSecret returns true if repo secret exists.
	HasSecret(path string) (bool, error)
}
