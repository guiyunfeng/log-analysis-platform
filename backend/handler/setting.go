package handler

import (
	"net/http"

	"log-analysis-platform/config"
	"log-analysis-platform/model"
	"log-analysis-platform/service"

	"github.com/gin-gonic/gin"
)

// GetSettings returns all settings
func GetSettings(c *gin.Context) {
	var settings []model.Setting
	if err := db.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := map[string]string{}
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	// Also return connection info
	result["loki_url"] = config.GlobalConfig.LokiURL
	result["dingtalk_webhook"] = config.GlobalConfig.DingTalkWebhook

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// UpdateSettings updates settings
func UpdateSettings(c *gin.Context) {
	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for key, value := range updates {
		setting := model.Setting{Key: key, Value: value}
		if err := db.Where(model.Setting{Key: key}).Assign(model.Setting{Value: value}).FirstOrCreate(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Update Loki URL dynamically if changed
		if key == "loki_url" {
			config.GlobalConfig.LokiURL = value
			service.DefaultLokiService = service.NewLokiService(value)
		}
		if key == "dingtalk_webhook" {
			config.GlobalConfig.DingTalkWebhook = value
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
