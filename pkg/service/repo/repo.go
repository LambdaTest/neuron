package repo

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
)

const (
	requestsLimit = 10
	repoLimit     = 100
)

type service struct {
	scmProvider core.SCMProvider
	repoStore   core.RepoStore
	logger      lumber.Logger
}

// New returns a new Repository service, providing access to the
// repository information from the source code management system.
func New(scmProvider core.SCMProvider, repoStore core.RepoStore, logger lumber.Logger) core.RepositoryService {
	return &service{
		scmProvider: scmProvider,
		repoStore:   repoStore,
		logger:      logger,
	}
}

func (s *service) Find(ctx context.Context, repoSlug string, token *core.Token, gitProvider core.SCMDriver) (*core.Repository, error) {
	gitSCM, err := s.scmProvider.GetClient(gitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", gitProvider, err)
		return nil, err
	}
	ctx = token.SetRequestContext(ctx)
	src, res, err := gitSCM.Client.Repositories.Find(ctx, repoSlug)
	if err != nil {
		s.logger.Errorf("failed to find repository %s for driver: %s, error: %v", repoSlug, gitProvider, err)
		if res != nil && res.Status == http.StatusNotFound {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return convertRepository(src), nil
}

func (s *service) List(ctx context.Context,
	page,
	size int,
	userID,
	orgID,
	orgName string,
	user *core.GitUser) ([]*core.Repository, int, error) {
	gitSCM, err := s.scmProvider.GetClient(user.GitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", user.GitProvider, err)
		return nil, 0, err
	}
	ctx = user.Oauth.SetRequestContext(ctx)
	wg := new(sync.WaitGroup)
	var m sync.Mutex
	var results []*scm.Repository
	errChan := make(chan error, requestsLimit)
	nextPageCheck := 0
	page = utils.Max(page, 1)
	nextPage := page
	size = utils.Min(size, repoLimit)

	wg.Add(requestsLimit)
	for i := 0; i < requestsLimit; i++ {
		go func(i int) {
			defer wg.Done()
			opts := scm.ListOptions{Size: size, Page: page + i}
			var result []*scm.Repository
			var meta *scm.Response
			var berr error

			switch user.GitProvider {
			case core.DriverBitbucket:
				// Second argument is orgSlug in case of bitbucket
				result, meta, berr = gitSCM.Client.Repositories.List2(ctx, orgName, opts)
			case core.DriverGithub:
				// Second argument is InstallationID in case of github
				result, meta, berr = gitSCM.Client.Repositories.List2(ctx, user.Oauth.InstallationID, opts)
			case core.DriverGitlab:
				result, meta, berr = gitSCM.Client.Repositories.List(ctx, opts)
			}

			errChan <- berr
			if berr == nil {
				m.Lock()
				results = append(results, result...)
				if meta.Page.Next == 0 {
					nextPageCheck = 1
				}
				nextPage = utils.Max(meta.Page.Next, nextPage)
				m.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if nextPageCheck == 1 {
		nextPage = 0
	}
	repos := make([]*core.Repository, 0, len(results))

	if cerr := processChannel(ctx, errChan, requestsLimit); cerr != nil {
		s.logger.Errorf("error in fetching repo list, error: %v", cerr)
		return nil, 0, cerr
	}
	repoActiveMap, err := s.repoStore.FindAllActiveMap(ctx, orgID)
	if err != nil && !errors.Is(err, errs.ErrRowsNotFound) {
		s.logger.Errorf("failed to find active repos %v", err)
		return nil, 0, err
	}

	for _, src := range results {
		r := convertRepository(src)
		if r.Namespace == orgName {
			if _, ok := repoActiveMap[r.Name]; ok {
				r.Active = true
			}
			repos = append(repos, r)
		}
	}
	// add next page in response
	return repos, nextPage, nil
}

func convertRepository(src *scm.Repository) *core.Repository {
	return &core.Repository{
		Namespace: src.Namespace,
		Name:      src.Name,
		HTTPURL:   src.Clone,
		SSHURL:    src.CloneSSH,
		Link:      src.Link,
		Private:   src.Private,
		Perm: &core.Perm{
			Write: src.Perm.Push,
			Read:  src.Perm.Pull,
			Admin: src.Perm.Admin,
		},
	}
}

func processChannel(ctx context.Context, errChan chan error, requestsLimit int) error {
Loop:
	for i := 0; i < requestsLimit; i++ {
		select {
		case cerr := <-errChan:
			if cerr != nil {
				return cerr
			}
		case <-ctx.Done():
			return ctx.Err()
		default:
			break Loop
		}
	}
	return nil
}

// ListBranches fetch the list of branch for a particular repo
func (s *service) ListBranches(ctx context.Context, page, size int, repoName string,
	user *core.GitUser) (branches []string, next int, err error) {
	branches = make([]string, 0)
	gitSCM, err := s.scmProvider.GetClient(user.GitProvider)
	if err != nil {
		s.logger.Errorf("failed to find git client for driver: %s, error: %v", user.GitProvider, err)
		return nil, 0, err
	}
	ctx = user.Oauth.SetRequestContext(ctx)
	opts := scm.ListOptions{Size: size, Page: page}

	branch, meta, berr := gitSCM.Client.Git.ListBranches(ctx, repoName, opts)
	if berr != nil {
		return nil, 0, berr
	}
	for i := range branch {
		branches = append(branches, branch[i].Name)
	}
	return branches, meta.Page.Next, nil
}
