package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dandelight/reno-arxiv/internal/config"
	"github.com/Dandelight/reno-arxiv/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportService struct {
	config    *config.Config
	db        *gorm.DB
	aiService *AIService
}

func NewReportService(cfg *config.Config, db *gorm.DB, aiService *AIService) *ReportService {
	return &ReportService{
		config:    cfg,
		db:        db,
		aiService: aiService,
	}
}

func (s *ReportService) GenerateDailyReport(date string, category string) (*models.DailyReport, error) {
	logrus.Infof("开始生成 %s 的每日报告...", date)

	// 获取当日论文
	papers, err := s.getPapersByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to get papers: %w", err)
	}

	if len(papers) == 0 {
		logrus.Warnf("未找到 %s 的论文", date)
		return nil, fmt.Errorf("no papers found for date %s", date)
	}

	// 转换为AI服务需要的格式
	var paperInfos []PaperInfo
	for _, paper := range papers {
		authors := strings.Split(paper.Authors, ", ")
		paperInfos = append(paperInfos, PaperInfo{
			Title:   paper.Title,
			Authors: authors,
			Summary: paper.Summary,
		})
	}

	// 使用AI生成总结
	aiSummary, err := s.aiService.SummarizePapers(paperInfos)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI summary: %w", err)
	}

	// 创建报告记录
	report := &models.DailyReport{
		Date:          date,
		Category:      category,
		PaperCount:    len(papers),
		AIProvider:    s.config.AI.Provider,
		Report:        aiSummary,
		TrendAnalysis: "", // 可以后续添加
		TopPapers:     "", // 可以后续添加推荐论文ID
	}

	// 保存到数据库
	if err := s.saveDailyReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	// 保存到文件
	if err := s.saveReportToFile(report, papers); err != nil {
		logrus.Errorf("Failed to save report to file: %v", err)
	}

	logrus.Infof("每日报告生成完成，共处理 %d 篇论文", len(papers))
	return report, nil
}

func (s *ReportService) getPapersByDate(date string) ([]models.Paper, error) {
	var papers []models.Paper
	
	// 查询指定日期的论文
	startDate := date + " 00:00:00"
	endDate := date + " 23:59:59"
	
	err := s.db.Where("published BETWEEN ? AND ?", startDate, endDate).Find(&papers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch papers from database: %w", err)
	}

	return papers, nil
}

func (s *ReportService) saveDailyReport(report *models.DailyReport) error {
	// 检查是否已存在该日期的报告
	var existing models.DailyReport
	result := s.db.Where("date = ? AND category = ?", report.Date, report.Category).First(&existing)

	if result.Error == nil {
		// 报告已存在，更新
		report.ID = existing.ID
		report.CreatedAt = existing.CreatedAt
		return s.db.Save(report).Error
	} else if result.Error == gorm.ErrRecordNotFound {
		// 报告不存在，创建新记录
		return s.db.Create(report).Error
	}

	return result.Error
}

func (s *ReportService) saveReportToFile(report *models.DailyReport, papers []models.Paper) error {
	// 创建报告目录
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// 生成文件名
	filename := fmt.Sprintf("daily_report_%s.md", strings.ReplaceAll(report.Date, "-", "_"))
	filepath := filepath.Join(reportsDir, filename)

	// 生成markdown内容
	content := s.generateMarkdownContent(report, papers)

	// 写入文件
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	logrus.Infof("报告已保存到文件: %s", filepath)
	return nil
}

func (s *ReportService) generateMarkdownContent(report *models.DailyReport, papers []models.Paper) string {
	var content strings.Builder

	// 标题
	content.WriteString(fmt.Sprintf("# arXiv %s 每日报告 - %s\n\n", report.Category, report.Date))
	
	// 基本信息
	content.WriteString(fmt.Sprintf("**论文数量**: %d 篇\n", report.PaperCount))
	content.WriteString(fmt.Sprintf("**AI模型**: %s\n", report.AIProvider))
	content.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// AI总结
	content.WriteString("## AI总结分析\n\n")
	content.WriteString(report.Report)
	content.WriteString("\n\n")

	// 论文详细列表
	content.WriteString("## 论文详细列表\n\n")
	for i, paper := range papers {
		content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, paper.Title))
		content.WriteString(fmt.Sprintf("**作者**: %s\n\n", paper.Authors))
		content.WriteString(fmt.Sprintf("**发布日期**: %s\n\n", paper.Published.Format("2006-01-02")))
		content.WriteString(fmt.Sprintf("**论文ID**: %s\n\n", paper.ArxivID))
		content.WriteString(fmt.Sprintf("**分类**: %s\n\n", paper.Categories))
		content.WriteString(fmt.Sprintf("**摘要**: %s\n\n", paper.Summary))
		if paper.PDFURL != "" {
			content.WriteString(fmt.Sprintf("**PDF链接**: [%s](%s)\n\n", paper.PDFURL, paper.PDFURL))
		}
		if paper.AISummary != "" {
			content.WriteString(fmt.Sprintf("**AI总结**: %s\n\n", paper.AISummary))
		}
		content.WriteString("---\n\n")
	}

	return content.String()
}

func (s *ReportService) GetReportByDate(date string) (*models.DailyReport, error) {
	var report models.DailyReport
	err := s.db.Where("date = ?", date).First(&report).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	return &report, nil
}

func (s *ReportService) ListReports(limit, offset int) ([]models.DailyReport, error) {
	var reports []models.DailyReport
	err := s.db.Order("date DESC").Limit(limit).Offset(offset).Find(&reports).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	return reports, nil
}

