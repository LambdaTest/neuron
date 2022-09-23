package synapse

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gorilla/websocket"
)

// constant related to synapse meta
const (
	Alive                         = "Alive"
	NotAlive                      = "NotAlive"
	synapseDisconnectErrorMessage = "Synapse disconnected during job execution, please make sure your synapse is up and running"
)

type synapsePool struct {
	ctx              context.Context
	wsToSynapse      map[*websocket.Conn]*core.SynapseMeta
	orgToSynapse     map[string][]*core.SynapseMeta
	idToSynapse      map[string]*core.SynapseMeta
	logger           lumber.Logger
	redisDB          core.RedisDB
	synapseStore     core.SynapseStore
	mutex            sync.Mutex
	buildStore       core.BuildStore
	repoStore        core.RepoStore
	taskStore        core.TaskStore
	taskQueueUtils   core.TaskQueueUtils
	taskQueueManager core.TaskQueueManager
	synapseManager   core.SynapseClientManager
	gracefulShutdown bool
}

// NewSynapsePoolManager returns new synapsePool
func NewSynapsePoolManager(ctx context.Context,
	logger lumber.Logger,
	redisDB core.RedisDB,
	synapseStore core.SynapseStore,
	buildStore core.BuildStore,
	repoStore core.RepoStore,
	taskStore core.TaskStore,
	taskQueueUtils core.TaskQueueUtils,
	taskQueueManager core.TaskQueueManager,
	synapseManager core.SynapseClientManager) core.SynapsePoolManager {
	return &synapsePool{
		ctx:              ctx,
		wsToSynapse:      make(map[*websocket.Conn]*core.SynapseMeta),
		orgToSynapse:     map[string][]*core.SynapseMeta{},
		idToSynapse:      make(map[string]*core.SynapseMeta),
		logger:           logger,
		redisDB:          redisDB,
		synapseStore:     synapseStore,
		buildStore:       buildStore,
		repoStore:        repoStore,
		taskStore:        taskStore,
		taskQueueUtils:   taskQueueUtils,
		taskQueueManager: taskQueueManager,
		synapseManager:   synapseManager,
		gracefulShutdown: false,
	}
}

func (p *synapsePool) RegisterSynapse(s *core.SynapseMeta) error {
	// if neuron is shutting down gracefully , do not register any new connections
	if p.gracefulShutdown {
		p.logger.Debugf("registeration of synapseID %s orgID %s rejected as pod is shutting down gracefully", s.ID, s.OrgID)
		return errs.GenericErrorMessage
	}
	key := getSynapseKey(s.OrgID, s.ID)
	synapse, err := p.synapseStore.GetSynapseMeta(context.Background(), key)
	if err != nil {
		return err
	}
	if synapse != nil {
		if synapse.IsAlive == Alive {
			return errs.ErrSynapseDuplicateConnection
		}
		s.Jobs = synapse.Jobs
	}

	p.logger.Debugf("registering connection synapseID %s for orgID %s", s.ID, s.OrgID)
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.wsToSynapse[s.Connection] = s
	p.orgToSynapse[s.OrgID] = append(p.orgToSynapse[s.OrgID], s)
	p.idToSynapse[s.ID] = s
	err = p.synapseStore.StoreSynapseMeta(p.ctx, s)
	if err != nil {
		p.logger.Errorf("error storing synapse metadata for synapseID %s orgID %s into redis error: %v", s.ID, s.OrgID, err)
		return err
	}
	return nil
}

func (p *synapsePool) DeRegisterSynapse(s *core.SynapseMeta) {
	err := p.synapseStore.DeleteSynapseMeta(p.ctx, s)
	if err != nil {
		p.logger.Errorf("error deleting metadata from redis for synapseID %s orgID %s error: %v",
			s.ID, s.OrgID, err)
	}
	p.markJobFailed(s)

	p.logger.Debugf("synapseID %s of orgID %s deregistered", s.ID, s.OrgID)
}

func (p *synapsePool) cleanUp(s *core.SynapseMeta) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.wsToSynapse, s.Connection)
	delete(p.idToSynapse, s.ID)
	if err := p.deleteFromOrgToSynapse(s); err != nil {
		p.logger.Errorf("error deleting synapseID %s of orgID %s error : %v", s.ID, s.OrgID, err)
		return
	}
}

