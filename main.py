#!/usr/bin/env python3
"""
获取指定日期arXiv cs.AI分类下的所有论文，并使用多种AI模型进行总结
支持: OpenAI GPT, Anthropic Claude, 以及其他兼容OpenAI协议的模型
"""

import glob
import json
import os
import re
from abc import ABC, abstractmethod
from datetime import datetime, timedelta
from enum import Enum
from typing import Any, Dict, List, Optional

import anthropic
from arxiv import Client, Search, SortCriterion, SortOrder
from loguru import logger
from openai import OpenAI


class ModelProvider(Enum):
    """支持的模型提供商"""

    OPENAI = "openai"
    CLAUDE = "claude"
    CUSTOM_OPENAI = "custom_openai"


class AIModelClient(ABC):
    """AI模型客户端的抽象基类"""

    @abstractmethod
    def generate_response(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 64000,
        temperature: float = 0.3,
    ) -> str:
        """生成AI回复"""


class OpenAIClient(AIModelClient):
    """OpenAI客户端"""

    def __init__(self, api_key: str, model: str = "gpt-3.5-turbo"):
        self.client = OpenAI(api_key=api_key)
        self.model = model

    def generate_response(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 64000,
        temperature: float = 0.3,
    ) -> str:
        try:
            response = self.client.chat.completions.create(
                model=self.model,
                messages=messages,
                max_tokens=max_tokens,
                temperature=temperature,
            )
            return response.choices[0].message.content.strip()
        except Exception as e:
            logger.error(f"OpenAI API调用失败: {e}")
            raise


class ClaudeClient(AIModelClient):
    """Claude客户端"""

    def __init__(self):
        self.client = anthropic.Anthropic(
            api_key=os.getenv("CLAUDE_API_KEY"),
            base_url="https://api.openai-proxy.org/anthropic",
        )
        self.model = "claude-sonnet-4-20250514"

    def generate_response(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 64000,
        temperature: float = 0.3,
    ) -> str:
        try:
            # Claude API需要分离system和user消息
            system_message = ""
            user_messages = []

            for msg in messages:
                if msg["role"] == "system":
                    system_message = msg["content"]
                else:
                    user_messages.append(msg)

            # 使用流式响应来避免超时
            full_response = ""
            with self.client.messages.stream(
                model=self.model,
                max_tokens=max_tokens,
                temperature=temperature,
                system=(
                    system_message
                    if system_message
                    else "你是一个AI研究领域的专家，擅长总结学术论文。"
                ),
                messages=user_messages,
            ) as stream:
                for text in stream.text_stream:
                    full_response += text

            return full_response.strip()

        except Exception as e:
            logger.error(f"Claude API调用失败: {e}")
            raise


class CustomOpenAIClient(AIModelClient):
    """自定义OpenAI兼容客户端（如本地部署的模型、其他API服务等）"""

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = OpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    def generate_response(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 300,
        temperature: float = 0.3,
    ) -> str:
        try:
            response = self.client.chat.completions.create(
                model=self.model,
                messages=messages,
                max_tokens=max_tokens,
                temperature=temperature,
            )
            return response.choices[0].message.content.strip()
        except Exception as e:
            logger.error(f"自定义API调用失败: {e}")
            raise


class AIModelFactory:
    """AI模型工厂类"""

    @staticmethod
    def create_client(provider: ModelProvider, **kwargs) -> AIModelClient:
        """创建AI模型客户端"""
        if provider == ModelProvider.OPENAI:
            api_key = kwargs.get("api_key") or os.getenv("OPENAI_API_KEY")
            model = kwargs.get("model", "gpt-5-mini")
            if not api_key:
                raise ValueError("请设置OPENAI_API_KEY环境变量或传入api_key参数")
            return OpenAIClient(api_key, model)

        elif provider == ModelProvider.CLAUDE:
            return ClaudeClient()

        elif provider == ModelProvider.CUSTOM_OPENAI:
            api_key = kwargs.get("api_key", "dummy")  # 有些本地服务不需要真实的key
            base_url = kwargs.get("base_url")
            model = kwargs.get("model")
            if not base_url or not model:
                raise ValueError("自定义OpenAI客户端需要base_url和model参数")
            return CustomOpenAIClient(api_key, base_url, model)

        else:
            raise ValueError(f"不支持的模型提供商: {provider}")