func (s *ReportService) UpdateReadmeWithLatestReport(report *models.DailyReport) error {
	readmePath := "README.md"
	
	// 读取现有README内容
	var existingContent string
	if content, err := os.ReadFile(readmePath); err == nil {
		existingContent = string(content)
	}

	// 生成新的README内容
	newContent := s.generateReadmeContent(report, existingContent)

	// 写入README文件
	if err := os.WriteFile(readmePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}

	logrus.Info("README.md已更新")
	return nil
}

func (s *ReportService) generateReadmeContent(report *models.DailyReport, existingContent string) string {
	var content strings.Builder

	// 项目标题和描述
	content.WriteString("# reno-arxiv\n\n")
	content.WriteString("🤖 基于AI的arXiv论文每日自动总结系统\n\n")
	content.WriteString("本项目使用Go语言构建高性能后端服务，每日自动获取arXiv cs.AI分类的最新论文，并使用AI模型进行智能总结和分析。\n\n")

	// 最新报告链接
	reportFilename := fmt.Sprintf("daily_report_%s.md", strings.ReplaceAll(report.Date, "-", "_"))
	content.WriteString("## 📈 最新报告\n\n")
	content.WriteString(fmt.Sprintf("**最新报告日期**: %s\n", report.Date))
	content.WriteString(fmt.Sprintf("**论文数量**: %d 篇\n", report.PaperCount))
	content.WriteString(fmt.Sprintf("**AI模型**: %s\n\n", report.AIProvider))
	content.WriteString(fmt.Sprintf("👉 [查看完整报告](reports/%s)\n\n", reportFilename))

	// 最新报告预览（前500字符）
	reportPreview := report.Report
	if len(reportPreview) > 500 {
		reportPreview = reportPreview[:500] + "..."
	}
	content.WriteString("### 📋 最新报告预览\n\n")
	content.WriteString(reportPreview)
	content.WriteString("\n\n")

	// 项目特性
	content.WriteString("## ✨ 项目特性\n\n")
	content.WriteString("- 🚀 **高性能Go后端**: 使用Gin框架构建的RESTful API\n")
	content.WriteString("- 🤖 **多AI模型支持**: 支持OpenAI GPT、Anthropic Claude等多种AI模型\n")
	content.WriteString("- 📊 **智能论文分析**: 自动评估论文的技术创新度、实用价值等维度\n")
	content.WriteString("- ⏰ **自动化工作流**: GitHub Actions每日自动执行论文总结\n")
	content.WriteString("- 📈 **趋势分析**: 识别AI研究的最新趋势和热点方向\n")
	content.WriteString("- 💾 **数据持久化**: MySQL数据库存储论文信息和分析结果\n\n")

	// 技术栈
	content.WriteString("## 🛠 技术栈\n\n")
	content.WriteString("- **后端**: Go 1.23, Gin, GORM\n")
	content.WriteString("- **数据库**: MySQL\n")
	content.WriteString("- **AI模型**: OpenAI GPT-4o, Anthropic Claude\n")
	content.WriteString("- **自动化**: GitHub Actions\n")
	content.WriteString("- **数据源**: arXiv API\n\n")

	// 快速开始
	content.WriteString("## 🚀 快速开始\n\n")
	content.WriteString("### 环境要求\n\n")
	content.WriteString("- Go 1.23+\n")
	content.WriteString("- MySQL 8.0+\n")
	content.WriteString("- AI API密钥 (OpenAI或Anthropic)\n\n")

	content.WriteString("### 安装运行\n\n")
	content.WriteString("```bash\n")
	content.WriteString("# 克隆项目\n")
	content.WriteString("git clone https://github.com/Dandelight/reno-arxiv.git\n")
	content.WriteString("cd reno-arxiv\n\n")
	content.WriteString("# 复制并配置环境变量\n")
	content.WriteString("cp .env.example .env\n")
	content.WriteString("# 编辑 .env 文件，填入你的配置\n\n")
	content.WriteString("# 安装依赖\n")
	content.WriteString("go mod tidy\n\n")
	content.WriteString("# 运行服务\n")
	content.WriteString("go run cmd/server/main.go\n")
	content.WriteString("```\n\n")

	// API文档
	content.WriteString("## 📚 API文档\n\n")
	content.WriteString("### 获取每日报告\n")
	content.WriteString("```http\n")
	content.WriteString("GET /api/reports/daily/{date}\n")
	content.WriteString("```\n\n")
	content.WriteString("### 获取论文列表\n")
	content.WriteString("```http\n")
	content.WriteString("GET /api/papers?date={date}&category={category}\n")
	content.WriteString("```\n\n")
	content.WriteString("### 手动触发论文总结\n")
	content.WriteString("```http\n")
	content.WriteString("POST /api/summarize\n")
	content.WriteString("{\n")
	content.WriteString(`  "date": "2024-01-15",` + "\n")
	content.WriteString(`  "category": "cs.AI"` + "\n")
	content.WriteString("}\n")
	content.WriteString("```\n\n")

	// 历史报告
	content.WriteString("## 📖 历史报告\n\n")
	content.WriteString("所有每日报告都保存在 [reports](reports/) 目录中，按日期组织。\n\n")

	// 贡献
	content.WriteString("## 🤝 贡献\n\n")
	content.WriteString("欢迎提交Issue和Pull Request！\n\n")

	// 许可证
	content.WriteString("## 📄 许可证\n\n")
	content.WriteString("MIT License\n\n")

	// 更新时间
	content.WriteString(fmt.Sprintf("---\n*最后更新: %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return content.String()
}