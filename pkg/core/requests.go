package core

import (
	"time"
)

// TestReportRequestPayload represents the request body for test and test suite report api.
type TestReportRequestPayload struct {
	TaskID   string            `json:"taskID" binding:"required"`
	BuildID  string            `json:"buildID" binding:"required"`
	RepoID   string            `json:"repoID" binding:"required"`
	OrgID    string            `json:"orgID" binding:"required"`
	CommitID string            `json:"commitID" binding:"required"`
	TaskType TaskType          `json:"taskType" binding:"required,oneof=discover execute flaky"`
	Results  []ExecutionResult `json:"results"`
}

// TestReportResponsePayload represents the response body for test and test suite report api.
type TestReportResponsePayload struct {
	TaskID     string `json:"taskID"`
	TaskStatus Status `json:"taskStatus"`
	Remark     string `json:"remark,omitempty"`
}

// ExecutionResult contains test results
type ExecutionResult struct {
	TestPayload      []TestPayload      `json:"testResults"`
	TestSuitePayload []TestSuitePayload `json:"testSuiteResults"`
}

// TestPayload represents the request body for test execution
type TestPayload struct {
	TestID          string              `json:"testID"`
	Detail          string              `json:"_detail"`
	SuiteID         string              `json:"suiteID"`
	Suites          []string            `json:"_suites"`
	Title           string              `json:"title"`
	FullTitle       string              `json:"fullTitle"`
	Name            string              `json:"name"`
	Duration        int                 `json:"duration"`
	FilePath        string              `json:"file"`
	Line            string              `json:"line"`
	Col             string              `json:"col"`
	CurrentRetry    int                 `json:"currentRetry"`
	Status          TestExecutionStatus `json:"status"`
	DAG             []string            `json:"dependsOn"`
	Filelocator     string              `json:"locator"`
	BlocklistSource string              `json:"blocklistSource"`
	Blocklisted     bool                `json:"blocklist"`
	StartTime       time.Time           `json:"start_time"`
	EndTime         time.Time           `json:"end_time"`
	Stats           []TestProcessStats  `json:"stats"`
	FailureMessage  string              `json:"failureMessage"`
}

// TestSuitePayload represents the request body for test suite execution
type TestSuitePayload struct {
	SuiteID         string                   `json:"suiteID"`
	SuiteName       string                   `json:"suiteName"`
	ParentSuiteID   string                   `json:"parentSuiteID"`
	BlocklistSource string                   `json:"blocklistSource"`
	Blocklisted     bool                     `json:"blocklist"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	Duration        int                      `json:"duration"`
	Status          TestSuiteExecutionStatus `json:"status"`
	Stats           []TestProcessStats       `json:"stats"`
}

// TestProcessStats process stats associated with each test
type TestProcessStats struct {
	Memory     uint64    `json:"memory_consumed,omitempty"`
	CPU        float64   `json:"cpu_percentage,omitempty"`
	Storage    uint64    `json:"storage,omitempty"`
	RecordTime time.Time `json:"record_time"`
}
