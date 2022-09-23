package buildabort

import (
	"context"
	"errors"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"golang.org/x/sync/errgroup"
)

type service struct {
	buildStore         core.BuildStore
	taskStore          core.TaskStore
	taskUpdateManager  core.TaskUpdateManager
	buildAbortProducer core.BuildAbortProducer
	logger             lumber.Logger
}

// New returns the BuildAbortService object
func New(buildStore core.BuildStore,
	taskStore core.TaskStore,
	taskUpdateManager core.TaskUpdateManager,
	buildAbortProducer core.BuildAbortProducer,
	logger lumber.Logger) core.BuildAbortService {
	return &service{
		buildStore:         buildStore,
		taskStore:          taskStore,
		taskUpdateManager:  taskUpdateManager,
		buildAbortProducer: buildAbortProducer,
		logger:             logger,
	}
}

func (s *service) AbortBuild(ctx context.Context,
	buildID,
	repoID,
	orgID string) (string, error) {
	if err := s.buildStore.UpdateBuildCache(ctx, buildID, map[string]interface{}{core.AbortedBuild: true}, false); err != nil {
		s.logger.Errorf("error while updating build cache for buildID: %s, error: %v", buildID, err)
		return core.JobCompletedMsg, nil
	}

	g, errCtx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		remark := core.JobAbortedByUser
		if errT := s.taskUpdateManager.UpdateAllTasksForBuild(errCtx, remark, buildID, orgID); errT != nil {
			s.logger.Errorf("error occurred while updating task status to aborted for buildID: %s, error: %v", buildID, errT)
			return errT
		}
		return nil
	})
	// push msg to build-abort topic
	g.Go(func() error {
		if errE := s.buildAbortProducer.Enqueue(buildID, orgID); errE != nil {
			s.logger.Errorf("failed to enqueue message on kafka for buildID: %s, orgID: %s, error: %v", buildID, orgID, errE)
			return errE
		}
		return nil
	})

	if errW := g.Wait(); errW != nil {
		s.logger.Errorf("error occurred while marking job as aborted for orgID %s, repoID %s, buildID %s, error %v",
			orgID, repoID, buildID, errW)
		if errors.Is(errW, errs.ErrRowsNotFound) {
			return "", errs.ErrRowsNotFound
		}
		return "", errs.GenericErrorMessage
	}
	return "", nil
}
