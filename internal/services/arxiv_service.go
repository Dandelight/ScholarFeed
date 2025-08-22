package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dandelight/reno-arxiv/internal/config"
	"github.com/Dandelight/reno-arxiv/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ArxivService struct {
	config *config.Config
	db     *gorm.DB
	client *http.Client
}

type ArxivResponse struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []ArxivEntry `xml:"entry"`
}

type ArxivEntry struct {
	ID        string      `xml:"id"`
	Title     string      `xml:"title"`
	Summary   string      `xml:"summary"`
	Published string      `xml:"published"`
	Authors   []Author    `xml:"author"`
	Links     []Link      `xml:"link"`
	Categories []Category `xml:"category"`
}

type Author struct {
	Name string `xml:"name"`
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Category struct {
	Term string `xml:"term,attr"`
}

func NewArxivService(cfg *config.Config, db *gorm.DB) *ArxivService {
	return &ArxivService{
		config: cfg,
		db:     db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *ArxivService) FetchPapersByDate(targetDate string, category string) ([]models.Paper, error) {
	logrus.Infof("开始获取 %s 的 %s 分类论文...", targetDate, category)

	// 解析日期
	date, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	nextDate := date.AddDate(0, 0, 1)

	// 构建查询参数
	query := fmt.Sprintf("cat:%s AND submittedDate:[%s0000 TO %s0000]",
		category,
		date.Format("20060102"),
		nextDate.Format("20060102"))

	params := url.Values{}
	params.Set("search_query", query)
	params.Set("start", "0")
	params.Set("max_results", fmt.Sprintf("%d", s.config.ArXiv.MaxResults))
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")

	// 构建请求URL
	baseURL := "http://export.arxiv.org/api/query"
	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	logrus.Debugf("请求URL: %s", reqURL)

	// 发送请求
	resp, err := s.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from arXiv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arXiv API returned status code: %d", resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析XML
	var arxivResp ArxivResponse
	if err := xml.Unmarshal(body, &arxivResp); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// 转换为Paper模型
	var papers []models.Paper
	for _, entry := range arxivResp.Entries {
		paper := s.convertToPaper(entry)
		papers = append(papers, paper)
	}

	logrus.Infof("成功获取到 %d 篇论文", len(papers))
	
	// 添加延迟以避免过于频繁的请求
	time.Sleep(time.Duration(s.config.ArXiv.DelaySeconds) * time.Second)

	return papers, nil
}

func (s *ArxivService) convertToPaper(entry ArxivEntry) models.Paper {
	// 提取arXiv ID
	arxivID := s.extractArxivID(entry.ID)

	// 提取作者
	var authors []string
	for _, author := range entry.Authors {
		authors = append(authors, author.Name)
	}

	// 提取PDF链接
	pdfURL := ""
	for _, link := range entry.Links {
		if link.Rel == "alternate" {
			pdfURL = link.Href
			break
		}
	}

	// 提取分类
	var categories []string
	for _, cat := range entry.Categories {
		categories = append(categories, cat.Term)
	}

	// 解析发布时间
	published, err := time.Parse("2006-01-02T15:04:05Z", entry.Published)
	if err != nil {
		logrus.Warnf("Failed to parse published date: %s", entry.Published)
		published = time.Now()
	}

	return models.Paper{
		ArxivID:    arxivID,
		Title:      strings.TrimSpace(entry.Title),
		Authors:    strings.Join(authors, ", "),
		Summary:    strings.TrimSpace(entry.Summary),
		Categories: strings.Join(categories, ", "),
		Published:  published,
		PDFURL:     pdfURL,
	}
}

func (s *ArxivService) extractArxivID(id string) string {
	// arXiv ID 格式通常是 http://arxiv.org/abs/2101.00001v1
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		arxivID := parts[len(parts)-1]
		// 移除版本号
		if idx := strings.Index(arxivID, "v"); idx > 0 {
			arxivID = arxivID[:idx]
		}
		return arxivID
	}
	return id
}

func (s *ArxivService) SavePapers(papers []models.Paper) error {
	if len(papers) == 0 {
		return nil
	}

	logrus.Infof("开始保存 %d 篇论文到数据库...", len(papers))

	for _, paper := range papers {
		// 检查是否已存在
		var existing models.Paper
		result := s.db.Where("arxiv_id = ?", paper.ArxivID).First(&existing)

		if result.Error == nil {
			// 论文已存在，更新信息
			s.db.Model(&existing).Updates(paper)
			logrus.Debugf("更新论文: %s", paper.Title)
		} else if result.Error == gorm.ErrRecordNotFound {
			// 论文不存在，创建新记录
			if err := s.db.Create(&paper).Error; err != nil {
				logrus.Errorf("保存论文失败: %s, 错误: %v", paper.Title, err)
				continue
			}
			logrus.Debugf("保存新论文: %s", paper.Title)
		} else {
			logrus.Errorf("查询论文时出错: %v", result.Error)
			continue
		}
	}

	logrus.Info("论文保存完成")
	return nil
}

func (s *ArxivService) GetPapersByDate(date string) ([]models.Paper, error) {
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