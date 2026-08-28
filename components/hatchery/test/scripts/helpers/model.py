"""
模型管理 —— 管理员接口 + 用户侧接口 + 模型 setup/teardown 脚手架
"""

from dataclasses import dataclass

from helpers import config
from helpers.api import admin_client, user_client, get_field


# ═══════════════════════════════════════════════════════════════════
# 上下文数据类
# ═══════════════════════════════════════════════════════════════════

@dataclass
class ModelContext:
    """模型上下文"""
    db_id: int
    model_id: str


# ═══════════════════════════════════════════════════════════════════
# 管理员接口
# ═══════════════════════════════════════════════════════════════════

def admin_create_model(
    admin_token,
    model_id,
    model_name,
    provider="openai",
    api_key=None,
    url=None,
    model_type=None,
    quota_day=-1,
):
    """创建自定义模型"""
    return admin_client(admin_token).post(
        "/admin/models/create",
        data={
            "model_id": model_id,
            "model_name": model_name,
            "provider": provider,
            "api_key": api_key or config.MODEL_API_KEY,
            "url": url or config.MODEL_URL,
            "model_type": model_type or config.MODEL_TYPE,
            "quota_day": str(quota_day),
        },
        timeout=60,
    )


def admin_get_models(admin_token):
    """获取管理员模型列表"""
    data = admin_client(admin_token).get("/admin/models")
    return data.get("models", [])


def admin_get_default_model_id(admin_token):
    """获取当前默认模型 ID（来自 /admin/models 响应顶层字段）"""
    data = admin_client(admin_token).get("/admin/models")
    return data.get("default_model_id", 0)


def admin_toggle_model(admin_token, model_db_id):
    """切换"用户端是否可见"（Visible）。

    后端 /admin/models/toggle 仅控制是否在用户端列表里展示，
    不影响 LLM 路由可用性。要彻底启用一个模型，需配合
    admin_toggle_model_enabled 一起翻开。
    """
    return admin_client(admin_token).post(
        "/admin/models/toggle", params={"id": model_db_id},
    )


def admin_toggle_model_enabled(admin_token, model_db_id):
    """切换"是否真正可用"（Enabled）。

    后端 /admin/models/toggle-enabled 控制 LLM 路由可用性，
    与 toggle (Visible) 解耦。
    """
    return admin_client(admin_token).post(
        "/admin/models/toggle-enabled", params={"id": model_db_id},
    )


def admin_toggle_default_model(admin_token, model_db_id):
    """设为/取消默认模型"""
    return admin_client(admin_token).post(
        "/admin/models/toggle-default", params={"id": model_db_id},
    )


def admin_update_model(admin_token, model_db_id, **fields):
    """更新模型"""
    return admin_client(admin_token).post(
        "/admin/models/update", params={"id": model_db_id}, data=fields,
    )


def admin_delete_model(admin_token, model_db_id):
    """删除模型"""
    return admin_client(admin_token).post(
        "/admin/models/delete", params={"id": model_db_id},
    )


# ═══════════════════════════════════════════════════════════════════
# 用户侧接口
# ═══════════════════════════════════════════════════════════════════

def user_get_models(user_token, agent_id=None):
    """用户侧查询模型列表"""
    params = {}
    if agent_id is not None:
        params["agent_id"] = agent_id
    data = user_client(user_token).get("/openclaw/models", params=params)
    return data.get("models", [])


def user_get_instance_models(user_token, instance_db_id):
    """查询实例绑定的模型列表"""
    return user_client(user_token).get(
        "/openclaw/instance-models", params={"id": instance_db_id},
    )


def user_add_model(user_token, instance_db_id, ai_model_id, **extra):
    """添加模型到实例（返回原始 Response 以便检查状态码）"""
    data = {
        "id": str(instance_db_id),
        "ai_model_id": str(ai_model_id),
    }
    data.update({k: str(v) for k, v in extra.items()})
    return user_client(user_token).post(
        "/openclaw/add-model", data=data, timeout=30, expect=None, raw=True,
    )


def user_set_model(user_token, instance_db_id, ai_model_id, **extra):
    """设置/替换实例主模型（返回原始 Response 以便检查状态码）。

    与 user_add_model 的差异：
      - /openclaw/set-model 维护一条 primary 记录（有则替换、无则新增），
        不支持 fallback 多模型；
      - /openclaw/add-model 支持多模型 fallback（首次 = primary，再次 = fallback）。

    适用场景：channel/skill 测试的 Setup 阶段（仅需绑定一个可用模型，
    不关心 primary/fallback 多模型语义）。
    """
    data = {
        "id": str(instance_db_id),
        "ai_model_id": str(ai_model_id),
    }
    data.update({k: str(v) for k, v in extra.items()})
    return user_client(user_token).post(
        "/openclaw/set-model", data=data, timeout=30, expect=None, raw=True,
    )


