package main

import (
	"fmt"
	"log"

	"github.com/Dandelight/reno-arxiv/internal/config"
	"github.com/Dandelight/reno-arxiv/internal/handlers"
	"github.com/Dandelight/reno-arxiv/internal/models"
	"github.com/Dandelight/reno-arxiv/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志级别
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 连接数据库
	db, err := connectDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移数据库结构
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化服务
	aiService := services.NewAIService(cfg)
	arxivService := services.NewArxivService(cfg, db)
	reportService := services.NewReportService(cfg, db, aiService)

	// 初始化处理器
	h := handlers.NewHandlers(arxivService, reportService)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由器
	router := gin.Default()

	// 配置CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://*.vercel.app", "https://*.netlify.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 健康检查
	router.GET("/health", h.HealthCheck)

	// API路由组
	api := router.Group("/api")
	{
		// 论文相关
		api.GET("/papers", h.GetPapers)
		
		// 报告相关
		api.GET("/reports", h.ListReports)
		api.GET("/reports/daily/:date", h.GetDailyReport)
		
		// 手动触发
		api.POST("/summarize", h.SummarizePapers)
		api.POST("/summarize/today", h.TriggerTodaySummary)
	}

	// 静态文件服务（用于提供报告文件）
	router.Static("/reports", "./reports")

	// 启动服务器
	addr := ":" + cfg.Server.Port
	logrus.Infof("服务器启动在端口 %s", cfg.Server.Port)
	logrus.Infof("API文档: http://localhost%s", addr)
	logrus.Infof("健康检查: http://localhost%s/health", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func connectDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	// 配置GORM日志
	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.Server.Mode == "release" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	logrus.Info("数据库连接成功")
	return db, nil
}