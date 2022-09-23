package core

import (
	"encoding/json"

	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/drone/go-scm/scm"
)

// SCMProvider represents new git scm provider
type SCMProvider interface {
	GetClient(scmClientID SCMDriver) (*SCM, error)
}

// SCM is wrapper around scm.Client
type SCM struct {
	Client     *scm.Client
	SelfHosted bool
	Name       string
}

// SCMDriver identifies source code management driver.
type SCMDriver string

// SCMDriver values.
const (
	DriverGithub    SCMDriver = "github"
	DriverGitlab    SCMDriver = "gitlab"
	DriverBitbucket SCMDriver = "bitbucket"
	// DriverGithubSelfHosted    SCMDriver = "github_self_hosted"
	// DriverGitlabSelfHosted    SCMDriver = "gitlab_self_hosted"
	// DriverBitbucketSelfHosted SCMDriver = "bitbucket_self_hosted"
)

// VerifyDriver verifies if the SCMDriver is valid.
func (d SCMDriver) VerifyDriver() error {
	switch d {
	case DriverGithub:
	// case DriverGithubSelfHosted:
	case DriverGitlab:
	// case DriverGitlabSelfHosted:
	case DriverBitbucket:
	default:
		return errs.ErrInvalidDriver
	}
	return nil
}

// VerifyDriver verifies if the SCMDriver is valid.
func (d SCMDriver) String() string {
	return string(d)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (d SCMDriver) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (d *SCMDriver) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, d)
}
