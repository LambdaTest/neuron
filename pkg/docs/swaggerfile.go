// Package docs classification TAS API
//
// Documentation of TAS API
//
// Terms of Service:
//
// Not any terms of service as of now
//
// Schemes: http
// Host: localhost:9876
// Version: 0.0.1
//
// 	Consumes:
// 	- application/json
//
// 	Produces:
// 	- application/json
// swagger:meta
package docs

import (
	"github.com/LambdaTest/neuron/pkg/api/build"
	"github.com/LambdaTest/neuron/pkg/api/coverage"
	"github.com/LambdaTest/neuron/pkg/api/organization"
	"github.com/LambdaTest/neuron/pkg/api/repo"
	"github.com/LambdaTest/neuron/pkg/api/secrets"
	"github.com/LambdaTest/neuron/pkg/api/user"
	"github.com/LambdaTest/neuron/pkg/core"
)

// swagger:route GET /health general healthAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: Authenticated

// swagger:route GET /blocklist nucleus blocklistAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 404: notFound
// 500: internal
// 200: testresponse
// 400: badReq

// swagger:route POST /test-list nucleus test-listAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 502: badGateway
// 500: internal

// swagger:route POST /report nucleus reportAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 500: internal

// swagger:route GET /coverage nucleus coverageFind
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 404: notFound
// 200: description: status ok
// 400: badReq
// 500: internal

// swagger:route POST /coverage nucleus coverageCreate
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 400: badReq
// 200: description: status ok

// swagger:parameters coverageCreate
type CoverageParamsWrapper struct {
	// in: body
	// required: true
	Body coverage.CoverageInput
}

// swagger:route PUT /task nucleus taskCreateAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: description: status ok
// 400: badReq
// 500: internal

// swagger:route GET /build build listBuildAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: buildresponse
// 500: internal
// 400: badReq

// swagger:route GET /coverage/info/{sha} coverage findInfoCoverageAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 404: notFound
// 200: description: statusOk
// 400: badReq
// 500: internal

// swagger:parameters findInfoCoverageAPI
type FindInfoRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: path
	// require: true
	CommitID string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:parameters listBuildAPI
type ListBuildRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Status string
	// in: query
	ID string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /build/{buildID}/aggregate build listBuildDetailsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: buildresponse
// 500: internal
// 400: badReq

// swagger:parameters listBuildDetailsAPI
type ListBuildDetailsRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route PUT /build/abort build abortBuildAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: abortBuildResponse
// 500: internal
// 400: abortBuildBadRequest
// 404: notFound

// swagger:parameters abortBuildAPI
type AbortBuildAPI struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	BuildID string
	// in: header
	// require: true
	Cookie string
}

// Status OK
// swagger:response abortBuildResponse
type AbortBuildResponseWrapper struct {
	// in:body
	Body struct {
		// HTTP status code 200 - Status Ok
		//
		// Example: 200
		Code int `json:"code"`
		// Detailed response message
		//
		// Example: aborting job inititated
		Message string `json:"message"`
	}
}

// BadRequest
// swagger:response abortBuildBadRequest
type AbortBuildBadRequestWrapper struct {
	// in:body
	Body struct {
		// HTTP status code 400 - Status BadRequest
		//
		// Example: 400
		Code int `json:"code"`
		// Detailed response message
		//
		// Example: missing buildID in parameters
		Message string `json:"message"`
	}
}

// swagger:route GET /build/{BuildID} build listSpecificBuildAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testresponse
// 500: internal
// 404: badReq

// swagger:parameters listSpecificBuildAPI
type ListSpecificBuildRequestWrapper struct {
	// in: path
	// require: true
	BuildID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
}

// swagger:route GET /build/{BuildID}/metrics/{TaskID} build listSpecificBuildMetricsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testmetricsresponse
// 500: internal
// 404: notFound

