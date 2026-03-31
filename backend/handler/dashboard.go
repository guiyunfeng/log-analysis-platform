package handler

import (
	"net/http"
	"strconv"
	"time"

	"log-analysis-platform/service"

	"github.com/gin-gonic/gin"
)

// GetErrorTrend returns error trend time series data
func GetErrorTrend(c *gin.Context) {
	project := c.Query("project")
	svc := c.Query("service")
	job := c.Query("job")
	callerFile := c.Query("caller_file")
	step := c.DefaultQuery("step", "5m")

	start, end, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	series, err := service.DefaultLokiService.QueryErrorTrend(project, svc, job, callerFile, start, end, step)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": series})
}

// GetErrorSummary returns error summary by various dimensions
func GetErrorSummary(c *gin.Context) {
	project := c.Query("project")
	svc := c.Query("service")
	job := c.Query("job")

	start, end, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	summary, err := service.DefaultLokiService.QueryErrorSummary(project, svc, job, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

// GetTopNServices returns top N services by error count
func GetTopNServices(c *gin.Context) {
	project := c.Query("project")
	svc := c.Query("service")
	job := c.Query("job")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	start, end, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := service.DefaultLokiService.QueryTopNServices(project, svc, job, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

// GetTopNCallers returns top N caller files by error count
func GetTopNCallers(c *gin.Context) {
	project := c.Query("project")
	svc := c.Query("service")
	job := c.Query("job")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	start, end, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := service.DefaultLokiService.QueryTopNCallers(project, svc, job, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

func parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	now := time.Now()
	endDefault := now.Unix()
	startDefault := now.Add(-1 * time.Hour).Unix()

	startStr := c.DefaultQuery("start", strconv.FormatInt(startDefault, 10))
	endStr := c.DefaultQuery("end", strconv.FormatInt(endDefault, 10))

	startUnix, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endUnix, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return time.Unix(startUnix, 0), time.Unix(endUnix, 0), nil
}
