package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gopkg.in/guregu/null.v4/zero"
)

type service struct {
	azureClient        core.AzureBlob
	commitService      core.CommitService
	pullRequestService core.PullRequestService
	tokenHandler       core.GitTokenHandler
	gitStatusService   core.GitStatusService
	coverageManager    core.CoverageManager
	licenseStore       core.LicenseStore
	logger             lumber.Logger
	runner             core.K8sRunner
	taskQueueManager   core.TaskQueueManager
	userStore          core.GitUserStore
	orgStore           core.OrganizationStore
	vaultStore         core.Vault
	hookStore          core.HookStore
	buildStore         core.BuildStore
}

// New creates a new BuildService
func New(
	azureClient core.AzureBlob,
	commitService core.CommitService,
	pullRequestService core.PullRequestService,
	tokenHandler core.GitTokenHandler,
	gitStatusService core.GitStatusService,
	coverageManager core.CoverageManager,
	licenseStore core.LicenseStore,
	logger lumber.Logger,
	runner core.K8sRunner,
	taskQueueManager core.TaskQueueManager,
	userStore core.GitUserStore,
	orgStore core.OrganizationStore,
	vaultStore core.Vault,
	hookStore core.HookStore,
	buildStore core.BuildStore,
) core.BuildService {
	return &service{
		azureClient:        azureClient,
		commitService:      commitService,
		pullRequestService: pullRequestService,
		tokenHandler:       tokenHandler,
		gitStatusService:   gitStatusService,
		coverageManager:    coverageManager,
		licenseStore:       licenseStore,
		logger:             logger,
		runner:             runner,
		taskQueueManager:   taskQueueManager,
		userStore:          userStore,
		orgStore:           orgStore,
		vaultStore:         vaultStore,
		hookStore:          hookStore,
		buildStore:         buildStore,
	}
}

func (s *service) ParseGitEvent(ctx context.Context,
	gitEvent *core.GitEvent,
	repo *core.Repository,
	buildTag core.BuildTag,
) (*core.ParserResponse, error) {
	user, err := s.userStore.FindByID(ctx, repo.Admin)
	if err != nil {
		s.logger.Errorf("failed to find user for id %s, repoID %s, orgID %s, error: %v",
			repo.Admin, repo.ID, repo.OrgID, err)
		return nil, err
	}
	license, err := s.licenseStore.Find(ctx, repo.OrgID)
	if err != nil {
		s.logger.Errorf("error while fetching license, repoID %s, orgID %s, error: %v",
			repo.ID, repo.OrgID, err)
		return nil, err
	}
	runnerType, err := s.orgStore.FindRunnerType(ctx, repo.OrgID)
	if err != nil {
		s.logger.Errorf("error while fetching runner type, repoID %s, orgID %s, error: %v",
			repo.ID, repo.OrgID, err)
		return nil, err
	}
	buildID := utils.GenerateUUID()
	k8namespace := utils.GetRunnerNamespaceFromOrgID(repo.OrgID)

	switch gitEvent.EventName {
	case core.EventPush:
		var v *scm.PushHook
		err := json.Unmarshal(gitEvent.EventPayload, &v)
		if err != nil {
			s.logger.Errorf("could not parse event payload to push hook, repoID %s, orgID %s, err: %v",
				repo.ID, repo.OrgID, err)
			return nil, err
		}
		repo.Namespace = v.Repo.Namespace
		oauthToken, tokenPath, installationTokenPath, secretPath, err := s.fetchTokenAndPaths(ctx,
			repo, user, k8namespace, gitEvent.GitProviderHandle)
		if err != nil {
			return nil, err
		}

		s.logger.Debugf("git push event %+v, repoID %s, orgID %s", v, repo.ID, repo.OrgID)
		return s.createPushHookBuild(ctx,
			gitEvent,
			repo,
			v,
			tokenPath,
			installationTokenPath,
			secretPath,
			buildID,
			license,
			buildTag,
			runnerType,
			oauthToken,
		)
	case core.EventPullRequest:
		var v *scm.PullRequestHook
		err := json.Unmarshal(gitEvent.EventPayload, &v)
		if err != nil {
			s.logger.Errorf("could not parse event payload to pull request hook, repoID %s, orgID %s,  error: %v",
				repo.ID, repo.OrgID, err)
			return nil, err
		}
		s.logger.Debugf("git pull-request event %+v, repoID %s, orgID %s", v, repo.ID, repo.OrgID)
		repo.Namespace = v.Repo.Namespace
		oauthToken, tokenPath, installationTokenPath, secretPath, err := s.fetchTokenAndPaths(ctx,
			repo, user, k8namespace, gitEvent.GitProviderHandle)
		if err != nil {
			return nil, err
		}
		return s.createPullRequestHookBuild(ctx,
			gitEvent,
			repo,
			v,
			tokenPath,
			installationTokenPath,
			secretPath,
			buildID,
			license,
			runnerType,
			oauthToken,
		)
	case core.EventPing:
		return nil, errs.ErrPingEvent
	default:
		return nil, errs.ErrNotSupported
	}
}

