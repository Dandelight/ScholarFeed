# reno-arxiv

🤖 基于AI的arXiv论文每日自动总结系统

本项目使用Go语言构建高性能后端服务，每日自动获取arXiv cs.AI分类的最新论文，并使用AI模型进行智能总结和分析。

## 📈 最新报告

**状态**: 系统正在初始化...

👉 [查看所有报告](reports/)

## ✨ 项目特性

- 🚀 **高性能Go后端**: 使用Gin框架构建的RESTful API
- 🤖 **多AI模型支持**: 支持OpenAI GPT、Anthropic Claude等多种AI模型
- 📊 **智能论文分析**: 自动评估论文的技术创新度、实用价值等维度
- ⏰ **自动化工作流**: GitHub Actions每日自动执行论文总结
- 📈 **趋势分析**: 识别AI研究的最新趋势和热点方向
- 💾 **数据持久化**: MySQL数据库存储论文信息和分析结果

## 🛠 技术栈

- **后端**: Go 1.23, Gin, GORM
- **数据库**: MySQL
- **AI模型**: OpenAI GPT-4o, Anthropic Claude
- **自动化**: GitHub Actions
- **数据源**: arXiv API

## 🚀 快速开始

### 环境要求

- Go 1.23+
- MySQL 8.0+
- AI API密钥 (OpenAI或Anthropic)

### 安装运行

```bash
# 克隆项目
git clone https://github.com/Dandelight/reno-arxiv.git
cd reno-arxiv

# 复制并配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入你的配置

# 安装依赖
go mod tidy

# 运行服务
go run cmd/server/main.go
```

### 环境变量配置

在 `.env` 文件中配置以下变量：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=arxiv_db

# AI配置
AI_PROVIDER=claude  # 或 openai
ANTHROPIC_API_KEY=your_claude_api_key
# 或
OPENAI_API_KEY=your_openai_api_key

# arXiv配置
ARXIV_CATEGORY=cs.AI
ARXIV_MAX_RESULTS=100
```

## 📚 API文档

### 基础信息

- **Base URL**: `http://localhost:8080/api`
- **Content-Type**: `application/json`

### 端点列表

#### 1. 健康检查
```http
GET /health
```

#### 2. 获取每日报告
```http
GET /api/reports/daily/{date}
```

**参数:**
- `date`: 日期，格式为 `YYYY-MM-DD`

#### 3. 获取报告列表
```http
GET /api/reports?limit=20&offset=0
```

#### 4. 获取论文列表
```http
GET /api/papers?date={date}&category={category}
```

**参数:**
- `date`: 必需，日期格式 `YYYY-MM-DD`
- `category`: 可选，默认为 `cs.AI`

#### 5. 手动触发论文总结
```http
POST /api/summarize
Content-Type: application/json

{
  "date": "2024-01-15",
  "category": "cs.AI"
}
```

#### 6. 触发今日总结
```http
POST /api/summarize/today?category=cs.AI
```

### 响应格式

**成功响应:**
```json
{
  "success": true,
  "data": { ... },
  "message": "操作成功"
}
```

**错误响应:**
```json
{
  "error": "error_code",
  "message": "错误描述"
}
```

## 🤖 自动化工作流

本项目使用GitHub Actions实现每日自动总结：

- **触发时间**: 每天北京时间上午9:00 (UTC 1:00)
- **执行内容**: 
  - 获取前一天的arXiv论文
  - 使用AI模型进行总结和分析
  - 生成markdown报告
  - 更新README.md
  - 自动提交到仓库

### 手动触发

你也可以通过以下方式手动触发工作流：

1. 在GitHub仓库页面进入Actions标签
2. 选择"每日论文总结"工作流
3. 点击"Run workflow"
4. 可选择特定日期和分类

## 📊 数据模型

### Paper (论文)
- ID, arXiv ID, 标题, 作者, 摘要
- 分类, 发布时间, PDF链接
- AI总结, 处理时间

### DailyReport (每日报告)
- 日期, 分类, 论文数量
- AI提供商, 报告内容
- 趋势分析, 推荐论文

### PaperEvaluation (论文评估)
- 技术创新度, 实用价值, 科学严谨性
- 研究前景, 社会影响, 关键词
- 评分理由, 总分

## 📖 历史报告

所有每日报告都保存在 [reports](reports/) 目录中，按日期组织。

## 🔧 开发指南

### 项目结构
```
reno-arxiv/
├── cmd/server/          # 服务器入口
├── internal/
│   ├── config/          # 配置管理
│   ├── models/          # 数据模型
│   ├── services/        # 业务逻辑
│   └── handlers/        # HTTP处理器
├── reports/             # 生成的报告
├── .github/workflows/   # GitHub Actions
└── go.mod              # Go模块定义
```

### 添加新的AI提供商

1. 在 `services/ai_service.go` 中添加新的客户端实现
2. 更新 `config/config.go` 中的配置选项
3. 在环境变量中添加相应的API密钥配置

### 自定义报告格式

修改 `services/report_service.go` 中的 `generateMarkdownContent` 方法来自定义报告格式。

## 🚨 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查MySQL服务是否运行
   - 验证数据库配置是否正确
   - 确保数据库用户有足够权限

2. **AI API调用失败**
   - 检查API密钥是否有效
   - 确认网络连接正常
   - 查看API使用额度

3. **arXiv API请求失败**
   - arXiv API有时不稳定，重试通常可解决
   - 检查日期格式是否正确
   - 确认分类名称是否有效

### 日志查看

```bash
# 查看服务器日志
go run cmd/server/main.go

# 查看GitHub Actions日志
# 在仓库的Actions标签页查看具体运行日志
```

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 开发流程

1. Fork本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开Pull Request

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [arXiv.org](https://arxiv.org/) - 提供开放的学术论文访问
- [OpenAI](https://openai.com/) - GPT模型支持
- [Anthropic](https://anthropic.com/) - Claude模型支持

---
*最后更新: 2024-01-15 12:00:00*