class ArxivPaperSummarizer:
    """arXiv论文总结"""

    def __init__(
        self,
        provider: ModelProvider = ModelProvider.OPENAI,
        model_config: Optional[Dict[str, Any]] = None,
    ):
        """
        初始化论文总结器

        Args:
            provider: 模型提供商
            model_config: 模型配置参数
        """
        if model_config is None:
            model_config = {}

        # 创建AI客户端
        self.ai_client = AIModelFactory.create_client(provider, **model_config)
        self.provider = provider

        # 初始化arXiv客户端
        self.arxiv_client = Client(page_size=100, delay_seconds=3.0, num_retries=3)

    def get_papers_by_date(
        self, target_date: str, category: str = "cs.AI"
    ) -> List[Dict[str, Any]]:
        """
        获取指定日期和分类的论文

        Args:
            target_date: 目标日期，格式为YYYY-MM-DD
            category: arXiv分类，默认为cs.AI

        Returns:
            论文列表，每个论文包含标题、摘要、作者等信息
        """
        try:
            # 解析日期
            date_obj = datetime.strptime(target_date, "%Y-%m-%d")
            next_date = date_obj + timedelta(days=1)

            # 构建查询字符串
            query = f"cat:{category} AND submittedDate:[{date_obj.strftime('%Y%m%d')}0000 TO {next_date.strftime('%Y%m%d')}0000]"

            logger.info(f"开始获取{target_date}的{category}分类论文...")

            # 创建搜索对象
            search = Search(
                query=query,
                max_results=None,
                sort_by=SortCriterion.SubmittedDate,
                sort_order=SortOrder.Descending,
            )

            papers = []
            paper_count = 0

            # 获取搜索结果
            for result in self.arxiv_client.results(search):
                paper_info = {
                    "id": result.get_short_id(),
                    "title": result.title.strip(),
                    "authors": [author.name for author in result.authors],
                    "summary": result.summary.strip(),
                    "published": result.published.strftime("%Y-%m-%d"),
                    "categories": result.categories,
                    "pdf_url": result.pdf_url,
                }
                papers.append(paper_info)
                paper_count += 1

                if paper_count % 10 == 0:
                    logger.info(f"已获取 {paper_count} 篇论文...")

            logger.info(f"共获取到 {len(papers)} 篇论文")
            return papers

        except Exception as e:
            logger.error(f"获取论文时出错: {e}")
            return []

    def format_paper(self, paper: Dict[str, Any]) -> str:
        """
        使用AI总结单篇论文

        Args:
            paper: 论文信息字典

        Returns:
            论文总结
        """
        author_limit = 5
        return f"""标题: {paper['title']}
作者: {', '.join(paper['authors'][:author_limit])}{'等' if len(paper['authors']) > author_limit else ''}
摘要: {paper['summary']}
---
"""

    def summarize_all_papers(self, papers: List[Dict[str, Any]]) -> str:
        """
        总结所有论文

        Args:
            papers: 论文列表

        Returns:
            包含总结的论文列表
        """
        formatted_papers = [self.format_paper(paper) for paper in papers]

        messages = [
            {"role": "system", "content": "你是一位专业的研究人员"},
            {
                "role": "user",
                "content": f"{formatted_papers}\n这是arXiv某一天的部分论文，请对其中与大语言模型、Multi-Agent相关的论文进行分类总结，描述研究趋势，然后以表格的形式，对每篇论文从技术创新度、实用价值、科学严谨性、研究前景、社会影响对这些论文进行打1-5分评价，并给出简要关键词（中文）、简要理由。你应该注意论文的泛化意义，而不是只解决某一个领域的小问题。请使用论文英文原名；最后，请重点推荐不超过3篇最有技术创新价值和长期影响力的论文。",
            },
        ]
        summary = self.ai_client.generate_response(messages)

        return summary

    def generate_daily_report(self, date: str, papers: List[Dict[str, Any]]) -> str:
        """
        生成每日报告

        Args:
            date: 日期
            papers: 论文列表（包含AI总结）

        Returns:
            格式化的报告文本
        """
        try:
            papers_info = "\n\n".join(
                [
                    f"论文{i+1}: {paper['title']}\n总结: {paper.get('ai_summary', '无总结')}"
                    for i, paper in enumerate(papers[:10])
                ]
            )

            prompt = f"""
基于以下{date}发布的AI论文，请生成一份简洁的每日研究趋势报告，包括：
1. 当日论文数量统计
2. 主要研究方向和热点
3. 值得关注的重要论文（2-3篇）
4. 技术趋势分析

论文信息：
{papers_info}

请用中文撰写，控制在500字以内。
"""

            messages = [
                {"role": "system", "content": "你是一个AI研究趋势分析专家。"},
                {"role": "user", "content": prompt},
            ]

            return self.ai_client.generate_response(
                messages, max_tokens=600, temperature=0.5
            )

        except Exception as e:
            logger.error(f"生成每日报告时出错: {e}")
            return f"生成报告失败: {e}"

    def save_results(self, date: str, papers: List[Dict[str, Any]], report: str):
        """
        保存结果到文件

        Args:
            date: 日期
            papers: 论文列表
            report: 每日报告
        """
        # 创建输出目录
        os.makedirs("arxiv_reports", exist_ok=True)

        # 保存详细数据
        filename = f"arxiv_reports/arxiv_cs_ai_{date.replace('-', '_')}.json"
        with open(filename, "w", encoding="utf-8") as f:
            json.dump(
                {
                    "date": date,
                    "category": "cs.AI",
                    "paper_count": len(papers),
                    "ai_provider": self.provider.value,
                    "papers": papers,
                    "daily_report": report,
                },
                f,
                ensure_ascii=False,
                indent=2,
            )

        # 保存简化报告
        report_filename = f"arxiv_reports/daily_report_{date.replace('-', '_')}.md"
        with open(report_filename, "w", encoding="utf-8") as f:
            f.write(f"# arXiv cs.AI 每日报告 - {date}\n\n")
            f.write(f"**论文数量**: {len(papers)} 篇\n")
            f.write(f"**AI模型**: {self.provider.value}\n\n")
            f.write("## 每日趋势分析\n\n")
            f.write(report)
            f.write("\n\n## 论文详细列表\n\n")

            for i, paper in enumerate(papers, 1):
                f.write(f"### {i}. {paper['title']}\n\n")
                f.write(f"**作者**: {', '.join(paper['authors'])}\n\n")
                f.write(f"**发布日期**: {paper['published']}\n\n")
                f.write(f"**论文ID**: {paper['id']}\n\n")
                if paper.get("ai_summary"):
                    f.write(f"**AI总结**: {paper['ai_summary']}\n\n")
                f.write(f"**PDF链接**: {paper['pdf_url']}\n\n")
                f.write("---\n\n")

        logger.info(f"结果已保存到: {filename} 和 {report_filename}")


