本文介绍 ClawPro 管控端的核心功能模块，及对应页面说明，帮助您快速掌握管理能力。



## 功能总览
<table>
<tr>
<td rowspan="1" colSpan="1" >**功能项**</td>

<td rowspan="1" colSpan="1" >**功能模块**</td>

<td rowspan="1" colSpan="1" >**说明**</td>
</tr>

<tr>
<td rowspan="2" colSpan="1" >基础信息</td>

<td rowspan="1" colSpan="1" >基础信息配置</td>

<td rowspan="1" colSpan="1" >**全局基础信息管理：**提供平台基础信息统一管理能力，覆盖地域、腾讯云账号、服务器 IP、访问域名、网站名称、描述、Logo 等核心配置项，支持管理员对指定字段进行编辑修改。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >成员管理</td>

<td rowspan="1" colSpan="1" >- **新增成员：**支持单个添加及批量导入。<br>- **管理成员：**<br>  - 可编辑成员信息、重置成员密码、启用/禁用成员。<br>  - 可配置成员角色，每人 OpenClaw 实例数量上限，及每日 Tokens 用量上限。</td>
</tr>

<tr>
<td rowspan="3" colSpan="1" >OpenClaw 配置</td>

<td rowspan="1" colSpan="1" >模型配置</td>

<td rowspan="1" colSpan="1" >- **模型配置：**统一管控用户可使用的 AI 大模型（例如 DeepSeek，混元等），支持自定义模型。<br>- **Tokens 数量上限设置：**可设置单个模型及企业全局每日 Tokens 数量上限。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >通道配置</td>

<td rowspan="1" colSpan="1" >**IM 通道管理：**统一管理成员在 OpenClaw 中可配置的 IM 通道（企业微信、QQ、飞书、钉钉）。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >技能配置</td>

<td rowspan="1" colSpan="1" >**地址设置：**支持自定义技能获取地址，默认从 ClawHub 官方获取。</td>
</tr>

<tr>
<td rowspan="2" colSpan="1" >云设备配置</td>

<td rowspan="1" colSpan="1" >镜像管理</td>

<td rowspan="1" colSpan="1" >- **镜像导入：**支持导入腾讯云控制台的所有镜像。<br>- **镜像管理：**支持启用/禁用、删除镜像。同一时间仅有一个镜像处于生效状态。<br>- **安全组管理：**统一配置服务器网络安全组规则。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >网络管理</td>

<td rowspan="1" colSpan="1" >- **规则新增：**支持自定义服务器网络安全组规则，包括入站规则和出站规则，<br>- **规则生效：**支持修改和删除安全组操作。规则按从上到下的顺序匹配，命中第一条匹配规则后即停止。规则修改后，对运行中的OpenClaw云服务器立即生效。</td>
</tr>

<tr>
<td rowspan="2" colSpan="1" >运营监控</td>

<td rowspan="1" colSpan="1" >OpenClaw 监控</td>

<td rowspan="1" colSpan="1" >- **全局 OpenClaw 监控**：支持监控企业全局 OpenClaw 的数量及运行状态。<br>- **一键删除**：支持一键删除违规或异常 OpenClaw 实例。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >Tokens 监控</td>

<td rowspan="1" colSpan="1" >- **全局 Tokens 监控**：支持实时监控全局 Tokens 数据。<br>- **自定义查询：**支持按指定时间查询模型维度、成员维度的 Tokens 数据。</td>
</tr>

<tr>
<td rowspan="3" colSpan="1" >会话与监控</td>

<td rowspan="1" colSpan="1" >会话管理</td>

<td rowspan="1" colSpan="1" >**会话监控与管理：**支持查看所有 OpenClaw 实例的会话列表，包括会话状态、创建时间、持续时间等信息。支持强制关闭异常会话。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >运维观测</td>

<td rowspan="1" colSpan="1" >**系统运维监控：**实时监控系统资源使用情况（CPU、内存、磁盘等），支持按时间范围查询系统性能指标。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >Token 监控（按会话）</td>

<td rowspan="1" colSpan="1" >**会话级 Token 监控：**支持按会话维度查询 Token 消耗情况，便于精细化管理和成本控制。</td>
</tr>

<tr>
<td rowspan="1" colSpan="1" >安全审计</td>

<td rowspan="1" colSpan="1" >操作记录</td>

<td rowspan="1" colSpan="1" >**操作事件审计：**提供所有成员活动事件的历史记录，便于审计追溯。</td>
</tr>
</table>


