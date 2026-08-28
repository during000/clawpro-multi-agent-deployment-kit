# 场景 D · 页面模板引用校验（SOP S1.5 存在性 + 挂载自洽）

> 日常触发语：「我改了页面模板引用的 spec。」
> 重点：页面模板是 Tier2 独有资产，reconcile **不覆盖**，只做「引用的 spec/组件存在且挂对」校验。演法（Q5=a）：**故意指向不存在的 spec → 报错 → 修复 → 通过**。
>
> ⚠️ 工具说明：`check-spec-symbols.mjs` 只扫 `component-specs`/`references`/`SKILL.md` 里 tsx 代码块的 import 标识符，**不扫 page-references**，且本仓基线有 2 条历史 ghost（永远 exit1）——**不适合**做本场景的 exit0 闭环。故本场景用 **SOP S1.5 的存在性校验**（`grep + test` 机械核对引用的 spec 文件是否存在，符合硬约束「结论以 grep 计数为准」）。

## 0. 人工触发（真实入口）📸 `D0-trigger.png`
改完页面模板引用（§2）后，对话框说口令：**`对齐 skill 规范`**。AI 读 `SOP.md` A 段（S1.5 页面模板校验）→ 执行下述存在性校验。

## 1. 基线校验（对照起点）
```
ROOT=.codebuddy/skills/clawpro-portable-design-skill
FILE=$ROOT/assets/page-references/admin-members.md
miss=0
for s in $(grep -oE "component-specs/[a-z0-9-]+\.md" "$FILE" | sort -u); do
  [ -f "$ROOT/$s" ] && echo "OK   $s" || { echo "MISS $s"; miss=$((miss+1)); }
done
echo "missing=$miss"; test $miss -eq 0; echo "exit=$?"
```
**实测**：11 个引用 spec 全部 OK · missing=0 · **exit=0**。

## 2. 制造 broken 引用
把 `admin-members.md` 状态行的 spec 引用从 `component-specs/status-tag.md` 改成不存在的 `component-specs/status-badge.md`。

## 3. 触发校验（报错）
```
# 同第 1 步命令
```
**实测证据**：`MISS component-specs/status-badge.md` · missing=1 · **exit=1** → 进「对不上清单」。

## 4. 修复并复跑（通过）
```
git checkout -- .codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.md
# 复跑第 1 步校验
```
**实测证据**：missing=0 · **exit=0**，引用重新挂对。

## 5. 亮点话术（PPT）
- **页面模板只校验、不覆盖**：Tier2 独有资产（页面模板、场景分流、portable fallback）在 reconcile 时受保护。
- **存在性 + 挂载自洽**：机械核对模板引用的 spec 文件真实存在，防 ghost / 错挂（对应历史上引用不存在 `tree-select.md` 的漂移）。

## 6. 录制取帧（截图存本文件夹）
| 文件名 | 截什么 |
|---|---|
| `D3-check-miss-exit1.png` | `MISS status-badge.md` + `exit=1` |
| `D4-check-ok-exit0.png` | 修复后 `missing=0` + `exit=0` |
| `D-recording.mp4` | 本场景全程录屏 |
