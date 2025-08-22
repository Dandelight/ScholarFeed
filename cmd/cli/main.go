package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Dandelight/reno-arxiv/internal/config"
	"github.com/Dandelight/reno-arxiv/internal/models"
	"github.com/Dandelight/reno-arxiv/internal/services"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	var (
		date     = flag.String("date", "", "指定日期 (YYYY-MM-DD)，默认为今天")
		category = flag.String("category", "cs.AI", "arXiv分类")
		action   = flag.String("action", "summarize", "执行的动作: summarize, fetch, report")
	)
	flag.Parse()

	// 设置默认日期
	if *date == "" {
		*date = time.Now().Format("2006-01-02")
	}

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", *date); err != nil {
		log.Fatalf("日期格式错误，应为 YYYY-MM-DD: %v", err)
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 连接数据库
	db, err := connectDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化服务
	aiService := services.NewAIService(cfg)
	arxivService := services.NewArxivService(cfg, db)
	reportService := services.NewReportService(cfg, db, aiService)

	// 执行指定动作
	switch *action {
	case "fetch":
		fetchPapers(arxivService, *date, *category)
	case "summarize":
		summarizePapers(arxivService, reportService, *date, *category)
	case "report":
		generateReport(reportService, *date, *category)
	default:
		log.Fatalf("未知的动作: %s", *action)
	}
}

func fetchPapers(arxivService *services.ArxivService, date, category string) {
	logrus.Infof("开始获取 %s 的 %s 分类论文...", date, category)

	papers, err := arxivService.FetchPapersByDate(date, category)
	if err != nil {
		log.Fatalf("获取论文失败: %v", err)
	}

	if len(papers) == 0 {
		logrus.Warnf("未找到 %s 的 %s 分类论文", date, category)
		return
	}

	if err := arxivService.SavePapers(papers); err != nil {
		log.Fatalf("保存论文失败: %v", err)
	}

	logrus.Infof("成功获取并保存 %d 篇论文", len(papers))
}

func summarizePapers(arxivService *services.ArxivService, reportService *services.ReportService, date, category string) {
	logrus.Infof("开始完整的论文总结流程: %s %s", date, category)

	// 1. 获取论文
	papers, err := arxivService.FetchPapersByDate(date, category)
	if err != nil {
		log.Fatalf("获取论文失败: %v", err)
	}

	if len(papers) == 0 {
		logrus.Warnf("未找到 %s 的 %s 分类论文", date, category)
		return
	}

	// 2. 保存论文
	if err := arxivService.SavePapers(papers); err != nil {
		log.Fatalf("保存论文失败: %v", err)
	}

	// 3. 生成报告
	report, err := reportService.GenerateDailyReport(date, category)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}

	// 4. 更新README
	if err := reportService.UpdateReadmeWithLatestReport(report); err != nil {
		logrus.Errorf("更新README失败: %v", err)
	}

	logrus.Infof("✅ 论文总结完成！共处理 %d 篇论文", len(papers))
	fmt.Printf("📊 报告ID: %d\n", report.ID)
	fmt.Printf("📝 论文数量: %d\n", report.PaperCount)
	fmt.Printf("🤖 AI提供商: %s\n", report.AIProvider)
}

func generateReport(reportService *services.ReportService, date, category string) {
	logrus.Infof("开始生成 %s 的 %s 分类报告...", date, category)

	report, err := reportService.GenerateDailyReport(date, category)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}

	if err := reportService.UpdateReadmeWithLatestReport(report); err != nil {
		logrus.Errorf("更新README失败: %v", err)
	}

	logrus.Infof("✅ 报告生成完成！")
	fmt.Printf("📊 报告ID: %d\n", report.ID)
	fmt.Printf("📝 论文数量: %d\n", report.PaperCount)
	fmt.Printf("🤖 AI提供商: %s\n", report.AIProvider)
}

func connectDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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