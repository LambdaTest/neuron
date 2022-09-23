package utils

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/drone/go-scm/scm"
	"github.com/google/uuid"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// GenerateUUID generates uuid v4
func GenerateUUID() string {
	uuidV4 := uuid.New() // panics on error
	return strings.Map(func(r rune) rune {
		if r == '-' {
			return -1
		}
		return r
	}, uuidV4.String())
}

// EncodeOffset base64 encode the offset value
func EncodeOffset(value int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%d", "offset", value)))
}

// EncodeCursor base64 encode the cursor value
func EncodeCursor(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "next_cursor", value)))
}

// GetRunnerNamespaceFromOrgID generates the namespace from orgID
func GetRunnerNamespaceFromOrgID(orgID string) string {
	return fmt.Sprintf("ns-orgid-%s", orgID)
}

// GetNucleusPodName generates the nucleus pod name from taskID
func GetNucleusPodName(taskID string) string {
	return fmt.Sprintf("nucleus-%s", taskID)
}

// GetOrgHashKey generates the redis hash from orgID
func GetOrgHashKey(orgID string) string {
	return fmt.Sprintf("org-%s", orgID)
}

// GetBuildHashKey generates the redis hash from buildID
func GetBuildHashKey(buildID string) string {
	return fmt.Sprintf("build-%s", buildID)
}

// RandString generates a random string of size n
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Int63()%int64(len(alphabet))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// Reverse reverses the 2d slice.
func Reverse(list [][]int) (reversedList [][]int) {
	length := len(list)
	if length == 0 {
		return reversedList
	}
	for i := length - 1; i >= 0; i-- {
		reversedList = append(reversedList, list[i])
	}
	return reversedList
}

// TaskFinished utility function to check if task finished.
func TaskFinished(status core.Status) bool {
	return status == core.TaskAborted || status == core.TaskFailed || status == core.TaskPassed || status == core.TaskError
}

// QueueJobFinshed utility function to check if queue job finished.
func QueueJobFinshed(status core.QueueStatus) bool {
	return status == core.Failed || status == core.Completed || status == core.Aborted
}

// QueueStatus utility function to map task status to queue status
func QueueStatus(status core.Status) core.QueueStatus {
	switch status {
	case core.TaskError, core.TaskFailed:
		return core.Failed
	case core.TaskPassed, core.TaskSkipped:
		return core.Completed
	case core.TaskInitiating:
		return core.Ready
	case core.TaskRunning:
		return core.Processing
	case core.TaskAborted:
		return core.Aborted
	default:
		return core.Processing
	}
}

// GetAuthorName utility function to identify name in a more reliable manner from scm
func GetAuthorName(author *scm.Signature) string {
	if author.Login != "" {
		return author.Login
	}
	return author.Name
}

// CreateRunnerLabels returns runner label
func CreateRunnerLabels(id, buildID, jobID, mode, repo string) map[string]string {
	var labels = map[string]string{}
	labels[synapse.BuildID] = buildID
	labels[synapse.ID] = id
	labels[synapse.JobID] = jobID
	labels[synapse.Mode] = mode
	labels[synapse.Repo] = repo
	return labels
}

// GetTokenSecretName returns the token secret name for k8s resource
func GetTokenSecretName(buildID string) string {
	return fmt.Sprintf("token-%s", buildID)
}

// GetRepoSecretName returns the repo secret name for k8s resource
func GetRepoSecretName(buildID string) string {
	return fmt.Sprintf("reposecrets-%s", buildID)
}

// AddMultipleFilters forms the IN query statements
func AddMultipleFilters(argType string, params []string,
	args map[string]interface{}) (inClause string, arg map[string]interface{}) {
	inClause = ""
	for i := range params {
		p := argType + strconv.Itoa(i)
		inClause += ":" + p
		if inClause != "" {
			inClause += `,`
		}
		args[p] = params[i]
	}
	inClause = inClause[:len(inClause)-1]
	return inClause, args
}

// GetSpawnedPodsMapKey returns the key for spawned pods map for build abort service
func GetSpawnedPodsMapKey(buildID, taskID string) string {
	return fmt.Sprintf("abortchan-%s-%s", buildID, taskID)
}

// GetFileNameFromTestLocator extracts the file name from the test locator
func GetFileNameFromTestLocator(testLocator string) (fileName string) {
	i := strings.Index(testLocator, constants.LocatorDelimiter)
	if i > 0 {
		return testLocator[:i]
	}
	return fileName
}

func GetFlakyComment(flakyTestCount int, frontendURL, gitProvider, repoSlug, buildID string) string {
	if flakyTestCount == 0 {
		return fmt.Sprintf("No flaky test detected. See [here](%s/%s/%s/jobs/%s/)",
			frontendURL, gitProvider, repoSlug, buildID)
	}
	if flakyTestCount == 1 {
		return fmt.Sprintf("%d test case was found **Flaky**. See [here](%s/%s/%s/jobs/%s/)",
			flakyTestCount, frontendURL, gitProvider, repoSlug, buildID)
	}
	return fmt.Sprintf("%d test cases were found **Flaky**. See [here](%s/%s/%s/jobs/%s/)",
		flakyTestCount, frontendURL, gitProvider, repoSlug, buildID)
}

func Chunk(chunkSize, total int, fn func(start int, end int) error) error {
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		if err := fn(i, end); err != nil {
			return err
		}
	}
	return nil
}
