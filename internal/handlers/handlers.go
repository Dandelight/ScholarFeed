package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Dandelight/reno-arxiv/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	arxivService  *services.ArxivService
	reportService *services.ReportService
}

func NewHandlers(arxivService *services.ArxivService, reportService *services.ReportService) *Handlers {
	return &Handlers{
		arxivService:  arxivService,
		reportService: reportService,
	}
}

type SummarizeRequest struct {
	Date     string `json:"date" binding:"required"`
	Category string `json:"category"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// SummarizePapers 手动触发论文总结
func (h *Handlers) SummarizePapers(c *gin.Context) {
	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 设置默认分类
	if req.Category == "" {
		req.Category = "cs.AI"
	}

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_date",
			Message: "日期格式应为 YYYY-MM-DD",
		})
		return
	}

	logrus.Infof("手动触发论文总结: 日期=%s, 分类=%s", req.Date, req.Category)

	// 获取论文
	papers, err := h.arxivService.FetchPapersByDate(req.Date, req.Category)
	if err != nil {
		logrus.Errorf("获取论文失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_papers_failed",
			Message: err.Error(),
		})
		return
	}

	if len(papers) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "no_papers_found",
			Message: "未找到指定日期和分类的论文",
		})
		return
	}

	// 保存论文
	if err := h.arxivService.SavePapers(papers); err != nil {
		logrus.Errorf("保存论文失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "save_papers_failed",
			Message: err.Error(),
		})
		return
	}

	// 生成报告
	report, err := h.reportService.GenerateDailyReport(req.Date, req.Category)
	if err != nil {
		logrus.Errorf("生成报告失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "generate_report_failed",
			Message: err.Error(),
		})
		return
	}

	// 更新README
	if err := h.reportService.UpdateReadmeWithLatestReport(report); err != nil {
		logrus.Errorf("更新README失败: %v", err)
		// 不影响主流程，只记录错误
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    report,
		Message: "论文总结完成",
	})
}

// GetDailyReport 获取每日报告
func (h *Handlers) GetDailyReport(c *gin.Context) {
	date := c.Param("date")

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_date",
			Message: "日期格式应为 YYYY-MM-DD",
		})
		return
	}

	report, err := h.reportService.GetReportByDate(date)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "report_not_found",
			Message: "未找到指定日期的报告",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    report,
	})
}

// ListReports 获取报告列表
func (h *Handlers) ListReports(c *gin.Context) {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	reports, err := h.reportService.ListReports(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_reports_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    reports,
	})
}

// GetPapers 获取论文列表
func (h *Handlers) GetPapers(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_date",
			Message: "请提供date参数",
		})
		return
	}

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_date",
			Message: "日期格式应为 YYYY-MM-DD",
		})
		return
	}

	papers, err := h.arxivService.GetPapersByDate(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "get_papers_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    papers,
	})
}

// HealthCheck 健康检查
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "服务运行正常",
		Data: gin.H{
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		},
	})
}

// TriggerTodaySummary 触发今日总结
func (h *Handlers) TriggerTodaySummary(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	category := c.DefaultQuery("category", "cs.AI")

	logrus.Infof("触发今日总结: 日期=%s, 分类=%s", today, category)

	// 获取论文
	papers, err := h.arxivService.FetchPapersByDate(today, category)
	if err != nil {
		logrus.Errorf("获取今日论文失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_papers_failed",
			Message: err.Error(),
		})
		return
	}

	if len(papers) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "no_papers_found",
			Message: "今日暂无论文",
		})
		return
	}

	// 保存论文
	if err := h.arxivService.SavePapers(papers); err != nil {
		logrus.Errorf("保存论文失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "save_papers_failed",
			Message: err.Error(),
		})
		return
	}

	// 生成报告
	report, err := h.reportService.GenerateDailyReport(today, category)
	if err != nil {
		logrus.Errorf("生成今日报告失败: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "generate_report_failed",
			Message: err.Error(),
		})
		return
	}

	// 更新README
	if err := h.reportService.UpdateReadmeWithLatestReport(report); err != nil {
		logrus.Errorf("更新README失败: %v", err)
		// 不影响主流程，只记录错误
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    report,
		Message: "今日总结完成",
	})
}