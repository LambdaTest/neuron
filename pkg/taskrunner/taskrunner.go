package taskrunner

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

type taskRunner struct {
	cfg                       *config.Config
	logger                    lumber.Logger
	runner                    core.K8sRunner
	orgStore                  core.OrganizationStore
	taskStore                 core.TaskStore
	synapseTaskQueue          core.SynapseTaskQueue
	nucleusPrivateDockerImage string
	nucleusPublicDockerImage  string
}

// New returns instance of core.TaskRunner
func New(
	cfg *config.Config,
	logger lumber.Logger,
	runner core.K8sRunner,
	orgStore core.OrganizationStore,
	taskStore core.TaskStore,
	synapseTaskQueue core.SynapseTaskQueue) core.TaskRunner {
	nucleusPrivateDockerImage, ok := constants.NucleusDockerImage[core.CloudRunner][cfg.Env]
	if !ok {
		logger.Fatalf("Invalid Environment")
	}
	nucleusPublicDockerImage, ok := constants.NucleusDockerImage[core.SelfHosted][cfg.Env]
	if !ok {
		logger.Fatalf("Invalid Environment")
	}
	return &taskRunner{
		logger:                    logger,
		runner:                    runner,
		orgStore:                  orgStore,
		taskStore:                 taskStore,
		synapseTaskQueue:          synapseTaskQueue,
		nucleusPublicDockerImage:  nucleusPublicDockerImage,
		nucleusPrivateDockerImage: nucleusPrivateDockerImage,
		cfg:                       cfg,
	}
}

func (t *taskRunner) ScheduleTask(
	ctx context.Context,
	r *core.RunnerOptions,
	buildID,
	jobID,
	taskID,
	customDockerImage string,
) <-chan error {
	errChan := make(chan error, 1)
	org, err := t.orgStore.FindByID(ctx, r.OrgID)
	if err != nil {
		t.logger.Errorf("error fetching runner type for orgID %s buildID %s jobID %s, error: %v",
			r.OrgID, buildID, jobID, err)
		errChan <- err
		return errChan
	}
	// overwrite the container image if custom image is provided in case of self-hosted runners
	t.setContainerImage(r, customDockerImage, org.RunnerType)
	t.logger.Debugf("jobID %s will be scheduled on %s for orgID %s buildID %s", jobID, org.RunnerType,
		org.ID, buildID)

	if org.RunnerType == core.SelfHosted {
		go func() {
			errChan <- <-t.synapseTaskQueue.Enqueue(r, time.Now())
		}()
	} else {
		go func() {
			errChan <- <-t.runner.Spawn(ctx, r, buildID, taskID)
		}()
	}
	return errChan
}

func (t *taskRunner) setContainerImage(r *core.RunnerOptions, customImage string, runnerType core.Runner) {
	// if not self hosted, set the runner image to the private nucleus image
	if runnerType != core.SelfHosted {
		r.DockerImage = t.nucleusPrivateDockerImage
		return
	}
	if r.PodType == core.NucleusPod {
		containerImage := t.nucleusPublicDockerImage
		if customImage != "" {
			containerImage = customImage
		}
		r.DockerImage = containerImage
		return
	}
	r.DockerImage = t.nucleusPublicDockerImage
}