func (p *synapsePool) markJobFailed(s *core.SynapseMeta) {
	for _, job := range s.Jobs {
		// do not mark the job failed if it is already completed
		if job.Status == core.JobCompleted {
			continue
		}
		jobInfo := core.JobInfo{
			Status:  core.StatusType(core.Failed),
			JobID:   job.ID,
			ID:      job.ID,
			Mode:    job.Mode,
			BuildID: job.BuildID,
		}
		p.logger.Infof("Marking orgID %s buildID %s and jobID %s failure", s.OrgID, jobInfo.BuildID, jobInfo.ID)
		p.synapseManager.MarkJobFailed(&jobInfo, synapseDisconnectErrorMessage)
	}
}

func (p *synapsePool) deleteFromOrgToSynapse(s *core.SynapseMeta) error {
	index := -1
	for key, value := range p.orgToSynapse[s.OrgID] {
		if value == s {
			index = key
		}
	}
	if index < 0 {
		return errors.New("no synapse found in org pool")
	}
	if index >= len(p.orgToSynapse[s.OrgID]) {
		return errors.New("found index is greater than array")
	}
	newList := make([]*core.SynapseMeta, 0)
	newList = append(newList, p.orgToSynapse[s.OrgID][:index]...)
	if index != len(p.orgToSynapse[s.OrgID])-1 {
		newList = append(newList, p.orgToSynapse[s.OrgID][index+1:]...)
	}
	p.orgToSynapse[s.OrgID] = newList
	return nil
}

func (p *synapsePool) GetSynapseFromWS(ws *websocket.Conn) *core.SynapseMeta {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.wsToSynapse[ws]
}

func (p *synapsePool) GetSynapseFromID(synapseID string) (*core.SynapseMeta, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if s, ok := p.idToSynapse[synapseID]; ok {
		return s, true
	}
	return nil, false
}

func (p *synapsePool) FindExecutor(r *core.RunnerOptions) (*core.SynapseMeta, error) {
	// change this with actual resources
	spec := GetResources(r.Tier)
	synapseList, err := p.synapseStore.ListSynapseMeta(p.ctx, r.OrgID)
	if err != nil {
		p.logger.Errorf("error fetching list of synapse")
	}
	for _, val := range synapseList {
		s, err := p.synapseStore.GetSynapseMeta(p.ctx, val)
		if err != nil {
			p.logger.Errorf("error fetching synapseID %s, of orgID %s error: %v", s.ID, s.OrgID, err)
			return nil, err
		}
		// this should not happen , but still keeping the null check
		if s == nil {
			continue
		}
		if s.IsAlive == NotAlive {
			// skip the synapses which are offline
			continue
		}
		// run all the nucleus and coverage task on same synapse
		if r.PodType == core.CoveragePod || r.PodType == core.NucleusPod {
			for _, j := range s.Jobs {
				// schedule all task of a build on same synapse
				if j.BuildID == r.Label["build-id"] {
					if s.CPU >= spec.CPU && s.RAM >= spec.RAM {
						return s, nil
					}
					return nil, errs.ErrSynapseNotFound
				}
			}
		}
		if s.CPU >= spec.CPU && s.RAM >= spec.RAM {
			return s, nil
		}
	}
	return nil, errs.ErrSynapseNotFound
}