// swagger:parameters listSpecificBuildMetricsAPI
type ListSpecificBuildMetricsRequestWrapper struct {
	// in: path
	// require: true
	BuildID string
	// in: path
	// require: true
	TaskID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /build/{BuildID}/logs build listBuildLogsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 500: internal
// 404: notFound

// swagger:parameters listBuildLogsAPI
type ListBuildLogsRequestWrapper struct {
	// in: path
	// require: true
	BuildID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	TaskID string
	// in: header
	// require: true
	LogsTag string
}

// swagger:route GET /build/task build listBuildTaskAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: taskResponse
// 500: internal
// 404: notFound

// swagger:parameters listBuildTaskAPI
type ListBuildTaskRequestWrapper struct {
	// in: path
	// require: true
	BuildID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	TaskID string
	// in: header
	// require: true
	LogsTag string
}

// swagger:route GET /build/{BuildID}/time-saved-impacted build listBuildImpactedTestsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testresponse
// 500: internal
// 404: notFound

// swagger:parameters listBuildImpactedTestsAPI
type ListBuildImpactedTestsRequestWrapper struct {
	// in: path
	// require: true
	BuildID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	CommitID string
}

// swagger:route GET /build/meta build listBuildMetaAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 500: internal
// 404: notFound

// swagger:parameters listBuildMetaAPI
type ListBuildMetaWrapper struct {
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /build/commit-diff build commitDiffAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 500: internal
// 404: notFound

// swagger:parameters commitDiffAPI
type ListCommitDiffWrapper struct {
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	BuildID string
	// in: query
	// require: true
	CommitID string
}

// swagger:route GET /commit commit listCommitsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: gitcommitresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitsAPI
type ListCommitsRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /commit/{commitID}/aggregate commit listCommitAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: gitcommitresponse
// 404: notFound
// 500: internal

// swagger:route GET /commit/{CommitID}/build commit listCommitBuildsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: buildresponse
// 404: notFound
// 500: internal
// 400: badReq

// swagger:parameters listCommitBuildsAPI
type ListCommitBuildsRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
	// in: path
	// require: true
	CommitID string
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /commit/{CommitID}/build/{BuildID}/impacted-tests commit listImpactedTestsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:parameters listImpactedTestsAPI
type ListImpactedTestsRequestWrapper struct {
	// in: path
	// require: true
	CommitID string
	// in: path
	// require: true
	BuildID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Text string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /commit/{CommitID}/build/{BuildID}/unimpacted-tests commit listUnimpactedTestsAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listUnimpactedTestsAPI
type ListUnmpactedTestsRequestWrapper struct {
	// in: path
	// require: true
	CommitID string
	// in: path
	// require: true
	BuildID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
}

// swagger:route GET /commit/build/{buildID} commit listCommitBuildAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: gitcommitresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitBuildAPI
type ListCommitBuildRequestWrapper struct {
	// in: path
	BuildID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
}

// swagger:route GET /commit/{CommitID}/build/{BuildID}/tests commit listTestsCommitBuildAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestsCommitBuildAPI
type ListTestsCommitBuildRequestWrapper struct {
	// in: path
	// require: true
	CommitID string
	// in: path
	// require: true
	BuildID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
}

// swagger:route GET /commit/meta commit listCommitMetaAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters listCommitMetaAPI
type ListCommitMetaRequestWrapper struct {
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
}

// swagger:route POST /repo repo enableRepoAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: Status ok
// 404: notFound
// 500: internal
// 400: badReq
// 502: badGateway
// 409: statusconflict

// swagger:parameters enableRepoAPI
type EnableRepoRequestWrapper struct {
	// body params of repository
	// in: body
	Body repo.RepoInput
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /repo repo listRepositoriesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: repositoryresponse
// 404: notFound
// 500: internal
// 502: badGateway

// swagger:parameters listRepositoriesAPI
type ListRepositoriesRequestWrapper struct {
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /repo/active repo getActiveRepositoriesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: repositoryresponse
// 404: notFound
// 500: internal

// swagger:parameters getActiveRepositoriesAPI
type GetActiveRepositorieRequestWrapper struct {
	// in: query
	// require: true
	Org string
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /repo/badge repo getRepositoriesBadgeAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters getRepositoriesBadgeAPI
type GetRepositoriesBadgeWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	Branch string
}

// swagger:route POST /repo/settings repo editConfigInfoAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal
// 400: badReq
// 502: badGateway
// 409: statusconflict

// swagger:parameters editConfigInfoAPI
type RepoEditConfigRequestWrapper struct {
	// body params of repository
	// in: body
	Body core.RepoSettingsInfo

	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /repo/settings repo fetchConfigInfoAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal
// 400: badReq
// 502: badGateway
// 409: statusconflict

// swagger:parameters fetchConfigInfoAPI
type RepoConfigInfoRequestWrapper struct {
	// query params of repository
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /v2/repo/badge repo getRepositoriesBadgev2API
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters getRepositoriesBadgev2API
type GetRepositoriesBadgev2Wrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /repo/badge/flaky repo getRepositoriesBadgev3API
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters getRepositoriesBadgev3API
type GetRepositoriesBadgev3Wrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in:query
	// require: true
	Branch string
}

// swagger:route DELETE /repo/deactivate repo repoDeactivateAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters repoDeactivateAPI
type RepoDeactivateWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /repo/user-branches repo getRepoBranchesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters getRepoBranchesAPI
type GetRepositoriesBranchesWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /suite suites listSuitesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testsuiteresponse
// 404: notFound
// 500: internal

// swagger:parameters listSuitesAPI
type ListSuitesRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /suite/{suiteID}/aggregate suites listSuiteAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testsuiteresponse
// 404: notFound
// 500: internal

// swagger:parameters listSuitesAPI
type ListSuiteRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: query
	// require: true
	SuiteID string
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /suite/{SuiteID} suites listSuiteDetailsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:parameters listSuiteDetailsAPI
type ListSuiteDetailsRequestWrapper struct {
	// in: path
	// require: true
	SuiteID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
	Branch string
	// in: query
	// require: true
	PerPage int
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
}

// swagger:route GET /suite/metrics suites listSuitesMetricsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testmetricsresponse
// 404: notFound
// 500: internal

// swagger:parameters listSuitesMetricsAPI
type ListSuitesMetricsRequestWrapper struct {
	// in: query
	// require: true
	TaskID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	ExecID string
	// in: query
	// require: true
	Org string
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	BuildID string
}

// swagger:route GET /suite/{SuiteID}/graphs/status-data suites listSuitesGraphAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:parameters listSuitesGraphAPI
type ListSuitesGraphRequestWrapper struct {

	// in: path
	// require: true
	SuiteID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
}

// swagger:route GET /analytics/graphs/status-data/tests analytics listMonthWiseNetTestsCountAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: monthwisetestcountresponse
// 404: notFound
// 500: internal

// swagger:parameters listMonthWiseNetTestsCountAPI
type ListMonthWiseNetTestsCountAPIWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
}

// swagger:route GET /suite/meta suites listSuitesMetaAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters listSuitesMetaAPI
type ListSuitesMetaRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	Branch string
}

// swagger:route GET /test test listTestsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestsAPI
type ListTestsRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
	Branch string
}

// swagger:route GET /test/{testID}/aggregate test listTestDetailsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestDetailsAPI
type ListTestRequestWrapper struct {
	// in: path
	TestID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	PerPage int
	// in: header
	// require: true
	Cookie string
	// in: query
	Status string
	// in: query
	Text string
	// in: query
	Branch string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
}

// swagger:route GET /test/{TestID} test listTestDetailsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:route GET /test/metrics test listTestMetricsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testmetricsresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestMetricsAPI
type ListTestMetricsRequestWrapper struct {
	// in: query
	// require: true
	TaskID string
	// in: query
	// require: true
	BuildID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	ExecID string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /test/{TestID}/graphs/status-data test listTestStatusGraphAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testexecutionresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestStatusGraphAPI
type ListTestStatusGraphRequestWrapper struct {
	// in: path
	// require: true
	TestID string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
}

// swagger:route GET /test/meta test listTestMetaAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters listTestMetaAPI
type ListTestMetaWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: header
	// require: true
	Cookie string
}

// swagger:route GET /login/github general loginAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: Status Ok
// 404: notFound
// 500: internal

// swagger:route GET /contributor contributors listContributorAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: Status Ok
// 404: notFound
// 500: internal

// swagger:parameters listContributorAPI
type ListContributorRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route GET /contributor/graphs/commit-activity contributors listContributorCommitAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: Status Ok
// 404: notFound
// 500: internal

// swagger:parameters listContributorCommitAPI
type ListContributorCommitRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Type string
}

// swagger:route GET /contributor/author/details contributors listContributorStatsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters listContributorStatsAPI
type ListContributorStatsRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	Author string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /contributor/graph contributors listContributorGraphAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters listContributorGraphAPI
type ListContributorGraphRequestWrapper struct {
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data analytics listTestStatusAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listTestStatusAPI
type ListTestStatusRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Text string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/build analytics listBuildStatusesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listBuildStatusesAPI
type ListBuildStatusRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/build/mttf-mttr analytics findMttfAndMttrAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: mttfmttrresponse
// 404: notFound
// 500: internal

// swagger:parameters findMttfAndMttrAPI
type FindMttfAndMttrRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	// require: true
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/commit analytics listCommitStatusesAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitStatusesAPI
type ListCommitStatusesRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/tests-added analytics findTestsAddedCountAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: addedTestResponse
// 404: commitNotFound
// 400: badRequest
// 500: internal

// swagger:parameters findTestsAddedCountAPI
type FindTestsAddedCountRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/commit/slowest analytics listCommitStatusesSlowestAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitStatusesSlowestAPI
type ListCommitStatusesSlowestRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/commit/irregular-tests analytics listCommitStatusesIrregularAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: irregulartestresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitStatusesIrregularAPI
type ListCommitStatusesIrregularTeststWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/commit/failed analytics listCommitStatusesFailedAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitStatusesFailedAPI
type ListCommitStatusesFailedRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/build/failed analytics listBuildStatusesFailedAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listBuildStatusesFailedAPI
type ListBuildStatusesFailedRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/commits/tests-data analytics listCommitStatusesTestDataAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listCommitStatusesTestDataAPI
type CommitStatusesTestDataRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
}

// swagger:route GET /analytics/graphs/status-data/tests-jobs analytics listBuildStatusesTestDataAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listBuildStatusesTestDataAPI
type BuildsTesStatustDataRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	Branch string
	// in: query
	// require: true
	Limit int
}

// swagger:route GET /analytics/graphs/jobs/status-data analytics listJobsStatusesTestDataAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: testresponse
// 404: notFound
// 500: internal

// swagger:parameters listJobsStatusesTestDataAPI
type JobsStatusTestDataRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
	// in: query
	Tag string
}