def user_del_model(user_token, instance_db_id, instance_model_id):
    """删除实例模型"""
    return user_client(user_token).post(
        "/openclaw/del-model",
        data={"id": str(instance_db_id), "instance_model_id": str(instance_model_id)},
    )


def user_switch_primary_model(user_token, instance_db_id, instance_model_id):
    """切换主模型"""
    return user_client(user_token).post(
        "/openclaw/switch-primary-model",
        data={"id": str(instance_db_id), "instance_model_id": str(instance_model_id)},
    )


# ═══════════════════════════════════════════════════════════════════
# 脚手架：模型 setup/teardown
# ═══════════════════════════════════════════════════════════════════

def setup_model(admin_token, model_id, model_name, enable=True, **kwargs):
    """创建模型并可选启用"""
    print(f">>> 创建模型: {model_id} ...")
    data = admin_create_model(
        admin_token,
        model_id=model_id,
        model_name=model_name,
        **kwargs,
    )
    assert data.get("ok"), f"创建模型失败: {data}"

    models = admin_get_models(admin_token)
    model = next(
        (m for m in models if get_field(m, "ModelName", "model_name") == model_name),
        None,
    )
    assert model, f"模型 {model_name} 未在列表中找到"
    db_id = get_field(model, "ID", "id")

    if enable:
        admin_toggle_model(admin_token, db_id)

    print(f"    模型创建成功 ✓  db_id={db_id}")
    return ModelContext(db_id=db_id, model_id=model_id)


def teardown_model(admin_token, model):
    """删除模型"""
    print(f">>> 清理模型: {model.model_id} ...")
    try:
        admin_delete_model(admin_token, model.db_id)
        print("    模型已清理 ✓")
    except Exception as e:
        print(f"    清理模型失败（忽略）: {e}")


# ═══════════════════════════════════════════════════════════════════
# 内置占位记录：自定义模型功能开关
# ═══════════════════════════════════════════════════════════════════

def ensure_custom_model_flag(admin_token):
    """确保内置占位记录 hatchery/custom 的 Enabled+Visible 都为 true。

    用户端 /openclaw/add-model 在 ai_model_id=0 时会进入 customModel()，
    要求 ai_models 表里 provider=hatchery & model_id=custom 这条记录
    Enabled=true AND Visible=true，否则返回 403 "自定义模型功能未开启"。

    新部署的环境（含 SQLite 集成测试 fixture）这条记录会以
    Enabled=false, Visible=false 的占位形态创建，因此需要"自定义模型
    功能已开启"前置条件的契约用例必须先调用本函数翻开。

    背景：后端 AIModel.MarshalJSON 把字段做了语义换名：
        JSON 里的 "Enabled"        = Go 字段 Visible（"用户端可见"开关）
        JSON 里的 "EnabledStatus"  = Go 字段 Enabled（"LLM 路由可用性"）
        JSON 里没有 "Visible" 键（被 MarshalJSON 显式 delete 掉）
    因此读取时必须按这个映射来判断真假；翻转时分别走两个端点：
        POST /admin/models/toggle           → 翻 Visible
        POST /admin/models/toggle-enabled   → 翻 Enabled
    """
    target = None
    for m in admin_get_models(admin_token):
        provider = get_field(m, "provider", "Provider")
        model_id = get_field(m, "model_id", "ModelID")
        if provider == "hatchery" and model_id == "custom":
            target = m
            break

    if not target:
        # 旧版后端可能没有这条占位记录，让调用方自行决定如何处理
        print("    NOTE: hatchery/custom 占位记录不存在，跳过")
        return

    db_id = get_field(target, "id", "ID")
    # 按 MarshalJSON 字段名映射读取真实语义：
    visible = bool(get_field(target, "Enabled", default=False))
    enabled = bool(get_field(target, "EnabledStatus", default=False))

    if not enabled:
        admin_toggle_model_enabled(admin_token, db_id)
    if not visible:
        admin_toggle_model(admin_token, db_id)

    if not (enabled and visible):
        # 翻完后做一次轻量校验，避免静默失败
        for m in admin_get_models(admin_token):
            if (get_field(m, "provider", "Provider") == "hatchery"
                    and get_field(m, "model_id", "ModelID") == "custom"):
                v = bool(get_field(m, "Enabled", default=False))
                e = bool(get_field(m, "EnabledStatus", default=False))
                assert v and e, (
                    f"翻开自定义模型开关失败 Visible={v} Enabled={e}"
                )
                break
