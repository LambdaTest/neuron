package synapse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform synapse transaction"
)

type synapseStore struct {
	redisDB core.RedisDB
	logger  lumber.Logger
	db      core.DB
}

type synapseRedis struct {
	ID             string  `redis:"id"`
	TotalRAM       int64   `redis:"total_ram"`
	TotalCPU       float32 `redis:"total_cpu"`
	IsAlive        string  `redis:"isalive"`
	CPU            float32 `redis:"cpu"`
	RAM            int64   `redis:"ram"`
	Jobs           string  `redis:"jobs"`
	NeuronID       string  `redis:"neuron_id"`
	LastAliveTime  int64   `redis:"last_alive_time"`
	SynapseVersion string  `redis:"synapse_version"`
}

// constant related to synapse status in redis
const (
	Alive    = "Alive"
	NotAlive = "NotAlive"
)

// Resource represents resource of synpase
type Resource struct {
	CPU float32
	RAM int64
}

// New returns new synapseStore
func New(redisDB core.RedisDB, logger lumber.Logger, db core.DB) core.SynapseStore {
	return &synapseStore{redisDB: redisDB, logger: logger, db: db}
}

func (s *synapseStore) CountSynapse(ctx context.Context, orgID string) (*core.SynapseStatusCount, error) {
	synapseCount := new(core.SynapseStatusCount)
	err := s.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"org_id": orgID,
		}
		rows, err := db.NamedQueryContext(ctx, countSynapseQuery, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		if rows.Err() != nil {
			return rows.Err()
		}
		if rows.Next() {
			if scanErr := rows.StructScan(synapseCount); scanErr != nil {
				return errs.SQLError(scanErr)
			}
			return nil
		}
		return errs.ErrRowsNotFound
	})
	return synapseCount, err
}

func (s *synapseStore) CreateOrUpdate(ctx context.Context, tx *sqlx.Tx, synapseMeta *core.SynapseMeta) error {
	now := time.Now()
	synapseData := core.Synapse{
		ID:             synapseMeta.ID,
		OrgID:          synapseMeta.OrgID,
		Name:           synapseMeta.Name,
		TotalCPU:       synapseMeta.TotalCPU,
		TotalRAM:       synapseMeta.TotalRAM,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
		SynapseVersion: synapseMeta.SynapseVersion,
	}
	synapseData.Status = core.Connected
	if synapseMeta.IsAlive == NotAlive {
		synapseData.Status = core.Disconnected
	}
	if _, err := tx.NamedExecContext(ctx, insertQuery, synapseData); err != nil {
		return err
	}
	return nil
}

func (s *synapseStore) UpdateIsActiveSynapse(ctx context.Context, isActive bool, synapseID, orgID string) error {
	return s.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"id":        synapseID,
			"org_id":    orgID,
			"is_active": isActive,
		}
		if _, err := db.NamedExecContext(ctx, isActiveUpdateQuery, args); err != nil {
			s.logger.Errorf("Error occurred in deletion of synapseID %s orgID %, error %v", synapseID, orgID, err)
			return errs.SQLError(err)
		}
		return nil
	})
}

func (s *synapseStore) StoreSynapseMeta(ctx context.Context, synapseMeta *core.SynapseMeta) error {
	jobs, err := json.Marshal(synapseMeta.Jobs)
	if err != nil {
		return err
	}
	synapseMetaMap := map[string]interface{}{
		"id":              synapseMeta.ID,
		"total_cpu":       synapseMeta.TotalCPU,
		"total_ram":       synapseMeta.TotalRAM,
		"cpu":             synapseMeta.CPU,
		"ram":             synapseMeta.RAM,
		"jobs":            jobs,
		"isalive":         synapseMeta.IsAlive,
		"neuron_id":       synapseMeta.NeuronID,
		"last_alive_time": synapseMeta.LastAliveTime,
		"synapse_version": synapseMeta.SynapseVersion,
	}
	key := getSynapseKey(synapseMeta.OrgID, synapseMeta.ID)
	return s.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		if errDB := s.CreateOrUpdate(ctx, tx, synapseMeta); errDB != nil {
			return errDB
		}
		_, err = s.redisDB.Client().Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.HSet(ctx, key, synapseMetaMap)
			return nil
		})
		return err
	})
}