## 

## 成员管理

面向企业管理员提供统一管理企业用户账号和资源配额功能。
- **添加成员：**单击右上角的**添加成员**后，可选择**单个添加**或**批量导入**成员。

- **管理成员**：可单击成员所在行右侧的**编辑**、**重置密码**、**启用/禁用**，按需进行成员管理操作。

   ![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/6f9bb01f1f0211f191f752540097cba1.png)


## **模型配置**

面向管理员提供统一管理成员使用的 AI 大模型功能，包括模型、单个模型和全局 Tokens 数量上限。
- **添加及管理模型：**单击右上角的**添加模型**后，管理员可以配置模型版本。配置模型后，可选择**启用/禁用**，或**删除**模型。
   

   > **说明：**
   > 

   > 成员只能使用已启用的模型。
   > 

- **设置模型或全局 Tokens 数量上限：**管理员可以设置每日单个模型或全局 Tokens 数量上限。

   ![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/7dda730e1f0211f18ab5525400a31896.png)


## **通道配置**

面向管理员提供预设 IM 通道（企业微信、QQ、飞书、钉钉）管理功能，控制成员在 OpenClaw 中可选择的 IM 通道。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/894b8b211f0211f1b3935254001d6acc.png)


## 技能配置

面向管理员提供自定义 SkillHub 地址功能，可从企业配置的专属 SkillHub 地址上或从默认的 ClawHub 上加载可用技能。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/968919fd1f0211f19bcc525400370dda.png)


## **镜像管理**

单击右上角的**导入镜像**后，可选择导入指定的腾讯云镜像，并且支持设置为**生效**，或**删除**镜像。

> **注意：**
> 

> 删除镜像后数据将无法恢复，请谨慎操作。
> 


![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/cca6ef121f0311f191f752540097cba1.png)


## **网络管理**

选择**网络管理**后，可单击右上角的**添加规则**进行安全组规则添加，或者**删除**已有的安全组规则。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/e01b8e0b1f0311f191f752540097cba1.png)


## OpenClaw 监控

面向管理员提供全局 OpenClaw 监控能力，可以查看企业所有的 OpenClaw 实例。

**删除 OpenClaw 实例：**如发现违规或异常 OpenClaw 实例，管理员可单击实例所在行右侧的**删除**来快速清理。

> **注意：**
> 

> 删除后原 OpenClaw 实例数据将无法找回，请谨慎操作。
> 


![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/fb90dff61f0311f18ab5525400a31896.png)


## Tokens 监控

面向管理员提供全局 Tokens 数量监控能力，可以查询指定时间范围内、或按模型、成员的 Tokens 使用数据。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/1c1599dc1f0411f18ab5525400a31896.png)


## 会话管理

面向管理员提供全局会话监控和管理能力，支持查看所有 OpenClaw 实例的活跃会话。

- **会话列表查看：**支持查看每个 OpenClaw 实例的会话信息，包括会话 ID、用户、创建时间、持续时间、会话状态等。

- **强制关闭会话：**当检测到异常或需要强制结束的会话时，管理员可单击会话所在行右侧的**关闭**按钮来强制结束会话。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/27f30b861f0411f191f752540097cba1.png)


## 运维观测

面向管理员提供实时系统运维监控能力，便于全面了解系统资源使用情况。

- **实时性能指标监控：**支持监控系统 CPU、内存、磁盘等资源使用情况，帮助管理员及时发现性能瓶颈。

- **自定义时间范围查询：**支持按指定时间范围查询系统性能指标，便于分析批量作业或高峰时段的资源消耗情况。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/27f30b861f0411f191f752540097cba1.png)


## Token 监控（按会话）

面向管理员提供会话级 Token 消耗监控能力，支持按会话维度精细化管理 Token 消耗。

- **会话级 Token 查询：**支持按会话 ID、用户、OpenClaw 实例等维度查询 Token 消耗情况，便于精细化成本控制。

- **自定义时间范围查询：**支持按指定时间范围查询不同会话的 Token 消耗数据，便于分析特定时间段的使用特点。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/1c1599dc1f0411f18ab5525400a31896.png)


## 操作记录

面向管理员提供查询所有成员活动事件历史记录能力，便于审计追溯。

![](https://write-document-release-1258344699.cos.ap-guangzhou.tencentcos.cn/100037379021/27f30b861f0411f191f752540097cba1.png)


