package api

import (
	"context"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/api/analytics"
	"github.com/LambdaTest/neuron/pkg/api/blocktest"
	"github.com/LambdaTest/neuron/pkg/api/build"
	"github.com/LambdaTest/neuron/pkg/api/commit"
	"github.com/LambdaTest/neuron/pkg/api/contributor"
	"github.com/LambdaTest/neuron/pkg/api/coverage"
	"github.com/LambdaTest/neuron/pkg/api/credits"
	"github.com/LambdaTest/neuron/pkg/api/flakyconfig"
	"github.com/LambdaTest/neuron/pkg/api/flakytest"
	"github.com/LambdaTest/neuron/pkg/api/health"
	"github.com/LambdaTest/neuron/pkg/api/license"
	"github.com/LambdaTest/neuron/pkg/api/login"
	"github.com/LambdaTest/neuron/pkg/api/logout"
	"github.com/LambdaTest/neuron/pkg/api/middleware"
	"github.com/LambdaTest/neuron/pkg/api/organization"
	"github.com/LambdaTest/neuron/pkg/api/postmergeconfig"
	"github.com/LambdaTest/neuron/pkg/api/repo"
	"github.com/LambdaTest/neuron/pkg/api/sas"
	"github.com/LambdaTest/neuron/pkg/api/secrets"
	"github.com/LambdaTest/neuron/pkg/api/submodule"
	"github.com/LambdaTest/neuron/pkg/api/synapse"
	"github.com/LambdaTest/neuron/pkg/api/task"
	"github.com/LambdaTest/neuron/pkg/api/test"
	"github.com/LambdaTest/neuron/pkg/api/testexecution"
	"github.com/LambdaTest/neuron/pkg/api/testlist"
	"github.com/LambdaTest/neuron/pkg/api/testsuite"
	"github.com/LambdaTest/neuron/pkg/api/user"
	"github.com/LambdaTest/neuron/pkg/api/yamlconfig"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/ws"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Router represents the routes for the http server.
type Router struct {
	cfg                      *config.Config
	signalCtx                context.Context
	scmProvider              core.SCMProvider
	loginProvider            core.GitLoginProvider
	tokenHandler             core.GitTokenHandler
	githubApp                core.GithubApp
	internalJWT              core.Session
	session                  core.Session
	vaultStore               core.Vault
	taskQueueManager         core.TaskQueueManager
	buildMonitor             core.BuildMonitor
	testStore                core.TestStore
	testSuiteStore           core.TestSuiteStore
	testExecutionStore       core.TestExecutionStore
	testSuiteExecutionStore  core.TestSuiteExecutionStore
	discoveryStore           core.DiscoveryStore
	executionStore           core.ExecutionStore
	orgStore                 core.OrganizationStore
	repoStore                core.RepoStore
	userStore                core.GitUserStore
	eventStore               core.GitEventStore
	commitStore              core.GitCommitStore
	branchStore              core.BranchStore
	branchCommitStore        core.BranchCommitStore
	buildStore               core.BuildStore
	taskStore                core.TaskStore
	taskQueueStore           core.TaskQueueStore
	hookStore                core.HookStore
	loginStore               core.LoginStore
	coverageStore            core.TestCoverageStore
	licenseStore             core.LicenseStore
	creditsUsageStore        core.CreditsUsageStore
	userOrgStore             core.UserOrgStore
	postMergeConfigStore     core.PostMergeConfigStore
	synapseStore             core.SynapseStore
	synapseManager           core.SynapseClientManager
	synapsePoolManager       core.SynapsePoolManager
	buildService             core.BuildService
	userService              core.GitUserService
	orgService               core.OrganizationService
	repoService              core.RepositoryService
	hookService              core.HookService
	testExecutionService     core.TestExecutionService
	redisDB                  core.RedisDB
	azureClient              core.AzureBlob
	gitStatusService         core.GitStatusService
	commentService           core.CommentService
	coverageManager          core.CoverageManager
	testSplitter             core.TestSplitter
	hookParser               core.HookParser
	k8sRunner                core.K8sRunner
	logger                   lumber.Logger
	flakytestStore           core.FlakyTestStore
	flakyconfigStore         core.FlakyConfigStore
	emailNotificationManager core.EmailNotificationManager
	flakyExecutionStore      core.FlakyExecutionStore
	blockTestStore           core.BlockTestStore
	userInfoStore            core.UserInfoStore
	userDemoStore            core.UserDemoStore
	buildAbortService        core.BuildAbortService
	taskUpdateManager        core.TaskUpdateManager
}

// New returns a New Router
func New(
	signalCtx context.Context,
	cfg *config.Config,
	scmProvider core.SCMProvider,
	loginProvider core.GitLoginProvider,
	tokenHandler core.GitTokenHandler,
	githubApp core.GithubApp,
	internalJWT core.Session,
	session core.Session,
	dbStores *core.DBStores,
	services *core.Services,
	vaultStore core.Vault,
	azureClient core.AzureBlob,
	redisDB core.RedisDB,
	buildMonitor core.BuildMonitor,
	hookParser core.HookParser,
	testSplitter core.TestSplitter,
	taskQueueManager core.TaskQueueManager,
	taskUpdateManager core.TaskUpdateManager,
	buildAbortQueueProducer core.BuildAbortProducer,
	coverageManager core.CoverageManager,
	synapseManager core.SynapseClientManager,
	synapsePoolManager core.SynapsePoolManager,
	runner core.K8sRunner,
	emailNotificationManager core.EmailNotificationManager,
	logger lumber.Logger) Router {
	return Router{
		cfg:                      cfg,
		signalCtx:                signalCtx,
		scmProvider:              scmProvider,
		loginProvider:            loginProvider,
		tokenHandler:             tokenHandler,
		githubApp:                githubApp,
		internalJWT:              internalJWT,
		session:                  session,
		vaultStore:               vaultStore,
		k8sRunner:                runner,
		buildMonitor:             buildMonitor,
		taskQueueManager:         taskQueueManager,
		testSplitter:             testSplitter,
		testStore:                dbStores.TestStore,
		testSuiteStore:           dbStores.TestSuiteStore,
		testExecutionStore:       dbStores.TestExecutionStore,
		testSuiteExecutionStore:  dbStores.TestSuiteExecutionStore,
		orgStore:                 dbStores.OrgStore,
		repoStore:                dbStores.RepoStore,
		userStore:                dbStores.UserStore,
		eventStore:               dbStores.EventStore,
		commitStore:              dbStores.CommitStore,
		branchStore:              dbStores.BranchStore,
		branchCommitStore:        dbStores.BranchCommitStore,
		buildStore:               dbStores.BuildStore,
		discoveryStore:           dbStores.DiscoveryStore,
		taskStore:                dbStores.TaskStore,
		taskQueueStore:           dbStores.TaskQueueStore,
		hookStore:                dbStores.HookStore,
		loginStore:               dbStores.LoginStore,
		coverageStore:            dbStores.CoverageStore,
		licenseStore:             dbStores.LicenseStore,
		creditsUsageStore:        dbStores.CreditsUsageStore,
		synapseStore:             dbStores.SynapseStore,
		flakyExecutionStore:      dbStores.FlakyExecutionStore,
		synapseManager:           synapseManager,
		synapsePoolManager:       synapsePoolManager,
		buildService:             services.BuildService,
		userOrgStore:             dbStores.UserOrgStore,
		postMergeConfigStore:     dbStores.PostMergeConfigStore,
		userService:              services.UserService,
		orgService:               services.OrgService,
		repoService:              services.RepoService,
		hookService:              services.HookService,
		gitStatusService:         services.GitStatusService,
		commentService:           services.CommentService,
		testExecutionService:     services.TestExecutionService,
		coverageManager:          coverageManager,
		hookParser:               hookParser,
		redisDB:                  redisDB,
		azureClient:              azureClient,
		logger:                   logger,
		executionStore:           dbStores.ExecutionStore,
		emailNotificationManager: emailNotificationManager,
		flakytestStore:           dbStores.FlakyTestStore,
		flakyconfigStore:         dbStores.FlakyConfigStore,
		blockTestStore:           dbStores.BlockTestStore,
		userInfoStore:            dbStores.UserInfoStore,
		userDemoStore:            dbStores.UserDemoStore,
		buildAbortService:        services.BuildAbortService,
		taskUpdateManager:        taskUpdateManager,
	}
}

//nolint:funlen
// Handler function will perform all route operations
func (r *Router) Handler() *gin.Engine {
	r.logger.Infof("Setting up routes")
	router := gin.New()
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := configureValidator(v); err != nil {
			r.logger.Fatalf("failed to configure validator %v", err)
		}
	}
	// skip /health API from logs as will be required in probes
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/health"))
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = constants.CorsAllowedOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowHeaders("authorization", "cache-control", "pragma")
	router.Use(cors.New(corsConfig))
	router.Use(otelgin.Middleware(constants.ServiceName))
	pprof.Register(router)

	router.GET("/health", health.Handler(r.signalCtx))

	internalRoutes := router.Group("/internal")

	// websocket routes for synapse
	wsRoutes := router.Group("/ws")
	ws.RegisterRoutes(wsRoutes, r.logger, r.synapseManager, r.synapsePoolManager)
	// all synapse related route with UI
	synapseRoutesExternal := router.Group("/synapse")
	synapseRoutesExternal.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	synapseRoutesExternal.GET("/list", synapse.HandleList(r.synapseStore, r.userStore, r.logger))

	// all synapse internal routes
	synapseRoute := router.Group("/synapse")
	synapseRoute.Use(middleware.HandleAuthenticationSynapse(r.orgStore, r.logger))
	synapseRoute.GET("/blocktest", blocktest.HandleFind(r.blockTestStore, r.taskStore, r.logger))
	synapseRoute.POST("/internal/sas-token", sas.HandleCreate(r.azureClient, r.logger))
	synapseRoute.POST("/test-list", testlist.HandleCreate(r.discoveryStore, r.buildStore, r.testSplitter, r.gitStatusService, r.logger))
	synapseRoute.PUT("/task", task.HandleUpdate(r.buildStore, r.taskUpdateManager, r.logger))
	synapseRoute.POST("/report", testexecution.HandleCreate(
		r.testStore,
		r.testSuiteStore,
		r.testExecutionStore,
		r.testSuiteExecutionStore,
		r.testExecutionService,
		r.executionStore,
		r.buildStore,
		r.taskStore,
		r.flakyExecutionStore,
		r.flakyconfigStore,
		r.commentService,
		r.logger))
	synapseRoute.GET("/coverage", coverage.HandleFind(r.coverageStore, r.logger))
	synapseRoute.POST("/coverage", coverage.HandleCreate(r.coverageManager, r.azureClient, r.logger))
	synapseRoute.POST("/submodule-list", submodule.HandleCreate(r.buildStore, r.logger))

	//TODO: restrict for nucleus
	// routes for nucleus
	router.GET("/blocktest", middleware.HandleJWTVerificationInternal(r.internalJWT, r.logger),
		blocktest.HandleFind(r.blockTestStore, r.taskStore, r.logger))
	internalRoutes.POST("/sas-token", middleware.HandleJWTVerificationInternal(r.internalJWT,
		r.logger), sas.HandleCreate(r.azureClient, r.logger))
	router.POST("/test-list", middleware.HandleJWTVerificationInternal(r.internalJWT, r.logger),
		testlist.HandleCreate(r.discoveryStore, r.buildStore, r.testSplitter, r.gitStatusService, r.logger))
	router.POST("/report", middleware.HandleJWTVerificationInternal(r.internalJWT, r.logger), testexecution.HandleCreate(
		r.testStore,
		r.testSuiteStore,
		r.testExecutionStore,
		r.testSuiteExecutionStore,
		r.testExecutionService,
		r.executionStore,
		r.buildStore,
		r.taskStore,
		r.flakyExecutionStore,
		r.flakyconfigStore,
		r.commentService,
		r.logger))

	router.POST("/submodule-list", middleware.HandleJWTVerificationInternal(r.internalJWT, r.logger),
		submodule.HandleCreate(r.buildStore, r.logger))

	coverageRoutes := router.Group("/coverage")
	coverageRoutes.GET("", coverage.HandleFind(r.coverageStore, r.logger))
	coverageRoutes.POST("", coverage.HandleCreate(r.coverageManager, r.azureClient, r.logger))
	coverageRoutes.GET("/info/:sha",
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
		middleware.HandlePage(),
		coverage.HandleFindCoverageInfo(r.userStore, r.coverageStore, r.logger))

	// taskRoutes
	taskRoutes := router.Group("/task")
	//TODO: restrict for nucleus
	taskRoutes.PUT("/", task.HandleUpdate(r.buildStore, r.taskUpdateManager, r.logger))

	// build api routes
	buildRoutes := router.Group("/build")
	buildRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)

	buildRoutes.PUT("/abort", build.HandleAbort(r.buildAbortService, r.logger))
	buildRoutes.GET("/:buildID/aggregate", middleware.HandlePage(), build.HandleList(r.buildStore, r.logger))
	buildRoutes.GET("", middleware.HandlePage(), build.HandleList(r.buildStore, r.logger))
	buildRoutes.GET("/:buildID/time-saved-impacted",
		middleware.HandlePage(), build.HandleFindTimeSaved(r.buildMonitor, r.logger))
	buildRoutes.GET("/:buildID", middleware.HandlePage(), build.HandleBuildDetails(r.buildStore, r.logger))
	buildRoutes.GET("/:buildID/logs", middleware.HandlePage(), build.HandleFetchLogs(r.azureClient, r.logger))
	buildRoutes.GET("/tasks", middleware.HandlePage(), build.HandleListTasks(r.taskStore, r.logger))
	buildRoutes.GET("/:buildID/metrics/:taskID", middleware.HandlePage(),
		build.HandleFetchMetrics(r.repoStore, r.testExecutionService, r.logger))
	buildRoutes.GET("/meta", middleware.HandlePage(), build.HandleListMeta(r.buildStore, r.logger))
	// Deprecated : added the same in commit aggregate api (to be removed later after checking with frontend)
	buildRoutes.GET("/commit-diff", build.HandleFindDiff(r.buildStore, r.logger))
	// rebuild route
	router.POST("/build/rebuild",
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		build.HandleRebuild(r.buildService, r.taskStore, r.logger, r.repoStore, r.userStore, r.buildStore, r.eventStore))

	// commit api routes
	commitRoutes := router.Group("/commit")
	commitRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	commitRoutes.GET("", middleware.HandlePage(), commit.HandleLists(r.commitStore, r.logger))
	commitRoutes.GET("/:sha/aggregate", middleware.HandlePage(), commit.HandleLists(r.commitStore, r.logger))
	commitRoutes.GET("/:sha/build", middleware.HandlePage(), commit.HandleListBuilds(r.buildStore, r.logger))
	commitRoutes.GET("/:sha/build/:buildID/impacted-tests",
		middleware.HandlePage(), commit.HandleListImpactedTests(r.commitStore, r.logger))
	commitRoutes.GET("/:sha/build/:buildID/unimpacted-tests",
		middleware.HandlePage(), commit.HandleListUnimpactedTests(r.testStore, r.logger))
	commitRoutes.GET("/build/:buildID", middleware.HandlePage(),
		commit.HandleListCommitsBuild(
			r.commitStore,
			r.logger))
	commitRoutes.GET("/:sha/build/:buildID/tests", middleware.HandlePage(),
		commit.HandleListTestsByCommit(
			r.commitStore,
			r.testExecutionStore,
			r.logger))
	commitRoutes.GET("/meta", middleware.HandlePage(), commit.HandleListMeta(r.commitStore, r.logger))

	// organization routes
	orgRoutes := router.Group("/org")
	orgRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	orgRoutes.GET("", organization.HandleList(r.orgStore, r.logger))
	orgRoutes.GET("/sync", organization.HandleSync(r.userStore, r.orgService, r.orgStore, r.tokenHandler, r.loginStore, r.logger))
	orgRoutes.PUT("/config", organization.HandleUpdateConfigInfo(r.orgStore, r.userStore, r.logger))
	orgRoutes.PUT("/synapse/token", organization.HandleUpdateSynapseToken(r.orgStore, r.userStore, r.logger))
	orgRoutes.GET("/synapse/list", organization.HandleSynapseList(r.synapseStore, r.userStore, r.logger))
	orgRoutes.GET("/synapse/test-connection", organization.HandleSynapseTestConnection(r.synapseStore, r.userStore, r.logger))
	orgRoutes.PATCH("/synapse", organization.HandleSynapsePatch(r.synapseStore, r.userStore, r.logger))
	orgRoutes.GET("/synapse/count", organization.HandleSynapseCount(r.synapseStore, r.userStore, r.logger))

	// user routes
	profileRoutes := router.Group("/profile")
	profileRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	profileRoutes.GET("", user.HandleUserData(r.userStore, r.logger))
	profileRoutes.POST("/info", user.HandleUserInfo(r.userStore, r.userInfoStore, r.logger))
	profileRoutes.GET("/info", user.HandleUserInfoData(r.userStore, r.userInfoStore, r.logger))
	profileRoutes.GET("/user/usage", user.HandleUserUsageInfo(r.userStore, r.logger))
	profileRoutes.PUT("/user/active", user.HandleMarkActive(r.userStore, r.userInfoStore, r.logger))
	router.POST("user/demo/info", user.HandleUserInfoDemoData(r.userDemoStore, r.emailNotificationManager, r.logger))
	router.POST("user/eligibility-info", user.HandleEligibilityInfo(r.emailNotificationManager, r.logger))

	// secrets routes
	secretsRoutes := router.Group("/secret")
	secretsRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	secretsRoutes.GET("", secrets.HandleList(r.repoStore, r.userStore, r.vaultStore, r.logger))
	secretsRoutes.POST("/add", secrets.HandleCreate(r.repoStore, r.userStore, r.vaultStore, r.logger))
	secretsRoutes.DELETE("/delete", secrets.HandleDelete(r.repoStore, r.userStore, r.vaultStore, r.logger))

	router.GET("/repo/branch",
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
		repo.HandleListBranch(r.branchStore, r.logger))

	// repo routes
	repoRoutes := router.Group("/repo")
	repoRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	repoRoutes.GET("/active", middleware.HandlePage(), repo.HandleListActive(r.userStore, r.repoStore, r.logger))
	repoRoutes.GET("", middleware.HandlePage(), repo.HandleList(r.repoService, r.userStore, r.tokenHandler, r.logger))
	repoRoutes.GET("/settings", middleware.HandlePage(), repo.HandleListRepoSettings(r.userStore, r.repoStore, r.logger))
	repoRoutes.POST("", repo.HandleCreate(r.repoStore, r.userStore, r.hookService, r.repoService, r.tokenHandler,
		r.emailNotificationManager, r.postMergeConfigStore, r.logger))
	repoRoutes.POST("/settings", repo.HandleCreateRepoSettings(r.userStore, r.repoStore, r.logger))
	repoRoutes.DELETE("/deactivate", middleware.HandlePage(),
		repo.HandleRemoveRepo(r.hookService, r.userStore, r.repoStore, r.tokenHandler, r.logger))
	router.GET("/repo/badge", middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger), repo.HandleBadgeData(r.repoStore, r.logger))
	router.GET("v2/repo/badge", middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
		repo.HandleBadgeTimeSaved(r.buildStore, r.testExecutionStore, r.repoStore, r.logger))
	router.GET("repo/badge/flaky", middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
		repo.HandleBadgeFlakyTests(r.repoStore, r.logger))
	repoRoutes.GET("/user-branches", middleware.HandleRepoValidation(r.repoStore, true, r.logger),
		middleware.HandlePage(),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
		repo.HandleListBranches(r.repoService, r.userStore, r.tokenHandler, r.logger))

	// test api routes
	testRoutes := router.Group("/test")
	testRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	testRoutes.GET("", middleware.HandlePage(), test.HandleList(r.testStore, r.logger))
	testRoutes.GET("/:testID/aggregate", middleware.HandlePage(), test.HandleList(r.testStore, r.logger))
	testRoutes.GET("/:testID", middleware.HandlePage(), testexecution.HandleList(r.testExecutionStore, r.logger))
	testRoutes.GET("/:testID/graphs/status-data", testexecution.HandleStatusData(r.testExecutionStore, r.logger))
	testRoutes.GET("/metrics", testexecution.HandleExecMetricsBlob(r.testExecutionService, r.logger))
	testRoutes.GET("/failure-logs", testexecution.HandleFailureLogs(r.testExecutionService, r.logger))
	testRoutes.GET("/meta", middleware.HandlePage(), test.HandleTestMeta(r.testStore, r.logger))

	// test suite routes
	testSuiteRoutes := router.Group("/suite")
	testSuiteRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	testSuiteRoutes.GET("", middleware.HandlePage(), testsuite.HandleList(r.testSuiteStore, r.logger))
	testSuiteRoutes.GET("/:suiteID/aggregate", middleware.HandlePage(), testsuite.HandleList(r.testSuiteStore, r.logger))
	testSuiteRoutes.GET("/:suiteID", middleware.HandlePage(), testsuite.HandleDetails(r.testSuiteStore, r.logger))
	testSuiteRoutes.GET("/:suiteID/graphs/status-data", testsuite.HandleStatusData(r.testSuiteStore, r.logger))
	testSuiteRoutes.GET("/metrics", testsuite.HandleExecMetricsBlob(r.testExecutionService, r.logger))
	testSuiteRoutes.GET("/meta", middleware.HandlePage(), testsuite.HandleMeta(r.testSuiteStore, r.logger))

	contributorRoutes := router.Group("/contributor")
	// git contributor routes
	contributorRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	contributorRoutes.GET("", middleware.HandlePage(), contributor.HandleLists(r.commitStore, r.logger))
	contributorRoutes.GET("/graphs/commit-activity", middleware.HandlePage(),
		contributor.HandleCommitActivityData(r.commitStore, r.logger))
	contributorRoutes.GET("/graph", contributor.HandleListGraph(r.commitStore, r.logger))
	contributorRoutes.GET("/author/details", contributor.HandleAuthorDetails(r.testStore, r.commitStore, r.logger))

	//  analytics routes
	analyticsRoutes := router.Group("/analytics")
	analyticsRoutes.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	analyticsRoutes.GET("/graphs/status-data", middleware.HandlePage(),
		analytics.HandleListTestStatuses(r.testStore, r.logger, false))
	analyticsRoutes.GET("/graphs/status-data/build", middleware.HandlePage(),
		analytics.HandleListBuildStatuses(r.buildStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/commit", middleware.HandlePage(),
		analytics.HandleListCommitStatuses(r.commitStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/commit/slowest", middleware.HandlePage(),
		analytics.HandleListTestStatusesSlowest(r.testStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/commit/failed", middleware.HandlePage(),
		analytics.HandleListTestStatusesFailed(r.testExecutionStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/build/failed", middleware.HandlePage(),
		analytics.HandleListBuildStatusesFailed(r.buildStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/commits/tests-data", middleware.HandlePage(),
		analytics.HandleListTestDataCommits(r.testStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/builds/flaky-tests",
		analytics.HandleListFlakyTestsJobs(r.flakytestStore, r.logger))
	analyticsRoutes.GET("/status-data/list-flaky-tests", middleware.HandlePage(),
		analytics.HandleListFlakyTests(r.flakytestStore, r.logger))
	analyticsRoutes.GET("/graphs/jobs/status-data", middleware.HandlePage(),
		analytics.HandleListTestStatuses(r.testStore, r.logger, true))
	analyticsRoutes.GET("/graphs/status-data/commit/irregular-tests", middleware.HandlePage(),
		analytics.HandleListTestStatusesIrregular(r.testStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/tests", middleware.HandlePage(),
		analytics.HandleListMonthWiseTest(r.testStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/build/mttf-mttr", middleware.HandlePage(),
		analytics.HandleListBuildMTTFandMTTR(r.buildStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/tests-jobs", middleware.HandlePage(),
		analytics.HandleListTestsWithJobsStatuses(r.buildStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/build/time-saved", middleware.HandlePage(),
		analytics.HandleListBuildWiseTimeSaved(r.buildStore, r.logger))
	analyticsRoutes.GET("/graphs/status-data/tests-added", middleware.HandlePage(),
		analytics.HandleListTestAdded(r.testStore, r.logger))

	// license routes
	licenseRoutes := router.Group("/license")
	licenseRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	licenseRoutes.GET("/:orgid", license.HandleFind(r.licenseStore, r.logger))
	//TODO: These should be an internal API, otherwise end-user can jack requests and get lifetime validity
	// licenseRoutes.POST("/", license.HandleCreate(r.licenseStore, r.logger))
	// licenseRoutes.PATCH("/", license.HandlePatch(r.licenseStore, r.logger))

	// usage credits routes
	creditsRoutes := router.Group("/credits")
	creditsRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	creditsRoutes.GET("/user/:username", middleware.HandlePage(), credits.HandleFind(r.userStore, r.creditsUsageStore, r.logger, "user"))
	creditsRoutes.GET("/org", middleware.HandlePage(), credits.HandleFind(r.userStore, r.creditsUsageStore, r.logger, "org"))
	creditsRoutes.GET("/consumed", credits.HandleFindTotalConsumed(r.creditsUsageStore, r.logger))

	if r.cfg.FrontendURL == "" {
		r.logger.Fatalf("frontend redirect url not found")
	}

	// OAuth login endpoint
	router.GET("/login/:scmdriver",
		login.Handler(
			r.loginProvider,
			r.scmProvider,
			r.userService,
			r.orgService,
			r.session,
			r.userStore,
			r.loginStore,
			r.orgStore,
			r.cfg.FrontendURL,
			r.cfg.GitHub.AppName,
			r.emailNotificationManager,
			r.tokenHandler,
			r.githubApp,
			r.logger))
	// logout route
	router.GET("/logout", middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		logout.HandleLogout(r.session, r.redisDB, r.logger))

	// PostMergeConfig routes
	postMergeConfigRoutes := router.Group("/postmergeconfig")
	postMergeConfigRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	postMergeConfigRoutes.GET("/list", postmergeconfig.HandleList(r.postMergeConfigStore,
		r.userStore, r.repoStore, r.logger))
	postMergeConfigRoutes.POST("", postmergeconfig.HandleCreate(r.postMergeConfigStore, r.userStore,
		r.repoStore, r.logger))
	postMergeConfigRoutes.PUT("", postmergeconfig.HandleUpdate(r.postMergeConfigStore,
		r.userStore, r.repoStore, r.logger))

	// FlakyConfig routes
	flakyConfigRoutes := router.Group("/flakyconfig")
	flakyConfigRoutes.Use(middleware.HandleJWTVerification(r.session, r.redisDB, r.logger))
	flakyConfigRoutes.GET("", flakyconfig.HandleList(r.flakyconfigStore,
		r.userStore, r.repoStore, r.logger))
	flakyConfigRoutes.POST("", flakyconfig.HandleCreate(r.flakyconfigStore, r.userStore,
		r.repoStore, r.logger))
	flakyConfigRoutes.PUT("", flakyconfig.HandleUpdate(r.flakyconfigStore,
		r.userStore, r.repoStore, r.logger))

	// FlakyExecutionTest Results routes
	flakyExecutionTestResults := router.Group("/flakybuild")
	flakyExecutionTestResults.Use(
		middleware.HandleRepoValidation(r.repoStore, false, r.logger),
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
		middleware.HandleAuthentication(r.userOrgStore, r.logger),
	)
	flakyExecutionTestResults.GET("/:buildID", middleware.HandlePage(), flakytest.HandleFlakyBuildDetails(r.flakytestStore,
		r.logger))

	// yamlConfig routes
	yamlConfigRoutes := router.Group("/yaml-config")
	yamlConfigRoutes.Use(
		middleware.HandleJWTVerification(r.session, r.redisDB, r.logger),
	)
	yamlConfigRoutes.POST("/validate", yamlconfig.HandleValidation(r.logger))

	return router
}
