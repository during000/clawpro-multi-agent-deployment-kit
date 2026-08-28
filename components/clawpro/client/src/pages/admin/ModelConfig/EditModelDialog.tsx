/**
 * EditModelDialog
 * 编辑已添加模型的弹窗。
 *
 * 与 Add Dialog 的区别：
 *   - 厂商不可更改（决定后续字段布局）
 *   - API Key 以脱敏值作为 placeholder 展示，输入框留空则保留原 Key
 *   - 保存时若 Key 输入框为空则保留原 Key
 *   - 支持高级配置（maxTokens / contextWindow / headers）编辑
 *   - 支持多模态属性编辑（所有模型均可切换）
 *   - 支持连通性检测
 */
import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { PanelTitle, MetaMedium, CardTitle, MetaText } from "@/components/ui/Typography";
import { toast } from "sonner";
import { Wifi } from "lucide-react";
import { AVAILABLE_MODELS } from "@/lib/mockData";
import {
  CUSTOM_PROVIDER_VALUE,
  maskApiKey,
  type ModelRow,
} from "@/lib/modelConfigStore";
import {
  AdvancedConfigSection,
  type AdvancedConfig,
} from "./AdvancedConfigSection";
import { ConnectFailDialog } from "./ConnectFailDialog";
import { CredentialKeySelect } from "@/components/CredentialKeySelect";

export interface EditModelDialogProps {
  model: ModelRow | null;
  open: boolean;
  onClose: () => void;
  onSave: (id: string, updates: Partial<ModelRow>) => void;
}