func (s *synapseStore) UpdateSynapseResources(ctx context.Context, synapseMeta *core.SynapseMeta, cpu float64, ram int64) error {
	key := getSynapseKey(synapseMeta.OrgID, synapseMeta.ID)
	_, err := s.redisDB.Client().Pipelined(ctx, func(pipe redis.Pipeliner) error {
		synapseRedis, errRedis := s.GetSynapseMeta(ctx, key)
		if errRedis != nil {
			return errRedis
		}
		if synapseRedis == nil {
			errMsg := fmt.Sprintf("no entry found in redis for synapseID %s orgID %s", synapseMeta.ID, synapseMeta.OrgID)
			s.logger.Errorf(errMsg)
			return errs.New(errMsg)
		}
		resultantCPU := synapseRedis.CPU + float32(cpu)
		resultantRAM := synapseRedis.RAM + ram
		if resultantCPU < 0 || resultantCPU > synapseRedis.TotalCPU || resultantRAM < 0 || resultantRAM > synapseRedis.TotalRAM {
			errMsg := fmt.Sprintf("invalid synapse redis update for orgID %s synapseID %s, resultant cpu: %f ram: %d, actual cpu %f, ram %d",
				synapseMeta.OrgID, synapseMeta.ID, resultantCPU, resultantRAM, synapseRedis.TotalCPU, synapseRedis.TotalRAM)
			s.logger.Errorf(errMsg)
			return errs.New(errMsg)
		}
		pipe.HIncrByFloat(ctx, key, "cpu", cpu)
		pipe.HIncrBy(ctx, key, "ram", ram)
		return nil
	})
	return err
}

func (s *synapseStore) UpdateSynapseJobs(ctx context.Context, synapseMeta *core.SynapseMeta) error {
	key := getSynapseKey(synapseMeta.OrgID, synapseMeta.ID)

	exists, err := s.redisDB.Client().Exists(ctx, key).Result()
	if exists == 0 {
		return errs.ErrRedisKeyNotFound
	}
	if err != nil {
		return err
	}
	jobs, err := json.Marshal(synapseMeta.Jobs)
	if err != nil {
		return err
	}
	_, err = s.redisDB.Client().Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, "jobs", jobs)
		return nil
	})
	return err
}

func (s *synapseStore) GetSynapseMeta(ctx context.Context, synapseID string) (*core.SynapseMeta, error) {
	var synapseRedisInput synapseRedis
	exists, err := s.redisDB.Client().Exists(ctx, synapseID).Result()
	if exists == 0 || err != nil {
		return nil, err
	}
	if err := s.redisDB.Client().HGetAll(ctx, synapseID).Scan(&synapseRedisInput); err != nil {
		return nil, err
	}
	jobs := []core.SynapseJobInfo{}
	if err := json.Unmarshal([]byte(synapseRedisInput.Jobs), &jobs); err != nil {
		return nil, err
	}
	return &core.SynapseMeta{
		CPU:            synapseRedisInput.CPU,
		RAM:            synapseRedisInput.RAM,
		ID:             synapseRedisInput.ID,
		Jobs:           jobs,
		IsAlive:        synapseRedisInput.IsAlive,
		TotalRAM:       synapseRedisInput.TotalRAM,
		TotalCPU:       synapseRedisInput.TotalCPU,
		NeuronID:       synapseRedisInput.NeuronID,
		LastAliveTime:  synapseRedisInput.LastAliveTime,
		SynapseVersion: synapseRedisInput.SynapseVersion,
	}, nil
}