func (s *service) CreateBuild(ctx context.Context,
	payload *core.ParserResponse,
	driver core.SCMDriver,
	isNewGitEvent bool,
) (int, error) {
	var err error
	commitID := payload.EventBlob.TargetCommit
	buildID := payload.EventBlob.BuildID
	orgID := payload.Repository.OrgID
	repoID := payload.Repository.ID
	defer func() {
		if err != nil {
			go func() {
				if errC := s.gitStatusService.UpdateGitStatus(context.Background(), driver,
					payload.EventBlob.RepoSlug, buildID, commitID,
					payload.TokenPath, payload.InstallationTokenPath, core.BuildError, payload.BuildTag); errC != nil {
					s.logger.Errorf("error while updating status on git scm for commitID: %s, repoId %s, buildID: %s, orgID: %s, err: %v",
						commitID, repoID, buildID, orgID, errC)
				}
			}()
		}
	}()
	createBuild := true
	if isNewGitEvent {
		createBuild, err = s.hookStore.CreateEntities(ctx, payload)
		if err != nil {
			s.logger.Errorf("failed to created hook entities, commitID: %s repoID %s, orgID %s, err: %v",
				commitID, repoID, orgID, err)
			return http.StatusInternalServerError, errs.GenericUserFacingBEErrRemark
		}
	}
	if !createBuild {
		s.logger.Debugf("skipping build creation for commitID: %s repoID %s, orgID %s, buildTag %s",
			commitID, repoID, orgID, payload.BuildTag)
		return http.StatusNoContent, nil
	}
	payloadAddress, err := s.getPayloadAddress(ctx, payload.EventBlob)
	if err != nil {
		s.logger.Errorf("failed to get payload address, commitID: %s repoID %s, orgID %s, err: %v",
			commitID, repoID, orgID, err)
	}
	if payload.RunnerType == core.CloudRunner {
		runnerOptions := &core.RunnerOptions{NameSpace: utils.GetRunnerNamespaceFromOrgID(orgID),
			PersistentVolumeClaimName: utils.GetBuildHashKey(buildID),
			Vault: &core.VaultOpts{
				TokenSecretName: utils.GetTokenSecretName(buildID),
			}}
		if payload.SecretPath != "" {
			runnerOptions.Vault.RepoSecretName = utils.GetRepoSecretName(buildID)
		}
		// initialize namespace based on license
		if err = s.runner.CreateNamespace(ctx, payload.License); err != nil {
			s.logger.Errorf("error while creating namespace to run buildID %s, orgID %s, error %v",
				payload.EventBlob.BuildID, payload.Repository.OrgID, err)
			return http.StatusInternalServerError, errs.GenericUserFacingBEErrRemark
		}
		if err = s.runner.Init(ctx, runnerOptions, payload); err != nil {
			s.logger.Errorf("failed to initialize  k8s resources for buildID %s, orgID %s, error %v",
				payload.EventBlob.BuildID, payload.Repository.OrgID, err)
			return http.StatusInternalServerError, errs.GenericUserFacingBEErrRemark
		}
		defer func() {
			if err != nil {
				// delete pvc if errors while running queries
				if pvcErr := s.runner.Cleanup(ctx, runnerOptions); pvcErr != nil {
					s.logger.Errorf("failed to delete pvc in k8s for buildID %s, orgID %s, error %v",
						buildID, orgID, pvcErr)
				}
			}
		}()
	}
	discoveryJob, err := s.hookStore.CreateBuild(ctx, payloadAddress, payload)
	if err != nil {
		s.logger.Errorf("error while running queries in transaction for webhook, buildID %s, orgID %s,  error %v",
			buildID, orgID, err)
		return http.StatusInternalServerError, errs.GenericUserFacingBEErrRemark
	}
	if err := s.taskQueueManager.EnqueueTasks(orgID, buildID, discoveryJob); err != nil {
		s.logger.Errorf("failed to enqueue tasks in queue buildID %s, orgID %s, repoID: %s, error: %v",
			buildID, payload.Repository.ID, orgID, err)
		return http.StatusInternalServerError, errs.GenericUserFacingBEErrRemark
	}

	return http.StatusNoContent, nil
}
func (s *service) fetchTokenAndPaths(ctx context.Context,
	repo *core.Repository,
	user *core.GitUser,
	k8namespace string,
	gitProvider core.SCMDriver,
) (oauthToken *core.Token, tokenPath, installationTokenPath, secretPath string, err error) {
	tokenPath = s.vaultStore.GetTokenPath(gitProvider, user.Mask, user.ID)
	installationTokenPath = s.vaultStore.GetInstallationTokenPath(gitProvider, k8namespace, repo.Namespace)

	g, errCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		oauthToken, _, err = s.tokenHandler.GetTokenFromPath(errCtx, gitProvider, tokenPath, installationTokenPath)
		if err != nil {
			s.logger.Errorf("failed to get clone token for path: %s, repoID %s, orgID %s, error: %v", tokenPath,
				repo.ID, repo.OrgID, err)
			return err
		}
		return nil
	})

	secretPath = s.vaultStore.GetSecretPath(gitProvider, k8namespace, repo.Namespace, repo.Mask, repo.ID)
	g.Go(func() error {
		ok, err := s.vaultStore.HasSecret(secretPath)
		if err != nil {
			if !errors.Is(err, errs.ErrSecretNotFound) {
				s.logger.Errorf("failed to find secrets repoID %s, orgID %s, error: %v",
					repo.ID, repo.OrgID, err)
				return err
			}
		}
		if !ok {
			// set secretPath to empty when repo secrets are empty
			secretPath = ""
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, "", "", "", err
	}
	return oauthToken, tokenPath, installationTokenPath, secretPath, nil
}

func (s *service) createPushHookBuild(ctx context.Context,
	gitEvent *core.GitEvent,
	repo *core.Repository,
	v *scm.PushHook,
	tokenPath,
	installationTokenPath,
	secretPath,
	buildID string,
	license *core.License,
	buildTag core.BuildTag,
	runnerType core.Runner,
	oauthToken *core.Token,
) (*core.ParserResponse, error) {
	commits := make([]*core.GitCommit, 0, len(v.Commits))
	parentCommitID := v.Before
	for idx := range v.Commits {
		c := v.Commits[idx]
		commitTime := c.Committer.Date
		if commitTime.IsZero() {
			commitTime = time.Now()
		}
		author, committer := s.getAuthorForPushHook(gitEvent.GitProviderHandle, v, &c)
		gitCommit := &core.GitCommit{ID: utils.GenerateUUID(),
			CommitID:       c.Sha,
			ParentCommitID: parentCommitID,
			Author:         author,
			Committer:      committer,
			RepoID:         repo.ID,
			Message:        c.Message,
			Link:           c.Link,
			Updated:        time.Now(),
			Created:        time.Now()}
		commits = append(commits, gitCommit)
		parentCommitID = gitCommit.CommitID
	}
	eventBody, err := json.Marshal(v)
	if err != nil {
		s.logger.Errorf("failed to marshal git event repoID %s, orgID %s, error: %v",
			repo.ID, repo.OrgID, err)
		return nil, errs.ErrMarshalJSON
	}
	rawTokenBody, err := json.Marshal(oauthToken)
	if err != nil {
		s.logger.Errorf("failed to marshal git event buildID %s, orgID %s, error: %v",
			buildID, repo.OrgID, err)
		return nil, errs.ErrMarshalJSON
	}

	branch := scm.TrimRef(v.Ref)
	baseCommit, rebuild, err := s.findBaseCommitForPostMerge(ctx, repo, branch, v.Commit.Sha, buildID)
	if err != nil {
		s.logger.Errorf("failed to find basecommit for orgID %s, repoID %s, buildID %s commitID %s, error %v",
			repo.OrgID, repo.ID, buildID, v.Commit.Sha, err)
		return nil, err
	}
	eventBlob := &core.EventBlob{EventID: gitEvent.ID,
		BuildID:         buildID,
		RepoID:          repo.ID,
		OrgID:           repo.OrgID,
		RepoSlug:        scm.Join(repo.Namespace, repo.Name),
		RepoLink:        v.Repo.Link,
		EventType:       core.EventPush,
		BaseCommit:      baseCommit,
		TargetCommit:    v.Commit.Sha,
		GitProvider:     gitEvent.GitProviderHandle,
		PrivateRepo:     repo.Private,
		TasFileName:     repo.TasFileName,
		EventBody:       eventBody,
		Commits:         v.Commits,
		BranchName:      branch,
		LicenseTier:     license.Tier,
		CollectCoverage: repo.CollectCoverage,
		RawToken:        rawTokenBody,
	}
	return s.parserResponseBuilder(repo, license,
		eventBlob, commits, tokenPath, installationTokenPath,
		secretPath, rebuild, buildTag, runnerType)
}

func (s *service) parserResponseBuilder(
	repo *core.Repository,
	license *core.License,
	eventBlob *core.EventBlob,
	commits []*core.GitCommit,
	tokenPath,
	installationTokenPath,
	secretPath string,
	rebuild bool,
	buildTag core.BuildTag,
	runnerType core.Runner) (*core.ParserResponse, error) {
	return &core.ParserResponse{
		Repository:            repo,
		EventBlob:             eventBlob,
		GitCommits:            commits,
		TokenPath:             tokenPath,
		InstallationTokenPath: installationTokenPath,
		SecretPath:            secretPath,
		License:               license,
		Rebuild:               rebuild,
		BuildTag:              buildTag,
		RunnerType:            runnerType,
	}, nil
}

func (s *service) getPayloadAddress(ctx context.Context, eventBlob *core.EventBlob) (string, error) {
	payloadBytes, err := json.Marshal(eventBlob)
	if err != nil {
		s.logger.Errorf("failed to marshal payload, buildID %s, orgID %s, repoID %s, error: %v",
			eventBlob.BuildID, eventBlob.OrgID, eventBlob.RepoID, err)
		return "", errs.ErrMarshalJSON
	}
	payloadAddress, err := s.azureClient.UploadBytes(ctx, fmt.Sprintf("%s/%s/build/%s", eventBlob.OrgID, eventBlob.RepoID, eventBlob.BuildID),
		payloadBytes, core.PayloadContainer, gin.MIMEJSON)
	if err != nil {
		s.logger.Errorf("failed to upload blob to azure storage for buildID %s, orgID %s, repoID %s, error: %v",
			eventBlob.BuildID, eventBlob.OrgID, eventBlob.RepoID, err)
		return "", errs.ErrAzureUpload
	}
	return s.azureClient.ReplaceWithCDN(payloadAddress), nil
}

func (s *service) createPullRequestHookBuild(
	ctx context.Context,
	gitEvent *core.GitEvent,
	repo *core.Repository,
	v *scm.PullRequestHook,
	tokenPath,
	installationTokenPath,
	secretPath,
	buildID string,
	license *core.License,
	runnerType core.Runner,
	oauthToken *core.Token,
) (*core.ParserResponse, error) {
	commits := make([]*core.GitCommit, 0, 1)

	baseCommit, branchName, targetCommit, err := s.getPullRequestCommit(ctx, gitEvent, repo, v, oauthToken)
	if err != nil {
		s.logger.Errorf("Failed to get pull request commit details for repoID %v, buildID %v, orgID %v err: %v",
			repo.ID, buildID, repo.OrgID, err)
		return nil, err
	}
	c, err := s.fetchCommitDetails(ctx,
		gitEvent.GitProviderHandle,
		oauthToken,
		repo,
		v.PullRequest.Fork,
		targetCommit)

	if err != nil {
		return nil, err
	}
	repoLink := v.Repo.Link
	targetSCMCommit := &scm.Commit{
		Sha:     c.CommitID,
		Link:    c.Link,
		Message: c.Message,
		Author:  scm.Signature{Name: c.Author.Name, Email: c.Author.Email, Date: c.Author.Date},
	}

	// added ifs for fail safe
	if gitEvent.GitProviderHandle == core.DriverGitlab {
		if v.PullRequest.Author.Login != "" {
			c.Author.Name = v.PullRequest.Author.Login
		}
		// don' t have committer username for gitlab
	}
	gitCommit := &core.GitCommit{ID: utils.GenerateUUID(),
		ParentCommitID: baseCommit, // temporarily we assume base commit as parent
		CommitID:       c.CommitID,
		Author:         c.Author,
		Committer:      c.Committer,
		RepoID:         repo.ID,
		Message:        c.Message,
		Link:           c.Link,
		Updated:        time.Now(),
		Created:        time.Now(),
	}
	commits = append(commits, gitCommit)
	eventBody, err := json.Marshal(v)
	if err != nil {
		s.logger.Errorf("failed to marshal git event buildID %s, orgID %s, error: %v",
			buildID, repo.OrgID, err)
		return nil, errs.ErrMarshalJSON
	}
	rawTokenBody, err := json.Marshal(oauthToken)
	if err != nil {
		s.logger.Errorf("failed to marshal oauth token buildID %s, orgID %s, error: %v",
			buildID, repo.OrgID, err)
		return nil, errs.ErrMarshalJSON
	}
	eventBlob := &core.EventBlob{
		EventID:           gitEvent.ID,
		BuildID:           buildID,
		RepoID:            repo.ID,
		OrgID:             repo.OrgID,
		RepoLink:          repoLink,
		RepoSlug:          scm.Join(repo.Namespace, repo.Name),
		ForkSlug:          v.PullRequest.Fork,
		EventType:         core.EventPullRequest,
		BaseCommit:        baseCommit,
		TargetCommit:      targetCommit,
		GitProvider:       gitEvent.GitProviderHandle,
		PrivateRepo:       repo.Private,
		TasFileName:       repo.TasFileName,
		EventBody:         eventBody,
		DiffURL:           v.PullRequest.Diff,
		PullRequestNumber: v.PullRequest.Number,
		Commits:           []scm.Commit{*targetSCMCommit},
		BranchName:        branchName,
		LicenseTier:       license.Tier,
		CollectCoverage:   repo.CollectCoverage,
		RawToken:          rawTokenBody,
	}
	return s.parserResponseBuilder(repo, license, eventBlob, commits,
		tokenPath, installationTokenPath, secretPath,
		false, core.PreMergeTag, runnerType)
}

func (s *service) findBaseCommitForPostMerge(
	ctx context.Context,
	repo *core.Repository,
	branch string,
	commitID string,
	buildID string) (baseCommit string, rebuild bool, err error) {
	rebuild = true
	// check if commit is being rebuild
	baseCommit, err = s.buildStore.FindBaseBuildCommitForRebuildPostMerge(ctx, repo.ID, branch, commitID)
	if err != nil {
		if !errors.Is(err, errs.ErrRowsNotFound) {
			s.logger.Errorf("failed to find rebuild basecommit for orgID %s, repoID %s, buildID %s commitID %s, branch %s, error %v",
				repo.OrgID, repo.ID, buildID, commitID, branch, err)
			return "", false, err
		}
		rebuild = false
	}
	if !rebuild {
		baseCommit, err = s.buildStore.FindLastBuildCommitSha(ctx, repo.ID, branch)
		if err != nil {
			if !errors.Is(err, errs.ErrRowsNotFound) {
				s.logger.Errorf("failed to find basecommit for orgID %s, repoID %s, buildID %s, branch %s, error %v",
					repo.OrgID, repo.ID, buildID, branch, err)
				return "", false, err
			}
			baseCommit = ""
			s.logger.Debugf("failed to find basecommit for orgID %s, repoID %s, buildID %s, error %v",
				repo.OrgID, repo.ID, buildID, err)
		}
	}

	return baseCommit, rebuild, nil
}
func (s *service) fetchCommitDetails(
	ctx context.Context,
	driver core.SCMDriver,
	oauthToken *core.Token,
	repo *core.Repository,
	forkSlug string,
	commitID string,
) (c *core.GitCommit, err error) {
	repoSlug := scm.Join(repo.Namespace, repo.Name)
	if driver == core.DriverBitbucket && forkSlug != "" {
		repoSlug = forkSlug
	}
	if c, err = s.commitService.Find(ctx, driver, oauthToken, repoSlug, commitID); err != nil {
		s.logger.Errorf("error while fetching commit details from git scm, repoID %s, orgID %s, error: %v",
			repo.ID, repo.OrgID, err)
		return nil, errs.GenericErrorMessage
	}
	return c, nil
}

func (s *service) getPullRequestCommit(ctx context.Context, gitEvent *core.GitEvent, repo *core.Repository,
	v *scm.PullRequestHook, oauthToken *core.Token) (baseCommit, branchName, targetCommit string, err error) {
	switch gitEvent.GitProviderHandle {
	case core.DriverGithub:
		baseCommit = v.PullRequest.Base.Sha
		branchName = v.PullRequest.Head.Name
		targetCommit = v.PullRequest.Sha

	case core.DriverGitlab:
		pullrequest, err := s.pullRequestService.Find(ctx, gitEvent.GitProviderHandle, oauthToken,
			scm.Join(repo.Namespace, repo.Name), v.PullRequest.Number)
		if err != nil {
			s.logger.Debugf("failed to find pullrequest object for orgID %v repoID %v gitprovider %v err %v",
				repo.OrgID, repo.ID, gitEvent.GitProviderHandle, err)
			return baseCommit, branchName, targetCommit, err
		}
		baseCommit = pullrequest.Base.Sha
		branchName = v.PullRequest.Source
		targetCommit = v.PullRequest.Sha

	case core.DriverBitbucket:
		pullrequest, err := s.pullRequestService.Find(ctx, gitEvent.GitProviderHandle, oauthToken,
			scm.Join(repo.Namespace, repo.Name), v.PullRequest.Number)
		if err != nil {
			s.logger.Debugf("failed to find pullrequest object for orgID %v repoID %v gitprovider %v err %v",
				repo.OrgID, repo.ID, gitEvent.GitProviderHandle, err)
			return baseCommit, branchName, targetCommit, err
		}

		baseCommitObj, err := s.commitService.Find(ctx,
			gitEvent.GitProviderHandle,
			oauthToken,
			scm.Join(repo.Namespace, repo.Name),
			pullrequest.Base.Sha)
		if err != nil {
			s.logger.Debugf("failed to find commit object for commitSha %v repoID %v gitprovider %v err %v",
				pullrequest.Base.Sha, repo.ID, gitEvent.GitProviderHandle, err)
			return baseCommit, branchName, targetCommit, err
		}

		targetCommitRepo := scm.Join(repo.Namespace, repo.Name)
		if v.PullRequest.Fork != "" {
			targetCommitRepo = v.PullRequest.Fork
		}

		tagerCommitObj, err := s.commitService.Find(ctx,
			gitEvent.GitProviderHandle,
			oauthToken,
			targetCommitRepo,
			v.PullRequest.Sha)
		if err != nil {
			s.logger.Debugf("failed to find commit object for commitSha %v repoID %v gitprovider %v err %v",
				pullrequest.Base.Sha, repo.ID, gitEvent.GitProviderHandle, err)
			return baseCommit, branchName, targetCommit, err
		}
		baseCommit = baseCommitObj.CommitID
		branchName = v.PullRequest.Source
		targetCommit = tagerCommitObj.CommitID

	default:
		return baseCommit, branchName, targetCommit, errs.ErrUnsupportedGitProvider
	}

	return baseCommit, branchName, targetCommit, nil
}

func (s *service) getAuthorForPushHook(gitProvider core.SCMDriver, v *scm.PushHook, commit *scm.Commit) (core.Author, core.Committer) {
	author := core.Author{Name: utils.GetAuthorName(&commit.Author), Email: commit.Author.Email, Date: commit.Author.Date}
	committer := core.Committer{
		Name:  zero.StringFrom(commit.Committer.Name),
		Email: zero.StringFrom(commit.Committer.Email),
		Date:  zero.TimeFrom(commit.Committer.Date)}

	switch gitProvider {
	case core.DriverGitlab:
		if v.Commit.Author.Login != "" {
			author.Name = v.Commit.Author.Login
		}
		if v.Commit.Committer.Login != "" {
			committer.Name = zero.StringFrom(v.Commit.Committer.Login)
		}

	case core.DriverBitbucket:
		author.Name = author.Email
		committer.Name = committer.Email

	case core.DriverGithub:
	}

	return author, committer
}
