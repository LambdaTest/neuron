package testexecution

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

var testMetricsColumnLabel = []string{"test_execution_id", "cpu", "memory", "record_time"}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var bufioWriterPool = sync.Pool{
	New: func() interface{} {
		// ioutil.Discard is just used to create the writer. Actual destination
		// writer is set later by Reset() before using it.
		return bufio.NewWriter(ioutil.Discard)
	},
}

// testExecutionService stores metrics in csv format
type testExecutionService struct {
	logger        lumber.Logger
	azureBlob     core.AzureBlob
	containerName string
}

// New returns TestExecutionService
func New(cfg *config.Config, azureBlob core.AzureBlob, logger lumber.Logger) core.TestExecutionService {
	return &testExecutionService{
		containerName: cfg.Azure.PayloadContainerName,
		logger:        logger,
		azureBlob:     azureBlob,
	}
}

func (tes *testExecutionService) StoreTestMetrics(ctx context.Context, path string, metrics [][]string) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	// Use bufio Writer to prevent csv.Writer from allocating a new buffer.
	bufioWriter := bufioWriterPool.Get().(*bufio.Writer)
	bufioWriter.Reset(buf)

	defer bufPool.Put(buf)
	defer bufioWriterPool.Put(bufioWriter)

	w := csv.NewWriter(bufioWriter)
	// write label
	if err := w.Write(testMetricsColumnLabel); err != nil {
		tes.logger.Errorf("error while writing column label %v", err)
		return err
	}

	if err := w.WriteAll(metrics); err != nil {
		tes.logger.Errorf("error while writing test metrics %v", err)
		return err
	}
	_, err := tes.azureBlob.UploadStream(ctx, path, buf, core.MetricsContainer, constants.MIMECSV)
	return err
}

// FetchMetrics fetches the metrics from Azure blob.
func (tes *testExecutionService) FetchMetrics(ctx context.Context, path, executionID string) (metrics []*core.TestMetrics, err error) {
	executionFilter := true
	if executionID == "" {
		executionFilter = false
	}
	metricsDownloaded, err := tes.azureBlob.DownloadStream(ctx, path, core.MetricsContainer)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(metricsDownloaded)
	_, err = reader.Read()
	if err != nil {
		tes.logger.Errorf("error while reading the Header of CSV file:", err)
		return nil, err
	}

	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		tes.logger.Errorf("error while reading the CSV file:", err)
		return nil, err
	}
	metrics, err = getMetrics(rows, executionID, executionFilter)
	if err != nil {
		tes.logger.Errorf("error while parsing csv values, %v", err)
		return nil, err
	}
	return metrics, nil
}

func (tes *testExecutionService) StoreTestFailures(ctx context.Context, path string, failureDetails map[string]string) error {
	failureDetailsJSON, err := json.Marshal(failureDetails)
	if err != nil {
		tes.logger.Errorf("error while marshaling failureDetails map %v", err)
		return err
	}
	buf := bytes.NewBuffer(failureDetailsJSON)
	_, err = tes.azureBlob.UploadStream(ctx, path, buf, core.LogsContainer, gin.MIMEJSON)
	return err
}

func (tes *testExecutionService) FetchTestFailures(ctx context.Context, path, executionID string) (failureMsg string, err error) {
	reader, err := tes.azureBlob.DownloadStream(ctx, path, core.LogsContainer)
	if err != nil {
		return "", err
	}
	jsonBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	failureDetails := make(map[string]string)
	err = json.Unmarshal(jsonBytes, &failureDetails)
	if err != nil {
		return "", err
	}
	return failureDetails[executionID], nil
}

func getMetrics(rows [][]string, executionID string,
	executionFilter bool) (metrics []*core.TestMetrics, err error) {
	metrics = make([]*core.TestMetrics, 0)
	for _, row := range rows {
		if len(row) < constants.BlobMetricsLength {
			continue
		}
		if !executionFilter {
			executionID = row[0]
		}
		if row[0] == executionID {
			cpu, cerr := strconv.ParseFloat(row[1], constants.FloatPrecision)
			if cerr != nil {
				return nil, cerr
			}
			memory, merr := strconv.ParseUint(row[2], 0, 0)
			if merr != nil {
				return nil, merr
			}
			recordTime, terr := time.Parse(constants.DefaultLayout, row[3])
			if terr != nil {
				return nil, terr
			}
			testMetrics := &core.TestMetrics{CPU: cpu,
				Memory:     memory,
				RecordTime: recordTime}
			metrics = append(metrics, testMetrics)
		}
	}
	return metrics, nil
}
