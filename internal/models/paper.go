package models

import (
	"time"

	"gorm.io/gorm"
)

type Paper struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ArxivID     string    `json:"arxiv_id" gorm:"uniqueIndex;not null"`
	Title       string    `json:"title" gorm:"not null"`
	Authors     string    `json:"authors" gorm:"type:text"`
	Summary     string    `json:"summary" gorm:"type:text"`
	Categories  string    `json:"categories"`
	Published   time.Time `json:"published"`
	PDFURL      string    `json:"pdf_url"`
	AISummary   string    `json:"ai_summary" gorm:"type:longtext"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DailyReport struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Date           string    `json:"date" gorm:"uniqueIndex;not null"`
	Category       string    `json:"category" gorm:"not null"`
	PaperCount     int       `json:"paper_count"`
	AIProvider     string    `json:"ai_provider"`
	Report         string    `json:"report" gorm:"type:longtext"`
	TrendAnalysis  string    `json:"trend_analysis" gorm:"type:longtext"`
	TopPapers      string    `json:"top_papers" gorm:"type:text"` // JSON array of paper IDs
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PaperEvaluation struct {
	ID               uint    `json:"id" gorm:"primaryKey"`
	PaperID          uint    `json:"paper_id" gorm:"not null"`
	Paper            Paper   `json:"paper" gorm:"foreignKey:PaperID"`
	TechInnovation   int     `json:"tech_innovation" gorm:"check:tech_innovation >= 1 AND tech_innovation <= 5"`
	PracticalValue   int     `json:"practical_value" gorm:"check:practical_value >= 1 AND practical_value <= 5"`
	ScientificRigor  int     `json:"scientific_rigor" gorm:"check:scientific_rigor >= 1 AND scientific_rigor <= 5"`
	ResearchProspect int     `json:"research_prospect" gorm:"check:research_prospect >= 1 AND research_prospect <= 5"`
	SocialImpact     int     `json:"social_impact" gorm:"check:social_impact >= 1 AND social_impact <= 5"`
	Keywords         string  `json:"keywords"`
	Reason           string  `json:"reason" gorm:"type:text"`
	TotalScore       float64 `json:"total_score"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Paper) TableName() string {
	return "papers"
}

func (DailyReport) TableName() string {
	return "daily_reports"
}

func (PaperEvaluation) TableName() string {
	return "paper_evaluations"
}

// AutoMigrate 自动迁移数据库结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Paper{},
		&DailyReport{},
		&PaperEvaluation{},
	)
}