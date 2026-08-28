/**
 * updateReminderStore - "有新版本时提醒用户更新"开关本地存储
 *
 * 每个 Agent 类型可独立控制是否在用户端展示更新提醒。
 * 开启后，用户端该类型的 Agent 将看到"有新版本可用"提示。
 *
 * 存储：
 *   - admin_update_reminder_v1: Record<string, boolean>
 */
const STORAGE_KEY = "admin_update_reminder_v1";

function readMap(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw) as Record<string, boolean>;
  } catch { /* ignore */ }
  return {};
}

function writeMap(map: Record<string, boolean>): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch { /* ignore */ }
}

/** 某 Agent 类型是否开启了更新提醒（系统预设类型默认开启） */
export function isReminderEnabled(agentType: string): boolean {
  const map = readMap();
  if (agentType in map) return map[agentType] ?? false;
  // 系统预设类型（OpenClaw / HermesAgent / LightClawACE）默认开启提醒
  const SYSTEM_PRESETS = new Set(["OpenClaw", "HermesAgent", "LightClawACE"]);
  return SYSTEM_PRESETS.has(agentType);
}

/** 设置某 Agent 类型的更新提醒开关 */
export function setReminderEnabled(agentType: string, enabled: boolean): void {
  const map = readMap();
  map[agentType] = enabled;
  writeMap(map);
}

/** 批量获取所有开启提醒的 Agent 类型 */
export function getEnabledReminderTypes(): string[] {
  const map = readMap();
  return Object.entries(map)
    .filter(([, v]) => v)
    .map(([k]) => k);
}
