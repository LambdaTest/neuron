package core

import (
	"context"
)

// UserSecretType represents the type of opaque k8s secret
type UserSecretType string

// supported secret types
const (
	Oauth      UserSecretType = "oauth"
	RepoSecret UserSecretType = "reposecrets"
)

// K8sRunner is the interface that must be implemented by an execution engine.
type K8sRunner interface {

	// Init the resources required for the execution engine.
	Init(ctx context.Context, r *RunnerOptions, payload *ParserResponse) error

	// Run runs the execution engine.
	Run(ctx context.Context, r *RunnerOptions) error

	// Destroy the execution engine.
	Destroy(ctx context.Context, r *RunnerOptions) error

	// Cleanup the resources required for the execution engine.
	Cleanup(ctx context.Context, r *RunnerOptions) error

	// CreateNamespace create a new k8 namespace
	CreateNamespace(ctx context.Context, l *License) error

	// CreateSecret creates a new k8s secret.
	CreateSecret(ctx context.Context, namespace, secretName string, secretType UserSecretType, data []byte) error

	// DeleteSecret deletes a k8s secret.
	DeleteSecret(ctx context.Context, namespace, secretName string) error

	// UpdateSecret updates a k8s secret.
	UpdateSecret(ctx context.Context, namespace, secretName string, secretKey UserSecretType, data []byte) error

	// CreatePersistentVolumeClaim create the persistent volume claim.
	CreatePersistentVolumeClaim(ctx context.Context, r *RunnerOptions, license *License) error

	// DeletePersistentVolumeClaim delete the persistent volume claim.
	DeletePersistentVolumeClaim(ctx context.Context, r *RunnerOptions) error

	// Spawn creates and runs the pod in k8s.
	Spawn(ctx context.Context, runnerOptions *RunnerOptions, buildID, taskID string) <-chan error

	// WaitForPodTermination waits for pods to get terminated before spawning next one.
	WaitForPodTermination(ctx context.Context, podName, podNamespace string) error
}

// RunnerOptions provides the the required instructions for execution engine.
type RunnerOptions struct {
	DockerImage               string            `json:"docker_image"`
	ContainerPort             int               `json:"container_port"`
	SecretName                string            `json:"-"`
	HostPort                  int               `json:"host_port"`
	Label                     map[string]string `json:"label"`
	NameSpace                 string            `json:"name_space"`
	PodName                   string            `json:"pod_name"`
	ContainerName             string            `json:"container_name"`
	ContainerArgs             []string          `json:"container_args"`
	ContainerCommands         []string          `json:"container_commands"`
	PersistentVolumeClaimName string            `json:"-"`
	Env                       []string          `json:"env"`
	OrgID                     string            `json:"org_id"`
	Vault                     *VaultOpts        `json:"-"`
	LogfilePath               string            `json:"logfile_path"`
	PodType                   PodType           `json:"pod_type"`
	Tier                      Tier              `json:"tier"`
}

// VaultOpts provides the vault path options
type VaultOpts struct {
	// SecretPath path of the repo secrets.
	SecretPath string
	// TokenPath path of the user token.
	TokenPath string
	// RoleName vault role name
	RoleName string
	// Namespace is the default vault namespace
	Namespace string
	// RepoSecretName the name of the k8s resource for the user's repository secrets
	RepoSecretName string
	// TokenSecretName the name of the k8s resource for the user's token
	TokenSecretName string
}

// PodType specifies the type of pod
type PodType string

// Values that PodType can take
const (
	NucleusPod  PodType = "nucleus"
	CoveragePod PodType = "coverage"
)