// swagger:route GET /analytics/graphs/status-data/build/time-saved analytics listBuildWiseTimeSavedAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: buildwisetimesavedresponse
// 404: notFound
// 500: internal

// swagger:parameters listBuildWiseTimeSavedAPI
type ListBuildWiseTimeSavedAPIRequestWrapper struct {
	// in: query
	// require: true
	Repo string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
	// in: query
	Branch string
	// in: query
	NextCursor string
	// in:query
	PerPage string
}

// swagger:route POST /secret/add Secrets secretsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:route DELETE /secret/delete Secrets secretsAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:route GET /secret Secrets secretsListAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters secretsAPI
type SecretsRequestWrapper struct {
	// body params of repository
	// in: body
	Body secrets.SecretInput

	// in: header
	// require: true
	Cookie string
}

// swagger:parameters secretsListAPI
type SecretsParamsWrapper struct {
	// in: header
	// require: true
	Cookie string

	// in: header
	// require: true
	Org string

	// in: header
	// require: true
	Repo string
}

// swagger:route POST /profile/info profile userInfoAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userInfoAPI
type UserInfoRequest struct {

	// in: body
	Body user.UserInfoDetails

	// in: header
	Cookie string
}

// swagger:route GET /profile profile userDataAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userDataAPI
type UserDataRequest struct {
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /profile/info profile userDataInfoAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userDataInfoAPI
type UserDataInfoRequest struct {
	// in: query
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /profile/user/usage profile userDataUsageAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userDataUsageAPI
type UserDataUsageRequest struct {
	// in: query
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route PUT /profile/user/active profile userActiveAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userActiveAPI
type UserActiveRequest struct {
	// in: query
	// require: true
	Org string

	// in: header
	Cookie string
}

// swagger:route POST /user/demo/info profile userDemoAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters userDemoAPI
type UserDemoRequest struct {
	// in: body
	Body core.UserInfoDemoDetails

	// in: header
	Cookie string
}

// swagger:route GET /repo/branch Branch branchListAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters branchListAPI
type BranchListRequest struct {
	// in: query
	Org string
	// in: query
	Repo string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /coverage/info/{sha} Coverage coverageInfoAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters coverageInfoAPI
type CoverageInfoRequest struct {
	// in: path
	Sha string
	// in: query
	Org string
	// in: query
	Repo string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /org Org orgListAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: orgResponse
// 404: notFound
// 500: internal

// swagger:parameters orgListAPI
type OrgListRequest struct {
	// in: query
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /org/sync Org orgListSyncAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: orgResponse
// 404: notFound
// 500: internal

// swagger:parameters orgListSyncAPI
type OrgListSyncRequest struct {
	// in: query
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /org/synapse/count Org orgSynapseAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapseAPI
type OrgSynapseRequest struct {
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Org string
}

// swagger:route PUT /org/synapse/token Org orgSynapseTokenAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapseTokenAPI
type OrgSynapseTokenRequest struct {
	// in: body
	Body organization.TokenInfo
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /org/synapse/list Org orgSynapseListAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapseListAPI
type OrgSynapseListRequest struct {
	// in: query
	// require: true
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route GET /org/synapse/test-connection Org orgSynapseTestConnectionAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapseTestConnectionAPI
type OrgSynapseTestConnectionRequest struct {
	// in: query
	// require: true
	Org string
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route PATCH /org/synapse Org orgSynapsePatchAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapsePatchAPI
type OrgSynapsePatchRequest struct {
	// in: header
	Cookie string
	// in: query
	// require: true
	Org string
}

// swagger:route PUT /org/synapse/config Org orgSynapseConfigAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters orgSynapseConfigAPI
type OrgSynapseConfigRequest struct {
	// in: body
	Body organization.OrgRunnerInfo
	// in: header
	Cookie string
	// in: query
	// require: true
	GitProvider string
}

// swagger:route POST /user/eligibility-info profile openSourceUserAPI
//
// Consumes:
// - application/json
// Produces:
// -application/json
// Schemes: http, https
// Responses:
// 200: statusOk
// 404: notFound
// 500: internal

// swagger:parameters openSourceUserAPI
type OpenSourceUserRequest struct {
	// in: body
	Body core.OpenSourceUserInfo
}

// status ok
// swagger:response orgResponse
type OrgResponseWrapper struct {
	// in: body
	Body []core.Organization
}

// status ok
// swagger:response testresponse
type TestResponseWrapper struct {
	// in: body
	Body []core.Test
}

// status ok
// swagger:response irregulartestresponse
type IrregularTestResponseWrapper struct {
	// in: body
	Body []core.IrregularTests
}

// status ok
// swagger:response testexecutionresponse
type CommitResponseWrapper struct {
	// in: body
	Body []core.TestExecution
}

// status ok
// swagger:response buildresponse
type BuildResponseWrapper struct {
	// in: body
	Body []core.Build
}

// status ok
// swagger:response testmetricsresponse
type TestMetricsResponseWrapper struct {
	// in: body
	Body []core.TestMetrics
}

// status ok
// swagger:response gitcommitresponse
type GitCommitResponseWrapper struct {
	// in: body
	Body []core.GitCommit
}

// status ok
// swagger:response repositoryresponse
type RepoResponseWrapper struct {
	// in: body
	Body []core.Repository
}

// status ok
// swagger:response mttfmttrresponse
type MttfMttrResponseWrapper struct {
	// in: body
	Body core.MttfAndMttr
}

// status ok
// swagger:response graphvisualizationresponse
type RepoGraphResponseWrapper struct {
	// in: body
	Body [][]int
}

// status ok
// swagger:response buildwisetimesavedresponse
type BuildWiseTimeSavedResponseWrapper struct {
	// in:body
	Body struct {
		Message []core.BuildWiseTimeSaved `json:"build_wise_time_saved"`
	}
}

// status ok
// swagger:response taskResponse
type ListBuildTaskResponseWrapper struct {
	// in:body
	Body []core.Task
}

// status ok
// swagger:response testsuiteresponse
type SuiteResponseWrapper struct {
	// in: body
	Body []core.TestSuite
}

// status ok
// swagger:response addedTestResponse
type AddedTestsCountResponseWrapper struct {
	// in:body
	Body struct {
		// Example: 50
		Message int `json:"added_tests_count"`
	}
}

// Error Commit Not Found
// swagger:response commitNotFound
type ErrCommitNotFound struct {
	// in:body
	Body struct {
		// HTTP status code 404 - Not Found
		//
		// Example: 404
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: Commit not found for given repository.
		Message string `json:"message"`
	}
}

// Error Bad Request
// swagger:response badRequest
type ErrBadRequest struct {
	// in:body
	Body struct {
		// HTTP status code 400 - Status Bad Request
		//
		// Example: 400
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: Missing org in request query parameters.
		Message string `json:"message"`
	}
}

// Error Bad Request
// swagger:response badReq
type ErrBadReq struct {
	// in:body
	Body struct {
		// HTTP status code 400 - Status Bad Request
		//
		// Example: 400
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: Invalid request payload.
		Message string `json:"message"`
	}
}

// Error Forbidden
// swagger:response forbidden
type ErrForbidden struct {
	// in:body
	Body struct {
		// HTTP status code 403 - Forbidden
		//
		// Example: 403
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Not Found
// swagger:response notFound
type ErrNotFound struct {
	// in:body
	Body struct {
		// HTTP status code 404 - Not Found
		//
		// Example: 404
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: User not found for given orgID.
		Message string `json:"message"`
	}
}

// Error Interval Server
// swagger:response internal
type ErrInternal struct {
	// in:body
	Body struct {
		// HTTP status code 500 - Internal server error
		//
		// Example: 500
		Code int `json:"code"`
		// Detailed error message
		Message string `json:"message"`
	}
}

// Error Bad Gateway
// swagger:response badGateway
type ErrBadGateway struct {
	// in:body
	Body struct {
		// HTTP status code 502 - Bad Gateway error
		//
		// Example: 502
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: Type assertion failed when parsing map.
		Message string `json:"message"`
	}
}

// Error Status Conflict
// swagger:response statusconflict
type ErrStatusConflict struct {
	// in:body
	Body struct {
		// HTTP status code 409 - Status Conflict error
		//
		// Example: 409
		Code int `json:"code"`
		// Detailed error message
		//
		// Example: Repo already active.
		Message string `json:"message"`
	}
}

// Status OK
// swagger:response statusOk
type StatusOkResponse struct {
	// in:body
	Body struct {
		// HTTP status code 200 - Status Ok
		//
		// Example: 200
		Code int `json:"code"`
		// Detailed response message
		//
		// Example: entity added/deleted successfully.
		Message string `json:"message"`
	}
}

// Unauthorized
// swagger:response unauthorized
type StatusUnauthorizedResponse struct {
	// in:body
	Body struct {
		// HTTP status code 401 - Unauthorized
		//
		// Example: 401
		Code int `json:"code"`
		// Detailed response message
		//
		// Example: entity added/deleted successfully.
		Message string `json:"message"`
	}
}

// Status OK
// swagger:response monthwisetestcountresponse
type MonthwisetestcountresponseWrapper struct {
	// in: body
	Body []core.MonthWiseNetTests
}

// swagger:route POST  /build/rebuild build rebuildAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http, https
// Responses:
// 200: description: Status ok
// 400: badReq
// 401: unauthorized
// 404: notFound
// 500: internal
// 502: badGateway

// swagger:parameters rebuildAPI
type RebuildRequestWrapper struct {

	// in: body
	// require: true
	Body build.ReBuildRequest
	// in: header
	Cookie string
}

// swagger:route POST /postmergeconfig postmergeconfig postmergeconfigAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 404: notFound
// 502: badGateway
// 500: internal
// swagger:parameters postmergeconfigAPI
type PostMergeConfigRequestWrapper struct {
	// body params of postmergeconfig

	// in: body
	// require: true
	Org string

	// in: body
	// require: true
	Repo string

	// in: body
	// require: true
	Branch string

	// in: body
	// require: true
	IsActive bool

	// in: body
	// require: true
	StrategyName string

	// in: body
	// require: true
	Threshold string
}

// swagger:route PUT /postmergeconfig postmergeconfig updatePostmergeconfigAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 404: notFound
// 502: badGateway
// 500: internal
// swagger:parameters updatePostmergeconfigAPI
type UpdatePostMergeConfigRequestWrapper struct {
	// body params of postmergeconfig

	// in: body
	// require: true
	ID string

	// in: body
	// require: true
	Org string

	// in: body
	// require: true
	Repo string

	// in: body
	// require: true
	Branch string

	// in: body
	// require: true
	IsActive bool

	// in: body
	// require: true
	StrategyName string

	// in: body
	// require: true
	Threshold string
}

// swagger:route GET /postmergeconfig/list postmergeconfig listPostmergeconfigAPI
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 401: unAuthorize
// 403: forbidden
// 404: notFound
// 500: internal
// swagger:parameters listPostmergeconfigAPI
type ListPostMergeConfigRequestWrapper struct {
	// in: query
	// require: true
	Org string

	// in: query
	// require: true
	Repo string
}

// swagger:route GET /credits/consumed Credits findTotalConmsumedCredits
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: description: status ok
// 400: badReq
// 401: unAuthorize
// 403: forbidden
// 500: internal
// swagger:parameters listPostmergeconfigAPI
type FindTotalConsumedCreditsRequestWrapper struct {
	// in: query
	// require: true
	OrgID string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: true
	StartDate string
	// in: query
	// require: true
	EndDate string
}

// swagger:route GET /flakybuild/:buildID FlakyBuild findFlakyTests
//
// Consumes:
// - application/json
// Produces:
// - application/json
// Schemes: http
// Responses:
// 200: flakyBuild
// 400: badReq
// 401: unAuthorize
// 404: notFound
// 500: internal
// swagger:parameters findFlakyTests
type FindFlakyTestsRequestWrapper struct {
	// in: query
	// require: true
	Branch string
	// in: header
	// require: true
	Cookie string
	// in: query
	// require: false
	TaskID string
	// in: query
	// require: true
	Org string
	// in: query
	// require: true
	GitProvider string
	// in: query
	// require: true
	Repo string
}

// status ok
// swagger:response flakyBuild
type FlakyBuildWrapper struct {
	// in: body
	Body []core.FlakyTestExecution
}
