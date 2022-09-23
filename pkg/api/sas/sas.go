package sas

import (
	"context"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// request body for getting SAS URL API.
type request struct {
	BlobPath string `json:"blob_path"`
	BlobType string `json:"blob_type"`
}

//  response body for  get SAS URL API.
type response struct {
	SASURL string `json:"sas_url"`
}

// HandleCreate handles the request for creating SAS URL.
func HandleCreate(azureClient core.AzureBlob, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestPayload request
		if err := c.ShouldBindJSON(&requestPayload); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		var containerID int
		// only cache and logs container are supported for now
		if requestPayload.BlobType == "cache" {
			containerID = core.CacheContainer
		} else if requestPayload.BlobType == "logs" {
			containerID = core.LogsContainer
		} else if requestPayload.BlobType == "container-payload" {
			containerID = core.PayloadContainer
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid blob type"})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		sasURL, err := azureClient.GenerateSasURL(ctx, requestPayload.BlobPath, containerID)
		if err != nil {
			logger.Errorf("error while generating SAS Token, error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, &response{
			SASURL: azureClient.ReplaceWithCDN(sasURL),
		})
	}
}
