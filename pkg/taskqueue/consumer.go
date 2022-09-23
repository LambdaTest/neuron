package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
	"github.com/segmentio/kafka-go"
	"gopkg.in/guregu/null.v4/zero"
)

var noContext = context.Background()

type consumer struct {
	cfg               *config.Config
	wg                *sync.WaitGroup
	internalJWT       core.Session
	topicName         string
	reader            *kafka.Reader
	logger            lumber.Logger
	startTime         time.Time
	taskStore         core.TaskStore
	taskUpdateManager core.TaskUpdateManager
	taskQueueStore    core.TaskQueueStore
	redisDB           core.RedisDB
	runner            core.K8sRunner
	buildMonitor      core.BuildMonitor
	buildStore        core.BuildStore
	taskQueueManager  core.TaskQueueManager
	orgStore          core.OrganizationStore
	taskQueueUtils    core.TaskQueueUtils
	taskRunner        core.TaskRunner
	tokenHandler      core.GitTokenHandler
	repoStore         core.RepoStore
}

// NewConsumer return a new taskqueue consumer.
func NewConsumer(cfg *config.Config,
	wg *sync.WaitGroup,
	internalJWT core.Session,
	taskStore core.TaskStore,
	taskUpdateManager core.TaskUpdateManager,
	buildStore core.BuildStore,
	buildMonitor core.BuildMonitor,
	taskQueueStore core.TaskQueueStore,
	taskQueueManager core.TaskQueueManager,
	redisDB core.RedisDB,
	tokenHandler core.GitTokenHandler,
	runner core.K8sRunner,
	gitStatusService core.GitStatusService,
	orgStore core.OrganizationStore,
	taskQueueUtils core.TaskQueueUtils,
	repoStore core.RepoStore,
	taskRunner core.TaskRunner,
	logger lumber.Logger) core.QueueConsumer {
	// offset retention time is 24h
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               strings.Split(cfg.Kafka.Brokers, ","),
		Topic:                 cfg.Kafka.TaskQueueConfig.Topic,
		ErrorLogger:           kafka.LoggerFunc(logger.Errorf),
		GroupID:               cfg.Kafka.TaskQueueConfig.ConsumerGroup,
		WatchPartitionChanges: true,
		GroupBalancers:        []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}}})
	logger.Infof("Kafka Consumer Group %s created successfully", cfg.Kafka.TaskQueueConfig.ConsumerGroup)
	return &consumer{
		cfg:               cfg,
		internalJWT:       internalJWT,
		redisDB:           redisDB,
		taskStore:         taskStore,
		taskUpdateManager: taskUpdateManager,
		topicName:         cfg.Kafka.TaskQueueConfig.Topic,
		logger:            logger,
		buildMonitor:      buildMonitor,
		startTime:         time.Now(),
		reader:            reader,
		runner:            runner,
		tokenHandler:      tokenHandler,
		buildStore:        buildStore,
		taskQueueManager:  taskQueueManager,
		taskQueueStore:    taskQueueStore,
		orgStore:          orgStore,
		taskQueueUtils:    taskQueueUtils,
		repoStore:         repoStore,
		taskRunner:        taskRunner,
		wg:                wg,
	}
}

