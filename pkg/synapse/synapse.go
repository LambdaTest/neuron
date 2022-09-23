package synapse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const dockerRunFailedMessage = "Docker run failed on synapse"

type synapseManager struct {
	ctx              context.Context
	logger           lumber.Logger
	redis            core.RedisDB
	orgStore         core.OrganizationStore
	synapseStore     core.SynapseStore
	buildStore       core.BuildStore
	repoStore        core.RepoStore
	taskStore        core.TaskStore
	taskQueueUtils   core.TaskQueueUtils
	taskQueueManager core.TaskQueueManager
	NeuronID         string
}

// NewSynapseManager returns new synapseManger
func NewSynapseManager(
	ctx context.Context,
	logger lumber.Logger,
	redis core.RedisDB,
	orgStore core.OrganizationStore,
	synapseStore core.SynapseStore,
	buildStore core.BuildStore,
	repoStore core.RepoStore,
	taskStore core.TaskStore,
	taskQueueUtils core.TaskQueueUtils,
	taskQueueManager core.TaskQueueManager) core.SynapseClientManager {
	return &synapseManager{ctx: ctx, logger: logger, redis: redis, orgStore: orgStore,
		synapseStore: synapseStore, repoStore: repoStore, buildStore: buildStore,
		taskStore: taskStore, taskQueueUtils: taskQueueUtils, taskQueueManager: taskQueueManager,
		NeuronID: uuid.New().String()}
}

func (s *synapseManager) NewClient(ws *websocket.Conn, loginDetails core.LoginDetails) *core.SynapseMeta {
	return &core.SynapseMeta{
		Name:            loginDetails.Name,
		ID:              loginDetails.SynapseID,
		SecretKey:       loginDetails.SecretKey,
		IsAuthenticated: false,
		CPU:             loginDetails.CPU,
		RAM:             loginDetails.RAM,
		TotalCPU:        loginDetails.CPU,
		TotalRAM:        loginDetails.RAM,
		Jobs:            []core.SynapseJobInfo{},
		Connection:      ws,
		AbortConnection: make(chan bool),
		IsAlive:         Alive,
		NeuronID:        s.NeuronID,
		LastAliveTime:   time.Now().Unix(),
		SynapseVersion:  loginDetails.SynapseVersion,
	}
}

func (s *synapseManager) AuthenticateClient(synapseMeta *core.SynapseMeta) bool {
	org, err := s.orgStore.FindBySecretKey(context.TODO(), synapseMeta.SecretKey)
	if err != nil {
		s.logger.Errorf("Error in finding org for synapseID %s error %v", synapseMeta.ID, err)
		return false
	}
	if org.SecretKey.String == synapseMeta.SecretKey {
		synapseMeta.OrgID = org.ID
		synapseMeta.IsAuthenticated = true
		return true
	}
	s.logger.Debugf("Authentication failed for synapseID %s orgID %s", synapseMeta.ID, synapseMeta.OrgID)
	return false
}

func (s *synapseManager) CaptureResources(synapseMeta *core.SynapseMeta, cpu float32, ram int64) error {
	s.logger.Debugf("capturing resources for orgID %s synapseID %s, cpu: %f ram: %d ", synapseMeta.OrgID, synapseMeta.ID, cpu, ram)
	err := s.synapseStore.UpdateSynapseResources(context.TODO(), synapseMeta, -(float64(cpu)), -(ram))
	if err != nil {
		return err
	}
	synapseMeta.CPU -= cpu
	synapseMeta.RAM -= ram
	return nil
}

func (s *synapseManager) ReleaseResources(synapseMeta *core.SynapseMeta, cpu float32, ram int64) error {
	s.logger.Debugf("releasing resources for orgID %s synapseID %s, cpu: %f ram: %d ", synapseMeta.OrgID, synapseMeta.ID, cpu, ram)
	err := s.synapseStore.UpdateSynapseResources(context.TODO(), synapseMeta, float64(cpu), ram)
	if err != nil {
		return err
	}
	synapseMeta.CPU += cpu
	synapseMeta.RAM += ram
	return nil
}

func (s *synapseManager) UpdateJobStatus(synapseMeta *core.SynapseMeta, jobInfo *core.JobInfo) error {
	switch jobInfo.Status {
	case core.JobCompleted:
		s.completeJob(synapseMeta, jobInfo)
		s.logger.Debugf("completed orgID %s buildID %s jobID %s on synapseID %s with mode %s", synapseMeta.OrgID, jobInfo.BuildID,
			jobInfo.ID, synapseMeta.ID, jobInfo.Mode)
	case core.JobStarted:
		s.logger.Debugf("Started orgID %s buildID %s jobID %s on synapseID %s with mode %s", synapseMeta.OrgID, jobInfo.BuildID,
			jobInfo.ID, synapseMeta.ID, jobInfo.Mode)
	case core.JobInitiated:
		s.intiateJob(synapseMeta, jobInfo)
		s.logger.Debugf("Initiated orgID %s buildID %s jobID %s on synapseID %s with mode %s", synapseMeta.OrgID, jobInfo.BuildID,
			jobInfo.ID, synapseMeta.ID, jobInfo.Mode)
	case core.JobFailed:
		s.failedJob(synapseMeta, jobInfo)
		s.logger.Errorf("job failed with orgID %s  buildID %s jobID %s on synapseID %s with mode : %+v", synapseMeta.OrgID,
			jobInfo.BuildID, jobInfo.ID, synapseMeta.ID, jobInfo.Mode)
	default:
		return errors.New("job update status not found")
	}
	return nil
}

