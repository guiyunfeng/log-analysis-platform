package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"log-analysis-platform/config"
)

// LokiQueryRangeResponse represents Loki query_range response
type LokiQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// LokiMatrixResponse represents Loki matrix query response
type LokiMatrixResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Caller    string `json:"caller"`
	Content   string `json:"content"`
	Trace     string `json:"trace"`
	Span      string `json:"span"`
	Level     string `json:"level"`
	Job       string `json:"job"`
	Project   string `json:"project"`
	Service   string `json:"service"`
}

// TrendPoint represents a single time series point
type TrendPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

// SeriesData represents a named time series
type SeriesData struct {
	Name   string       `json:"name"`
	Data   []TrendPoint `json:"data"`
	Labels map[string]string `json:"labels"`
}

// TopNItem represents a top-n result
type TopNItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
	Extra map[string]string `json:"extra"`
}

// LokiService handles all Loki API queries
type LokiService struct {
	baseURL string
	client  *http.Client
}

var DefaultLokiService *LokiService

func NewLokiService(baseURL string) *LokiService {
	return &LokiService{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func InitLoki() {
	DefaultLokiService = NewLokiService(config.GlobalConfig.LokiURL)
}

// buildSelector builds a LogQL stream selector
func buildSelector(project, service, job, callerFile, env string) string {
	selectors := []string{}
	// Always filter for error logs
	selectors = append(selectors, `logtype="error"`)
	if env != "" {
		selectors = append(selectors, fmt.Sprintf(`env="%s"`, env))
	}
	if project != "" {
		selectors = append(selectors, fmt.Sprintf(`project="%s"`, project))
	}
	if service != "" {
		selectors = append(selectors, fmt.Sprintf(`service="%s"`, service))
	}
	if job != "" {
		selectors = append(selectors, fmt.Sprintf(`job="%s"`, job))
	}
	if callerFile != "" {
		selectors = append(selectors, fmt.Sprintf(`caller_file="%s"`, callerFile))
	}
	return "{" + strings.Join(selectors, ", ") + "}"
}

// QueryLogs queries Loki for raw log entries
func (s *LokiService) QueryLogs(project, service, job, callerFile, keyword string, start, end time.Time, limit int) ([]LogEntry, int, error) {
	selector := buildSelector(project, service, job, callerFile, "")
	logQL := selector
	if keyword != "" {
		escaped := strings.ReplaceAll(keyword, `"`, `\"`)
		logQL = fmt.Sprintf(`%s |= "%s"`, selector, escaped)
	}

	params := url.Values{}
	params.Set("query", logQL)
	params.Set("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Set("end", fmt.Sprintf("%d", end.UnixNano()))
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("direction", "backward")

	apiURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", s.baseURL, params.Encode())
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, 0, fmt.Errorf("loki request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	var lokiResp LokiQueryRangeResponse
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}

	var entries []LogEntry
	for _, stream := range lokiResp.Data.Result {
		for _, val := range stream.Values {
			if len(val) < 2 {
				continue
			}
			entry := parseLogEntry(val[1], stream.Stream)
			if entry != nil {
				entries = append(entries, *entry)
			}
		}
	}

	return entries, len(entries), nil
}

func parseLogEntry(rawMsg string, labels map[string]string) *LogEntry {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawMsg), &parsed); err != nil {
		// not JSON, treat as plain text
		return &LogEntry{
			Content: rawMsg,
			Job:     labels["job"],
			Project: labels["project"],
			Service: labels["service"],
		}
	}

	entry := &LogEntry{
		Job:     labels["job"],
		Project: labels["project"],
		Service: labels["service"],
	}
	if v, ok := parsed["@timestamp"]; ok {
		entry.Timestamp = fmt.Sprintf("%v", v)
	}
	if v, ok := parsed["caller"]; ok {
		entry.Caller = fmt.Sprintf("%v", v)
	}
	if v, ok := parsed["content"]; ok {
		entry.Content = fmt.Sprintf("%v", v)
	}
	if v, ok := parsed["trace"]; ok {
		entry.Trace = fmt.Sprintf("%v", v)
	}
	if v, ok := parsed["span"]; ok {
		entry.Span = fmt.Sprintf("%v", v)
	}
	if v, ok := parsed["level"]; ok {
		entry.Level = fmt.Sprintf("%v", v)
	}
	return entry
}

// QueryErrorTrend queries error trends as time series, grouped by a dimension
func (s *LokiService) QueryErrorTrend(project, service, job, callerFile string, start, end time.Time, step string) ([]SeriesData, error) {
	selector := buildSelector(project, service, job, callerFile, "")

	// Determine groupBy label
	groupBy := "service"
	if service != "" {
		groupBy = "caller_file"
	}

	logQL := fmt.Sprintf(`sum by (%s) (rate(%s[%s]))`, groupBy, selector, step)

	params := url.Values{}
	params.Set("query", logQL)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", step)

	apiURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", s.baseURL, params.Encode())
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("loki request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var lokiResp LokiMatrixResponse
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var series []SeriesData
	for _, result := range lokiResp.Data.Result {
		name := result.Metric[groupBy]
		if name == "" {
			name = "unknown"
		}
		var points []TrendPoint
		for _, v := range result.Values {
			if len(v) < 2 {
				continue
			}
			ts, _ := v[0].(float64)
			valStr, _ := v[1].(string)
			val, _ := strconv.ParseFloat(valStr, 64)
			points = append(points, TrendPoint{
				Time:  int64(ts) * 1000,
				Value: val,
			})
		}
		series = append(series, SeriesData{
			Name:   name,
			Data:   points,
			Labels: result.Metric,
		})
	}
	return series, nil
}

// QueryErrorSummary returns error counts grouped by multiple dimensions
func (s *LokiService) QueryErrorSummary(project, service, job string, start, end time.Time) (map[string]interface{}, error) {
	selector := buildSelector(project, service, job, "", "")
	duration := end.Sub(start)
	window := formatDuration(duration)

	result := map[string]interface{}{}

	// By service
	byService, err := s.queryCountByLabel(fmt.Sprintf(`sum by (service) (count_over_time(%s[%s]))`, selector, window), start, end, "service")
	if err != nil {
		log.Printf("QueryErrorSummary by service error: %v", err)
	} else {
		result["by_service"] = byService
	}

	// By project
	byProject, err := s.queryCountByLabel(fmt.Sprintf(`sum by (project) (count_over_time(%s[%s]))`, selector, window), start, end, "project")
	if err != nil {
		log.Printf("QueryErrorSummary by project error: %v", err)
	} else {
		result["by_project"] = byProject
	}

	// By job
	byJob, err := s.queryCountByLabel(fmt.Sprintf(`sum by (job) (count_over_time(%s[%s]))`, selector, window), start, end, "job")
	if err != nil {
		log.Printf("QueryErrorSummary by job error: %v", err)
	} else {
		result["by_job"] = byJob
	}

	return result, nil
}

func (s *LokiService) queryCountByLabel(logQL string, start, end time.Time, labelKey string) ([]TopNItem, error) {
	params := url.Values{}
	params.Set("query", logQL)
	params.Set("time", fmt.Sprintf("%d", end.Unix()))

	apiURL := fmt.Sprintf("%s/loki/api/v1/query?%s", s.baseURL, params.Encode())
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lokiResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return nil, err
	}

	var items []TopNItem
	for _, r := range lokiResp.Data.Result {
		name := r.Metric[labelKey]
		if name == "" {
			name = "unknown"
		}
		var count int64
		if len(r.Value) >= 2 {
			valStr, _ := r.Value[1].(string)
			count, _ = strconv.ParseInt(valStr, 10, 64)
		}
		items = append(items, TopNItem{
			Name:  name,
			Count: count,
			Extra: r.Metric,
		})
	}
	return items, nil
}

// QueryTopNServices returns top N services by error count
func (s *LokiService) QueryTopNServices(project, service, job string, start, end time.Time, limit int) ([]TopNItem, error) {
	selector := buildSelector(project, service, job, "", "")
	duration := end.Sub(start)
	window := formatDuration(duration)
	logQL := fmt.Sprintf(`topk(%d, sum by (service, project) (count_over_time(%s[%s])))`, limit, selector, window)

	items, err := s.queryCountByLabelMulti(logQL, start, end, "service", "project")
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// QueryTopNCallers returns top N caller files by error count
func (s *LokiService) QueryTopNCallers(project, service, job string, start, end time.Time, limit int) ([]TopNItem, error) {
	selector := buildSelector(project, service, job, "", "")
	duration := end.Sub(start)
	window := formatDuration(duration)
	logQL := fmt.Sprintf(`topk(%d, sum by (caller_file, service) (count_over_time(%s[%s])))`, limit, selector, window)

	items, err := s.queryCountByLabelMulti(logQL, start, end, "caller_file", "service")
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *LokiService) queryCountByLabelMulti(logQL string, start, end time.Time, primaryLabel string, extraLabels ...string) ([]TopNItem, error) {
	params := url.Values{}
	params.Set("query", logQL)
	params.Set("time", fmt.Sprintf("%d", end.Unix()))

	apiURL := fmt.Sprintf("%s/loki/api/v1/query?%s", s.baseURL, params.Encode())
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lokiResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return nil, err
	}

	var items []TopNItem
	for _, r := range lokiResp.Data.Result {
		name := r.Metric[primaryLabel]
		if name == "" {
			name = "unknown"
		}
		var count int64
		if len(r.Value) >= 2 {
			valStr, _ := r.Value[1].(string)
			count, _ = strconv.ParseInt(valStr, 10, 64)
		}
		items = append(items, TopNItem{
			Name:  name,
			Count: count,
			Extra: r.Metric,
		})
	}
	return items, nil
}

// CountErrors counts errors matching the given filters in a time window
func (s *LokiService) CountErrors(project, service, job, callerFile, contentPattern string, start, end time.Time) (int64, string, error) {
	selector := buildSelector(project, service, job, callerFile, "")
	logQL := selector
	if contentPattern != "" {
		logQL = fmt.Sprintf(`%s |~ "%s"`, selector, contentPattern)
	}

	duration := end.Sub(start)
	window := formatDuration(duration)
	countQL := fmt.Sprintf(`count_over_time(%s[%s])`, logQL, window)

	params := url.Values{}
	params.Set("query", countQL)
	params.Set("time", fmt.Sprintf("%d", end.Unix()))

	apiURL := fmt.Sprintf("%s/loki/api/v1/query?%s", s.baseURL, params.Encode())
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}

	var lokiResp struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return 0, "", err
	}

	var total int64
	for _, r := range lokiResp.Data.Result {
		if len(r.Value) >= 2 {
			valStr, _ := r.Value[1].(string)
			v, _ := strconv.ParseInt(valStr, 10, 64)
			total += v
		}
	}

	// Get a sample content
	sample := ""
	entries, _, err := s.QueryLogs(project, service, job, callerFile, contentPattern, start, end, 1)
	if err == nil && len(entries) > 0 {
		sample = entries[0].Content
	}

	return total, sample, nil
}

// GetSampleContent returns a sample error content for a given filter
func (s *LokiService) GetSampleContent(project, service, callerFile string, start, end time.Time) string {
	entries, _, err := s.QueryLogs(project, service, "", callerFile, "", start, end, 1)
	if err != nil || len(entries) == 0 {
		return ""
	}
	return entries[0].Content
}

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 1 {
		return "1m"
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%dd", days)
}
