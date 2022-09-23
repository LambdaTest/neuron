package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/LambdaTest/neuron/pkg/buildabortqueue"
	"github.com/LambdaTest/neuron/pkg/coverage"
	"github.com/LambdaTest/neuron/pkg/emailnotifications"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/flakytaskbuilder"
	"github.com/LambdaTest/neuron/pkg/githubapp"
	"github.com/LambdaTest/neuron/pkg/opentelemetry"
	"github.com/LambdaTest/neuron/pkg/postprocessingqueue"
	"github.com/LambdaTest/neuron/pkg/requestutils"
	"github.com/LambdaTest/neuron/pkg/startup"
	"github.com/LambdaTest/neuron/pkg/store/branchcommit"
	"github.com/LambdaTest/neuron/pkg/store/buildcoveragestatus"
	"github.com/LambdaTest/neuron/pkg/store/commitdiscovery"
	"github.com/LambdaTest/neuron/pkg/store/discovery"
	"github.com/LambdaTest/neuron/pkg/store/eventqueue"
	"github.com/LambdaTest/neuron/pkg/store/execution"
	"github.com/LambdaTest/neuron/pkg/store/flakyconfigstore"
	"github.com/LambdaTest/neuron/pkg/store/flakyexecution"
	"github.com/LambdaTest/neuron/pkg/store/flakyteststore"
	"github.com/LambdaTest/neuron/pkg/store/postmerge"
	"github.com/LambdaTest/neuron/pkg/store/synapsequeue"
	"github.com/LambdaTest/neuron/pkg/store/testbranch"
	"github.com/LambdaTest/neuron/pkg/store/testsuitebranch"
	"github.com/LambdaTest/neuron/pkg/store/userdemo"
	"github.com/LambdaTest/neuron/pkg/store/userinfo"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/LambdaTest/neuron/pkg/taskbuilder"
	"github.com/LambdaTest/neuron/pkg/taskrunner"
	"github.com/LambdaTest/neuron/pkg/taskupdate"
	"github.com/LambdaTest/neuron/pkg/testsplitter"
	"github.com/LambdaTest/neuron/pkg/token"
	"github.com/LambdaTest/neuron/pkg/webhook"

	"github.com/LambdaTest/neuron/pkg/service/buildabort"
	"github.com/LambdaTest/neuron/pkg/service/comment"
	"github.com/LambdaTest/neuron/pkg/service/gitstatus"
	"github.com/LambdaTest/neuron/pkg/service/pullrequest"
	"github.com/LambdaTest/neuron/pkg/store/branch"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/api"
	"github.com/LambdaTest/neuron/pkg/azure"
	"github.com/LambdaTest/neuron/pkg/buildmonitor"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/db"
	"github.com/LambdaTest/neuron/pkg/gitscm"
	"github.com/LambdaTest/neuron/pkg/jwt"
	"github.com/LambdaTest/neuron/pkg/login"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/redis"
	"github.com/LambdaTest/neuron/pkg/runner/kube"
	"github.com/LambdaTest/neuron/pkg/secrets/vault"
	"github.com/LambdaTest/neuron/pkg/server"
	buildz "github.com/LambdaTest/neuron/pkg/service/build"
	"github.com/LambdaTest/neuron/pkg/service/commit"
	"github.com/LambdaTest/neuron/pkg/service/gituser"
	"github.com/LambdaTest/neuron/pkg/service/hook"
	repoz "github.com/LambdaTest/neuron/pkg/service/repo"
	taskqueuestore "github.com/LambdaTest/neuron/pkg/store/taskqueue"

	orgsz "github.com/LambdaTest/neuron/pkg/service/organizations"

	"github.com/LambdaTest/neuron/pkg/store/blocktests"
	"github.com/LambdaTest/neuron/pkg/store/builds"
	"github.com/LambdaTest/neuron/pkg/store/credits"
	"github.com/LambdaTest/neuron/pkg/store/gitcommits"
	"github.com/LambdaTest/neuron/pkg/store/gitevents"
	"github.com/LambdaTest/neuron/pkg/store/gitusers"
	hookStore "github.com/LambdaTest/neuron/pkg/store/hook"
	loginStore "github.com/LambdaTest/neuron/pkg/store/login"

	"github.com/LambdaTest/neuron/pkg/store/license"
	"github.com/LambdaTest/neuron/pkg/store/organizations"

	testexecutionz "github.com/LambdaTest/neuron/pkg/service/testexecution"
	repo "github.com/LambdaTest/neuron/pkg/store/repos"
	"github.com/LambdaTest/neuron/pkg/store/schedulerstore"
	synapseStore "github.com/LambdaTest/neuron/pkg/store/synapse"
	"github.com/LambdaTest/neuron/pkg/store/task"
	"github.com/LambdaTest/neuron/pkg/store/test"
	"github.com/LambdaTest/neuron/pkg/store/testcoverage"
	"github.com/LambdaTest/neuron/pkg/store/testexecution"
	"github.com/LambdaTest/neuron/pkg/store/testsuite"
	"github.com/LambdaTest/neuron/pkg/store/testsuiteexecution"
	"github.com/LambdaTest/neuron/pkg/store/userorg"
	"github.com/LambdaTest/neuron/pkg/taskqueue"
	"github.com/LambdaTest/neuron/pkg/webhook/parser"
	"github.com/spf13/cobra"
)

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd := cobra.Command{
		Use:     "neuron",
		Long:    `neuron is the brain of TAS which runs the business logic by executing an action in response to the request/event received.`,
		Version: constants.BinaryVersion,
		RunE:    run,
	}

	// define flags used for this command
	AttachCLIFlags(&rootCmd)

	return &rootCmd
}