func (p *synapsePool) MonitorConnection(ctx context.Context, synapse *core.SynapseMeta) {
	ticker := time.NewTicker(PingInterval)
	normalCloser := make(chan struct{}, 1)
	setPongHandler(synapse)
	defer p.checkAndDeRegister(synapse, normalCloser, ticker)
	for {
		select {
		case <-synapse.AbortConnection:
			p.logger.Warnf("Abort connection detected for synapseID %s orgID %s", synapse.ID, synapse.OrgID)
			normalCloser <- struct{}{}
			return
		case <-ticker.C:
			if err := synapse.Connection.SetWriteDeadline(time.Now().Add(WsWait)); err != nil {
				synapse.IsAlive = NotAlive
				synapse.LastAliveTime = time.Now().Unix()
				key := getSynapseKey(synapse.OrgID, synapse.ID)
				synapseRedis, errRedis := p.synapseStore.GetSynapseMeta(ctx, key)
				if errRedis != nil {
					p.logger.Errorf("Error reading synapseMeta from redis for synpaseID %s orgID %s error: %v", synapse.ID, synapse.OrgID, errRedis)
					return
				}
				if synapseRedis != nil && synapseRedis.NeuronID == synapse.NeuronID {
					if err := p.synapseStore.StoreSynapseMeta(ctx, synapse); err != nil {
						p.logger.Errorf("Failed to update synapseID %s orgID %s with isalive %s", synapse.ID, synapse.OrgID, NotAlive)
					}
				}
				return
			}
			if err := synapse.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				synapse.IsAlive = NotAlive
				synapse.LastAliveTime = time.Now().Unix()
				key := getSynapseKey(synapse.OrgID, synapse.ID)
				synapseRedis, errRedis := p.synapseStore.GetSynapseMeta(ctx, key)
				if errRedis != nil {
					p.logger.Errorf("Error reading synapseMeta from redis for synpaseID %s orgID %s error : %v", synapse.ID, synapse.OrgID, errRedis)
					return
				}
				if synapseRedis != nil && synapseRedis.NeuronID == synapse.NeuronID {
					if err := p.synapseStore.StoreSynapseMeta(ctx, synapse); err != nil {
						p.logger.Errorf("Failed to update synapseID %s of orgID %s with isalive %s , error: %v", synapse.ID, synapse.OrgID, NotAlive, err)
					}
				}
				return
			}
		}
	}
}

func (p *synapsePool) checkAndDeRegister(synapse *core.SynapseMeta, normalCloser chan struct{}, ticker *time.Ticker) {
	p.logger.Debugf("closing the ping sender go routine for synapseID %s orgID %s", synapse.ID, synapse.OrgID)
	synapse.Connection.Close()
	p.cleanUp(synapse)
	ticker.Stop()
	select {
	case <-normalCloser:
		p.logger.Warnf("normal closer for orgID %s synpaseID %s", synapse.OrgID, synapse.ID)
		p.DeRegisterSynapse(synapse)

	case <-time.After(DeRegisterWaitDuration):
		key := getSynapseKey(synapse.OrgID, synapse.ID)
		synapseRedis, errRedis := p.synapseStore.GetSynapseMeta(context.Background(), key)
		if errRedis != nil {
			p.logger.Errorf("Error reading synapseMeta from redis for synpaseID %s orgID %s with error %v", synapse.ID, synapse.OrgID, errRedis)
			return
		}
		if synapseRedis == nil {
			return
		}
		if synapseRedis.IsAlive == NotAlive && synapseRedis.NeuronID == synapse.NeuronID {
			lastAliveTime := time.Unix(synapseRedis.LastAliveTime, 0)
			if time.Since(lastAliveTime) >= DeRegisterWaitDuration {
				p.DeRegisterSynapse(synapse)
			}
		}
		if synapseRedis.NeuronID != synapse.NeuronID {
			return
		}
	}
}

func (p *synapsePool) SetAllSynapseToNotAlive(ctx context.Context) {
	<-ctx.Done()
	p.gracefulShutdown = true
	p.logger.Warnf("Setting all register synapse to offline")
	for _, synapse := range p.idToSynapse {
		if err := synapse.Connection.Close(); err != nil {
			p.logger.Debugf("error occurred while closing the synapse connections, error %v", err)
		}
		key := getSynapseKey(synapse.OrgID, synapse.ID)
		p.logger.Infof("SynapseID %s orgID %s will be set as offline in redis", synapse.ID, synapse.OrgID)
		synapseRedis, err := p.synapseStore.GetSynapseMeta(context.TODO(), key)
		if err != nil {
			p.logger.Errorf("Error reading synapseMeta from redis for synpaseID %s orgID %s error %s", synapse.ID, synapse.OrgID, err)
			return
		}
		if synapseRedis != nil && synapseRedis.NeuronID == synapse.NeuronID {
			synapse.IsAlive = NotAlive
			if err := p.synapseStore.StoreSynapseMeta(context.TODO(), synapse); err != nil {
				p.logger.Errorf("Error updating synapseID %s orgID %s with error : %v", synapse.ID, synapse.OrgID, err)
			}
		}
	}
}

func setPongHandler(synapse *core.SynapseMeta) {
	synapse.Connection.SetReadLimit(MaxMessageSize)
	synapse.Connection.SetPongHandler(func(string) error {
		if err := synapse.Connection.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
			return err
		}
		return nil
	})
}