// nolint:funlen
func (c *consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			c.logger.Errorf("Kafka ReadMessage of topic: %v failed: %v", c.topicName, err)
			continue
		}
		c.logger.Debugf("Kafka: Message received on partition: %d, offset: %d, topic: %s", msg.Partition, msg.Offset, msg.Topic)
		c.wg.Add(1)
		// perform tasks in go routine
		go func(msg kafka.Message) {
			defer c.wg.Done()
			var job core.Job
			if err := json.Unmarshal(msg.Value, &job); err != nil {
				c.logger.Errorf("failed to unmarshal job payload, error: %v, message %s", err, string(msg.Value))
				return
			}
			c.logger.Debugf("initiating job %s for taskID %s, orgID %s", job.ID, job.TaskID, job.OrgID)
			task, err := c.taskStore.Find(noContext, job.TaskID)
			if err != nil {
				c.logger.Errorf("failed to find task details in database for orgID %s, taskID %s, error: %v",
					job.OrgID, job.TaskID, err)
				return
			}
			if utils.TaskFinished(task.Status) {
				c.logger.Debugf("task with taskID: %s, buildID: %s, status: %s, has been completed, aborting further processing for it",
					task.ID, task.BuildID, task.Status)
				return
			}
			build, err := c.buildStore.GetBuildCache(noContext, task.BuildID)
			if err != nil {
				c.logger.Errorf("failed to find build details in redis for orgID %s, taskID  %s, buildID %s, error: %v",
					job.OrgID, task.ID, task.BuildID, err)
				task.Remark = zero.StringFrom(errs.GenericUserFacingBEErrRemark.Error())
				c.taskQueueUtils.MarkTaskToStatus(task, job.OrgID, core.TaskError)
				return
			}
			if build.Aborted {
				c.markTaskAborted(ctx, job.OrgID, task)
				return
			}
			if err = c.updateSecret(ctx, build, job.OrgID, task.BuildID); err != nil {
				c.logger.Errorf("failed to update secret for orgID %s, taskID  %s, buildID %s, error: %v",
					job.OrgID, task.ID, task.BuildID, err)
				task.Remark = zero.StringFrom(errs.GenericUserFacingBEErrRemark.Error())
				c.taskQueueUtils.MarkTaskToStatus(task, job.OrgID, core.TaskError)
				return
			}
			r, err := c.getRunnerOptions(build, &job, task)
			if err != nil {
				task.Remark = zero.StringFrom(errs.GenericUserFacingBEErrRemark.Error())
				c.taskQueueUtils.MarkTaskToStatus(task, job.OrgID, core.TaskError)
				return
			}
			runnerErr := <-c.taskRunner.ScheduleTask(noContext, r, task.BuildID, job.ID, task.ID, task.ContainerImage)
			if runnerErr != nil {
				// incase of nucleus crash mark task as stopped and schedule the next task
				c.logger.Errorf("error while spawning nucleus pod, orgID %s, taskID  %s, buildID %s, error: %v",
					job.OrgID, task.ID, task.BuildID, runnerErr)
				var status core.Status
				task.Remark = getTaskErrorRemark(runnerErr)
				status, err = c.getTaskStatus(ctx, task.BuildID, runnerErr)
				if err != nil {
					c.logger.Errorf("failed to get task status for buildID: %s, taskID, error: %s", task.BuildID, task.ID, err)
				}
				c.taskQueueUtils.MarkTaskToStatus(task, job.OrgID, status)
				c.wg.Add(1)
				go func(orgID string) {
					defer c.wg.Done()
					if err := c.taskQueueManager.DequeueTasks(orgID); err != nil {
						c.logger.Errorf("error while dequeueing tasks from queue for orgID %s, error: %v", orgID, err)
					}
				}(job.OrgID)
			}
		}(msg)
	}
	if err := c.Close(); err != nil {
		c.logger.Errorf("failed to closed kafka reader, error: %v", err)
		return
	}
	c.logger.Debugf("Kafka consumer closed successfully for topic %s", c.topicName)
}

func (c *consumer) Close() error {
	return c.reader.Close()
}

func (c *consumer) markTaskAborted(ctx context.Context, orgID string, task *core.Task) {
	c.logger.Debugf("build is set to abort for buildID: %s, marking initiating task with taskID: %s to abort", task.BuildID, task.ID)
	now := time.Now()
	task.Status = core.TaskAborted
	task.Updated = now
	task.EndTime = zero.TimeFrom(now)
	task.Remark = zero.StringFrom(core.JobAbortedByUser)
	if errT := c.taskUpdateManager.TaskUpdate(ctx, task, orgID); errT != nil {
		c.logger.Errorf("failed to update task for taskID: %s, buildID: %s, orgID: %s, error: %v", task.ID, task.BuildID, orgID, errT)
		return
	}
}

func (c *consumer) getTaskStatus(ctx context.Context, buildID string, err error) (core.Status, error) {
	status := core.TaskError
	build, errB := c.buildStore.FindByBuildID(ctx, buildID)
	if errB != nil {
		c.logger.Errorf("failed to build status for buildID: %s, error: %s", buildID, errB)
		return status, errB
	}
	if build.Status == core.BuildAborted || errors.Is(err, context.Canceled) {
		status = core.TaskAborted
	}
	return status, nil
}