def main():
    """主函数 - 演示不同模型的使用"""

    model_config_claude = {
        "api_key": os.getenv("ANTHROPIC_API_KEY"),
        "model": "claude-sonnet-4-20250514",
    }

    model_config_openai = {
        "api_key": os.getenv("OPENAI_API_KEY"),
        "model": "gpt-4o",
    }

    model_config_custom = {
        "api_key": "ollama",
        "base_url": "http://localhost:11434/v1",
        "model": "llama2",
    }

    # 选择要使用的模型 - 修改这里来切换模型
    provider = ModelProvider.CLAUDE  # 改为OPENAI或CUSTOM_OPENAI
    if provider == ModelProvider.CLAUDE:
        config = model_config_claude
    elif provider == ModelProvider.OPENAI:
        config = model_config_openai
    else:
        config = model_config_custom

    category = "cs.AI"
    # 初始化总结器
    for day in range(1, 31):
        target_date = f"2025-04-{day:02d}"
        papers = get_summary_by_date_and_category(
            target_date, category, provider, config
        )

        logger.info(f"{target_date} 共处理 {len(papers)} 篇论文")


def get_summary_by_date_and_category(target_date, category, provider, config):
    summarizer = ArxivPaperSummarizer(provider=provider, model_config=config)

    # 获取论文
    papers = summarizer.get_papers_by_date(target_date, category)

    if not papers:
        logger.warning(f"未找到{target_date}的{category}分类论文")
        return

    # 保存一下
    with open(f"arxiv_papers_{target_date}.json", "w", encoding="utf-8") as f:
        json.dump(papers, f, ensure_ascii=False, indent=2)

    # 总结论文
    logger.info(f"开始使用{provider.value}总结论文...")
    summarized_papers = summarizer.summarize_all_papers(papers)

    file_version = find_correct_version(f"daily_report_{target_date}")
    logger.info(f"生成每日报告到 {file_version}")
    with open(file_version, "w", encoding="utf-8") as f:
        f.write(summarized_papers)
    return papers


def find_correct_version(filename: str) -> str:
    """查找正确的文件版本，例如已经有 filename.md, filename_v1.md，则返回 filename_v2.md"""

    pattern = re.escape(filename) + r"(_v(\d+))?.md"
    files = glob.glob(pattern)

    if not files:
        return filename + ".md"

    versions = [
        int(re.search(r"_v(\d+)", f).group(1))
        for f in files
        if re.search(r"_v(\d+)", f)
    ]
    if not versions:
        return filename + "_v1"

    max_version = max(versions)
    return filename + f"_v{max_version+1}.md"


if __name__ == "__main__":
    main()