// nolint:funlen,gocyclo
func run(cmd *cobra.Command, args []string) error {
	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}
	// a WaitGroup to check if all pods spawned by neuron have stopped
	runnerWg := sync.WaitGroup{}

	cfg, err := config.Load(cmd)
	if err != nil {
		fmt.Printf("Failed to load config: %v", err)
		return err
	}

	// patch logconfig file location with root level log file location
	if cfg.LogFile != "" {
		cfg.LogConfig.FileLocation = filepath.Join(cfg.LogFile, "nu.log")
	}

	// You can also use logrus implementation
	// by using lumber.InstanceLogrusLogger
	logger, err := lumber.NewLogger(&cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	if err != nil {
		log.Printf("could not instantiate logger %s", err.Error())
		return err
	}
	database, err := db.Connect(cfg, logger)
	if err != nil {
		logger.Errorf("failed to create database connection %v", err)
		return err
	}
	// create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// initialize tracer
	if cfg.Tracing.OtelEndpoint != "" {
		tracerCleanup := opentelemetry.InitTracer(ctx, cfg, logger)
		defer func() {
			if tracerErr := tracerCleanup(context.Background()); tracerErr != nil {
				logger.Errorf("Failed to cleanup the tracer %v", tracerErr)
			}
		}()
	}

	redisDB, err := redis.New(ctx, cfg, logger)
	if err != nil {
		logger.Errorf("failed to create redis database connection %v", err)
		return err
	}

	azureClient, err := azure.NewAzureBlobEnv(cfg, logger)
	if err != nil {
		logger.Errorf("could not instantiate azure client %v", err)
		return err
	}
	vaultStore := vault.New(ctx, cfg, logger)

	buildAbortMap := buildabort.NewMap(logger)
	runner, err := kube.New(azureClient, vaultStore, logger, cfg, buildAbortMap)
	if err != nil {
		logger.Errorf("could not instantiate k8s runner %v", err)
		return err
	}
	requests := requestutils.New(logger)
	session, err := jwt.New(cfg, logger)
	if err != nil {
		logger.Errorf("could not instantiate jwt authenticator %v", err)
		return err
	}
	internalJWT, err := jwt.New(cfg, logger)
	if err != nil {
		logger.Errorf("could not instantiate internal jwt authenticator %v", err)
		return err
	}

	githubApp, err := githubapp.New(cfg, requests, logger)
	if err != nil {
		logger.Errorf("could not instantiate githubApp service %v", err)
		return err
	}

	testExecutionService := testexecutionz.New(cfg, azureClient, logger)
	loginProvider := login.New(cfg, logger)
	scmProvider := gitscm.New(logger)
	tokenHandler := token.New(cfg, vaultStore, githubApp, logger)
	dbStores := &core.DBStores{
		TestStore:                        test.New(database, logger),
		TestSuiteStore:                   testsuite.New(database, logger),
		TestExecutionStore:               testexecution.New(database, logger),
		TestSuiteExecutionStore:          testsuiteexecution.New(database, logger),
		TestBranchStore:                  testbranch.New(database, logger),
		TestSuiteBranchStore:             testsuitebranch.New(database, logger),
		OrgStore:                         organizations.New(database, redisDB, logger),
		RepoStore:                        repo.New(database, logger),
		UserStore:                        gitusers.New(database, logger),
		EventStore:                       gitevents.New(database, logger),
		CommitDiscoveryStore:             commitdiscovery.New(database, logger),
		CommitStore:                      gitcommits.New(database, logger),
		BranchStore:                      branch.New(database, logger),
		BranchCommitStore:                branchcommit.New(database, logger),
		BuildStore:                       builds.New(database, redisDB, runner, logger),
		TaskStore:                        task.New(database, redisDB, logger),
		CoverageStore:                    testcoverage.New(database, logger),
		LicenseStore:                     license.New(database, logger),
		CreditsUsageStore:                credits.New(database, logger),
		UserOrgStore:                     userorg.New(database, logger),
		BuildCoverageStatusStore:         buildcoveragestatus.New(database, logger),
		PostMergeConfigStore:             postmerge.NewPostMergeConfigStore(database, logger),
		CommitCntSinceLastPostMergeStore: postmerge.NewCommitCntSinceLastPostMergeStore(database, logger),
		EventQueueStore:                  eventqueue.New(database, logger),
		FlakyTestStore:                   flakyteststore.New(database, logger),
		FlakyConfigStore:                 flakyconfigstore.New(database, logger),
		BlockTestStore:                   blocktests.New(database, logger),
		SynapseStore:                     synapseStore.New(redisDB, logger, database),
		UserInfoStore:                    userinfo.New(database, logger),
		UserDemoStore:                    userdemo.New(database, logger),
	}
	dbStores.TaskQueueStore = taskqueuestore.New(database,
		redisDB,
		dbStores.TaskStore,
		dbStores.BuildStore,
		dbStores.OrgStore,
		dbStores.LicenseStore,
		dbStores.CreditsUsageStore, dbStores.EventQueueStore, logger)

	dbStores.PostMergeStoreManager = postmerge.NewPostMergeStoreManager(database,
		dbStores.PostMergeConfigStore,
		dbStores.CommitCntSinceLastPostMergeStore,
		logger)
	dbStores.HookStore = hookStore.New(
		database,
		dbStores.CommitStore,
		dbStores.EventStore,
		dbStores.TaskStore,
		dbStores.BuildStore,
		dbStores.BranchStore,
		dbStores.BranchCommitStore,
		dbStores.PostMergeStoreManager,
		logger)
	dbStores.LoginStore = loginStore.New(database,
		dbStores.OrgStore,
		dbStores.UserStore,
		dbStores.UserOrgStore,
		vaultStore,
		tokenHandler,
		dbStores.LicenseStore,
		logger)
	dbStores.DiscoveryStore = discovery.New(database,
		dbStores.TestStore,
		dbStores.TestSuiteStore,
		dbStores.TestBranchStore,
		dbStores.TestSuiteBranchStore,
		dbStores.CommitDiscoveryStore,
		logger,
	)
	dbStores.ExecutionStore = execution.New(database,
		dbStores.TestExecutionStore,
		dbStores.TestSuiteExecutionStore,
		dbStores.TestStore,
		dbStores.TestSuiteStore,
		testExecutionService,
		logger)

	dbStores.FlakyExecutionStore = flakyexecution.New(database,
		dbStores.FlakyTestStore,
		dbStores.BlockTestStore,
		logger)

	commitService := commit.New(scmProvider, logger)
	pullRequestService := pullrequest.New(scmProvider, logger)
	gitStatusService := gitstatus.New(cfg, dbStores.RepoStore, scmProvider, tokenHandler, logger)
	commentService := comment.New(cfg, scmProvider, tokenHandler, logger)

	// initialize queue producers
	postProcessingQueueProducer := postprocessingqueue.NewProducer(cfg, logger)
	buildAbortQueueProducer := buildabortqueue.NewProducer(cfg, logger)
	taskQueueManager := taskqueue.NewManager(cfg,
		dbStores.TaskQueueStore,
		dbStores.OrgStore,
		dbStores.BuildStore,
		dbStores.TaskStore,
		gitStatusService,
		logger)
	taskBuilder := taskbuilder.New(dbStores.TaskStore, taskQueueManager, dbStores.BuildStore, azureClient, logger)
	flakyTaskBuilder := flakytaskbuilder.New(dbStores.TestExecutionStore, taskBuilder, dbStores.BuildStore, dbStores.FlakyConfigStore, logger)
	emailNotificationManager := emailnotifications.New(cfg, dbStores.UserStore,
		dbStores.OrgStore, requests, logger)
	buildMonitor := buildmonitor.New(cfg, dbStores.BuildStore, dbStores.BuildCoverageStatusStore,
		postProcessingQueueProducer, gitStatusService, dbStores.FlakyConfigStore, flakyTaskBuilder,
		dbStores.RepoStore, emailNotificationManager, dbStores.TestExecutionStore, logger)

	taskQueueUtils := taskqueue.NewTaskQueueUtils(logger, dbStores.TaskQueueStore, gitStatusService, buildMonitor)
	synapseManager := synapse.NewSynapseManager(ctx, logger, redisDB,
		dbStores.OrgStore,
		dbStores.SynapseStore,
		dbStores.BuildStore,
		dbStores.RepoStore,
		dbStores.TaskStore,
		taskQueueUtils,
		taskQueueManager)
	synapsePoolManager := synapse.NewSynapsePoolManager(ctx, logger, redisDB,
		dbStores.SynapseStore,
		dbStores.BuildStore,
		dbStores.RepoStore,
		dbStores.TaskStore,
		taskQueueUtils,
		taskQueueManager,
		synapseManager)
	synapseQueue := synapsequeue.New(redisDB, logger, synapseManager, synapsePoolManager, dbStores.SynapseStore)
	synapseTaskQueue, err := synapsequeue.NewSynapseTaskQueue(redisDB, synapsePoolManager,
		synapseManager, synapseQueue, dbStores.BuildStore, logger)
	if err != nil {
		logger.Errorf("Error occurred in creation of synapse queue %+v", err)
		return err
	}
	taskRunner := taskrunner.New(cfg, logger, runner, dbStores.OrgStore, dbStores.TaskStore, synapseTaskQueue)
	coverageManager := coverage.New(cfg, internalJWT,
		database,
		runner,
		dbStores.BuildCoverageStatusStore,
		dbStores.CoverageStore,
		dbStores.TaskStore,
		dbStores.BuildStore,
		dbStores.OrgStore,
		dbStores.RepoStore,
		taskRunner,
		logger)
	taskUpdateManager := taskupdate.New(dbStores.TaskQueueStore,
		dbStores.BuildStore,
		dbStores.TaskStore,
		buildMonitor,
		taskQueueManager,
		runner,
		logger)
	buildAbortService := buildabort.New(dbStores.BuildStore, dbStores.TaskStore, taskUpdateManager, buildAbortQueueProducer, logger)
	services := &core.Services{
		UserService: gituser.New(scmProvider, logger),
		BuildService: buildz.New(
			azureClient, commitService, pullRequestService, tokenHandler,
			gitStatusService, coverageManager, dbStores.LicenseStore,
			logger, runner, taskQueueManager,
			dbStores.UserStore, dbStores.OrgStore, vaultStore, dbStores.HookStore,
			dbStores.BuildStore,
		),
		OrgService:           orgsz.New(scmProvider, logger),
		RepoService:          repoz.New(scmProvider, dbStores.RepoStore, logger),
		HookService:          hook.New(scmProvider, cfg, logger),
		CommitService:        commitService,
		TestExecutionService: testExecutionService,
		GitStatusService:     gitStatusService,
		CommentService:       commentService,
		PullRequestService:   pullRequestService,
		BuildAbortService:    buildAbortService,
	}
	testSplitter := testsplitter.NewTestSplitter(dbStores.TaskStore,
		dbStores.BuildStore,
		dbStores.TestExecutionStore,
		taskQueueManager,
		azureClient,
		taskBuilder,
		logger)

	webhookParser := parser.New(
		dbStores.RepoStore,
		services.BuildService,
		logger)
	// schedulers
	startupScheduler := startup.NewScheduler(logger, dbStores.TaskQueueStore,
		taskQueueManager,
		dbStores.EventQueueStore)
	schedulerManager := schedulerstore.New(database, dbStores.OrgStore, dbStores.BuildStore, dbStores.TaskStore,
		dbStores.EventQueueStore, logger)
	buildScheduler := buildmonitor.NewScheduler(logger, schedulerManager)
	// initialize queue consumers
	webhookConsumer := webhook.New(cfg, webhookParser, logger)
	taskQueueConsumer := taskqueue.NewConsumer(cfg, &runnerWg, internalJWT,
		dbStores.TaskStore,
		taskUpdateManager,
		dbStores.BuildStore,
		buildMonitor,
		dbStores.TaskQueueStore,
		taskQueueManager,
		redisDB,
		tokenHandler,
		runner,
		services.GitStatusService,
		dbStores.OrgStore,
		taskQueueUtils,
		dbStores.RepoStore,
		taskRunner,
		logger)
	postProcessingQueueConsumer := postprocessingqueue.NewConsumer(cfg, coverageManager, runner, logger)
	buildAbortQueueConsumer := buildabortqueue.NewConsumer(cfg, dbStores.BuildStore, dbStores.TaskStore,
		taskUpdateManager, buildAbortMap, logger)
	// create child context so as to close kafka consumers and schedulers on SIGTERM/SIGINT
	// and fail health API.
	childCtx, childCancel := context.WithCancel(ctx)
	defer childCancel()
	routers := api.New(
		childCtx,
		cfg,
		scmProvider,
		loginProvider,
		tokenHandler,
		githubApp,
		internalJWT,
		session,
		dbStores,
		services,
		vaultStore,
		azureClient,
		redisDB,
		buildMonitor,
		webhookParser,
		testSplitter,
		taskQueueManager,
		taskUpdateManager,
		buildAbortQueueProducer,
		coverageManager,
		synapseManager,
		synapsePoolManager,
		runner,
		emailNotificationManager,
		logger)
	wg.Add(1)
	// setup http server
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(ctx, &routers, cfg, logger); err != nil {
			logger.Errorf("error while running http server %v", err)
		}
	}()

	wg.Add(1)
	// start webhook consumers
	go func() {
		defer wg.Done()
		webhookConsumer.Run(childCtx)
	}()
	wg.Add(1)
	// start taskqueue consumer
	go func() {
		defer wg.Done()
		taskQueueConsumer.Run(childCtx)
	}()

	wg.Add(1)
	// start post processing queue consumer.
	go func() {
		defer wg.Done()
		postProcessingQueueConsumer.Run(childCtx)
	}()
	wg.Add(1)
	// start build abort queue consumer.
	go func() {
		defer wg.Done()
		// passing parent context to avoid closing of kafka-consumer on receiving SIGTERM
		buildAbortQueueConsumer.Run(ctx)
	}()
	// start schedulers
	wg.Add(1)
	go func() {
		defer wg.Done()
		startupScheduler.Run(childCtx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		buildScheduler.Run(childCtx)
	}()

	// starts synapse task queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		synapseQueue.Run(childCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		synapseTaskQueue.InitConsumer(childCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go synapsePoolManager.SetAllSynapseToNotAlive(childCtx)
	}()

	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	// channel to track if all runners spawned by neuron have stopped
	runnersDone := make(chan struct{})

	// create channel to mark status of waitgroup
	// this is required to brutally kill application in case of
	// timeout
	done := make(chan struct{})

	// asynchronously wait for all the go routines
	go func() {
		// and wait for all go routines
		wg.Wait()
		logger.Debugf("main: all goroutines have finished.")
		close(done)
	}()
	// wait for signal channel
	<-c
	logger.Debugf("main: received close signal - attempting graceful shutdown ....")
	childCancel()
	// asynchronously wait for all the runner to exit
	go func() {
		// and wait for all runners go routines
		runnerWg.Wait()
		close(runnersDone)
	}()
	// add some delay so as to allow all go queue consumer to exit, in case no runners are active
	time.Sleep(cfg.ShutDownDelay)
	select {
	case <-runnersDone:
		logger.Debugf("All runners have finished within the specified timeout.")
	case <-time.After(cfg.RunnerWaitTimeout):
		logger.Errorf("Timeout waiting for runners to exit, Brutally killing all runners.")
	}
	// tell the goroutines to stop
	logger.Debugf("main: telling all goroutines to stop")
	cancel()
	select {
	case <-done:
		logger.Debugf("Go routines exited within timeout")
	case <-time.After(cfg.GracefulTimeout):
		logger.Errorf("Graceful timeout exceeded. Brutally killing the application")
		return errs.ErrTimeoutExceeded
	}
	return nil
}
