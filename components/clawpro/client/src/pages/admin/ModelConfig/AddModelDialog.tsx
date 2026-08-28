import { useEffect, useMemo, useState } from "react";
import { Wifi } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  CardTitle,
  HelperText,
  MetaMedium,
  MetaText,
  PanelTitle,
} from "@/components/ui/Typography";
import { CredentialKeySelect } from "@/components/CredentialKeySelect";
import { AVAILABLE_MODELS } from "@/lib/mockData";
import {
  CUSTOM_MODEL_DEFAULT_JSON,
  CUSTOM_PROVIDER_VALUE,
  loadAdminModels,
  saveAdminModels,
  type ModelRow,
} from "@/lib/modelConfigStore";
import {
  AdvancedConfigSection,
  type AdvancedConfig,
} from "./AdvancedConfigSection";
import { ConnectFailDialog } from "./ConnectFailDialog";

const PROVIDER_OPTIONS = [
  ...AVAILABLE_MODELS.map((model) => ({ value: model.value, label: model.label })),
  { value: CUSTOM_PROVIDER_VALUE, label: "自定义模型" },
];

const DEFAULT_NEW_MODEL = {
  provider: PROVIDER_OPTIONS[0]?.value ?? "",
  version: "",
  modelUrl: "",
  apiKey: "",
  dailyLimit: 100000,
};

const DEFAULT_CUSTOM_FORM = {
  provider: "",
  base_url: "",
  api: "",
  api_key: "",
  model_id: "",
  model_name: "",
  dailyLimit: 100000,
  isMultimodal: false,
};

const DEFAULT_ADVANCED_CONFIG: AdvancedConfig = {
  contextWindow: "",
  maxTokens: "",
  headers: [{ key: "", value: "" }],
};

const CUSTOM_FORM_FIELDS = [
  { key: "provider", label: "请输入自定义模型 provider" },
  { key: "base_url", label: "请输入自定义模型 base_url" },
  { key: "api", label: "请输入自定义模型 api" },
  { key: "model_id", label: "请输入自定义模型 model.id" },
  { key: "model_name", label: "请输入自定义模型 model.name" },
] as const;

interface AddModelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdded?: (model: ModelRow) => void;
}

