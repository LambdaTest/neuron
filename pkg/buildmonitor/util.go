package buildmonitor

import (
	"context"
	"errors"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"
	"golang.org/x/sync/errgroup"
)

func (m *buildMonitor) FindTimeSavedData(ctx context.Context, buildID, commitID, repoID string) (*core.TimeSavedData, error) {
	var timeSavedData core.TimeSavedData
	g, errCtx := errgroup.WithContext(ctx)
	var totalTest, timeAllTest int
	isImpacted := true
	g.Go(func() error {
		totalTests, timeAllTests, cerr := m.testExecutionStore.FindTimeByRunningAllTests(errCtx, buildID, commitID, repoID)
		if cerr != nil || timeAllTests == 0 {
			if timeAllTests == 0 {
				cerr = errs.ErrRowsNotFound
			}
			return cerr
		}
		totalTest = totalTests
		timeAllTest = timeAllTests
		return nil
	})
	var totalImpactedTest, timeAllImpactedTest int
	g.Go(func() error {
		totalImpactedTests, timeAllImpactedTests, terr := m.testExecutionStore.FindExecutionTimeImpactedTests(errCtx, commitID,
			buildID, repoID)
		if terr != nil {
			if errors.Is(terr, errs.ErrRowsNotFound) {
				isImpacted = false
			}
			return terr
		}
		totalImpactedTest = totalImpactedTests
		timeAllImpactedTest = timeAllImpactedTests
		return nil
	})
	if err := g.Wait(); err != nil {
		if errors.Is(err, errs.ErrRowsNotFound) {
			if !isImpacted {
				timeSavedData = core.TimeSavedData{
					PercentTimeSaved:         100,
					TimeTakenByAllTests:      timeAllTest,
					TimeTakenByImpactedTests: 0,
					TotalTests:               totalTest,
					TotalImpactedTests:       totalImpactedTest,
				}
				return &timeSavedData, nil
			}
			return nil, errs.ErrRowsNotFound
		}
		m.logger.Errorf("error while fetching info from database for repoID %s, buildID %s, error %v",
			repoID, buildID, err)
		return nil, err
	}

	percentTimeSaved := 0.0
	if totalTest > totalImpactedTest {
		timeDifference := utils.Max(0, (timeAllTest - timeAllImpactedTest))
		percentTimeSaved = (float64(timeDifference) / float64(timeAllTest)) * 100
	}
	if percentTimeSaved == 0.0 {
		timeAllTest = timeAllImpactedTest
	}
	timeSavedData = core.TimeSavedData{
		PercentTimeSaved:         percentTimeSaved,
		TimeTakenByAllTests:      timeAllTest,
		TimeTakenByImpactedTests: timeAllImpactedTest,
		TotalTests:               totalTest,
		TotalImpactedTests:       totalImpactedTest,
	}
	return &timeSavedData, nil
}
