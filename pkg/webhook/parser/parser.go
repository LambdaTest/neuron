package parser

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
)

type parser struct {
	logger       lumber.Logger
	repoStore    core.RepoStore
	buildService core.BuildService
}

// New returns a new HookParser.
func New(
	repoStore core.RepoStore,
	buildService core.BuildService,
	logger lumber.Logger,
) core.HookParser {
	return &parser{
		repoStore:    repoStore,
		buildService: buildService,
		logger:       logger}
}

func (p *parser) CreateAndScheduleBuild(eventType core.EventType, driver core.SCMDriver, repoID string, msg []byte) error {
	eventID := utils.GenerateUUID()
	repo, err := p.repoStore.FindByID(context.Background(), repoID)
	if err != nil {
		p.logger.Errorf("failed to find repoID %s, error %v", repoID, err)
		return err
	}
	event := &core.GitEvent{
		ID:                eventID,
		RepoID:            repo.ID,
		EventName:         eventType,
		GitProviderHandle: driver,
		EventPayload:      msg,
	}
	buildTag := core.PreMergeTag
	if eventType == core.EventPush {
		buildTag = core.PostMergeTag
	}
	payload, err := p.buildService.ParseGitEvent(context.Background(), event, repo, buildTag)
	if err != nil {
		p.logger.Errorf("failed to parse webhook, repoID %s, orgID %s, error %v", repoID, repo.OrgID, err)
		return err
	}

	_, err = p.buildService.CreateBuild(context.Background(), payload, driver, true)
	if err != nil {
		p.logger.Errorf("failed to create build for repoID %s, orgID %s, error %v", repoID, repo.OrgID, err)
		return err
	}
	return nil
}
