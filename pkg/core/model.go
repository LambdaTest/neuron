package core

// DBStores contains collection of neuron dbstores
type DBStores struct {
	TestStore                        TestStore
	TestSuiteStore                   TestSuiteStore
	TestExecutionStore               TestExecutionStore
	TestSuiteExecutionStore          TestSuiteExecutionStore
	TestBranchStore                  TestBranchStore
	TestSuiteBranchStore             TestSuiteBranchStore
	DiscoveryStore                   DiscoveryStore
	ExecutionStore                   ExecutionStore
	OrgStore                         OrganizationStore
	RepoStore                        RepoStore
	UserStore                        GitUserStore
	EventStore                       GitEventStore
	CommitDiscoveryStore             CommitDiscoveryStore
	CommitStore                      GitCommitStore
	BranchStore                      BranchStore
	BranchCommitStore                BranchCommitStore
	BuildStore                       BuildStore
	TaskStore                        TaskStore
	CoverageStore                    TestCoverageStore
	LicenseStore                     LicenseStore
	CreditsUsageStore                CreditsUsageStore
	UserOrgStore                     UserOrgStore
	TaskQueueStore                   TaskQueueStore
	HookStore                        HookStore
	LoginStore                       LoginStore
	BuildCoverageStatusStore         BuildCoverageStatusStore
	PostMergeConfigStore             PostMergeConfigStore
	PostMergeStoreManager            PostMergeStoreManager
	CommitCntSinceLastPostMergeStore CommitCntSinceLastPostMergeStore
	EventQueueStore                  EventQueueStore
	SynapseStore                     SynapseStore
	FlakyConfigStore                 FlakyConfigStore
	FlakyTestStore                   FlakyTestStore
	BlockTestStore                   BlockTestStore
	FlakyExecutionStore              FlakyExecutionStore
	UserInfoStore                    UserInfoStore
	UserDemoStore                    UserDemoStore
}

// Services contains collection of neuron services
type Services struct {
	BuildService         BuildService
	UserService          GitUserService
	OrgService           OrganizationService
	RepoService          RepositoryService
	HookService          HookService
	CommitService        CommitService
	GitStatusService     GitStatusService
	CommentService       CommentService
	TestExecutionService TestExecutionService
	PullRequestService   PullRequestService
	BuildAbortService    BuildAbortService
}
