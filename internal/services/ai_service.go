package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Dandelight/reno-arxiv/internal/config"
	"github.com/sirupsen/logrus"
)

type AIService struct {
	config *config.Config
	client *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type AnthropicRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	System      string    `json:"system"`
	Messages    []Message `json:"messages"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func NewAIService(cfg *config.Config) *AIService {
	return &AIService{
		config: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (s *AIService) GenerateResponse(messages []Message, maxTokens int, temperature float64) (string, error) {
	switch s.config.AI.Provider {
	case "openai":
		return s.generateOpenAIResponse(messages, maxTokens, temperature)
	case "claude":
		return s.generateClaudeResponse(messages, maxTokens, temperature)
	default:
		return "", errors.New("unsupported AI provider")
	}
}

func (s *AIService) generateOpenAIResponse(messages []Message, maxTokens int, temperature float64) (string, error) {
	if s.config.AI.OpenAIAPIKey == "" {
		return "", errors.New("OpenAI API key is required")
	}

	request := OpenAIRequest{
		Model:       s.config.AI.OpenAIModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.AI.OpenAIAPIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no choices in response")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

func (s *AIService) generateClaudeResponse(messages []Message, maxTokens int, temperature float64) (string, error) {
	if s.config.AI.AnthropicAPIKey == "" {
		return "", errors.New("Anthropic API key is required")
	}

	// 分离 system 和 user 消息
	var systemMessage string
	var userMessages []Message

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			userMessages = append(userMessages, msg)
		}
	}

	if systemMessage == "" {
		systemMessage = "你是一个AI研究领域的专家，擅长总结学术论文。"
	}

	request := AnthropicRequest{
		Model:       s.config.AI.AnthropicModel,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		System:      systemMessage,
		Messages:    userMessages,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.AI.AnthropicBaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.AI.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Claude API request failed with status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", errors.New("no content in response")
	}

	return strings.TrimSpace(response.Content[0].Text), nil
}

func (s *AIService) SummarizePapers(papers []PaperInfo) (string, error) {
	var formattedPapers strings.Builder
	
	for i, paper := range papers {
		formattedPapers.WriteString(fmt.Sprintf("论文%d: %s\n", i+1, paper.Title))
		
		// 限制作者数量
		authors := paper.Authors
		if len(authors) > 5 {
			authors = append(authors[:5], "等")
		}
		formattedPapers.WriteString(fmt.Sprintf("作者: %s\n", strings.Join(authors, ", ")))
		formattedPapers.WriteString(fmt.Sprintf("摘要: %s\n", paper.Summary))
		formattedPapers.WriteString("---\n\n")
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "你是一位专业的研究人员",
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`%s

这是arXiv某一天的部分论文，请对其中与大语言模型、Multi-Agent相关的论文进行分类总结，描述研究趋势，然后以表格的形式，对每篇论文从技术创新度、实用价值、科学严谨性、研究前景、社会影响对这些论文进行打1-5分评价，并给出简要关键词（中文）、简要理由。你应该注意论文的泛化意义，而不是只解决某一个领域的小问题。请使用论文英文原名；最后，请重点推荐不超过3篇最有技术创新价值和长期影响力的论文。`, 
				formattedPapers.String()),
		},
	}

	return s.GenerateResponse(messages, 64000, 0.3)
}

type PaperInfo struct {
	Title   string
	Authors []string
	Summary string
}