package handler

import (
	"net/http"
	"strconv"

	"log-analysis-platform/service"

	"github.com/gin-gonic/gin"
)

// GetLogs returns paginated log entries
func GetLogs(c *gin.Context) {
	project := c.Query("project")
	svc := c.Query("service")
	callerFile := c.Query("caller_file")
	job := c.Query("job")
	keyword := c.Query("keyword")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	start, end, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries, total, err := service.DefaultLokiService.QueryLogs(project, svc, job, callerFile, keyword, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  entries,
		"total": total,
	})
}