func (s *synapseManager) SendMessage(synapseMeta *core.SynapseMeta, message *core.Message) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = synapseMeta.Connection.WriteMessage(websocket.TextMessage, messageJSON)
	if err != nil {
		return err
	}
	return nil
}

// private helper functions
func (s *synapseManager) intiateJob(synapseMeta *core.SynapseMeta, jobInfo *core.JobInfo) {
	synapseJobInfo := core.SynapseJobInfo{
		BuildID: jobInfo.BuildID,
		Mode:    jobInfo.Mode,
		ID:      jobInfo.ID,
		Status:  core.JobInitiated,
	}

	synapseMeta.Lock.Lock()
	defer synapseMeta.Lock.Unlock()
	synapseMeta.Jobs = append(synapseMeta.Jobs, synapseJobInfo)
	if err := s.synapseStore.UpdateSynapseJobs(s.ctx, synapseMeta); err != nil {
		s.logger.Errorf("job update on redis for orgID %s buildID %s jobID %s failed with error %s", synapseMeta.OrgID, jobInfo.BuildID,
			jobInfo.ID, err)
	}
}

func (s *synapseManager) failedJob(synapseMeta *core.SynapseMeta, jobInfo *core.JobInfo) {
	s.removeJobFromsynapseMeta(synapseMeta, jobInfo)
	// TODO : message should come from  jobInfo.Message
	errMsg := fmt.Sprintf("%s: %s", dockerRunFailedMessage, jobInfo.Message)
	s.MarkJobFailed(jobInfo, errMsg)
}
func (s *synapseManager) MarkJobFailed(jobInfo *core.JobInfo, remarks string) {
	build, err := s.buildStore.FindByBuildID(s.ctx, jobInfo.BuildID)
	if err != nil {
		s.logger.Errorf("Error finding entry in build table for  buildID %s", jobInfo.BuildID)
		return
	}
	// We are storing all nucleus jobs in redis even the jobs are completed
	// as we need to run all nucleus and coverage jobs on same machine
	// TODO : after coverage mode will be enabled add logic to remove all jobs from
	// redis if coverage job completed
	if build.Status != core.BuildInitiating && build.Status != core.BuildRunning {
		s.logger.Infof("Build with buildID %s is neither in initiating nor in running state, not marking it as failure", jobInfo.BuildID)
		return
	}
	buildCache, err := s.buildStore.GetBuildCache(s.ctx, jobInfo.BuildID)
	if err != nil {
		s.logger.Errorf("Error finding build cache for buildID %s", jobInfo.BuildID)
		return
	}
	repo, err := s.repoStore.FindByID(s.ctx, buildCache.RepoID)
	if err != nil {
		s.logger.Infof("Error while finding repo for id : %s, buildID %s jobID %s", buildCache.RepoID, jobInfo.BuildID, jobInfo.ID)
		return
	}
	updateRemarks := true
	if buildCache.BuildFailedRemarks != "" {
		updateRemarks = false
	}
	s.handleFailedTestExecJob(s.ctx, jobInfo, repo, s.taskStore,
		s.taskQueueUtils, s.taskQueueManager, remarks, updateRemarks, s.logger)
}

func (s *synapseManager) removeJobFromsynapseMeta(synapseMeta *core.SynapseMeta, jobInfo *core.JobInfo) {
	synapseJobInfo := core.SynapseJobInfo{
		BuildID: jobInfo.BuildID,
		Mode:    jobInfo.Mode,
		ID:      jobInfo.ID,
	}

	synapseMeta.Lock.Lock()
	defer synapseMeta.Lock.Unlock()
	index := s.findIndex(synapseMeta, &synapseJobInfo)
	if index == -1 {
		return
	}
	synapseMeta.Jobs = removeAt(synapseMeta.Jobs, index)
	if err := s.synapseStore.UpdateSynapseJobs(s.ctx, synapseMeta); err != nil {
		s.logger.Errorf("Synapse job update on redis failed for synapseID %s orgID %s  with error %v",
			synapseMeta.ID, synapseMeta.OrgID, err)
	}
}

func (s *synapseManager) findIndex(synapseMeta *core.SynapseMeta, synapseJobInfo *core.SynapseJobInfo) int {
	index := -1
	for idx, job := range synapseMeta.Jobs {
		if job.ID == synapseJobInfo.ID && job.BuildID == synapseJobInfo.BuildID {
			index = idx
		}
	}
	return index
}

func (s *synapseManager) completeJob(synapseMeta *core.SynapseMeta, jobInfo *core.JobInfo) {
	// any other task than test execution , ex - coverage
	if jobInfo.Mode != TestExecutionMode {
		s.removeJobFromsynapseMeta(synapseMeta, jobInfo)
	} else {
		// if we are retaining the job in redis and in memory , then we must update the status
		// of job
		synapseJobInfo := core.SynapseJobInfo{
			BuildID: jobInfo.BuildID,
			Mode:    jobInfo.Mode,
			ID:      jobInfo.ID,
		}
		if idx := s.findIndex(synapseMeta, &synapseJobInfo); idx != -1 {
			synapseMeta.Lock.Lock()
			defer synapseMeta.Lock.Unlock()
			synapseMeta.Jobs[idx].Status = core.JobCompleted
			if err := s.synapseStore.UpdateSynapseJobs(s.ctx, synapseMeta); err != nil {
				s.logger.Errorf("error while updating job info in redis for synapseID %s, orgID %s, buildID %s, taskID %s , err : %v",
					synapseMeta.ID, synapseMeta.OrgID, jobInfo.BuildID, jobInfo.ID, err)
			}
		}
	}
}