func (s *synapseStore) ListSynapseMeta(ctx context.Context, orgID string) ([]string, error) {
	key := getSynapseKey(orgID, "*")
	result, err := s.redisDB.Client().Keys(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *synapseStore) DeleteSynapseMeta(ctx context.Context, synapseMeta *core.SynapseMeta) error {
	return s.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		synapseMeta.IsAlive = NotAlive
		if errDB := s.CreateOrUpdate(ctx, tx, synapseMeta); errDB != nil {
			return errDB
		}
		key := getSynapseKey(synapseMeta.OrgID, synapseMeta.ID)
		_, err := s.redisDB.Client().Del(ctx, key).Result()
		return err
	})
}

func getSynapseKey(orgID, synapseID string) string {
	return fmt.Sprintf("synapse-%s-%s", orgID, synapseID)
}

func (s *synapseStore) GetSynapseList(ctx context.Context, orgID string) ([]*core.Synapse, error) {
	synapseList := make([]*core.Synapse, 0)
	synapseIDList, err := s.ListSynapseMeta(ctx, orgID)
	if err != nil {
		return synapseList, err
	}
	synapseMap := map[string]Resource{}
	for _, synapseID := range synapseIDList {
		meta, redisErr := s.GetSynapseMeta(ctx, synapseID)
		if redisErr != nil {
			return synapseList, redisErr
		}
		// this should not happen but still keeping the null check here
		if meta == nil {
			continue
		}
		synapseMap[synapseID] = Resource{CPU: meta.CPU, RAM: meta.RAM}
	}
	err = s.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"org_id": orgID,
		}
		rows, dbErr := db.NamedQueryContext(ctx, findAllSynapseQuery, args)
		if err != nil {
			return errs.SQLError(dbErr)
		}
		for rows.Next() {
			sc := new(core.Synapse)
			if scanErr := rows.StructScan(sc); err != nil {
				return errs.SQLError(scanErr)
			}
			key := getSynapseKey(orgID, sc.ID)
			if val, ok := synapseMap[key]; ok {
				sc.AvailableRAM = val.RAM
				sc.AvailableCPU = val.CPU
			} else {
				sc.AvailableRAM = sc.TotalRAM
				sc.AvailableCPU = sc.TotalCPU
			}
			synapseList = append(synapseList, sc)
		}
		return nil
	})
	return synapseList, err
}

func (s *synapseStore) TestSynapseConnection(ctx context.Context, orgID string) (bool, error) {
	synapses, err := s.ListSynapseMeta(ctx, orgID)
	if err != nil {
		return false, err
	}
	if len(synapses) > 0 {
		return true, nil
	}
	return false, nil
}

const insertQuery = `INSERT 
INTO synapse(
	id,
	org_id,
	name,
	total_ram_mib,
	total_cpu_core,
	status,
	is_active,
	created_at,
	updated_at,
	synapse_version
	)
VALUES (
	:id,
	:org_id,
	:name,
	:total_ram_mib,
	:total_cpu_core,
	:status,
	:is_active,
	:created_at,
	:updated_at,
	:synapse_version
	)
ON DUPLICATE KEY UPDATE 
	name = VALUES(name),
	total_cpu_core  = VALUES(total_cpu_core),
	total_ram_mib = VALUES(total_ram_mib),
	status = VALUES(status),
	is_active = VALUES(is_active),
	updated_at = VALUES(updated_at),
	synapse_version = VALUES(synapse_version)`

const countSynapseQuery = `
SELECT 
	COUNT(CASE WHEN status = 'connected' THEN s.id END ) as connected,
	COUNT(CASE WHEN status = 'disconnected' THEN s.id END ) as disconnected,
	COUNT(1) as total_register 
FROM 
	synapse s
WHERE 
	is_active = TRUE
	AND org_id = :org_id`

const findAllSynapseQuery = `
SELECT 
	name,
	id,
	status,
	total_ram_mib ,
	total_cpu_core,
	created_at,
	updated_at,
	synapse_version
FROM 
	synapse 
WHERE 
	org_id = :org_id
	AND is_active = TRUE`

const isActiveUpdateQuery = `
UPDATE 
	synapse 
SET 
	is_active = :is_active 
WHERE 
	id = :id 
	AND org_id = :org_id`
