package core

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// SynapseMeta stores metadata for synapse
// TODO: convert job array to set.
type SynapseMeta struct {
	ID              string `redis:"id"`
	Name            string
	SecretKey       string
	OrgID           string
	IsAuthenticated bool
	CPU             float32          `redis:"cpu"`
	RAM             int64            `redis:"ram"`
	Jobs            []SynapseJobInfo `redis:"jobs"`
	IsAlive         string           `redis:"isalive"`
	TotalRAM        int64            `redis:"total_ram"`
	TotalCPU        float32          `redis:"total_cpu"`
	Connection      *websocket.Conn
	AbortConnection chan bool
	Lock            sync.Mutex
	NeuronID        string `redis:"neuron_id"`
	LastAliveTime   int64  `redis:"last_alive_time"`
	SynapseVersion  string `redis:"synapse_version"`
}

// SynapseStatus represent synapse status in db
type SynapseStatus string

// const related to SynapseStatus
const (
	Connected    SynapseStatus = "connected"
	Disconnected SynapseStatus = "disconnected"
)

// Synapse represents db entry of synapse table
type Synapse struct {
	ID             string        `db:"id" json:"id"`
	OrgID          string        `db:"org_id" json:"-,omitempty"`
	Name           string        `db:"name" json:"name"`
	TotalCPU       float32       `db:"total_cpu_core" json:"total_cpu_core"`
	TotalRAM       int64         `db:"total_ram_mib" json:"total_ram_mib"`
	Status         SynapseStatus `db:"status" json:"status,omitempty"`
	CreatedAt      time.Time     `db:"created_at" json:"first_connected,omitempty"`
	UpdatedAt      time.Time     `db:"updated_at" json:"last_connected,omitempty"`
	AvailableCPU   float32       `json:"available_cpu_core"`
	AvailableRAM   int64         `json:"available_ram_mib"`
	IsActive       bool          `db:"is_active" json:"is_active,omitempty"`
	SynapseVersion string        `db:"synapse_version" json:"synapse_version"`
}

// SynapseJobInfo describes the job description for synapse
type SynapseJobInfo struct {
	BuildID string
	Mode    string
	ID      string
	Status  StatusType
}

// SynapseTask holds info for publishing task in redis
type SynapseTask struct {
	SynapseID string
	Task      *RunnerOptions
}

// SynapseStatusCount defines synapse status count output
type SynapseStatusCount struct {
	Connected     int `db:"connected" json:"connected"`
	Disconnected  int `db:"disconnected" json:"disconnected"`
	TotalRegister int `db:"total_register" json:"total_register"`
}

// SynapseClientManager defines operations for managing synapse
type SynapseClientManager interface {
	// NewClient creates a new client
	NewClient(ws *websocket.Conn, loginDetails LoginDetails) *SynapseMeta
	// AuthenticateClient authenticates the client
	AuthenticateClient(synapseMeta *SynapseMeta) bool
	// UpdateJobStatus updates job status - attach/dettach jobs to synapse
	UpdateJobStatus(synapseMeta *SynapseMeta, jobInfo *JobInfo) error
	// CaptureResources captures resources
	CaptureResources(synapseMeta *SynapseMeta, cpu float32, ram int64) error
	// ReleaseResources releases resources
	ReleaseResources(synapseMeta *SynapseMeta, cpu float32, ram int64) error
	// SendMessage sends message
	SendMessage(synapseMeta *SynapseMeta, message *Message) error
	// MarkJobFailed marks the job failed
	MarkJobFailed(jobInfo *JobInfo, remarks string)
}

// SynapsePoolManager defines operations for managing synapsepool
type SynapsePoolManager interface {
	// RegisterSynapse registers new clients in pool
	RegisterSynapse(s *SynapseMeta) error
	// DeRegisterSynapse deregister client from pool
	DeRegisterSynapse(s *SynapseMeta)
	// GetSynapseFromWS retrieve synapse from websocket information
	GetSynapseFromWS(ws *websocket.Conn) *SynapseMeta
	// FindExecutor schedules job on synapse
	FindExecutor(r *RunnerOptions) (*SynapseMeta, error)
	// GetSynapseFromID get synapse from id
	GetSynapseFromID(synapseID string) (*SynapseMeta, bool)
	// MonitorConnection monitors the synapse for disconnections
	MonitorConnection(ctx context.Context, synapse *SynapseMeta)
	// SetAllSynapseToNotAlive sets IsAlive to NotAlive for all registered synapse when graceful timeout is called
	SetAllSynapseToNotAlive(ctx context.Context)
}

// SynapseStore defines operations for managing database for synapse
type SynapseStore interface {
	// StoreSynapseMeta stores synapse metadata in to redis
	StoreSynapseMeta(ctx context.Context, synapseMeta *SynapseMeta) error
	// UpdateSynapseResources updates cpu, memory for a synapse in redis
	UpdateSynapseResources(ctx context.Context, synapseMeta *SynapseMeta, cpu float64, ram int64) error
	// UpdateSynapseJobs updates job status in redis
	UpdateSynapseJobs(ctx context.Context, synapseMeta *SynapseMeta) error
	// GetSynapseMeta get synapse metadata from the redis
	GetSynapseMeta(ctx context.Context, synapseID string) (*SynapseMeta, error)
	// ListSynapseMeta retrives the available synapses for an organization
	ListSynapseMeta(ctx context.Context, orgID string) ([]string, error)
	// DeleteSynapseMeta delete synapse metadata from redis
	DeleteSynapseMeta(ctx context.Context, synapseMeta *SynapseMeta) error
	// GetSynapseList lists the synapse info of orgs
	GetSynapseList(ctx context.Context, orgID string) ([]*Synapse, error)
	// TestSynapseConnection checks if synapse is connected or not
	TestSynapseConnection(ctx context.Context, orgID string) (isExist bool, err error)
	// UpdateIsActiveSynapse updates is_active for given synapse in DB
	UpdateIsActiveSynapse(ctx context.Context, isActive bool, synapseID, orgID string) error
	// CountSynapse returns list of status and their count of synapse
	CountSynapse(ctx context.Context, orgID string) (*SynapseStatusCount, error)
}

// SynapseQueueManager defines operations for pub/sub for synapse
type SynapseQueueManager interface {
	// Run consumes synapse messages
	Run(ctx context.Context)
	// ScheduleTask publishes task for synapse
	ScheduleTask(r *RunnerOptions, synapseID string) error
}

// SynapseTaskQueue defines operation synapse task queue
type SynapseTaskQueue interface {
	// Enqueue enqueues task message
	Enqueue(r *RunnerOptions, createdAt time.Time) <-chan error
	// InitConsumer intiate the consumer
	InitConsumer(ctx context.Context)
}
