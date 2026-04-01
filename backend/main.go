package main

import (
	"log"

	"log-analysis-platform/config"
	"log-analysis-platform/handler"
	"log-analysis-platform/job"
	"log-analysis-platform/model"
	"log-analysis-platform/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	config.Load()

	// Initialize database
	db, err := gorm.Open(mysql.Open(config.GlobalConfig.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.AlertRule{},
		&model.AlertHistory{},
		&model.Setting{},
		&model.NotifyChannel{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed default data
	seedDefaults(db)

	// Initialize services
	service.InitLoki()
	service.InitNotifyService(db)
	service.InitAnalyzer(db)
	service.InitAlerter(db)

	// Initialize handler DB
	handler.SetDB(db)

	// Start scheduler
	scheduler := job.NewScheduler()
	scheduler.Start()
	defer scheduler.Stop()

	// Setup Gin
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	// API routes
	api := r.Group("/api")
	{
		// Dashboard
		api.GET("/dashboard/error-trend", handler.GetErrorTrend)
		api.GET("/dashboard/error-summary", handler.GetErrorSummary)

		// TopN
		api.GET("/topn/services", handler.GetTopNServices)
		api.GET("/topn/callers", handler.GetTopNCallers)

		// Log detail
		api.GET("/logs", handler.GetLogs)

		// Alert rules
		api.GET("/alert-rules", handler.ListAlertRules)
		api.POST("/alert-rules", handler.CreateAlertRule)
		api.PUT("/alert-rules/:id", handler.UpdateAlertRule)
		api.DELETE("/alert-rules/:id", handler.DeleteAlertRule)
		api.PUT("/alert-rules/:id/toggle", handler.ToggleAlertRule)

		// Alert history
		api.GET("/alert-history", handler.ListAlertHistory)
		api.PUT("/alert-history/:id/resolve", handler.ResolveAlertHistory)

		// Settings
		api.GET("/settings", handler.GetSettings)
		api.PUT("/settings", handler.UpdateSettings)

		// Notify channels
		api.GET("/notify-channels", handler.ListNotifyChannels)
		api.POST("/notify-channels", handler.CreateNotifyChannel)
		api.PUT("/notify-channels/:id", handler.UpdateNotifyChannel)
		api.DELETE("/notify-channels/:id", handler.DeleteNotifyChannel)
		api.PUT("/notify-channels/:id/toggle", handler.ToggleNotifyChannel)
		api.POST("/notify-channels/:id/test", handler.TestNotifyChannel)
	}

	log.Printf("Server starting on :%s", config.GlobalConfig.ServerPort)
	if err := r.Run(":" + config.GlobalConfig.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func seedDefaults(db *gorm.DB) {
	// Seed default alert rules if none exist
	var count int64
	db.Model(&model.AlertRule{}).Count(&count)
	if count == 0 {
		rules := []model.AlertRule{
			{
				Name:           "DB扫描错误",
				Severity:       "critical",
				ContentPattern: "Scan error on column index",
				TimeWindow:     300,
				Threshold:      0,
				SilenceMinutes: 30,
				Enabled:        true,
			},
			{
				Name:           "连接失败",
				Severity:       "critical",
				ContentPattern: "connect failed|connection refused",
				TimeWindow:     300,
				Threshold:      0,
				SilenceMinutes: 30,
				Enabled:        true,
			},
			{
				Name:           "用户未找到",
				Severity:       "warning",
				ContentPattern: "not found",
				TimeWindow:     300,
				Threshold:      50,
				SilenceMinutes: 30,
				Enabled:        true,
			},
			{
				Name:           "风控报单错误",
				Severity:       "warning",
				ContentPattern: "ErrCode:205010",
				TimeWindow:     300,
				Threshold:      20,
				SilenceMinutes: 30,
				Enabled:        true,
			},
			{
				Name:           "未授权请求噪音",
				Severity:       "noise",
				ContentPattern: "no token present in request",
				TimeWindow:     300,
				Threshold:      1,
				SilenceMinutes: 60,
				Enabled:        true,
			},
			{
				Name:           "扫描器噪音",
				Severity:       "noise",
				ContentPattern: "CensysInspect",
				TimeWindow:     300,
				Threshold:      1,
				SilenceMinutes: 60,
				Enabled:        true,
			},
		}
		db.Create(&rules)
		log.Println("Seeded default alert rules")
	}

	// Seed default settings
	defaultSettings := []model.Setting{
		{Key: "spike_multiplier", Value: "10", Description: "突增倍数阈值"},
		{Key: "global_threshold", Value: "100", Description: "全局默认5分钟错误阈值"},
		{Key: "global_time_window", Value: "300", Description: "全局默认时间窗口（秒）"},
		{Key: "global_silence_minutes", Value: "30", Description: "全局静默时间（分钟）"},
		{Key: "warning_batch_interval", Value: "5", Description: "warning聚合推送间隔（分钟）"},
	}
	for _, s := range defaultSettings {
		db.Where(model.Setting{Key: s.Key}).FirstOrCreate(&s)
	}
}
