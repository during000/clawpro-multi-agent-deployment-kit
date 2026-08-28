---
description: 当用户对任何 Skill 生成结果表达不满、纠错、补充需求或高度赞赏时，AI 必须调用 feedback Skill，对反馈做 FEC 象限判定并把象限 A/B 增量落盘到 .codebuddy/feedback/{skill-name}/ 下。适用于所有 Skill 的对话。
alwaysApply: true
---

# Global Feedback Rule（薄触发层）

> **定位**：项目级全局触发声明。本文件**只负责"何时该启动反馈治理"**，不再重复判定标准与模板。
> **权威来源**：判定、落盘、模板一律以 `skills/feedback/` Skill 为准。
> **说明**：在 hook 自动化机制就绪前，本 `alwaysApply` 规则作为兜底触发；就绪后主触发改由 hook 承担，本规则降级为交互式兜底。

---

## 何时触发（触发即调用 feedback Skill）

当用户在对话中对当前 Skill 的产物出现以下任一信号时，AI 必须在**同一轮**调用 `feedback` Skill 走完整流程：

- 否定/纠错："这个不对""不符合预期""有问题""应该改成""不需要这段"
- 缺漏/冗余："漏了…""少了…""太啰嗦""太简略""删掉…"
- 赞赏（优秀案例）："这个效果很好，记录一下"
- **平静提出的新需求**，但暴露了 AI 默认做法中会复发的系统性缺陷（元原则见 Skill）

## 调用后由 Skill 负责的事（本文件不再展开）

- FEC 2×2 象限判定 → `skills/feedback/references/fec-framework.md`
- 落盘路径 / 幂等 / 追加不新建 / 收口时机 → `skills/feedback/SKILL.md` 第四节
- 反馈单模板 → `skills/feedback/references/feedback-template.md`

> **强制**：内容修改与反馈记录必须**同轮完成**，严禁"先改内容、回头再补反馈"。象限 C/D 不生成文件。
