"""
集成测试配置

所有外部参数通过环境变量注入，测试框架读取后作为全局配置使用。

连接参数（API / ADMIN_TOKEN）在模块加载时立即校验；
模型配置、通道凭证等均采用惰性加载 — 只在首次访问时才检查环境变量，
避免仅跑管理类测试（admin_sg/quota 等）时因无关变量缺失而 crash。
"""

import os
import string
import random


def _require_env(name: str) -> str:
    """读取必填环境变量，未设置时抛出明确异常。"""
    value = os.environ.get(name)
    if not value:
        raise EnvironmentError(f"必填环境变量 {name} 未设置")
    return value


class _LazyEnv:
    """惰性环境变量描述符 — 首次访问时才读取并缓存。"""

    def __init__(self, name: str, *, required: bool = True, default: str = ""):
        self._name = name
        self._required = required
        self._default = default
        self._value = None
        self._resolved = False

    def __set_name__(self, owner, name):
        self._attr = name

    def __get__(self, obj, objtype=None):
        if not self._resolved:
            if self._required:
                self._value = _require_env(self._name)
            else:
                self._value = os.environ.get(self._name, self._default)
            self._resolved = True
        return self._value


class _ModelConfig:
    """模型配置容器 — 所有属性惰性加载。"""

    MODEL_ID = _LazyEnv("MODEL_ID")
    MODEL_API_KEY = _LazyEnv("MODEL_API_KEY")
    MODEL_URL = _LazyEnv("MODEL_URL")
    MODEL_TYPE = _LazyEnv("MODEL_TYPE")

    # 第二个模型（Fallback 测试需要，非强制）
    MODEL_ID_2 = _LazyEnv("MODEL_ID_2", required=False)
    # 第三个模型（多模型 Fallback 需要 ≥ 3 个模型，非强制）
    MODEL_ID_3 = _LazyEnv("MODEL_ID_3", required=False)


class _ChannelCredentials:
    """通道凭证容器 — 所有属性惰性加载。"""

    # 飞书
    FEISHU_APP_ID = _LazyEnv("FEISHU_APP_ID")
    FEISHU_APP_SECRET = _LazyEnv("FEISHU_APP_SECRET")

    # 企微机器人
    WECOM_BOT_ID = _LazyEnv("WECOM_BOT_ID")
    WECOM_SECRET = _LazyEnv("WECOM_SECRET")

    # 钉钉
    DDINGTALK_CLIENT_ID = _LazyEnv("DDINGTALK_CLIENT_ID")
    DDINGTALK_CLIENT_SECRET = _LazyEnv("DDINGTALK_CLIENT_SECRET")

    # QQ 机器人
    QQBOT_APP_ID = _LazyEnv("QQBOT_APP_ID")
    QQBOT_APP_SECRET = _LazyEnv("QQBOT_APP_SECRET")

    # 企微应用
    WECOM_APP_CORP_ID = _LazyEnv("WECOM_APP_CORP_ID")
    WECOM_APP_CORP_SECRET = _LazyEnv("WECOM_APP_CORP_SECRET")
    WECOM_APP_AGENT_ID = _LazyEnv("WECOM_APP_AGENT_ID")
    WECOM_APP_TOKEN = _LazyEnv("WECOM_APP_TOKEN")
    WECOM_APP_ENCODING_AES_KEY = _LazyEnv("WECOM_APP_ENCODING_AES_KEY")

    # Discord（overseas-only，Hermes 支持）
    DISCORD_BOT_TOKEN = _LazyEnv("DISCORD_BOT_TOKEN")
    DISCORD_USER_ID = _LazyEnv("DISCORD_USER_ID")

    # Lark（overseas-only，Hermes 支持；复用 feishu 的 app_id/app_secret 协议）
    LARK_APP_ID = _LazyEnv("LARK_APP_ID")
    LARK_APP_SECRET = _LazyEnv("LARK_APP_SECRET")

    # LINE（overseas-only，Hermes 支持）
    LINE_CHANNEL_ACCESS_TOKEN = _LazyEnv("LINE_CHANNEL_ACCESS_TOKEN")
    LINE_CHANNEL_SECRET = _LazyEnv("LINE_CHANNEL_SECRET")

    # 通道测试用户 ID
    FEISHU_OPEN_ID = _LazyEnv("FEISHU_OPEN_ID")
    WECOM_USER_ID = _LazyEnv("WECOM_USER_ID")
    QQBOT_C2C_OPEN_ID = _LazyEnv("QQBOT_C2C_OPEN_ID")
    DINGTALK_USER_ID = _LazyEnv("DINGTALK_USER_ID")


# 惰性单例
_model = _ModelConfig()
_channel = _ChannelCredentials()


def generate_password(length: int = 16) -> str:
    """生成随机测试密码，满足大小写+数字+特殊字符要求。"""
    special = "!@#$%&"
    # 确保至少包含各类字符
    password = [
        random.choice(string.ascii_uppercase),
        random.choice(string.ascii_lowercase),
        random.choice(string.digits),
        random.choice(special),
    ]
    pool = string.ascii_letters + string.digits + special
    password += [random.choice(pool) for _ in range(length - 4)]
    random.shuffle(password)
    return "".join(password)


# ──────────────────────────── 连接参数（立即校验）────────────────────────────

BASE_URL: str = _require_env("API").rstrip("/")
SEED_ADMIN_TOKEN: str = _require_env("ADMIN_TOKEN")

# ──────────────────────────── 模型配置（惰性加载）────────────────────────────
# 通过模块级 __getattr__ 代理到 _model 单例，仅在首次访问时校验。

# ──────────────────────────── 通道凭证（惰性加载）────────────────────────────
# 通过模块级 __getattr__ 代理到 _channel 单例，首次读取时才校验环境变量。


def __getattr__(name: str):
    """模块级 __getattr__，将模型配置和通道凭证属性代理到对应惰性单例。"""
    if hasattr(_ModelConfig, name) and not name.startswith("_"):
        return getattr(_model, name)
    if hasattr(_ChannelCredentials, name) and not name.startswith("_"):
        return getattr(_channel, name)
    raise AttributeError(f"module 'helpers.config' has no attribute {name!r}")


# ──────────────────────────── 固定测试参数 ────────────────────────────

AGENT_TYPE: str = "openclaw"
INSTANCE_NAME_PREFIX: str = "inttest-"
USERNAME_PREFIX: str = "inttest-user-"
ADMIN_USERNAME_PREFIX: str = "inttest-admin-"
DEFAULT_INSTANCE_QUOTA: int = 10
DEFAULT_TOKEN_QUOTA_DAY: int = -1  # -1 表示不限制

POLL_INTERVAL: int = 10        # 状态轮询间隔（秒）
POLL_TIMEOUT: int = 600        # 状态轮询上限（秒）= 10 分钟
SKILL_POLL_TIMEOUT: int = 180  # 技能安装轮询上限（秒）= 3 分钟
