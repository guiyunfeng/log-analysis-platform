package handler

import (
	"net/http"
	"strconv"
	"time"

	"log-analysis-platform/model"

	"github.com/gin-gonic/gin"
)

// ListAlertHistory returns paginated alert history
func ListAlertHistory(c *gin.Context) {
	severity := c.Query("severity")
	svc := c.Query("service")
	resolved := c.Query("resolved")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := db.Model(&model.AlertHistory{})
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if svc != "" {
		query = query.Where("service = ?", svc)
	}
	if resolved == "true" {
		query = query.Where("resolved = ?", true)
	} else if resolved == "false" {
		query = query.Where("resolved = ?", false)
	}

	// Time range filter
	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr != "" {
		startUnix, err := strconv.ParseInt(startStr, 10, 64)
		if err == nil {
			query = query.Where("created_at >= ?", time.Unix(startUnix, 0))
		}
	}
	if endStr != "" {
		endUnix, err := strconv.ParseInt(endStr, 10, 64)
		if err == nil {
			query = query.Where("created_at <= ?", time.Unix(endUnix, 0))
		}
	}

	var total int64
	query.Count(&total)

	var history []model.AlertHistory
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      history,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ResolveAlertHistory marks an alert history as resolved
func ResolveAlertHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	now := time.Now()
	if err := db.Model(&model.AlertHistory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": &now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resolved"})
}
