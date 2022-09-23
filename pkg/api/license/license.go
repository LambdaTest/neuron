package license

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

// HandleFind finds license pertaining to an organization
func HandleFind(licenseStore core.LicenseStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID := c.Param("orgid")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		license, err := licenseStore.Find(ctx, orgID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"message": "License not found."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"org_id":          license.OrgID,
			"license_type":    license.Type,
			"credits_bought":  license.CreditsBought,
			"credits_balance": license.CreditsBalance,
			"credits_expiry":  license.CreditsExpiry,
			"concurrency":     license.Concurrency,
			"tier":            license.Tier,
		})
	}
}

// HandleCreate creates a new license for an organization
// TODO: This should be an internal API, otherwise end-user can jack requests and get lifetime validity
func HandleCreate(licenseStore core.LicenseStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newLicense = &core.License{}
		if err := c.ShouldBindJSON(newLicense); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		newLicense.ID = utils.GenerateUUID()
		if err := licenseStore.Create(ctx, newLicense); err != nil {
			logger.Errorf("error while adding license %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"license_id": newLicense.ID})
	}
}

// HandlePut updates a license for an organization
// TODO: This should be an internal API, otherwise end-user can jack requests and get lifetime validity
func HandlePut(licenseStore core.LicenseStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var license = &core.License{}
		if err := c.ShouldBindJSON(license); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		if err := licenseStore.Update(ctx, license); err != nil {
			logger.Errorf("error while updating license %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"license_id": license.ID})
	}
}