func (c *consumer) getRunnerOptions(build *core.BuildCache,
	job *core.Job,
	task *core.Task) (r *core.RunnerOptions, err error) {
	cmdFlags := []string{"--payloadAddress", build.PayloadAddress, "--subModule", task.SubModule}
	if task.Type == core.DiscoveryTask {
		cmdFlags = append(cmdFlags, "--discover")
	} else if task.Type == core.ExecutionTask {
		cmdFlags = append(cmdFlags, "--execute", "--locatorAddress", task.TestLocators.String, "--collectStats")
	} else {
		consecutiveRuns := build.FlakyConsecutiveRuns
		if consecutiveRuns < 1 {
			consecutiveRuns = 1
		}
		cmdFlags = append(cmdFlags, "--flaky", "--locatorAddress", task.TestLocators.String,
			"--consecutiveRuns", strconv.Itoa(consecutiveRuns))
	}
	orgName, repoName := scm.Split(build.RepoSlug)
	if orgName == "" || repoName == "" {
		c.logger.Errorf("failed to find orgName or repoName for orgID %s, job %s, buildID %s",
			job.OrgID, job.ID, task.BuildID)
		return nil, errs.ErrInvalidSlug
	}
	vaultOpts := getVaultOptions(build, job.OrgID, task.BuildID)

	token, err := c.internalJWT.CreateTokenInternal(&core.BuildData{
		BuildID: task.BuildID,
		OrgID:   job.OrgID,
		RepoID:  build.RepoID,
	})
	if err != nil {
		c.logger.Errorf("error generating internal token: %v", err)
		return nil, errs.ErrInvalidJWTToken
	}
	envList := []string{
		"TASK_ID=" + task.ID,
		"BUILD_ID=" + task.BuildID,
		"REPO_ID=" + build.RepoID,
		"ORG_ID=" + job.OrgID,
		"TASK_TYPE=" + string(task.Type),
		"TOKEN=" + token,
	}

	r = &core.RunnerOptions{ContainerName: job.TaskID,
		Label:                     utils.CreateRunnerLabels(task.ID, task.BuildID, job.ID, synapse.TestExecutionMode, repoName),
		NameSpace:                 utils.GetRunnerNamespaceFromOrgID(job.OrgID),
		PodName:                   utils.GetNucleusPodName(job.TaskID),
		PersistentVolumeClaimName: utils.GetBuildHashKey(task.BuildID),
		ContainerPort:             constants.DefaultRunnerPort,
		ContainerArgs:             cmdFlags,
		LogfilePath:               fmt.Sprintf("%s/%s/internal/%s/%s-nucleus.log", job.OrgID, task.BuildID, task.Type, job.TaskID),
		Vault:                     vaultOpts,
		PodType:                   core.NucleusPod,
		Tier:                      build.Tier,
		Env:                       envList,
		OrgID:                     job.OrgID}
	return r, nil
}

func getVaultOptions(buildCache *core.BuildCache, orgID, buildID string) *core.VaultOpts {
	vaultOpts := &core.VaultOpts{
		TokenPath:       buildCache.TokenPath,
		RoleName:        orgID,
		TokenSecretName: utils.GetTokenSecretName(buildID),
	}
	if buildCache.SecretPath != "" {
		vaultOpts.SecretPath = buildCache.SecretPath
		vaultOpts.RepoSecretName = utils.GetRepoSecretName(buildID)
	}
	return vaultOpts
}

func getTaskErrorRemark(err error) zero.String {
	var remark string
	if errors.Is(err, context.Canceled) {
		remark = errs.GenericTaskAbortedErrorRemark.Error()
	} else if errors.Is(err, context.DeadlineExceeded) {
		remark = errs.GenericTaskTimeoutErrorRemark.Error()
	} else if errors.Is(err, errs.ErrOOMKilled) {
		remark = errs.ErrOOMKilled.Error()
	} else {
		remark = errs.GenericUserFacingBEErrRemark.Error()
	}
	return zero.StringFrom(remark)
}

func (c *consumer) updateSecret(ctx context.Context, build *core.BuildCache, orgID, buildID string) error {
	if build.GitProvider != core.DriverBitbucket {
		return nil
	}
	oauthToken, refreshed, err := c.tokenHandler.GetTokenFromPath(ctx, build.GitProvider, build.TokenPath, build.InstallationTokenPath)
	if err != nil {
		c.logger.Errorf("failed to get oauth token for orgID %s, buildID %s, error: %v",
			orgID, buildID, err)
		return err
	}

	if refreshed {
		secretBody, err := json.Marshal(oauthToken)
		if err != nil {
			c.logger.Errorf("failed to marshal oauth token for orgID %s, buildID %s, error: %v",
				orgID, buildID, err)
			return err
		}
		err = c.runner.UpdateSecret(ctx, utils.GetRunnerNamespaceFromOrgID(orgID), utils.GetTokenSecretName(buildID), core.Oauth, secretBody)
		if err != nil {
			c.logger.Errorf("failed to marshal oauth token for orgID %s, buildID %s, error: %v",
				orgID, buildID, err)
			return err
		}
		c.logger.Debugf("updated secret for orgID %s, buildID %s", orgID, buildID)
	}
	return nil
}
