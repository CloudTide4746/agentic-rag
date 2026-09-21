"""全局配置 — pydantic-settings 读取 backend/.env（密钥不入库不入仓）"""
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict

BASE_DIR = Path(__file__).resolve().parent.parent


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=BASE_DIR / ".env", env_file_encoding="utf-8", extra="ignore")

    # LLM（deepseek-v4-flash-vision-exp，OpenAI 兼容，多模态视觉模型）
    deepseek_api_key: str = ""
    deepseek_base_url: str = "https://api.deepseek.com"
    llm_model: str = "deepseek-v4-flash-vision-exp"

    # 问答图片（多模态）
    chat_image_max_count: int = 4  # 单条消息最多图片数
    chat_image_max_bytes: int = 5 * 1024 * 1024  # 单张图片上限（解码后）

    # Embedding（阿里云百炼 text-embedding-v4，OpenAI 兼容接口）
    dashscope_api_key: str = ""
    dashscope_base_url: str = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    embed_model: str = "text-embedding-v4"
    embed_dims: int = 1024

    # MySQL 8.x（utf8mb4）
    mysql_host: str = "localhost"
    mysql_port: int = 3306
    mysql_user: str = "root"
    mysql_password: str = ""
    mysql_db: str = "agentic_rag"

    # Elasticsearch 8.x/9.x
    es_url: str = "http://localhost:9200"
    es_user: str = "elastic"
    es_password: str = ""

    # 应用
    data_dir: str = "data"
    chunk_size: int = 500
    chunk_overlap: int = 50
    retrieval_top_k: int = 5
    fused_top_n: int = 8
    grade_pass: float = 7.0
    grade_partial: float = 5.0
    crag_max_rounds: int = 2
    reflect_max_rounds: int = 2
    overall_timeout: int = 120  # 单轮问答总超时（秒）
    history_window: int = 8  # 多轮上下文条数


settings = Settings()

UPLOAD_DIR = BASE_DIR / settings.data_dir / "uploads"