export function AddModelDialog({ open, onOpenChange, onAdded }: AddModelDialogProps) {
  const [customInputMode, setCustomInputMode] = useState<"json" | "form">("form");
  const [newModel, setNewModel] = useState(DEFAULT_NEW_MODEL);
  const [customForm, setCustomForm] = useState(DEFAULT_CUSTOM_FORM);
  const [customJson, setCustomJson] = useState(CUSTOM_MODEL_DEFAULT_JSON);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [advancedConfig, setAdvancedConfig] = useState<AdvancedConfig>(DEFAULT_ADVANCED_CONFIG);
  const [connectTesting, setConnectTesting] = useState(false);
  const [connectFailResult, setConnectFailResult] = useState<string | null>(null);
  // API Key 由用户端自行配置：管理员不填密钥，且不支持每日 Tokens 上限与连通性检测
  const [userProvidedKey, setUserProvidedKey] = useState(false);

  useEffect(() => {
    if (!open) return;
    setCustomInputMode("form");
    setNewModel(DEFAULT_NEW_MODEL);
    setCustomForm(DEFAULT_CUSTOM_FORM);
    setCustomJson(CUSTOM_MODEL_DEFAULT_JSON);
    setAdvancedOpen(false);
    setAdvancedConfig(DEFAULT_ADVANCED_CONFIG);
    setConnectTesting(false);
    setConnectFailResult(null);
    setUserProvidedKey(false);
  }, [open]);

  const isCustomProvider = newModel.provider === CUSTOM_PROVIDER_VALUE;
  const selectedProviderData = useMemo(
    () => AVAILABLE_MODELS.find((model) => model.value === newModel.provider),
    [newModel.provider],
  );

  const handleConnectTest = async () => {
    if (userProvidedKey) {
      toast.error("由用户端自行配置密钥时不支持连通性检测");
      return;
    }
    if (isCustomProvider) {
      if (customInputMode === "form") {
        if (!customForm.base_url || !customForm.api_key || !customForm.model_id) {
          toast.error("请填写完整的模型配置信息");
          return;
        }
      } else if (!customJson.trim()) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else if (!newModel.version || !newModel.modelUrl) {
      toast.error("请填写完整的模型配置信息");
      return;
    }

    setConnectTesting(true);
    await new Promise((resolve) => setTimeout(resolve, 1500));
    setConnectTesting(false);
    setConnectFailResult(JSON.stringify({
      error: {
        message: "Invalid API Key",
        param: "Please provide valid API Key",
        code: "401",
        type: "invalid_key",
      },
    }, null, 2));
  };

  const handleAddModel = () => {
    let model: ModelRow;
    if (isCustomProvider) {
      const name = customInputMode === "form" ? (customForm.provider || "自定义模型") : "自定义模型";
      model = {
        id: String(Date.now()),
        name,
        version: customInputMode === "form" ? customForm.model_name : "自定义",
        modelUrl: customForm.base_url || "",
        apiKey: userProvidedKey ? undefined : (customForm.api_key || undefined),
        visible: true,
        isDefault: false,
        isMultimodal: customForm.isMultimodal,
        dailyLimit: userProvidedKey ? 0 : customForm.dailyLimit,
        userProvidedKey,
        provider: CUSTOM_PROVIDER_VALUE,
        versions: [],
        visibilityScope: "all",
        visibilityGroupIds: [],
      };
    } else {
      if (!newModel.provider || !newModel.modelUrl) {
        toast.error("请填写完整信息");
        return;
      }
      const versions = selectedProviderData?.versions ?? [];
      model = {
        id: String(Date.now()),
        name: selectedProviderData?.label || newModel.provider,
        version: newModel.version || (versions[0] ?? "自动"),
        modelUrl: newModel.modelUrl,
        apiKey: userProvidedKey ? undefined : (newModel.apiKey || undefined),
        visible: true,
        isDefault: false,
        dailyLimit: userProvidedKey ? 0 : newModel.dailyLimit,
        userProvidedKey,
        provider: newModel.provider,
        versions,
        visibilityScope: "all",
        visibilityGroupIds: [],
      };
    }

    saveAdminModels([...loadAdminModels(), model]);
    onAdded?.(model);
    onOpenChange(false);
    toast.success(isCustomProvider ? "自定义模型已添加" : "模型已添加");
  };

  const addDisabled = (() => {
    if (!newModel.provider) return true;
    if (isCustomProvider) {
      if (customInputMode === "form") {
        if (!customForm.provider || !customForm.base_url || !customForm.api || !customForm.model_id || !customForm.model_name) return true;
        // 由用户端自行配置密钥时，管理员无需填写 API Key
        if (!userProvidedKey && !customForm.api_key) return true;
      } else if (!customJson.trim()) {
        return true;
      }
      // 用户端配置模式不设置每日上限
      if (userProvidedKey) return false;
      return !customForm.dailyLimit || customForm.dailyLimit <= 0;
    }
    if (!newModel.version || !newModel.modelUrl.trim()) return true;
    // 用户端配置模式不设置每日上限
    if (userProvidedKey) return false;
    return !newModel.dailyLimit || newModel.dailyLimit <= 0;
  })();

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg flex flex-col max-h-[90vh]">
          <DialogHeader className="shrink-0">
            <DialogTitle asChild>
              <PanelTitle>添加模型</PanelTitle>
            </DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 space-y-4">
            <div>
              <MetaMedium as="label" tone="secondary">
                模型厂商<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Select
                value={newModel.provider}
                onValueChange={(provider) => {
                  setNewModel({ ...newModel, provider, version: "" });
                  setUserProvidedKey(false);
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择模型厂商或自定义模型" />
                </SelectTrigger>
                <SelectContent>
                  {PROVIDER_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {newModel.provider && !isCustomProvider && (
              <>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    模型名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  {selectedProviderData && selectedProviderData.versions.length > 0 ? (
                    <Select
                      value={newModel.version}
                      onValueChange={(version) => setNewModel({ ...newModel, version })}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="选择模型版本" />
                      </SelectTrigger>
                      <SelectContent>
                        {selectedProviderData.versions.map((version) => (
                          <SelectItem key={version} value={version}>{version}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : (
                    <div className="w-full px-3 py-2">
                      <HelperText>暂无可用的模型版本</HelperText>
                    </div>
                  )}
                </div>
                <div>
                  <MetaMedium as="label" tone="secondary">
                    模型 URL<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    type="text"
                    placeholder="请输入模型 URL地址"
                    value={newModel.modelUrl}
                    onChange={(event) => setNewModel({ ...newModel, modelUrl: event.target.value })}
                  />
                </div>
                <CredentialKeySelect
                  value={newModel.apiKey}
                  onChange={(apiKey) => setNewModel({ ...newModel, apiKey })}
                />
                {!userProvidedKey && (
                  <div>
                    <MetaMedium as="label" tone="secondary">
                      每日 Tokens 数量上限<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      type="number"
                      value={newModel.dailyLimit}
                      onChange={(event) => setNewModel({ ...newModel, dailyLimit: Number(event.target.value) })}
                    />
                  </div>
                )}
              </>
            )}

            {isCustomProvider && (
              <>
                <Tabs value={customInputMode} onValueChange={(value) => { setCustomInputMode(value as "json" | "form"); setUserProvidedKey(false); }}>
                  <TabsList className="w-full">
                    <TabsTrigger value="json" className="flex-1">JSON 输入</TabsTrigger>
                    <TabsTrigger value="form" className="flex-1">表单输入</TabsTrigger>
                  </TabsList>
                  <TabsContent value="json" className="mt-3">
                    <Textarea
                      value={customJson}
                      onChange={(event) => setCustomJson(event.target.value)}
                      className="font-mono text-xs min-h-48"
                    />
                  </TabsContent>
                  <TabsContent value="form" className="mt-3 space-y-3">
                    {CUSTOM_FORM_FIELDS.map((field) => (
                      <Input
                        key={field.key}
                        placeholder={field.label}
                        value={customForm[field.key]}
                        onChange={(event) => setCustomForm({ ...customForm, [field.key]: event.target.value })}
                      />
                    ))}
                    <CredentialKeySelect
                      label="API Key"
                      value={customForm.api_key}
                      onChange={(api_key) => setCustomForm({ ...customForm, api_key })}
                      allowUserSide
                      onUserSideChange={setUserProvidedKey}
                    />
                  </TabsContent>
                </Tabs>
                {!userProvidedKey && (
                  <div>
                    <MetaMedium as="label" tone="secondary">
                      每日 Tokens 数量上限<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      type="number"
                      value={customForm.dailyLimit}
                      onChange={(event) => setCustomForm({ ...customForm, dailyLimit: Number(event.target.value) })}
                    />
                  </div>
                )}
                <div className="rounded-[var(--radius)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-3 flex items-center justify-between">
                  <div>
                    <CardTitle as="p">多模态模型</CardTitle>
                    <MetaText as="p" className="mt-0.5">支持图片、文字多模态输入</MetaText>
                  </div>
                  <Switch
                    checked={customForm.isMultimodal}
                    onCheckedChange={(isMultimodal) => setCustomForm({ ...customForm, isMultimodal })}
                  />
                </div>
                {customInputMode === "form" && (
                  <AdvancedConfigSection
                    open={advancedOpen}
                    onToggle={() => setAdvancedOpen((value) => !value)}
                    config={advancedConfig}
                    onChange={setAdvancedConfig}
                    showContextWindow
                  />
                )}
              </>
            )}

            {newModel.provider && !isCustomProvider && (
              <AdvancedConfigSection
                open={advancedOpen}
                onToggle={() => setAdvancedOpen((value) => !value)}
                config={advancedConfig}
                onChange={setAdvancedConfig}
                showContextWindow={false}
              />
            )}
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
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={handleAddModel}
              disabled={addDisabled}
            >
              确认添加
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConnectFailDialog
        result={connectFailResult}
        onClose={() => setConnectFailResult(null)}
      />
    </>
  );
}
