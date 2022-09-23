package ws

import (
	"context"
	"encoding/json"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func processIncomingMessage(
	ws *websocket.Conn,
	msg string,
	logger lumber.Logger,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
	abortConnection chan struct{},
) {
	var message core.Message
	err := json.Unmarshal([]byte(msg), &message)
	if err != nil {
		logger.Errorf("error in unmarshaling core.message %v", err)
		return
	}

	switch message.Type {
	case core.MsgTask, core.MsgError, core.MsgInfo:
		break
	case core.MsgLogin:
		if err := processLogin(ws, logger, message, synapseManager, synapsePool); err != nil {
			abortConnection <- struct{}{}
			break
		}
	case core.MsgLogout:
		processLogout(ws, logger, synapsePool)
	case core.MsgJobInfo:
		processJobInfo(ws, logger, message, synapseManager, synapsePool)
	case core.MsgResourceStats:
		processResourceStats(ws, logger, message, synapseManager, synapsePool)
	}
}

func processLogin(
	ws *websocket.Conn,
	logger lumber.Logger,
	message core.Message,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager) error {
	var loginDetails core.LoginDetails
	err := json.Unmarshal(message.Content, &loginDetails)
	if err != nil {
		return err
	}

	synapseMeta := synapseManager.NewClient(ws, loginDetails)
	// Do not register synapse with specs less than minimum requirement
	if !synapse.CheckSynpaseMinRequirement(loginDetails.CPU, loginDetails.RAM) {
		logger.Errorf("synapseID %s does not meet the min requirement: %v, actual spec {CPU %f RAM %d}",
			loginDetails.SynapseID, synapse.TierOpts[core.Small], loginDetails.CPU, loginDetails.RAM)
		reqErr := errs.ErrSynapseMinRequirement
		errMsg := wsError(reqErr.Error())
		if err := synapseManager.SendMessage(synapseMeta, &errMsg); err != nil {
			logger.Errorf("error sending message to synapseID %s of orgID %s error :%v", synapseMeta.ID, synapseMeta.OrgID, err)
			return err
		}
		return reqErr
	}
	isAuthenticated := synapseManager.AuthenticateClient(synapseMeta)
	if isAuthenticated {
		if err := synapsePool.RegisterSynapse(synapseMeta); err != nil {
			var errMsg core.Message
			if errors.Is(errs.ErrSynapseDuplicateConnection, err) {
				errMsg = wsError(err.Error())
			} else {
				errMsg = wsError(errs.GenericErrorMessage.Error())
			}
			if errSend := synapseManager.SendMessage(synapseMeta, &errMsg); errSend != nil {
				logger.Errorf("error sending message to synapseID %s of orgID %s error :%v", synapseMeta.ID, synapseMeta.OrgID, errSend)
			}
			if synapseMeta != nil {
				logger.Errorf("Error occurred in login for orgID %s synapseID %s, error: %v", synapseMeta.OrgID, synapseMeta.ID, err)
			}

			return err
		}
		infoMsg := wsInfo(AuthenticationPassedMsg)
		if err := synapseManager.SendMessage(synapseMeta, &infoMsg); err != nil {
			logger.Errorf("error sending message to synapseID %s of orgID %s error :%v", synapseMeta.ID, synapseMeta.OrgID, err)
			return err
		}
		go synapsePool.MonitorConnection(context.Background(), synapseMeta)
	} else {
		errAuth := errs.ErrSynapseAuthFailed
		errMsg := wsError(errAuth.Error())
		if err := synapseManager.SendMessage(synapseMeta, &errMsg); err != nil {
			logger.Errorf("error sending message to synapseID %s of orgID %s error :%v", synapseMeta.ID, synapseMeta.OrgID, err)
			return err
		}
		if synapseMeta != nil {
			logger.Errorf("Error occurred in login for orgID %s synapseID %s, error: %v", synapseMeta.OrgID, synapseMeta.ID, errAuth)
		}
		return errAuth
	}
	return nil
}

func processLogout(
	ws *websocket.Conn,
	logger lumber.Logger,
	synapsePool core.SynapsePoolManager,
) {
	synapseMeta := synapsePool.GetSynapseFromWS(ws)
	if synapseMeta == nil {
		return
	}
	synapseMeta.AbortConnection <- true
	logger.Debugf("synapseID %s logged out from orgID %s", synapseMeta.ID, synapseMeta.OrgID)
}

func processResourceStats(
	ws *websocket.Conn,
	logger lumber.Logger,
	message core.Message,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
) {
	var resourceStats core.ResourceStats
	err := json.Unmarshal(message.Content, &resourceStats)
	if err != nil {
		logger.Errorf("Error during unmarshal of ResourceStats message %s", err.Error())
		return
	}
	synapseMeta := synapsePool.GetSynapseFromWS(ws)
	if synapseMeta == nil {
		logger.Errorf("No synapse found for connected websocket")
		return
	}
	logger.Debugf("Resource update message received for orgID %s synapseID %s, message: %+v",
		synapseMeta.OrgID, synapseMeta.ID, resourceStats)
	if resourceStats.Status == core.ResourceRelease {
		if err := synapseManager.ReleaseResources(synapseMeta, resourceStats.CPU, resourceStats.RAM); err != nil {
			logger.Errorf("error releasing resources for synapseID %s of orgID %s, error %v", synapseMeta.ID, synapseMeta.OrgID, err)
		}
	}
}

func processJobInfo(
	ws *websocket.Conn,
	logger lumber.Logger,
	message core.Message,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
) {
	var jobInfo core.JobInfo
	err := json.Unmarshal(message.Content, &jobInfo)
	if err != nil {
		logger.Errorf("Error in unmarshal logs %s", err.Error())
		return
	}

	synapseMeta := synapsePool.GetSynapseFromWS(ws)
	if synapseMeta == nil {
		logger.Errorf("No synapse found for buildID %s jobID %s ", jobInfo.BuildID, jobInfo.ID)
		return
	}
	if err := synapseManager.UpdateJobStatus(synapseMeta, &jobInfo); err != nil {
		logger.Errorf("failed to update jobID %s status, buildID %s error: %s", jobInfo.ID, jobInfo.BuildID, err.Error())
	}
}