export function EditModelDialog({ model, open, onClose, onSave }: EditModelDialogProps) {
  const isCustom = model?.provider === CUSTOM_PROVIDER_VALUE;
  const selectedProviderData = AVAILABLE_MODELS.find((m) => m.value === model?.provider);
  // 由用户端自行填写 Key 的模型：编辑时不支持修改凭据
  const isUserProvidedKeyModel = model?.userProvidedKey ?? false;

  // 厂商模型编辑字段
  const [version, setVersion] = useState("");
  const [modelUrl, setModelUrl] = useState("");
  const [dailyLimit, setDailyLimit] = useState(100000);
  const [isMultimodal, setIsMultimodal] = useState(false);

  // 自定义模型编辑字段
  const [customForm, setCustomForm] = useState({
    provider: "", base_url: "", api: "", api_key: "", model_id: "", model_name: "", dailyLimit: 100000, isMultimodal: false,
  });

  // API Key 输入（留空则保留原 Key）
  const [apiKeyInput, setApiKeyInput] = useState("");

  // 高级配置
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [advancedConfig, setAdvancedConfig] = useState<AdvancedConfig>({
    contextWindow: "",
    maxTokens: "",
    headers: [{ key: "", value: "" }],
  });

  // 连通性检测
  const [connectTesting, setConnectTesting] = useState(false);
  const [connectFailResult, setConnectFailResult] = useState<string | null>(null);

  // API Key 由用户端自行配置：管理员不填密钥，且不支持每日 Tokens 上限与连通性检测
  const [userProvidedKey, setUserProvidedKey] = useState(false);

  // 每次打开弹窗时用 model 数据初始化
  useEffect(() => {
    if (!model || !open) return;
    setVersion(model.version);
    setModelUrl(model.modelUrl);
    setDailyLimit(model.dailyLimit);
    setIsMultimodal(model.isMultimodal ?? false);
    setApiKeyInput("");
    setUserProvidedKey(model.userProvidedKey ?? false);
    setAdvancedOpen(false);
    setAdvancedConfig({
      contextWindow: model.contextWindow ?? "",
      maxTokens: model.maxTokens ?? "",
      headers: model.headers?.length ? model.headers : [{ key: "", value: "" }],
    });
    setConnectFailResult(null);
    if (model.provider === CUSTOM_PROVIDER_VALUE) {
      setCustomForm({
        provider: model.name,
        base_url: model.modelUrl,
        api: "",
        api_key: model.apiKey ?? "",
        model_id: "",
        model_name: model.version,
        dailyLimit: model.dailyLimit,
        isMultimodal: model.isMultimodal ?? false,
      });
    }
  }, [model, open]);

  if (!model) return null;

  // 连通性检测：校验必填字段后模拟请求
  const handleConnectTest = async () => {
    if (userProvidedKey) {
      toast.error("由用户端自行配置密钥时不支持连通性检测");
      return;
    }
    if (isCustom) {
      if (!customForm.base_url) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else {
      if (!modelUrl.trim()) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    }
    setConnectTesting(true);
    await new Promise((r) => setTimeout(r, 1500));
    setConnectTesting(false);
    setConnectFailResult(JSON.stringify({
      error: {
        message: "Invalid API Key",
        param: "Please provide valid API Key",
        code: "401",
        type: "invalid_key",
      }
    }, null, 2));
  };

  const handleSave = () => {
    // 过滤掉空 headers
    const cleanHeaders = advancedConfig.headers.filter((h) => h.key || h.value);

    if (isCustom) {
      if (!customForm.base_url || !customForm.model_name) {
        toast.error("请填写完整信息");
        return;
      }
      if (!userProvidedKey && (!customForm.dailyLimit || customForm.dailyLimit <= 0)) {
        toast.error("请填写有效的每日配额");
        return;
      }
      const updates: Partial<ModelRow> = {
        name: customForm.provider || "自定义模型",
        version: customForm.model_name,
        modelUrl: customForm.base_url,
        dailyLimit: userProvidedKey ? 0 : customForm.dailyLimit,
        isMultimodal: customForm.isMultimodal,
        maxTokens: advancedConfig.maxTokens || undefined,
        contextWindow: advancedConfig.contextWindow || undefined,
        headers: cleanHeaders.length > 0 ? cleanHeaders : undefined,
        userProvidedKey,
      };
      if (userProvidedKey) {
        updates.apiKey = undefined;
      } else if (apiKeyInput.trim()) {
        updates.apiKey = apiKeyInput.trim();
      }
      onSave(model.id, updates);
    } else {
      if (!modelUrl.trim()) {
        toast.error("请填写模型 URL");
        return;
      }
      if (!userProvidedKey && (!dailyLimit || dailyLimit <= 0)) {
        toast.error("请填写有效的每日配额");
        return;
      }
      const updates: Partial<ModelRow> = {
        version,
        modelUrl: modelUrl.trim(),
        dailyLimit: userProvidedKey ? 0 : dailyLimit,
        isMultimodal,
        maxTokens: advancedConfig.maxTokens || undefined,
        headers: cleanHeaders.length > 0 ? cleanHeaders : undefined,
        userProvidedKey,
      };
      if (userProvidedKey) {
        updates.apiKey = undefined;
      } else if (apiKeyInput.trim()) {
        updates.apiKey = apiKeyInput.trim();
      }
      onSave(model.id, updates);
    }
    onClose();
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-lg flex flex-col max-h-[90vh]">
          <DialogHeader className="shrink-0">
            <DialogTitle asChild>
              <PanelTitle>编辑模型</PanelTitle>
            </DialogTitle>
            <MetaText as="p" tone="weak" className="mt-1">保存后修改内容立即生效，用户端无需重复添加</MetaText>
          </DialogHeader>
          <DialogBody className="px-6 space-y-4 overflow-y-auto">
            {/* 厂商（只读） */}
            <div>
              <MetaMedium as="label" tone="secondary">模型厂商</MetaMedium>
              <div className="w-full px-3 py-2 rounded-[var(--radius)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)]">
                <MetaText>{isCustom ? "自定义模型" : (selectedProviderData?.label ?? model.provider)}</MetaText>
              </div>
            </div>

            {/* 厂商模型字段 */}
            {!isCustom && (
              <>
                {/* 模型名称（版本） */}
                {selectedProviderData && selectedProviderData.versions.length > 0 && (
                  <div>
                    <MetaMedium as="label" tone="secondary">
                      模型名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Select value={version} onValueChange={setVersion}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="选择模型版本" />
                      </SelectTrigger>
                      <SelectContent>
                        {selectedProviderData.versions.map((v) => (
                          <SelectItem key={v} value={v}>{v}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}

                {/* 模型 URL */}
                <div>
                  <MetaMedium as="label" tone="secondary">
                    模型 URL<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入模型 URL地址"
                    value={modelUrl}
                    onChange={(e) => setModelUrl(e.target.value)}
                  />
                </div>
              </>
            )}

            {/* 自定义模型字段 */}
            {isCustom && (
              <>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    Provider<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入自定义模型 provider"
                    value={customForm.provider}
                    onChange={(e) => setCustomForm({ ...customForm, provider: e.target.value })}
                  />
                </div>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    Base URL<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入自定义模型 base_url"
                    value={customForm.base_url}
                    onChange={(e) => setCustomForm({ ...customForm, base_url: e.target.value })}
                  />
                </div>
                <div>
                  <MetaMedium as="label" tone="secondary">API 协议</MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入自定义模型 api"
                    value={customForm.api}
                    onChange={(e) => setCustomForm({ ...customForm, api: e.target.value })}
                  />
                </div>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    Model ID<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入自定义模型 model.id"
                    value={customForm.model_id}
                    onChange={(e) => setCustomForm({ ...customForm, model_id: e.target.value })}
                  />
                </div>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    Model Name<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入自定义模型 model.name"
                    value={customForm.model_name}
                    onChange={(e) => setCustomForm({ ...customForm, model_name: e.target.value })}
                  />
                </div>
              </>
            )}

            {/* API Key：手动输入或选择凭据；留空则保留原 Key */}
            {/* 由用户端自行填写 Key 的模型：管理员不可修改凭据，仅只读展示 */}
            {isUserProvidedKeyModel ? (
              <div>
                <MetaMedium as="label" tone="secondary">API Key</MetaMedium>
                <div className="w-full px-3 py-2 rounded-[var(--radius)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)]">
                  <MetaText tone="weak">该模型由用户端自行填写 Key，管理员不可修改凭据</MetaText>
                </div>
              </div>
            ) : (
              <CredentialKeySelect
                value={apiKeyInput}
                onChange={setApiKeyInput}
                placeholder={model.apiKey ? `当前：${maskApiKey(model.apiKey)}（留空保留原值）` : "请输入 API Key"}
                initialApiKey={model.apiKey}
                allowUserSide={isCustom}
                initialUserSide={model.userProvidedKey ?? false}
                onUserSideChange={setUserProvidedKey}
              />
            )}

            {/* 每日配额 */}
            {!userProvidedKey && (
              <div>
                <MetaMedium as="label" tone="secondary">
                  每日 Tokens 数量上限<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  type="number"
                  value={isCustom ? customForm.dailyLimit : dailyLimit}
                  onChange={(e) => {
                    const val = Number(e.target.value);
                    if (isCustom) {
                      setCustomForm({ ...customForm, dailyLimit: val });
                    } else {
                      setDailyLimit(val);
                    }
                  }}
                />
              </div>
            )}

            {/* 多模态开关 — 所有模型均可切换 */}
            <div className="rounded-[var(--radius)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-3 flex items-center justify-between">
              <div>
                <CardTitle as="p">多模态模型</CardTitle>
                <MetaText as="p" className="mt-0.5">支持图片、文字多模态输入</MetaText>
              </div>
              <Switch
                checked={isCustom ? customForm.isMultimodal : isMultimodal}
                onCheckedChange={(v) => {
                  if (isCustom) {
                    setCustomForm({ ...customForm, isMultimodal: v });
                  } else {
                    setIsMultimodal(v);
                  }
                }}
              />
            </div>

            {/* 高级配置 */}
            <AdvancedConfigSection
              open={advancedOpen}
              onToggle={() => setAdvancedOpen((v) => !v)}
              config={advancedConfig}
              onChange={setAdvancedConfig}
              showContextWindow={isCustom}
            />
          </DialogBody>
          <DialogFooter className="flex items-center justify-between gap-2 shrink-0">
            {userProvidedKey ? (
              <span />
            ) : (
              <Button
                variant="claw-outline"
                size="claw-sm"
                className="gap-1.5"
                disabled={connectTesting}
                onClick={handleConnectTest}
              >
                <Wifi className="w-4 h-4" />
                {connectTesting ? "检测中…" : "连通性检测"}
              </Button>
            )}
            <div className="flex items-center gap-2">
              <Button variant="claw-outline" size="claw-sm" onClick={onClose}>取消</Button>
              <Button variant="dialog-confirm" size="claw-sm" onClick={handleSave}>
                保存
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 连通性检测失败弹窗 */}
      <ConnectFailDialog
        result={connectFailResult}
        onClose={() => setConnectFailResult(null)}
      />
    </>
  );
}
