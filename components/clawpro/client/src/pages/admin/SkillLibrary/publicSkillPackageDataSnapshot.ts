/**
 * SkillHub 公共技能包数据快照
 *
 * 由 `scripts/fetch-skillhub-snapshot.mjs` 自动生成，请勿手工编辑。
 * 数据来源：https://api.skillhub.cn/api/v1/skillsets
 * 生成时间：2026-05-26T07:47:15.400Z
 * 共 42 个技能包
 */

import type { PublicSkillPackageRaw } from './publicSkillPackageMockData';

export const SKILLHUB_PACKAGE_SNAPSHOT: PublicSkillPackageRaw[] = [
  {
    "id": 1,
    "slug": "academic-academic-writing",
    "displayName": "学术写作",
    "summary": "从论文素材收集与研究框架构建（主题聚焦/文献整理/论点提炼）、学术论文润色与语言精炼（语法修正/术语规范/逻辑优化/风格统一）、论文辅助写作与结构化撰写（摘要/引言/方法/结果/讨论/结论）与学术论文框架搭建（章节规划/段落组织/论证逻辑），到论文正文撰写（学术规范写作/引用格式/可提交全文）与查重降重优化（原创性检测/重复率降低/表述改写）的完整学术写作工作流。覆盖素材收集、润色精炼、结构撰写、框架搭建、正文写作、查重优化全链路。",
    "scene": "academic",
    "subScene": "academic-writing",
    "category": "academic",
    "content": "---\nscene: \"academic\"\nsub_scene: \"academic-writing\"\nskills:\n  - \"research-paper-writer\"\n  - \"academic-writing-refiner\"\n  - \"academic-paper-assistant\"\n  - \"zeelin-academic-paper\"\n  - \"academic-writer\"\n  - \"paper-originality-studio\"\n---\n\n# 学术写作工作流\n\n你现在要完成一项学术论文写作任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：论文素材收集与框架构建（获取层）\n使用 **Research Paper Writer** 完成：\n- 围绕研究主题收集和整理论文素材\n- 提炼核心论点和研究问题\n- 构建论文整体研究框架和写作大纲\n- 确定章节结构和论证逻辑路径\n- 生成初步的论文写作计划\n\n输出论文框架大纲和素材清单。\n\n## 步骤 2：学术语言润色与精炼（获取层）\n使用 **academic-writing-refiner** 完成：\n- 对已有草稿进行学术语言润色\n- 修正语法错误和术语使用不当之处\n- 优化句式结构和论述逻辑\n- 统一全文学术写作风格\n- 确保表述符合学术规范要求\n\n输出润色后的学术文本。\n\n## 步骤 3：论文结构化辅助撰写（分析层）\n使用 **Academic Paper Assistant** 完成：\n- 按学术论文标准结构进行辅助撰写\n- 完善摘要、引言、方法、结果、讨论、结论各章节\n- 确保各章节之间逻辑衔接紧密\n- 规范参考文献引用格式\n- 检查论文整体结构完整性\n\n输出结构化论文草稿。\n\n## 步骤 4：论文框架搭建与章节规划（分析层）\n使用 **ZeeLin Academic Paper** 完成：\n- 细化论文各章节的段落组织\n- 搭建完整的论证逻辑框架\n- 优化章节间的过渡和衔接\n- 确保研究方法与结论的一致性\n- 补充必要的图表和数据说明\n\n输出完善的论文章节框架。\n\n## 步骤 5：论文正文学术规范写作（输出层）\n使用 **Academic Writer** 完成：\n- 按学术规范完成论文正文撰写\n- 严格遵守目标期刊/学位论文格式要求\n- 确保引用标注准确、参考文献完整\n- 保持论述逻辑严密、语言专业准确\n- 生成可直接提交的学术论文全文\n\n输出学术规范的论文正文。\n\n## 步骤 6：查重降重与原创性优化（输出层）\n使用 **paper-originality-studio** 完成：\n- 对论文全文进行原创性检测\n- 识别重复率较高的段落和句子\n- 通过表述改写降低重复率\n- 保持改写后的学术准确性和可读性\n- 输出降重优化后的最终版本\n\n输出查重报告和降重优化后的论文终稿。\n\n## 最终输出\n将以上步骤的结果整合为完整的学术写作成果包，交付以下文件：\n1. **论文框架大纲**：研究框架、章节结构、写作计划\n2. **润色文本**：语言精炼、术语规范、风格统一\n3. **结构化草稿**：各章节完整撰写、引用规范\n4. **章节框架**：段落组织、论证逻辑、过渡衔接\n5. **论文正文**：学术规范写作、可提交全文\n6. **查重优化稿**：原创性检测、降重改写、最终版本",
    "skillSlugs": [
      "research-paper-writer",
      "academic-writing-refiner",
      "academic-paper-assistant",
      "zeelin-academic-paper",
      "academic-writer",
      "paper-originality-studio"
    ],
    "skillCount": 6
  },
  {
    "id": 2,
    "slug": "academic-literature-review",
    "displayName": "文献综述",
    "summary": "从透明严谨的双轮深度调研（APA引用/证据分级/用户确认）与专精文献综述论文分析学术写作、12要素结构化论文拆解（研究背景/问题/方法/贡献/不足/展望）与8阶段系统性中英文文献回顾，到论文大纲文献综述框架摘要生成引用格式规范与学术写作文献综述研究方法撰写的完整文献综述工作流。覆盖文献调研、论文拆解、综述撰写、格式规范全链路。",
    "scene": "academic",
    "subScene": "literature-review",
    "category": "academic",
    "content": "---\nscene: \"academic\"\nsub_scene: \"literature-review\"\nskills:\n  - \"academic-deep-research\"\n  - \"academic-researcher\"\n  - \"paper-analyzer\"\n  - \"literature-reviewer-skill\"\n  - \"thesis-helper\"\n  - \"academic-writing\"\n---\n\n# 文献综述工作流\n\n你现在要完成一项文献综述的撰写任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：双轮深度调研（获取层）\n使用 **Academic Deep Research** 完成：\n- 对每个主题强制执行双轮调研确保覆盖全面\n- 采用 APA 7 版引用格式标注所有来源\n- 对证据进行分级评估（强/中/弱）\n- 通过 3 次用户确认保证研究方向准确\n- 生成详尽的调研报告和文献清单\n\n输出双轮调研报告和分级证据清单。\n\n## 步骤 2：文献综述与论文分析（获取层）\n使用 **Academic Researcher** 完成：\n- 审阅已收集的学术论文\n- 开展系统性文献综述分析\n- 识别研究主题、趋势和空白\n- 梳理各论文间的逻辑关系和理论脉络\n- 形成文献综述的整体框架\n\n输出文献综述框架和论文关系图谱。\n\n## 步骤 3：论文结构化深度拆解（分析层）\n使用 **Paper Analyzer** 完成：\n- 基于 12 个阅读要素深度拆解每篇论文\n- 提取研究背景、研究问题、研究方法和理论框架\n- 分析一致性发现和不一致性发现\n- 评估研究贡献、研究不足和未来展望\n- 将拆解结果保存为结构化 Excel 文件\n\n输出论文深度拆解表和要素分析。\n\n## 步骤 4：系统性中英文文献回顾（分析层）\n使用 **Literature Reviewer Skill** 完成：\n- 采用 8 阶段工作流进行系统性文献回顾\n- 支持 CNKI、Web of Science、ScienceDirect 等数据库\n- 无需 API 配置，通过浏览器自动化获取文献\n- 中英文文献同步回顾和交叉分析\n- 生成文献回顾的完整报告\n\n输出系统性文献回顾报告。\n\n## 步骤 5：综述框架与大纲生成（输出层）\n使用 **Thesis Helper** 完成：\n- 生成论文大纲和文献综述框架\n- 撰写研究摘要和各章节概要\n- 转换和规范引用格式\n- 执行格式规范检查\n- 准备答辩相关材料\n\n输出综述大纲、框架和格式规范文档。\n\n## 步骤 6：学术综述正文撰写（输出层）\n使用 **academic-writing** 完成：\n- 按学术规范撰写文献综述正文\n- 包含引言、主题分析、批判性讨论和结论\n- 严格遵守引用标准和学术写作规范\n- 保持论述逻辑严密、语言专业准确\n- 输出可直接提交的文献综述全文\n\n## 最终输出\n将以上步骤的结果整合为完整的文献综述包，交付以下文件：\n1. **深度调研报告**：双轮调研、证据分级、APA 引用\n2. **文献框架图谱**：主题梳理、趋势识别、研究空白\n3. **论文拆解表**：12 要素分析、结构化 Excel\n4. **系统性回顾报告**：中英文文献、8 阶段工作流\n5. **综述大纲与框架**：章节结构、引用格式、格式规范\n6. **综述正文**：学术写作、批判性讨论、可提交全文",
    "skillSlugs": [
      "academic-deep-research",
      "academic-researcher",
      "paper-analyzer",
      "literature-reviewer-skill",
      "thesis-helper",
      "academic-writing"
    ],
    "skillCount": 6
  },
  {
    "id": 3,
    "slug": "academic-paper-search",
    "displayName": "论文检索",
    "summary": "从多数据库学术论文检索下载与引文提取（arXiv/PubMed/Semantic Scholar/Google Scholar）、跨平台系统性文献检索（IEEE/ACM/Scopus/Web of Science）与批量PDF下载元数据提取索引生成，到科研文献智能监测与中文摘要定时推送、知网CNKI高级检索自动化与真实参考文献引用规范管理的完整论文检索工作流。覆盖文献搜索、批量下载、智能监测、引用管理全链路。",
    "scene": "academic",
    "subScene": "paper-search",
    "category": "academic",
    "content": "---\nscene: \"academic\"\nsub_scene: \"paper-search\"\nskills:\n  - \"academic-research-hub\"\n  - \"literature-search\"\n  - \"scholar-paper-downloader\"\n  - \"research-paper-monitor\"\n  - \"cnki-advanced-search\"\n  - \"academic-citation-manager\"\n---\n\n# 论文检索工作流\n\n你现在要完成一项论文检索与文献获取任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：多数据库学术检索（获取层）\n使用 **Academic Research Hub** 完成：\n- 在 arXiv、PubMed、Semantic Scholar、Google Scholar 等数据库检索论文\n- 根据关键词、作者、年份等条件精确搜索\n- 下载文献 PDF 并提取引文信息\n- 获取论文摘要、引用次数和影响因子\n- 建立初始文献检索结果集\n\n输出多数据库检索结果和文献列表。\n\n## 步骤 2：跨平台系统性文献检索（获取层）\n使用 **literature-search** 完成：\n- 在 Google Scholar、PubMed、arXiv、IEEE、ACM、Scopus、Web of Science 上系统检索\n- 获取引用列表和相关文献推荐\n- 查找特定主题的关键论文\n- 支持多字段搜索和高级过滤\n- 补充步骤 1 未覆盖的数据库来源\n\n输出跨平台系统性检索结果。\n\n## 步骤 3：文献批量下载与索引（分析层）\n使用 **scholar-paper-downloader** 完成：\n- 从 arXiv、PubMed、PMC、Semantic Scholar 等批量下载论文 PDF\n- 自动提取论文元数据（标题/作者/年份/期刊）\n- 生成结构化索引列表\n- 优先从官方免费渠道下载\n- 付费文献提供手动下载指引\n\n输出批量下载的 PDF 文件和索引清单。\n\n## 步骤 4：科研文献智能监测（分析层）\n使用 **Research Paper Monitor** 完成：\n- 自动监测 arXiv、PubMed、CNKI 等多个学术信源\n- 根据关注领域和关键词采集最新论文\n- 生成中文摘要便于快速浏览\n- 设置定时推送和提醒机制\n- 持续跟踪学术前沿动态\n\n输出文献监测报告和定时推送配置。\n\n## 步骤 5：知网高级检索（输出层）\n使用 **知网高级检索** 完成：\n- 在知网（CNKI）高级检索页面自动化检索\n- 选择学术期刊类别并勾选 CSSCI 来源\n- 输入主题关键词（含同义词和同位词）\n- 多组关键词用 OR 关系连接\n- 获取中文核心期刊论文\n\n输出知网检索结果和中文文献列表。\n\n## 步骤 6：引用管理与规范化（输出层）\n使用 **Academic Citation Manager** 完成：\n- 为科研论文和毕业论文添加真实参考文献\n- 规范引用标注格式（APA/MLA/Chicago/GB/T 7714）\n- 生成 BibTeX 和参考文献列表\n- 检查引用完整性和格式一致性\n- 输出规范化的参考文献清单\n\n## 最终输出\n将以上步骤的结果整合为完整的论文检索包，交付以下文件：\n1. **多数据库检索结果**：文献列表、摘要、引用次数\n2. **跨平台系统检索**：IEEE/ACM/Scopus 补充文献\n3. **批量下载文件**：PDF 文件、元数据索引\n4. **文献监测报告**：最新论文、中文摘要、推送配置\n5. **知网检索结果**：CSSCI 文献、中文核心期刊\n6. **规范引用清单**：参考文献列表、BibTeX、格式校验",
    "skillSlugs": [
      "academic-research-hub",
      "literature-search",
      "scholar-paper-downloader",
      "research-paper-monitor",
      "cnki-advanced-search",
      "academic-citation-manager"
    ],
    "skillCount": 6
  },
  {
    "id": 4,
    "slug": "academic-statistical-analysis",
    "displayName": "统计分析",
    "summary": "从R语言82种统计方法（回归/生存/贝叶斯/荟萃/因果推断/结构方程）与高级生物统计分析（贝叶斯推断/蒙特卡洛模拟/机器学习/生存模型）、标准化数据分析工作流（数据清洗/统计分析/科学可视化）与37种统计检验方法（正态性/位置/相关性/时间序列/模型诊断），到中文数据分析报告（统计分析/可视化建议/趋势检测）与数学统计绘图引擎（图表生成/计算验证/可视化呈现）的完整统计分析工作流。覆盖统计方法选择、高级建模、标准化流程、检验验证、分析报告、图表输出全链路。",
    "scene": "academic",
    "subScene": "statistical-analysis",
    "category": "academic",
    "content": "---\nscene: \"academic\"\nsub_scene: \"statistical-analysis\"\nskills:\n  - \"r-stats\"\n  - \"biostatistics\"\n  - \"data-analysis-workflow\"\n  - \"statistics-2\"\n  - \"data-analyst-cn\"\n  - \"mathgraphs\"\n---\n\n# 统计分析工作流\n\n你现在要完成一项科研数据的统计分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：统计方法选择与建模（获取层）\n使用 **R Stats — 82 Statistical Methods** 完成：\n- 根据研究设计和数据类型选择合适的统计方法\n- 覆盖回归分析、生存分析、贝叶斯统计、荟萃分析等 82 种方法\n- 执行因果推断、结构方程模型等高级分析\n- 进行项目反应理论和临床试验设计相关统计\n- 输出统计建模的初步结果和模型参数\n\n输出统计方法选择方案和初步建模结果。\n\n## 步骤 2：高级统计建模与模拟（获取层）\n使用 **Biostatistics: Actuarial-Level Statistical Analysis** 完成：\n- 执行贝叶斯推断和后验概率估计\n- 运行蒙特卡洛模拟评估模型稳健性\n- 构建生存模型和风险评估框架\n- 结合机器学习方法进行特征筛选和预测\n- 对复杂数据进行多层次统计建模\n\n输出高级统计模型和模拟分析报告。\n\n## 步骤 3：标准化数据分析流程（分析层）\n使用 **Data Analysis Workflow** 完成：\n- 按标准化工作流执行数据清洗和预处理\n- 进行描述性统计和探索性数据分析\n- 执行核心统计分析并记录分析过程\n- 生成科学可视化图表辅助理解\n- 确保分析流程可复现、结果可追溯\n\n输出标准化分析流程文档和中间结果。\n\n## 步骤 4：统计检验与模型诊断（分析层）\n使用 **Pywayne Statistics** 完成：\n- 执行正态性检验验证数据分布假设\n- 进行位置检验和相关性检验\n- 运行时间序列检验评估数据时序特征\n- 执行模型诊断检验评估拟合质量\n- 覆盖 37 种以上统计检验方法确保结论稳健\n\n输出统计检验结果汇总和模型诊断报告。\n\n## 步骤 5：中文数据分析报告（输出层）\n使用 **Data Analyst Cn** 完成：\n- 将统计分析结果整理为中文分析报告\n- 提供数据可视化建议和图表方案\n- 进行趋势检测和异常值识别\n- 用通俗语言解读统计结论\n- 给出基于数据的决策建议\n\n输出中文数据分析报告和可视化方案。\n\n## 步骤 6：统计图表生成与验证（输出层）\n使用 **Math & Statistics Graphing Engine** 完成：\n- 绘制统计分析所需的各类专业图表\n- 生成回归图、箱线图、分布图、热力图等\n- 对计算结果进行图形化验证\n- 确保图表符合学术发表规范\n- 输出可直接用于论文的高质量统计图表\n\n输出学术级统计图表集。\n\n## 最终输出\n将以上步骤的结果整合为完整的统计分析成果包，交付以下文件：\n1. **统计方法方案**：方法选择、初步建模、参数估计\n2. **高级模型报告**：贝叶斯推断、蒙特卡洛模拟、生存模型\n3. **标准化流程文档**：数据清洗、探索性分析、过程记录\n4. **检验诊断报告**：37+ 统计检验、模型诊断、稳健性评估\n5. **中文分析报告**：结果解读、趋势检测、决策建议\n6. **统计图表集**：专业绘图、图形验证、学术发表级图表",
    "skillSlugs": [
      "r-stats",
      "biostatistics",
      "data-analysis-workflow",
      "statistics-2",
      "data-analyst-cn",
      "mathgraphs"
    ],
    "skillCount": 6
  },
  {
    "id": 7,
    "slug": "design-brand-visual",
    "displayName": "品牌视觉",
    "summary": "从品牌圣经构建（品牌定位/语调/视觉识别/规范文档）、平面设计与品牌调性升级指引、品牌配色方案生成与WCAG对比度校验，到几何图元与负空间极简Logo设计、基于算法设计哲学的品牌海报与视觉物料制作、SVG矢量插图与品牌图形资源生成的完整品牌视觉工作流。覆盖品牌策略、视觉识别系统、配色排版和物料输出全链路。",
    "scene": "design",
    "subScene": "brand-visual",
    "category": "design",
    "content": "---\nscene: \"design\"\nsub_scene: \"brand-visual\"\nskills:\n  - \"brand-cog\"\n  - \"visual\"\n  - \"color-palette-cn\"\n  - \"logo-creator\"\n  - \"poster\"\n  - \"svg-draw\"\n---\n\n# 品牌视觉工作流\n\n你现在要完成一项品牌视觉识别系统的设计任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：品牌圣经构建（获取层）\n使用 **brand-cog** 完成：\n- 基于品牌简报（Brief）梳理品牌定位和核心价值主张\n- 定义品牌个性特质和语调风格（Tone of Voice）\n- 构建视觉识别系统框架（色彩方向、字体方向、图像风格）\n- 制定品牌规范文档（Brand Guideline）大纲\n- 输出 Logo 设计方向和概念关键词\n\n输出品牌圣经文档和视觉识别系统框架。\n\n## 步骤 2：视觉设计指导与调性升级（分析层）\n使用 **Visual** 完成：\n- 基于品牌定位提供平面设计方向指引\n- 评估和优化品牌视觉调性（高端/年轻/科技/温暖等）\n- 制定排版系统规范（标题/正文/辅助文字层级）\n- 定义图像和插画风格指南\n- 提供 UI 交互场景下的品牌视觉应用建议\n\n输出视觉设计指引文档。\n\n## 步骤 3：品牌配色系统设计（分析层）\n使用 **color-palette-cn** 完成：\n- 基于品牌调性生成主色、辅色和点缀色方案\n- 运用色彩和谐理论（互补色/类似色/三色）确保配色协调\n- 检查配色的 WCAG 对比度合规性（文字/背景可读性）\n- 输出多种格式的色值（HEX/RGB/HSL）\n- 生成浅色/深色模式的配色变体\n\n输出品牌配色方案和对比度检查报告。\n\n## 步骤 4：Logo 设计（输出层）\n使用 **Logo Creator** 完成：\n- 运用几何图元和负空间技法设计品牌 Logo\n- 确保 Logo 在不同尺寸下的可识别性（favicon 到海报级）\n- 设计 Logo 的标准版、反白版和单色版\n- 定义 Logo 安全区域和最小使用尺寸\n- 生成矢量风格的极简 Logo 方案\n\n输出 Logo 设计方案（含变体）。\n\n## 步骤 5：品牌海报与视觉物料（输出层）\n使用 **海报设计skill** 完成：\n- 基于品牌视觉规范构建设计哲学（Algorithmic Philosophy）\n- 设计品牌主视觉海报（品牌发布/活动/宣传）\n- 制作社交媒体视觉物料（封面图/头像/配图模板）\n- 应用品牌排版系统和配色方案\n- 生成可渲染的 HTML 视觉稿\n\n输出品牌海报和社交媒体视觉物料。\n\n## 步骤 6：SVG 品牌图形资源（输出层）\n使用 **SVG Draw** 完成：\n- 生成品牌专属的矢量插图和图标\n- 创建品牌辅助图形元素（装饰线条/几何图案/纹理）\n- 制作可缩放的品牌图标集（SVG 格式）\n- 将 SVG 转换为 PNG 格式供不同场景使用\n- 确保图形风格与品牌视觉系统统一\n\n输出 SVG 品牌图形资源包。\n\n## 最终输出\n将以上步骤的结果整合为完整的品牌视觉设计包，交付以下文件：\n1. **品牌圣经**：品牌定位、个性、语调、视觉识别框架\n2. **视觉设计指引**：排版系统、图像风格、调性规范\n3. **品牌配色方案**：主色/辅色/点缀色、对比度报告、多模式变体\n4. **Logo 设计方案**：标准版/反白版/单色版 + 使用规范\n5. **品牌视觉物料**：海报、社交媒体模板\n6. **SVG 图形资源**：品牌图标、辅助图形、矢量插图",
    "skillSlugs": [
      "brand-cog",
      "visual",
      "color-palette-cn",
      "logo-creator",
      "poster",
      "svg-draw"
    ],
    "skillCount": 6
  },
  {
    "id": 8,
    "slug": "design-interaction-design",
    "displayName": "交互设计",
    "summary": "从数据驱动的用户画像构建与旅程映射、UI/UX信息架构与视觉风格设计、自动化UX可用性审计（视觉层级/认知负荷/导航评估）、微交互设计与WCAG 2.2无障碍规范，到量化美学与动效优化、WCAG 2.1 AA合规实现的完整交互设计工作流。覆盖用户研究、交互方案、可用性评估和无障碍实现全链路。",
    "scene": "design",
    "subScene": "interaction-design",
    "category": "design",
    "content": "---\nscene: \"design\"\nsub_scene: \"interaction-design\"\nskills:\n  - \"ux-researcher-designer\"\n  - \"ui-ux-pro-max\"\n  - \"ui-audit\"\n  - \"ui-ux-design\"\n  - \"human-optimized-frontend\"\n  - \"accessibility\"\n---\n\n# 交互设计工作流\n\n你现在要完成一项全面的交互设计任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：用户研究与画像构建（获取层）\n使用 **Ux Researcher Designer** 完成：\n- 基于数据驱动方法生成用户画像（Persona）\n- 绘制用户旅程地图（Journey Map），标注关键触点和情绪曲线\n- 设计可用性测试框架和测试任务\n- 识别用户痛点、需求和行为模式\n- 产出用户研究洞察报告\n\n输出用户画像文档、旅程地图和研究洞察。\n\n## 步骤 2：信息架构与交互方案设计（分析层）\n使用 **UI/UX Pro Max** 完成：\n- 设计信息架构（IA），定义内容层级和导航结构\n- 制定视觉风格方向（色彩/字体/间距/图标风格）\n- 设计组件规格和交互状态（默认/悬停/按下/禁用/加载）\n- 编写界面文案和微文案（按钮/提示/错误信息）\n- 构建设计系统令牌和组件文档\n\n输出信息架构图、交互规格和设计系统文档。\n\n## 步骤 3：UX 可用性审计（分析层）\n使用 **UI Audit** 完成：\n- 基于 UX 成熟原则自动审计界面设计\n- 评估视觉层级是否清晰（主次关系、视觉引导）\n- 检查认知负荷是否合理（信息密度、决策复杂度）\n- 审查导航设计（可发现性、一致性、回退路径）\n- 评估无障碍性基础指标\n\n输出 UX 审计报告和优化建议清单。\n\n## 步骤 4：微交互与动效设计（分析层）\n使用 **UI/UX Design Guide** 完成：\n- 设计移动优先的交互模式和手势方案\n- 定义微交互（按钮反馈、加载动画、状态切换）\n- 制定色彩系统和字体排版层级\n- 确保设计符合 WCAG 2.2 无障碍标准\n- 集成 Tailwind + Shadcn/ui 的交互组件规范\n\n输出微交互规范和动效设计方案。\n\n## 步骤 5：交互原型优化与实现（输出层）\n使用 **human-optimized-frontend** 完成：\n- 通过量化评估共同优化美学、动效和用户体验\n- 确保交互动效流畅自然（缓动曲线、时序控制）\n- 优化界面视觉细节（阴影、渐变、过渡）\n- 生成视觉美观、体验良好的前端界面代码\n- 量化评分指导迭代优化\n\n输出优化后的前端交互代码和评估报告。\n\n## 步骤 6：无障碍合规实现（输出层）\n使用 **Accessibility** 完成：\n- 实现语义化 HTML 结构和适当的 ARIA 标签\n- 确保键盘导航完整（Tab 顺序、焦点管理、快捷键）\n- 验证颜色对比度（文本 4.5:1，大文本 3:1）\n- 测试屏幕阅读器兼容性\n- 实现实时区域（Live Region）和表单标签\n\n输出 WCAG 2.1 AA 合规的前端代码。\n\n## 最终输出\n将以上步骤的结果整合为完整的交互设计包，交付以下文件：\n1. **用户研究报告**：用户画像、旅程地图、痛点洞察\n2. **信息架构与交互规格**：IA 图、组件规格、状态定义\n3. **UX 审计报告**：视觉层级/认知负荷/导航评估和优化建议\n4. **微交互与动效方案**：交互模式、动效规范、时序定义\n5. **前端交互代码**：量化优化后的高质量界面实现\n6. **无障碍合规报告**：WCAG 2.1 AA 检查结果和合规代码",
    "skillSlugs": [
      "ux-researcher-designer",
      "ui-ux-pro-max",
      "ui-audit",
      "ui-ux-design",
      "human-optimized-frontend",
      "accessibility"
    ],
    "skillCount": 6
  },
  {
    "id": 9,
    "slug": "design-ui-prototype",
    "displayName": "UI 原型设计",
    "summary": "从产品需求零提问直出PRD到设计方向、设计稿像素级还原，到设计令牌体系与组件规范制定、UI设计质量审查与反模式识别，再到低保真线框图绘制和高保真生产级HTML/Tailwind原型输出的完整UI原型设计工作流。支持Figma/Sketch设计稿解析、响应式布局、移动优先设计、设计系统构建和交互原型生成。",
    "scene": "design",
    "subScene": "ui-prototype",
    "category": "design",
    "content": "---\nscene: \"design\"\nsub_scene: \"ui-prototype\"\nskills:\n  - \"prd-to-prototype\"\n  - \"design-to-code\"\n  - \"ui-design-system\"\n  - \"ui-design\"\n  - \"wireframe\"\n  - \"frontend-design-ultimate\"\n---\n\n# UI 原型设计工作流\n\n你现在要完成一项从产品需求到高保真原型的 UI 原型设计任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：需求分析与 PRD 输出（获取层）\n使用 **PRD to Prototype** 完成：\n- 接收用户的产品想法或需求描述\n- 零提问模式直接输出结构化 PRD（产品需求文档）\n- 明确目标用户、核心功能、页面列表和交互要求\n- 确认目标平台（移动端/PC端/双端）\n- 生成设计方向建议和功能优先级排序\n\n输出 PRD 文档和平台选择确认。\n\n## 步骤 2：设计稿解析与像素还原（获取层）\n使用 **Design To Code** 完成：\n- 如有现成设计稿（Figma、Sketch 或截图），进行像素级解析\n- 提取设计稿中的布局结构、间距、字体、颜色等参数\n- 识别响应式断点和自适应规则\n- 将设计稿转为结构化的设计规格说明\n\n输出设计规格参数和布局结构描述。若无现成设计稿则跳过此步，由后续步骤从零设计。\n\n## 步骤 3：设计系统与规范制定（分析层）\n使用 **Ui Design System** 完成：\n- 基于 PRD 和设计规格，构建设计令牌体系（颜色、字体、间距、圆角等）\n- 定义组件库规范（按钮、输入框、卡片、导航等基础组件）\n- 设计响应式布局策略和断点规则\n- 生成设计令牌文档（CSS 变量 / Tailwind 配置）\n- 建立组件命名和状态管理规范\n\n输出设计令牌文件和组件规范文档。\n\n## 步骤 4：UI 设计质量审查（分析层）\n使用 **UI Design** 完成：\n- 审查设计方案的布局合理性（视觉层级、信息密度）\n- 检查排版系统（字号层级、行高、对齐）\n- 验证色彩搭配（对比度、可访问性 WCAG 标准）\n- 评估间距一致性和组件复用性\n- 识别常见 UI 反模式并给出修正建议\n\n输出设计质量审查报告和优化建议。\n\n## 步骤 5：低保真线框图绘制（输出层）\n使用 **Wireframe** 完成：\n- 绘制核心页面的低保真线框图（ASCII 或 SVG 格式）\n- 标注页面间的跳转关系和用户流程\n- 定义每个页面的功能区块和内容占位\n- 导出线框图用于团队评审\n\n输出线框图文件和用户流程图。\n\n## 步骤 6：高保真原型生成（输出层）\n使用 **Frontend Design Ultimate** 完成：\n- 基于线框图和设计系统，生成高保真 HTML/Tailwind 原型\n- 实现移动优先的响应式布局\n- 添加微交互动效和过渡效果\n- 确保视觉效果达到生产级水准（Awwwards 级别）\n- 输出单文件可预览的 HTML 原型\n\n## 最终输出\n将以上步骤的结果整合为完整的 UI 原型设计包，交付以下文件：\n1. **PRD 文档**：产品需求、功能列表、目标平台\n2. **设计令牌与组件规范**：颜色/字体/间距令牌、组件库定义\n3. **设计审查报告**：布局/排版/色彩/可访问性检查结果\n4. **线框图**：核心页面低保真布局和用户流程\n5. **高保真 HTML 原型**：可直接在浏览器中预览的交互原型",
    "skillSlugs": [
      "prd-to-prototype",
      "design-to-code",
      "ui-design-system",
      "ui-design",
      "wireframe",
      "frontend-design-ultimate"
    ],
    "skillCount": 6
  },
  {
    "id": 10,
    "slug": "ecommerce-bidding-strategy",
    "displayName": "竞价策略",
    "summary": "从亚马逊Listing优化与PPC广告管理（标题优化/五点描述/后台关键词/PPC广告/评价管理/排名提升）与跨平台PPC策略规划（Google Ads/Meta等平台推荐/利润分析/广告投放方案），经亚马逊PPC广告构建与优化（关键词组织/出价策略/广告活动结构/效果优化）与多平台预算分配出价优化（Meta/Google/TikTok/YouTube/Amazon出价策略），到多平台广告创建优化（A/B测试/成本分析/ROI分析/精准定向/素材推荐）与Google Shopping广告优化（商品信息流/竞价策略/广告系列结构/绩效分析/ROAS优化）的完整竞价策略工作流。覆盖Listing优化、策略规划、广告构建、出价优化、效果监控、ROAS提升全链路。",
    "scene": "ecommerce",
    "subScene": "bidding-strategy",
    "category": "ecommerce",
    "content": "---\nscene: \"ecommerce\"\nsub_scene: \"bidding-strategy\"\nskills:\n  - \"amazon-listing\"\n  - \"ecommerce-ppc-strategy-planner\"\n  - \"amazon-ppc-campaign\"\n  - \"budget-bidding-optimizer\"\n  - \"ad-campaign-optimizer\"\n  - \"google-shopping-optimization\"\n---\n\n# 竞价策略工作流\n\n你现在要完成一项竞价策略制定任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：亚马逊 Listing 优化与 PPC 基础（获取层）\n使用 **amazon-listing** 完成：\n- 优化商品标题、五点描述和后台关键词\n- 分析当前 PPC 广告的投放表现\n- 评估关键词覆盖率和排名情况\n- 整理评价管理和排名提升数据\n- 为后续竞价策略提供 Listing 基础数据\n\n输出 Listing 优化报告和 PPC 现状分析。\n\n## 步骤 2：跨平台 PPC 策略规划（获取层）\n使用 **Ecommerce Ppc Strategy Planner** 完成：\n- 分析产品特性和利润结构\n- 评估 Google Ads、Meta、TikTok 等平台的适配度\n- 推荐最优广告投放平台组合\n- 制定各平台的初始预算分配方案\n- 规划跨平台 PPC 整体投放策略\n\n输出跨平台 PPC 策略规划方案。\n\n## 步骤 3：亚马逊 PPC 广告构建与优化（分析层）\n使用 **Amazon Ppc Campaign** 完成：\n- 从零设计完整的广告活动结构\n- 进行关键词组织和匹配类型设置\n- 制定分层出价策略（自动/手动/精准）\n- 优化现有广告活动的表现指标\n- 管理否定关键词减少无效花费\n\n输出亚马逊 PPC 广告方案和优化建议。\n\n## 步骤 4：多平台预算分配与出价优化（分析层）\n使用 **Ads Bid Optimizer** 完成：\n- 优化 Meta、Google、TikTok、YouTube、Amazon 的预算分配\n- 制定各平台的差异化出价策略\n- 基于历史数据调整出价参数\n- 平衡各渠道的投入产出比\n- 动态调整预算以应对竞争变化\n\n输出多平台预算分配方案和出价策略。\n\n## 步骤 5：广告创建与 A/B 测试优化（输出层）\n使用 **Ad Campaign Optimizer** 完成：\n- 自动创建多平台广告内容\n- 设计 A/B 测试方案对比不同素材效果\n- 分析广告成本和 ROI 表现\n- 实现精准受众定向和素材推荐\n- 持续迭代优化广告投放效果\n\n输出广告创建方案和 A/B 测试报告。\n\n## 步骤 6：Google Shopping 广告与 ROAS 优化（输出层）\n使用 **Google Shopping Optimization** 完成：\n- 优化 Google Shopping 商品信息流\n- 制定 Shopping 广告的竞价策略\n- 设计广告系列结构和组织逻辑\n- 分析绩效数据并优化 ROAS\n- 实现最大曝光率和转化效率\n\n输出 Google Shopping 优化报告和 ROAS 分析。\n\n## 最终输出\n将以上步骤的结果整合为完整的竞价策略成果包，交付以下文件：\n1. **Listing 基础**：标题优化、关键词覆盖、PPC 现状\n2. **策略规划**：平台选择、预算分配、投放方案\n3. **亚马逊 PPC**：广告结构、出价策略、否定关键词\n4. **出价优化**：多平台预算、差异化出价、动态调整\n5. **A/B 测试**：素材对比、ROI 分析、受众定向\n6. **Shopping 优化**：信息流、竞价策略、ROAS 分析",
    "skillSlugs": [
      "amazon-listing",
      "ecommerce-ppc-strategy-planner",
      "amazon-ppc-campaign",
      "budget-bidding-optimizer",
      "ad-campaign-optimizer",
      "google-shopping-optimization"
    ],
    "skillCount": 6
  },
  {
    "id": 11,
    "slug": "education-lesson-planning",
    "displayName": "教案设计",
    "summary": "从基于教学评一致性原则的专业教案写作（新课标/个性化教案/教学计划）与教师工具箱（教案设计/评分标准Rubric/课堂活动/评估设计）、课程目标驱动的教案生成（互动题/作业/分层教学建议）与课程模块结构规划（单元目标/作业设计/里程碑），到教师备课自动化（资料搜索/课文分析/生字词/教学设计）与教案内容PPT转化（讲师活动提取/幻灯片备注/一键更新）的完整教案设计工作流。覆盖教案撰写、工具辅助、分层设计、模块规划、备课自动化、PPT转化全链路。",
    "scene": "education",
    "subScene": "lesson-planning",
    "category": "education",
    "content": "---\nscene: \"education\"\nsub_scene: \"lesson-planning\"\nskills:\n  - \"teaching-plan-writer\"\n  - \"teacher-toolkit\"\n  - \"classroom-lesson-pack\"\n  - \"creator-course-outline\"\n  - \"teacher-prep\"\n  - \"ppt-lecture-notes\"\n---\n\n# 教案设计工作流\n\n你现在要完成一项教案设计任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：专业教案撰写（获取层）\n使用 **teaching-plan-writer** 完成：\n- 基于教学评一致性原则设计教案\n- 符合《义务教育课程标准（2022年版）》要求\n- 明确教学目标、重难点和教学过程\n- 设计导入环节、新授环节、练习环节和总结环节\n- 生成个性化的完整教案文档\n\n输出符合新课标的完整教案初稿。\n\n## 步骤 2：教学工具与评估设计（获取层）\n使用 **Teacher Toolkit** 完成：\n- 设计配套的评分标准（Rubric）\n- 规划课堂互动活动和教学策略\n- 制定学生评估方案和反馈机制\n- 准备家长沟通材料\n- 完善教案的评价维度\n\n输出评分标准、课堂活动方案和评估设计。\n\n## 步骤 3：分层教学与互动题设计（分析层）\n使用 **Classroom Lesson Pack** 完成：\n- 根据课程目标生成配套互动题\n- 设计分层教学建议适应不同学生水平\n- 制定课堂作业和课后练习\n- 确保教学内容与目标对齐\n- 生成完整的课堂教学资源包\n\n输出互动题库、分层教学方案和作业设计。\n\n## 步骤 4：课程模块与单元规划（分析层）\n使用 **Creator Course Outline** 完成：\n- 设计课程模块结构和教学单元\n- 明确每个单元的学习目标和能力要求\n- 规划作业节点和学习里程碑\n- 建立单元间的知识衔接关系\n- 生成结构化的课程大纲\n\n输出课程模块结构和单元规划文档。\n\n## 步骤 5：备课资料自动整理（输出层）\n使用 **teacher-prep** 完成：\n- 自动搜索课文相关资料（原文、作者、背景）\n- 整理生字词、重点段落和教学要点\n- 梳理课文知识结构和教学脉络\n- 补充拓展资源和参考材料\n- 输出结构化的备课资料包\n\n输出备课资料包和教学参考材料。\n\n## 步骤 6：教案PPT转化（输出层）\n使用 **教案内容写入PPT备注** 完成：\n- 从教案中提取各环节的讲师活动\n- 自动写入对应PPT幻灯片的备注区域\n- 确保教案内容与PPT页面一一对应\n- 支持一键批量更新备注内容\n- 输出带完整讲师备注的教学PPT\n\n输出带讲师备注的教学PPT文件。\n\n## 最终输出\n将以上步骤的结果整合为完整的教案设计成果包，交付以下文件：\n1. **完整教案**：教学评一致性、新课标、个性化教案\n2. **评估工具**：Rubric评分标准、活动方案、评估设计\n3. **分层资源包**：互动题、分层教学、作业设计\n4. **课程大纲**：模块结构、单元目标、里程碑规划\n5. **备课资料**：课文资料、生字词、教学参考\n6. **教学PPT**：讲师备注、幻灯片对应、一键更新",
    "skillSlugs": [
      "teaching-plan-writer",
      "teacher-toolkit",
      "classroom-lesson-pack",
      "creator-course-outline",
      "teacher-prep",
      "ppt-lecture-notes"
    ],
    "skillCount": 6
  },
  {
    "id": 12,
    "slug": "education-training-program",
    "displayName": "培训方案",
    "summary": "从培训计划整体设计（培训方案/课程体系/效果评估/日程安排/证书模板）与结构化员工培训计划（模块设计/评估机制/日程跟踪/入职合规技能提升领导力），企业培训专业课程设计（一键生成全套培训资料/课件/营销文案）与培训课程大纲教学设计（课程大纲/效果评估/内部分享材料），到工作坊大纲与培训议程生成（课程结构/日程规划/工作坊设计）与培训会议记录转化学习材料（结构化学习指南/速查表/检查清单/复习内容）的完整培训方案工作流。覆盖方案设计、员工培训、课程开发、教学设计、议程规划、材料转化全链路。",
    "scene": "education",
    "subScene": "training-program",
    "category": "education",
    "content": "---\nscene: \"education\"\nsub_scene: \"training-program\"\nskills:\n  - \"training-plan\"\n  - \"afrexai-training-program\"\n  - \"training-course-designer\"\n  - \"instructional-design-cn\"\n  - \"jackyshen-design-workshop-outline\"\n  - \"transcript-to-content\"\n---\n\n# 培训方案工作流\n\n你现在要完成一项培训方案设计任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：培训计划整体设计（获取层）\n使用 **Training Plan** 完成：\n- 明确培训目标和受众分析\n- 设计完整的课程体系和模块结构\n- 制定培训日程安排和时间规划\n- 建立培训效果评估机制\n- 准备证书模板和结业标准\n\n输出培训计划总方案和日程表。\n\n## 步骤 2：结构化员工培训计划（获取层）\n使用 **Employee Training Program** 完成：\n- 设计入职培训、合规培训和技能提升模块\n- 为每个模块制定学习目标和评估标准\n- 建立培训进度跟踪和完成度监测\n- 规划领导力发展和职业成长路径\n- 输出结构化的培训项目文档\n\n输出员工培训项目方案和跟踪表。\n\n## 步骤 3：专业培训课件开发（分析层）\n使用 **training-course-designer** 完成：\n- 针对企业培训需求设计专业课程\n- 一键生成包含课件在内的全套培训资料\n- 制作配套的营销文案和宣传材料\n- 确保课程内容与培训目标对齐\n- 输出可直接使用的培训课件包\n\n输出培训课件和配套资料。\n\n## 步骤 4：培训课程大纲与教学设计（分析层）\n使用 **教学设计** 完成：\n- 设计培训课程的详细大纲\n- 规划教学环节和互动方式\n- 制定效果评估方案和考核标准\n- 生成内部分享和交流材料\n- 确保教学策略与学习目标匹配\n\n输出课程大纲和教学设计文档。\n\n## 步骤 5：工作坊议程与日程规划（输出层）\n使用 **jackyshen-design-workshop-outline** 完成：\n- 生成结构化的工作坊大纲\n- 设计培训议程和时间分配\n- 规划每个环节的活动形式和产出物\n- 制定工作坊日程和分段脚本\n- 输出可直接执行的培训议程表\n\n输出工作坊大纲和培训议程。\n\n## 步骤 6：培训记录转化学习材料（输出层）\n使用 **Transcript to Content** 完成：\n- 将培训会议记录转化为结构化学习材料\n- 提取关键知识点生成学习指南\n- 制作速查表和检查清单\n- 整理可操作的复习内容\n- 输出完整的培训知识沉淀文档\n\n输出学习指南、速查表和知识沉淀文档。\n\n## 最终输出\n将以上步骤的结果整合为完整的培训方案成果包，交付以下文件：\n1. **培训计划**：目标分析、课程体系、日程安排、证书模板\n2. **培训项目**：模块设计、评估标准、进度跟踪、发展路径\n3. **培训课件**：专业课程、全套资料、营销文案\n4. **教学设计**：课程大纲、教学环节、效果评估\n5. **培训议程**：工作坊大纲、时间分配、分段脚本\n6. **知识沉淀**：学习指南、速查表、检查清单、复习材料",
    "skillSlugs": [
      "training-plan",
      "afrexai-training-program",
      "training-course-designer",
      "instructional-design-cn",
      "jackyshen-design-workshop-outline",
      "transcript-to-content"
    ],
    "skillCount": 6
  },
  {
    "id": 13,
    "slug": "finance-business-analysis",
    "displayName": "经营分析",
    "summary": "从A股上市公司业绩快报数据获取到自动化财务分析、杜邦体系ROE拆解、预算执行差异分析，再到KPI决策仪表盘和智能图表生成的完整经营分析工作流。支持同比环比趋势分析、多维度KPI计算、预算vs实际差异识别和管理层决策简报输出。",
    "scene": "finance",
    "subScene": "business-analysis",
    "category": "finance",
    "content": "---\nscene: \"finance\"\nsub_scene: \"business-analysis\"\nskills:\n  - \"test-stock-performance-express\"\n  - \"auto-data-analysis-claw\"\n  - \"financial-roe-analysis\"\n  - \"budget-vs-actual\"\n  - \"business-intelligence\"\n  - \"smart-charts\"\n---\n\n# 经营分析工作流\n\n你现在要完成一项企业经营状况的深度分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：获取经营数据（获取层）\n使用 **A股业绩快报查询** 完成：\n- 查询目标上市公司的业绩快报数据（支持沪深京）\n- 获取核心经营指标：营收、净利润、EPS、ROE\n- 拉取同比增长率，对比一季报、半年报、三季报、年报\n- 收集历史财务指标用于趋势分析\n\n将经营数据整理为结构化格式待用。\n\n## 步骤 2：自动化数据分析（分析层）\n使用 **Auto Data Analysis Claw** 完成：\n- 对财务报表（利润表、资产负债表、现金流量表）进行自动化分析\n- 执行 KPI 计算：毛利率、净利率、资产周转率、应收账款周转天数等\n- 进行同比环比分析，识别营收和利润的增长趋势\n- 执行多维度差异分析，定位业绩波动的核心驱动因素\n- 数据清洗和异常值标记\n\n输出数据分析底稿和 KPI 计算结果。\n\n## 步骤 3：杜邦体系深度拆解（分析层）\n使用 **financial roe analysis** 完成：\n- 基于杜邦分析体系对公司进行深度财务分析\n- 拆解 ROE 驱动因素：净利率 × 资产周转率 × 权益乘数\n- 评估盈利质量（扣非净利润占比、经营性现金流/净利润）\n- 分析资产运营效率和杠杆风险水平\n- 与行业均值和竞争对手对标\n\n输出杜邦分析报告和财务健康度评估。\n\n## 步骤 4：预算执行差异分析（分析层）\n使用 **Budget Vs Actual** 完成：\n- 对比月度/季度的预算与实际执行数据\n- 计算收入、费用和利润率的差异\n- 识别有利差异和不利差异\n- 分析差异产生的原因（量差、价差、结构差）\n- 标记需要管理层关注的重大偏差项\n\n输出预算执行差异分析报告。\n\n## 步骤 5：KPI 模型与决策仪表盘（输出层）\n使用 **Business Intelligence** 完成：\n- 建立业务绩效模型，定义核心 KPI 体系\n- 将分析结果转化为面向管理层的决策仪表盘\n- 生成运营简报和经营分析周报/月报\n- 设定运营节奏（周例会/月度经营分析会的数据支撑）\n\n## 步骤 6：智能图表生成（输出层）\n使用 **smart-charts** 完成：\n- 读取分析数据（CSV/Excel/JSON），自动推荐最佳图表类型\n- 生成交互式 ECharts 图表（趋势图、对比图、占比图）\n- 制作营收趋势折线图、成本结构饼图、KPI 达成率仪表盘\n- 输出可嵌入报告的高质量可视化图表\n\n## 最终输出\n将以上步骤的结果整合为完整的经营分析包，交付以下文件：\n1. **经营分析报告**：核心 KPI、同比环比趋势、驱动因素分析\n2. **杜邦分析报告**：ROE 拆解、盈利质量、运营效率评估\n3. **预算差异报告**：预算vs实际对比、有利/不利差异清单\n4. **管理层决策简报**：一页纸经营分析摘要 + 行动建议\n5. **可视化图表集**：交互式经营数据图表",
    "skillSlugs": [
      "test-stock-performance-express",
      "auto-data-analysis-claw",
      "financial-roe-analysis",
      "budget-vs-actual",
      "business-intelligence",
      "smart-charts"
    ],
    "skillCount": 6
  },
  {
    "id": 14,
    "slug": "finance-quant-backtesting",
    "displayName": "量化回测",
    "summary": "从A股/期货历史数据获取到量化策略编写、多因子选股模型、回测引擎执行，再到生成包含胜率、收益率、夏普比率、最大回撤等核心指标的专业回测报告的完整量化交易工作流。支持经典策略（双均线、网格交易、突破策略）、事件驱动策略和因子分析。",
    "scene": "finance",
    "subScene": "quant-backtesting",
    "category": "finance",
    "content": "---\nscene: \"finance\"\nsub_scene: \"quant-backtesting\"\nskills:\n  - \"joinquant\"\n  - \"stock-strategy-backtester\"\n  - \"bitsoul-china-stock-quantization\"\n  - \"quant-strategy\"\n  - \"quant\"\n  - \"backtesting-trading-strategies\"\n---\n\n# 量化回测工作流\n\n你现在要完成一项量化交易策略的开发与回测任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：获取历史行情与因子数据（获取层）\n使用 **joinquant** 完成：\n- 获取 A 股、期货、基金的历史行情数据（日线/分钟线）\n- 查询目标标的的财务数据和估值指标\n- 获取因子数据（市值因子、动量因子、价值因子等）\n- 准备事件驱动策略所需的公告、分红除权等事件数据\n\n将历史数据和因子数据整理为策略开发所需的标准格式。\n\n## 步骤 2：策略回测执行（分析层）\n使用 **Stock Strategy Backtester** 完成：\n- 在历史 OHLCV 数据上回测交易策略\n- 计算核心绩效指标：胜率、收益率、CAGR、夏普比率\n- 计算风险指标：最大回撤、波动率\n- 输出完整的交易日志（每笔交易的进出时间、价格、盈亏）\n\n记录回测结果和交易明细。\n\n## 步骤 3：多因子量化选股（分析层）\n使用 **BitSoulStockSkill** 完成：\n- 使用内置的上百种量化指标进行多因子选股\n- 基于 MOE 混合因子专家模型计算股票买卖点\n- 执行股票筛选策略（市值、成长性、估值、质量等维度）\n- 输出因子得分和选股结果\n\n输出多因子选股清单和买卖信号。\n\n## 步骤 4：策略编写与因子分析（分析层）\n使用 **Quant Strategy** 完成：\n- 辅助编写量化交易策略代码\n- 进行因子有效性分析（IC/IR、组织回测）\n- 优化策略参数（网格搜索、滚动优化）\n- 评估策略在不同市场环境下的稳健性\n\n输出策略代码和因子分析报告。\n\n## 步骤 5：多引擎回测与风控（分析层）\n使用 **Quant** 完成：\n- 多数据源交叉验证回测结果\n- 多引擎回测对比（避免过拟合）\n- 实时风控指标监控（仓位限制、回撤预警）\n- 交易信号推送和组合管理\n\n输出风控参数和多引擎回测对比结果。\n\n## 步骤 6：生成回测报告（输出层）\n使用 **Backtesting Trading Strategies** 完成：\n- 汇总所有回测数据，生成完整绩效报告\n- 计算夏普比率、索提诺比率、最大回撤、卡玛比率等指标\n- 绘制权益曲线、回撤曲线和月度收益热力图\n- 对比基准（沪深300/标普500）的超额收益分析\n\n## 最终输出\n将以上步骤的结果整合为完整的量化回测包，交付以下文件：\n1. **策略回测报告**：胜率、收益率、夏普比率、最大回撤等核心指标\n2. **多因子选股清单**：因子得分、买卖信号、标的推荐\n3. **策略代码**：完整的可执行量化策略代码\n4. **权益曲线图**：含基准对比的策略净值走势图",
    "skillSlugs": [
      "joinquant",
      "stock-strategy-backtester",
      "bitsoul-china-stock-quantization",
      "quant-strategy",
      "quant",
      "backtesting-trading-strategies"
    ],
    "skillCount": 6
  },
  {
    "id": 16,
    "slug": "hr-ppt-creation",
    "displayName": "PPT 制作",
    "summary": "从PPT大纲与演示结构规划生成、PowerPoint文件全功能创建编辑（布局/模板/图表/备注）、麦肯锡风格专业顾问级演示文稿设计，到参考模板视觉风格提取与幻灯片重建、乔布斯风极简科技感HTML演示稿一键生成、python-pptx多布局多图文混排科技风PPT制作的完整PPT制作工作流。覆盖大纲规划、风格设计、文件生成、多格式输出全链路。",
    "scene": "hr",
    "subScene": "ppt-creation",
    "category": "hr",
    "content": "---\nscene: \"hr\"\nsub_scene: \"ppt-creation\"\nskills:\n  - \"ppt-outline\"\n  - \"powerpoint-pptx\"\n  - \"mck-ppt-design\"\n  - \"ppt-from-template\"\n  - \"ppt\"\n  - \"dragon-ppt-maker\"\n---\n\n# PPT 制作工作流\n\n你现在要完成一项 PPT 演示文稿的制作任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：PPT 大纲与结构规划（获取层）\n使用 **PPT Outline** 完成：\n- 根据演示主题生成 PPT 大纲和页面结构\n- 规划每页的核心信息和叙事逻辑\n- 设计演示文稿的整体故事线\n- 确定页数、章节划分和过渡方式\n- 支持生成 HTML 演示文稿预览\n\n输出 PPT 大纲和页面结构规划。\n\n## 步骤 2：PowerPoint 文件全功能编辑（获取层）\n使用 **Powerpoint / PPTX** 完成：\n- 创建、检查和编辑 PPTX 文件\n- 配置幻灯片布局、模板和占位符\n- 添加演讲者备注和批注\n- 插入图表、表格和数据可视化\n- 执行视觉质检确保格式一致\n\n输出基础 PPTX 文件和布局配置。\n\n## 步骤 3：麦肯锡风格专业设计（分析层）\n使用 **Mck Ppt Design Skill** 完成：\n- 按麦肯锡咨询风格从零创建专业 PPT\n- 应用顾问级排版规范和配色方案\n- 设计数据驱动的图表和分析页面\n- 确保每页有明确的 Takeaway 信息\n- 输出高标准的商业演示文稿\n\n输出麦肯锡风格专业 PPT。\n\n## 步骤 4：模板风格提取与重建（分析层）\n使用 **PPT from Template** 完成：\n- 从参考模板中提取视觉风格（配色/字体/布局）\n- 使用 PptxGenJS 从头重新创建幻灯片\n- 保持品牌视觉一致性\n- 适配不同场景的模板需求（汇报/路演/培训）\n- 输出风格统一的演示文稿\n\n输出基于模板风格重建的 PPT。\n\n## 步骤 5：极简科技感演示稿生成（输出层）\n使用 **ppt** 完成：\n- 将讲稿一键生成乔布斯风极简科技感演示稿\n- 输出单个可直接运行的 HTML 文件\n- 支持竖屏和横屏两种模式\n- 自动排版和视觉设计\n- 适合科技产品发布、技术分享等场景\n\n输出极简科技感 HTML 演示文件。\n\n## 步骤 6：科技风多布局 PPT 制作（输出层）\n使用 **PPT制作** 完成：\n- 使用 python-pptx 制作科技风 PPT\n- 支持多种幻灯片布局和图文混排\n- 嵌入 HTML 内容和数据可视化\n- 生成可编辑的 .pptx 文件\n- 适合工作汇报、项目展示等正式场景\n\n## 最终输出\n将以上步骤的结果整合为完整的 PPT 制作包，交付以下文件：\n1. **PPT 大纲**：页面结构、叙事逻辑、章节划分\n2. **基础 PPTX 文件**：布局配置、图表表格、备注批注\n3. **麦肯锡风格 PPT**：顾问级设计、数据图表、Takeaway\n4. **模板风格 PPT**：品牌一致性、视觉提取、风格重建\n5. **极简科技感演示**：乔布斯风格、HTML 文件、一键运行\n6. **科技风 PPTX 文件**：多布局、图文混排、可编辑格式",
    "skillSlugs": [
      "ppt-outline",
      "powerpoint-pptx",
      "mck-ppt-design",
      "ppt-from-template",
      "ppt",
      "dragon-ppt-maker"
    ],
    "skillCount": 6
  },
  {
    "id": 17,
    "slug": "hr-resume-screening",
    "displayName": "简历筛选",
    "summary": "从简历漏斗式量化评估（基础匹配度/能力匹配度/动机稳定匹配度）与结构化信息提取解析、四阶段一站式招聘流程（筛选→面试设计→面试评估→推荐）与JD自动匹配排序，到批量简历解析评分排名与循证面试策略设计的完整简历筛选工作流。覆盖简历解析、量化评估、人岗匹配、面试设计全链路。",
    "scene": "hr",
    "subScene": "resume-screening",
    "category": "hr",
    "content": "---\nscene: \"hr\"\nsub_scene: \"resume-screening\"\nskills:\n  - \"resume-screening\"\n  - \"resume-parser\"\n  - \"resume-screener-pro\"\n  - \"easy-recruitment\"\n  - \"ai-resume-screener\"\n  - \"interview-designer\"\n---\n\n# 简历筛选工作流\n\n你现在要完成一项简历筛选与候选人评估任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：简历量化筛选（获取层）\n使用 **专业简历筛选技能** 完成：\n- 按\"先硬后软，先筛后读\"的漏斗式流程筛选简历池\n- 从基础匹配度、能力匹配度、动机稳定匹配度三个维度量化评估\n- 对候选人进行初步分级（强匹配/一般匹配/不匹配）\n- 标记简历中的关键亮点和风险信号\n- 生成初筛通过名单和淘汰原因说明\n\n输出初筛评估报告和候选人分级名单。\n\n## 步骤 2：简历结构化解析（获取层）\n使用 **resume-parser** 完成：\n- 解析候选人简历文件（PDF/Word/图片格式）\n- 提取核心结构化信息（教育背景/工作经历/技能/项目经验）\n- 计算简历与目标岗位 JD 的匹配度得分\n- 识别简历中的信息缺失和不一致之处\n- 导出标准化候选人数据供后续环节使用\n\n输出结构化简历数据和 JD 匹配度分析。\n\n## 步骤 3：四阶段招聘评估（分析层）\n使用 **Resume Screener Pro** 完成：\n- 执行简历深度筛选（Topgrading 方法论）\n- 设计结构化面试方案和评分标准\n- 进行绩效预测和 ATS 优化评估\n- 控制招聘偏见（bias control）\n- 生成候选人综合评估和最终推荐排序\n\n输出四阶段评估报告和候选人推荐排序。\n\n## 步骤 4：JD 智能匹配与排序（分析层）\n使用 **recruitment-assistant** 完成：\n- 根据职位 JD 自动筛选和评估全部简历\n- 输出候选人匹配度排序报告\n- 为每位候选人生成定制化面试问题清单\n- 标注候选人的核心优势和潜在风险\n- 给出各候选人的录用优先级建议\n\n输出候选人排序报告和定制面试问题。\n\n## 步骤 5：批量简历评分排名（分析层）\n使用 **AI Resume Screener** 完成：\n- 批量解析大量简历（支持 1000+ 份）\n- 按岗位要求进行无偏见自动评分\n- 生成候选人排名和评分明细\n- 提供多维度数据分析（技能分布/经验分布/学历分布）\n- 输出可视化筛选报告\n\n输出批量评分排名和数据分析报告。\n\n## 步骤 6：循证面试策略设计（输出层）\n使用 **Interview Designer** 完成：\n- 基于简历分析设计针对性面试策略\n- 制定结构化面试指南和评分标准\n- 设计胜任力验证问题（行为面试/情境面试）\n- 融合 Topgrading 方法论和绩效导向招聘理论\n- 准备深度追问清单和评估维度\n\n## 最终输出\n将以上步骤的结果整合为完整的简历筛选包，交付以下文件：\n1. **初筛评估报告**：候选人分级、匹配度评分、淘汰原因\n2. **结构化简历数据**：核心信息提取、JD 匹配度分析\n3. **四阶段评估报告**：Topgrading 评估、绩效预测、推荐排序\n4. **候选人排序报告**：JD 匹配排序、定制面试问题、录用优先级\n5. **批量评分报告**：评分明细、多维数据分析、可视化报告\n6. **面试策略方案**：结构化面试指南、胜任力验证问题、追问清单",
    "skillSlugs": [
      "resume-screening",
      "resume-parser",
      "resume-screener-pro",
      "easy-recruitment",
      "ai-resume-screener",
      "interview-designer"
    ],
    "skillCount": 6
  },
  {
    "id": 18,
    "slug": "hr-weekly-daily-report",
    "displayName": "周报日报",
    "summary": "从工作周报日报月报全类型模板化生成与职场人日报周报结构化输出（含问题复盘）、飞书聊天记录与daily memory自动汇总生成周报与根据职业角色智能提炼关键信息，到工作汇报PPT自动生成与多格式多平台报告输出的完整周报日报工作流。覆盖内容收集、结构化提炼、格式输出、多平台发布全链路。",
    "scene": "hr",
    "subScene": "weekly-daily-report",
    "category": "hr",
    "content": "---\nscene: \"hr\"\nsub_scene: \"weekly-daily-report\"\nskills:\n  - \"work-report-writer\"\n  - \"chinese-daily-report-generator\"\n  - \"feishu-weekly-report\"\n  - \"smart-weekly-report\"\n  - \"work-report\"\n  - \"smart-reporter\"\n---\n\n# 周报日报工作流\n\n你现在要完成一项周报日报的生成任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：工作报告全类型生成（获取层）\n使用 **Work Report Writer** 完成：\n- 根据报告类型选择模板（日报/周报/月报）\n- 收集本周期工作内容、成果和数据\n- 按标准结构填充报告各板块\n- 自动生成工作总结和下期计划\n- 支持多种报告风格切换\n\n输出标准格式的工作报告初稿。\n\n## 步骤 2：日报周报结构化生成（获取层）\n使用 **Chinese Daily Report Generator** 完成：\n- 根据工作记录自动生成结构化日报/周报\n- 按职场规范格式化输出\n- 包含问题复盘板块（遇到的问题/解决方案/经验教训）\n- 自动提炼工作亮点和关键成果\n- 输出可直接发送的日报/周报文本\n\n输出结构化日报/周报和问题复盘。\n\n## 步骤 3：飞书数据自动汇总（分析层）\n使用 **飞书周报** 完成：\n- 调用飞书 API 拉取指定时间范围的聊天记录\n- 读取本地 daily memory 日志\n- 合并两个数据源的工作内容\n- 按用户指定的周报模板自动整理\n- 输出飞书格式的周报\n\n输出飞书周报和数据汇总。\n\n## 步骤 4：智能提炼关键信息（分析层）\n使用 **Report Generator** 完成：\n- 根据职业角色定制报告风格和重点\n- 从本周工作内容中提炼关键信息\n- 生成具有洞察力的分析和建议\n- 包含问题分析与下周计划\n- 输出专业且结构化的周报\n\n输出智能提炼的专业周报。\n\n## 步骤 5：工作汇报 PPT 生成（输出层）\n使用 **工作报告PPT生成技能** 完成：\n- 使用 python-pptx 创建专业演示文稿\n- 按规范格式排版（标题宋体48号/内容36号/正文18号）\n- 控制总页数在 10 页以内\n- 支持工作周报、月报、项目汇报等类型\n- 输出可直接使用的 PPT 文件\n\n输出工作汇报 PPT 文件。\n\n## 步骤 6：多格式多平台输出（输出层）\n使用 **智能报告生成器** 完成：\n- 自动分析数据并生成专业报告\n- 支持日报、周报、月报、分析报告等多种类型\n- 输出到飞书文档或本地文件\n- 适配不同平台的格式要求\n- 确保报告质量和格式一致性\n\n## 最终输出\n将以上步骤的结果整合为完整的周报日报包，交付以下文件：\n1. **工作报告初稿**：标准模板、全类型覆盖、多风格切换\n2. **结构化日报周报**：职场规范格式、问题复盘、亮点提炼\n3. **飞书数据汇总**：聊天记录整合、daily memory 合并、模板输出\n4. **智能提炼周报**：角色定制、关键信息、洞察分析、下周计划\n5. **工作汇报 PPT**：专业排版、规范格式、演示文稿\n6. **多平台报告**：飞书文档、本地文件、多格式适配",
    "skillSlugs": [
      "work-report-writer",
      "chinese-daily-report-generator",
      "feishu-weekly-report",
      "smart-weekly-report",
      "work-report",
      "smart-reporter"
    ],
    "skillCount": 6
  },
  {
    "id": 19,
    "slug": "legal-contract-drafting",
    "displayName": "合同起草",
    "summary": "从多类型合同模板生成（劳动合同/服务协议/合作协议/租赁合同/协议书模板）与专业合同生成审查（销售合同/租赁合同/劳动合同/NDA/服务协议/法律咨询），法律文书模板与撰写指导（劳务纠纷/离婚协议/交通事故/租房协议/文书模板）与中国财税法律文书起草（税务筹划报告/法律意见书/合同税务条款/行政复议/行政诉讼），到保密协议NDA起草与DOCX生成（双向NDA/单向NDA/Common Paper/Bonterms标准）与法律文书模板专家（500+案例/自我进化/多类型文书模板）的完整合同起草工作流。覆盖合同模板、专业生成、文书指导、财税文书、NDA起草、模板库全链路。",
    "scene": "legal",
    "subScene": "contract-drafting",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"contract-drafting\"\nskills:\n  - \"contract-template\"\n  - \"zhang-contract-generator\"\n  - \"legal-document-assistant\"\n  - \"legal-doc-writer\"\n  - \"nda\"\n  - \"zhang-legal-templates\"\n---\n\n# 合同起草工作流\n\n你现在要完成一项合同起草任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：合同模板选择与生成（获取层）\n使用 **Contract Template** 完成：\n- 根据合同类型选择适配的标准模板\n- 覆盖劳动合同、服务协议、合作协议等\n- 生成包含必要条款的合同框架\n- 支持自定义条款和特殊约定\n- 输出结构化的合同模板初稿\n\n输出合同模板和框架文档。\n\n## 步骤 2：专业合同内容生成（获取层）\n使用 **九章合同生成器 V1.4.0** 完成：\n- 专业生成销售、租赁、劳动等合同\n- 自动填充NDA和服务协议关键条款\n- 提供合同内容的法律审查\n- 支持案例分析辅助条款设计\n- 确保合同条款的法律有效性\n\n输出专业合同正文和审查意见。\n\n## 步骤 3：法律文书撰写指导（分析层）\n使用 **legal document assistant** 完成：\n- 提供常见法律文书模板\n- 给出文书撰写的专业指导\n- 覆盖劳务纠纷、离婚协议等场景\n- 指导租房协议和交通事故文书\n- 确保文书格式和内容规范\n\n输出法律文书撰写指南和参考模板。\n\n## 步骤 4：财税法律文书起草（分析层）\n使用 **Legal Doc Writer** 完成：\n- 起草税务筹划报告和法律意见书\n- 撰写合同中的税务条款\n- 制作税务行政复议申请书\n- 起草税务行政诉讼相关文书\n- 审查和修改现有法律文书\n\n输出财税法律文书和意见书。\n\n## 步骤 5：保密协议专项起草（输出层）\n使用 **NDA** 完成：\n- 起草双向或单向保密协议\n- 填写NDA模板的关键信息\n- 符合Common Paper和Bonterms标准\n- 生成可直接签署的DOCX文件\n- 支持员工保密协议等场景\n\n输出可签署的NDA文件。\n\n## 步骤 6：法律文书模板库输出（输出层）\n使用 **九章法律文书模板 V1.4.0** 完成：\n- 基于500+案例库选择最佳文书模板\n- 生成各类法律文书的标准格式\n- 支持文书模板的自我进化和优化\n- 覆盖诉讼、仲裁、调解等多种场景\n- 输出格式规范的法律文书终稿\n\n输出法律文书终稿和模板库。\n\n## 最终输出\n将以上步骤的结果整合为完整的合同起草成果包，交付以下文件：\n1. **合同模板**：标准模板、框架文档、条款结构\n2. **专业合同**：正文生成、法律审查、条款设计\n3. **撰写指南**：文书模板、撰写指导、格式规范\n4. **财税文书**：税务筹划、法律意见书、税务条款\n5. **NDA文件**：双向/单向NDA、DOCX可签署文件\n6. **文书终稿**：500+案例、标准格式、终稿输出",
    "skillSlugs": [
      "contract-template",
      "zhang-contract-generator",
      "legal-document-assistant",
      "legal-doc-writer",
      "nda",
      "zhang-legal-templates"
    ],
    "skillCount": 6
  },
  {
    "id": 20,
    "slug": "marketing-ad-copywriting",
    "displayName": "广告文案",
    "summary": "从23模块营销策略体系（CRO/SEO/文案/分析/广告/发布）、AIDA/PAS/FAB等经典文案框架与说服力技巧、中文营销文案100个标题公式与痛点挖掘，到全平台广告文案生成（信息流/朋友圈/搜索/直通车/巨量引擎/Google Ads）含A/B测试、高转化落地页自动生成、电商爆款文案（淘宝/拼多多/抖音/京东）的完整广告文案工作流。",
    "scene": "marketing",
    "subScene": "ad-copywriting",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"ad-copywriting\"\nskills:\n  - \"marketing-skills\"\n  - \"copywriting\"\n  - \"copywriter-cn\"\n  - \"ad-copywriter\"\n  - \"landing-page-generator\"\n  - \"ecommerce-copywriter\"\n---\n\n# 广告文案工作流\n\n你现在要完成一项广告文案创作任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：营销策略与受众分析（获取层）\n使用 **Marketing Skills** 完成：\n- 明确广告投放目标（品牌曝光/线索获取/直接转化）\n- 分析目标受众画像（人口统计/兴趣/痛点/购买动机）\n- 选择合适的营销框架和渠道策略\n- 制定文案风格指南（语调/用词/情感方向）\n- 确定关键卖点和差异化定位\n\n输出营销策略简报和受众画像。\n\n## 步骤 2：文案框架与说服力构建（分析层）\n使用 **Copywriting** 完成：\n- 运用 AIDA（注意-兴趣-欲望-行动）框架构建文案结构\n- 运用 PAS（痛点-激化-解决）和 FAB（特性-优势-利益）公式\n- 设计高点击率标题（提问式/数字式/悬念式/利益式）\n- 编写说服力 CTA（行动号召）变体\n- 植入情感触发点和紧迫感元素\n\n输出文案框架和核心卖点提炼。\n\n## 步骤 3：中文营销文案撰写（分析层）\n使用 **Copywriter Cn** 完成：\n- 从 100 个中文标题公式中匹配最佳模板\n- 深度挖掘用户痛点并转化为文案卖点\n- 撰写符合中文语境的正文（开头钩子/中间论证/结尾推动）\n- 编写客户证言和社会认同文案\n- 适配中文平台的阅读习惯和表达方式\n\n输出中文广告文案初稿。\n\n## 步骤 4：全平台广告文案生成（输出层）\n使用 **Ad Copywriter** 完成：\n- 生成信息流广告文案（今日头条/抖音/微信朋友圈）\n- 生成搜索广告文案（百度/Google Ads）\n- 生成电商平台广告（直通车标题/巨量引擎创意）\n- 为每组文案提供 A/B 测试变体\n- 计算预估 ROI 并给出投放建议\n\n输出各平台适配的广告文案集。\n\n## 步骤 5：高转化落地页生成（输出层）\n使用 **Landing Page Generator** 完成：\n- 根据广告文案生成配套落地页内容\n- 设计首屏（主标题+副标题+CTA+主视觉）\n- 构建社会认同区（客户评价/数据背书/合作品牌）\n- 编排功能特性和利益点展示\n- 优化移动端适配和加载体验\n\n输出高转化率落地页。\n\n## 步骤 6：电商爆款文案（输出层）\n使用 **电商爆款文案生成器** 完成：\n- 生成高转化商品标题（关键词+卖点+规格）\n- 撰写详情页文案（痛点场景→解决方案→产品优势→用户证言→促销信息）\n- 提炼核心卖点（3 秒抓住注意力）\n- 生成促销活动文案（限时折扣/满减/赠品）\n- 适配淘宝/拼多多/抖音/京东各平台规则\n\n## 最终输出\n将以上步骤的结果整合为完整的广告文案包，交付以下文件：\n1. **营销策略简报**：受众画像、卖点定位、渠道策略\n2. **核心文案库**：标题变体、CTA变体、正文模板\n3. **全平台广告文案**：信息流/搜索/电商各平台适配版本（含A/B变体）\n4. **高转化落地页**：配套的产品/服务落地页\n5. **电商文案集**：商品标题/详情页/促销文案",
    "skillSlugs": [
      "marketing-skills",
      "copywriting",
      "copywriter-cn",
      "ad-copywriter",
      "landing-page-generator",
      "ecommerce-copywriter"
    ],
    "skillCount": 6
  },
  {
    "id": 21,
    "slug": "marketing-event-planning",
    "displayName": "活动策划",
    "summary": "从商业活动全品类策划（会议/发布会/展览/企业聚会）与可落地SOP生成、定价供应商后勤预算时间表的运营规划，到活动全流程管理与ROI评估分析，再到H5/网页营销活动方案与页面结构生成、营销创意战略与活动概念开发的完整活动策划工作流。覆盖活动策划、执行管控、创意输出全链路。",
    "scene": "marketing",
    "subScene": "event-planning",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"event-planning\"\nskills:\n  - \"afrexai-event-planner\"\n  - \"campaign-planning\"\n  - \"afrexai-event-planning\"\n  - \"afrexai-event-management\"\n  - \"activity-campaign-from-ui\"\n  - \"marketing-designer\"\n---\n\n# 活动策划工作流\n\n你现在要完成一项活动策划的全流程任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：商业活动整体策划（获取层）\n使用 **Event Planner Pro** 完成：\n- 明确活动类型（会议/网络研讨会/产品发布会/展览/企业聚会/社交活动）\n- 确定活动目标、目标受众和预期规模\n- 制定活动总体方案框架\n- 评估场地需求和基础预算范围\n- 输出活动概念提案和可行性评估\n\n输出活动整体策划方案。\n\n## 步骤 2：落地执行 SOP 生成（获取层）\n使用 **活动策划** 完成：\n- 生成活动策划可落地指南与标准操作流程（SOP）\n- 细化各阶段执行清单（筹备期/执行期/收尾期）\n- 明确各环节责任分工和时间节点\n- 制定风险预案和应急处理方案\n- 输出可直接执行的活动策划 SOP 文档\n\n输出活动执行 SOP 和阶段清单。\n\n## 步骤 3：运营规划与供应商管理（分析层）\n使用 **Event Planning Business Operations** 完成：\n- 制定活动定价策略和收支预算明细\n- 筛选和管理供应商（场地/餐饮/设备/搭建）\n- 规划后勤保障方案（交通/住宿/物料）\n- 制定人员配置和排班计划\n- 输出详细时间表和里程碑节点\n\n输出运营预算和供应商管理方案。\n\n## 步骤 4：活动全流程管理与效果评估（分析层）\n使用 **Event Management** 完成：\n- 执行活动当天流程管控和现场协调\n- 监控关键执行指标（签到率/参与度/满意度）\n- 收集活动数据（参会人数/互动数据/媒体曝光）\n- 进行活动后 ROI 评估和效果复盘\n- 输出活动效果分析报告和改进建议\n\n输出活动执行报告和 ROI 分析。\n\n## 步骤 5：营销活动页面与方案生成（输出层）\n使用 **Activity Campaign from UI** 完成：\n- 根据 UI 参考创建 H5/网页营销活动方案\n- 设计活动页面结构和交互逻辑\n- 在固定技术栈上生成高保真 HTML/CSS/JS 前端原型\n- 制作活动报名页、详情页、倒计时页等核心页面\n- 输出可直接部署的活动营销页面\n\n输出活动营销页面方案和前端原型。\n\n## 步骤 6：创意策略与概念开发（输出层）\n使用 **Marketing Designer** 完成：\n- 策划营销活动创意概念和主题\n- 融合数据洞察与创意直觉制定传播策略\n- 设计活动视觉风格和品牌调性\n- 生成多版本创意方案供选择\n- 制定跨渠道传播计划和内容排期\n\n## 最终输出\n将以上步骤的结果整合为完整的活动策划包，交付以下文件：\n1. **活动整体策划方案**：活动类型、目标、受众、概念提案\n2. **执行 SOP 文档**：阶段清单、责任分工、时间节点、风险预案\n3. **运营预算与供应商方案**：预算明细、供应商清单、后勤保障、排班计划\n4. **活动效果分析报告**：执行数据、ROI 评估、复盘总结、改进建议\n5. **营销活动页面**：H5/网页方案、页面结构、前端原型\n6. **创意传播策略**：创意概念、视觉风格、跨渠道传播计划",
    "skillSlugs": [
      "afrexai-event-planner",
      "campaign-planning",
      "afrexai-event-planning",
      "afrexai-event-management",
      "activity-campaign-from-ui",
      "marketing-designer"
    ],
    "skillCount": 6
  },
  {
    "id": 22,
    "slug": "marketing-social-media-operation",
    "displayName": "社媒运营",
    "summary": "从小红书/抖音/B站热门内容抓取与趋势分析、小红书/公众号/抖音/私域全平台运营策略与涨粉规划，到多平台发布级自媒体文案生成（小红书/知乎/公众号/抖音）、各平台内容优化与互动率提升、微信公众号爆文创作与SEO优化，再到全链路新媒体运营闭环（行业分析→竞品→养号→爆款→互动钩子）的完整社媒运营工作流。",
    "scene": "marketing",
    "subScene": "social-media-operation",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"social-media-operation\"\nskills:\n  - \"content-hunter\"\n  - \"social-media-operator\"\n  - \"content-writer\"\n  - \"social-media-optimizer\"\n  - \"wechat-mp-writer\"\n  - \"newmedia-operations\"\n---\n\n# 社媒运营工作流\n\n你现在要完成一项社交媒体运营任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：热门内容抓取与趋势分析（获取层）\n使用 **内容捕手** 完成：\n- 抓取小红书、抖音、B站等平台的热门内容\n- 分析当前热点话题和流量趋势\n- 提取爆款内容的标题结构、标签策略和互动数据\n- 分批抓取并生成热门内容汇报\n- 为后续内容创作提供选题参考\n\n输出热门内容报告和选题建议。\n\n## 步骤 2：运营策略与涨粉规划（分析层）\n使用 **自媒体运营全能助手** 完成：\n- 制定小红书/公众号/抖音/私域的运营策略\n- 规划内容日历和发布排期\n- 设计涨粉策略（内容钩子、互动话术、粉丝活动）\n- 分析账号数据，优化内容方向\n- 制定私域流量转化路径\n\n输出运营策略方案和内容日历。\n\n## 步骤 3：多平台自媒体文案生成（分析层）\n使用 **Content Writer 自媒体内容生成器** 完成：\n- 为小红书生成种草笔记（标题+正文+标签）\n- 为知乎生成专业回答和文章\n- 为微信公众号生成推文初稿\n- 为抖音生成短视频文案和脚本\n- 确保各平台内容风格适配（小红书活泼/知乎专业/公众号深度）\n\n输出各平台适配的内容文案。\n\n## 步骤 4：内容优化与互动率提升（输出层）\n使用 **Social Media Optimizer** 完成：\n- 针对微信、小红书、知乎、B站、抖音分别优化内容\n- 优化标题吸引力和首图/封面质量\n- 优化标签（hashtag）策略和关键词布局\n- 提升内容曝光率和互动率的具体建议\n- 适配各平台算法推荐机制\n\n输出优化建议和修改后的内容。\n\n## 步骤 5：微信公众号爆文创作（输出层）\n使用 **WeChat MP Writer** 完成：\n- 撰写 10 万+潜力的公众号文章\n- 设计爆款标题（悬念/数字/情绪/反转）\n- 优化文章结构（开头钩子→故事→干货→行动号召）\n- 微信 SEO 关键词布局\n- 生成订阅号/服务号适配版本\n\n输出微信公众号推文。\n\n## 步骤 6：全链路运营闭环（输出层）\n使用 **运营工具skill** 完成：\n- 执行行业分析和竞品对标\n- 制定账号养号策略和人设定位\n- 设计互动钩子（评论区话术/投票/抽奖）\n- 违禁词检测和内容合规审查\n- 覆盖抖音/视频号/小红书三大平台\n\n## 最终输出\n将以上步骤的结果整合为完整的社媒运营包，交付以下文件：\n1. **热门趋势报告**：各平台热门内容分析和选题建议\n2. **运营策略方案**：涨粉策略、内容日历、私域转化路径\n3. **多平台内容集**：小红书笔记/知乎文章/公众号推文/抖音脚本\n4. **内容优化报告**：各平台算法适配和互动率提升建议\n5. **运营工具配置**：违禁词检测、互动钩子、账号管理方案",
    "skillSlugs": [
      "content-hunter",
      "social-media-operator",
      "content-writer",
      "social-media-optimizer",
      "wechat-mp-writer",
      "newmedia-operations"
    ],
    "skillCount": 6
  },
  {
    "id": 23,
    "slug": "marketing-user-growth",
    "displayName": "用户增长",
    "summary": "从AARRR+增长框架与北极星指标定义、获客闭环与实验框架设计、转化漏斗创建与流失环节诊断，到新用户入职流程与激活指标优化、客户留存策略与赢回活动设计、流失风险AI评估与定制化留存推荐的完整用户增长工作流。覆盖获客、激活、留存、转化、推荐全链路增长体系。",
    "scene": "marketing",
    "subScene": "user-growth",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"user-growth\"\nskills:\n  - \"afrexai-growth-engine\"\n  - \"cgo\"\n  - \"funnel-analyzer\"\n  - \"customer-onboarding-2\"\n  - \"customer-retention\"\n  - \"afrexai-churn-analyzer\"\n---\n\n# 用户增长工作流\n\n你现在要完成一项用户增长体系的搭建与优化任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：增长体系设计与北极星指标（获取层）\n使用 **Growth Engineering Mastery** 完成：\n- 定义产品的北极星指标（NSM）和关键增长指标\n- 构建 AARRR+ 增长框架（获取/激活/留存/推荐/变现）\n- 设计病毒式增长循环（Viral Loop）机制\n- 制定增长实验测试计划和优先级排序\n- 明确各阶段的关键转化节点\n\n输出增长体系框架和北极星指标定义。\n\n## 步骤 2：获客闭环与实验框架（分析层）\n使用 **CGO / Chief Growth Officer** 完成：\n- 设计获客闭环策略（有机增长/付费增长/病毒式增长）\n- 建立增长实验框架（假设→实验→数据→迭代）\n- 制定留存系统和产品驱动增长（PLG）策略\n- 规划渠道组合和预算分配\n- 设定实验优先级（ICE/RICE 评分）\n\n输出获客策略和实验计划。\n\n## 步骤 3：转化漏斗分析（分析层）\n使用 **Funnel Analyzer** 完成：\n- 创建从曝光到付费的完整转化漏斗\n- 诊断每一步的流失率和流失原因\n- 与行业基准对标，找到最大优化空间\n- 生成漏斗对比报告（当前 vs 目标）\n- 给出各环节的具体优化建议\n\n输出漏斗分析报告和优化方案。\n\n## 步骤 4：新用户入职与激活优化（输出层）\n使用 **Customer Onboarding** 完成：\n- 设计新用户入职引导流程（应用内/邮件/人工）\n- 定义激活标准和「Aha Moment」指标\n- 识别入职流程中的摩擦点并消除\n- 优化价值实现时间（Time to Value）\n- 降低前 30 天流失率\n\n输出入职流程设计和激活指标。\n\n## 步骤 5：客户留存与赢回策略（输出层）\n使用 **Customer Retention** 完成：\n- 制定留存策略（功能引导/使用习惯养成/价值强化）\n- 设计忠诚度计划和奖励机制\n- 构建生命周期营销自动化（邮件/推送/消息）\n- 策划休眠用户赢回活动（优惠/内容/功能提醒）\n- 计算客户终身价值（LTV）和获客成本（CAC）对比\n\n输出留存策略方案和 LTV/CAC 模型。\n\n## 步骤 6：流失风险预警（输出层）\n使用 **Churn Risk Analyzer** 完成：\n- 分析客户数据评估流失风险等级\n- 将客户按风险程度分层（高危/中危/低危）\n- 为每层客户推荐定制化留存干预措施\n- 设置流失预警阈值和自动触发机制\n- 持续追踪干预效果\n\n## 最终输出\n将以上步骤的结果整合为完整的用户增长包，交付以下文件：\n1. **增长体系框架**：北极星指标、AARRR+ 框架、增长飞轮\n2. **获客策略与实验计划**：渠道策略、实验优先级、预算分配\n3. **漏斗分析报告**：各环节转化率、流失诊断、优化建议\n4. **入职与激活方案**：用户引导流程、Aha Moment、摩擦消除\n5. **留存与赢回策略**：忠诚度计划、生命周期营销、赢回活动\n6. **流失预警模型**：风险分层、干预措施、预警阈值",
    "skillSlugs": [
      "afrexai-growth-engine",
      "cgo",
      "funnel-analyzer",
      "customer-onboarding-2",
      "customer-retention",
      "afrexai-churn-analyzer"
    ],
    "skillCount": 6
  },
  {
    "id": 28,
    "slug": "tech-api-documentation",
    "displayName": "API 文档",
    "summary": "从REST/GraphQL API规范设计到代码自动解析生成OpenAPI文档、API开发全流程管理、接口文档撰写、请求构造与Mock测试，再到OpenAPI 3.0规范输出与SDK快速入门指南生成的完整API文档工作流。支持资源建模、版本策略、分页模式、错误处理规范和多语言SDK示例。",
    "scene": "tech",
    "subScene": "api-documentation",
    "category": "tech",
    "content": "---\nscene: \"tech\"\nsub_scene: \"api-documentation\"\nskills:\n  - \"api-designer\"\n  - \"sovereign-api-docs-generator\"\n  - \"api-dev\"\n  - \"api-doc-writer\"\n  - \"api-tester-cn\"\n  - \"afrexai-api-docs\"\n---\n\n# API 文档工作流\n\n你现在要完成一项 API 接口文档的编写与生成任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：API 规范设计（获取层）\n使用 **Api Designer** 完成：\n- 设计 REST 或 GraphQL API 的整体架构\n- 定义资源模型和 URL 路径命名规范\n- 制定版本策略（URL path / Header / Query param）\n- 设计分页模式（offset / cursor / keyset）\n- 规范错误处理格式（错误码、错误消息结构）\n- 编写 OpenAPI 规范草案\n\n输出 API 设计规范文档和 OpenAPI 草案。\n\n## 步骤 2：从代码自动生成文档（分析层）\n使用 **API Documentation Generator** (sovereign-api-docs-generator) 完成：\n- 扫描项目代码，自动识别 API 端点\n- 提取路由、参数、请求体和响应体结构\n- 支持 REST、GraphQL、WebSocket 接口\n- 生成包含示例和数据模式的完整 API 文档\n- 与步骤 1 的设计规范对齐补充\n\n输出自动生成的 API 参考文档。\n\n## 步骤 3：API 开发与调试验证（分析层）\n使用 **API Development** 完成：\n- 搭建 API 端点并验证接口实现是否符合文档规范\n- 使用 curl 命令测试每个端点的请求/响应\n- 编写集成测试验证接口行为\n- 生成 OpenAPI 规范并校验一致性\n- 模拟 API 响应用于前端对接\n\n输出接口测试结果和一致性校验报告。\n\n## 步骤 4：API 文档撰写（输出层）\n使用 **API Doc Writer** 完成：\n- 编写 REST API 接口文档的文字描述部分\n- 定义接口规范（请求方法、路径、参数说明）\n- 编写认证授权说明（OAuth、JWT、API Key）\n- 撰写使用指南和快速入门章节\n- 添加请求/响应示例和错误码说明\n\n输出结构化的 API 接口文档。\n\n## 步骤 5：请求构造与 Mock 测试（输出层）\n使用 **api-tester-cn** 完成：\n- 构造 API 请求并生成 curl 命令\n- 生成 Mock 数据用于前端开发对接\n- 提供 HTTP 状态码速查和 Headers 说明\n- 验证文档中的请求示例是否可执行\n\n输出可执行的 curl 命令集和 Mock 数据。\n\n## 步骤 6：OpenAPI 规范与 SDK 指南（输出层）\n使用 **API Documentation Generator** (afrexai-api-docs) 完成：\n- 生成完整的 OpenAPI 3.0 规范文件（YAML/JSON）\n- 输出 Markdown 格式的 API 参考文档\n- 生成 SDK 快速入门指南（含多语言代码示例）\n- 包含错误码完整列表和认证流程说明\n\n## 最终输出\n将以上步骤的结果整合为完整的 API 文档包，交付以下文件：\n1. **API 设计规范**：资源模型、版本策略、错误处理规范\n2. **OpenAPI 3.0 规范文件**：YAML/JSON 格式，可导入 Swagger UI\n3. **API 参考文档**：完整的端点说明、参数定义、请求/响应示例\n4. **SDK 快速入门指南**：多语言代码示例和认证配置说明\n5. **curl 命令集**：每个端点的可执行测试命令",
    "skillSlugs": [
      "api-designer",
      "sovereign-api-docs-generator",
      "api-dev",
      "api-doc-writer",
      "api-tester-cn",
      "afrexai-api-docs"
    ],
    "skillCount": 6
  },
  {
    "id": 29,
    "slug": "tech-bug-troubleshooting",
    "displayName": "Bug 排查",
    "summary": "从多格式日志解析与错误模式分析到七步调试法、运行时执行追踪、四阶段系统化根因分析，再到零回归修复工作流和快速错误解释的完整Bug排查工作流。支持Python、Node.js、Java等多语言运行时调试，覆盖堆栈跟踪解析、错误模式识别、假设检验和修复验证。",
    "scene": "tech",
    "subScene": "bug-troubleshooting",
    "category": "tech",
    "content": "---\nscene: \"tech\"\nsub_scene: \"bug-troubleshooting\"\nskills:\n  - \"log-analyzer\"\n  - \"debug-pro\"\n  - \"runtime-debugging-skill\"\n  - \"superpowers-systematic-debugging\"\n  - \"bug-fixing\"\n  - \"nexus-error-explain\"\n---\n\n# Bug 排查工作流\n\n你现在要完成一项软件 Bug 的定位与修复任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：日志收集与错误模式分析（获取层）\n使用 **Log Analyzer** 完成：\n- 解析应用程序日志文件（支持多种格式）\n- 搜索错误信息、异常堆栈和告警记录\n- 分析错误发生的时间线和频率模式\n- 跨服务关联事件，追踪请求链路\n- 解析堆栈跟踪（Stack Trace），定位异常抛出位置\n\n输出日志分析摘要和关键错误事件时间线。\n\n## 步骤 2：系统化调试（分析层）\n使用 **Debug Pro** 完成：\n- 执行七步调试协议：环境确认 → 问题复现 → 最小化复现 → 假设生成 → 假设验证 → 修复 → 回归验证\n- 使用语言特定调试命令（Python pdb、Node.js inspector、Java jdb 等）\n- 跨多环境（开发/测试/生产）系统化排查\n- 记录每步调试证据和中间结果\n\n输出调试过程记录和初步定位结果。\n\n## 步骤 3：运行时执行追踪（分析层）\n使用 **Runtime Debugging Skill** 完成：\n- 对 Python、Node.js 或 Java 应用进行运行时执行追踪\n- 动态插入断点和日志输出，观察变量状态变化\n- 追踪函数调用链，定位异常传播路径\n- 分析运行时内存、线程和I/O状态\n\n输出运行时诊断数据和异常传播路径。\n\n## 步骤 4：四阶段根因分析（分析层）\n使用 **Superpowers Systematic Debugging** 完成：\n- **根因调查**：收集所有相关证据（日志、状态、配置）\n- **模式分析**：识别错误模式，对比历史类似问题\n- **假设检验**：生成并逐一验证可能的根因假设\n- **修复验证**：基于证据确认根因，验证修复方案有效性\n\n输出根因分析报告和验证结论。\n\n## 步骤 5：零回归修复工作流（输出层）\n使用 **bug-fixing** 完成：\n- **分诊**：评估 Bug 严重等级和影响范围\n- **复现**：确认稳定复现路径\n- **影响分析**：评估修复的波及面\n- **修复**：实施修复方案\n- **验证**：执行回归测试确认无新问题引入\n- **知识沉淀**：记录 Bug 根因和修复方案到知识库\n\n输出修复方案和回归验证结果。\n\n## 步骤 6：快速错误解释（输出层）\n使用 **NEXUS Error Explain** 完成：\n- 粘贴任意错误信息、堆栈跟踪或异常\n- 即时获得简明的根因解释\n- 输出可操作的修复建议和代码示例\n\n## 最终输出\n将以上步骤的结果整合为完整的 Bug 排查包，交付以下文件：\n1. **Bug 排查报告**：问题描述、复现步骤、根因分析、修复方案\n2. **调试过程记录**：七步调试协议执行记录和证据链\n3. **修复验证报告**：修复前后对比、回归测试结果\n4. **知识沉淀文档**：Bug 根因模式和防范建议",
    "skillSlugs": [
      "log-analyzer",
      "debug-pro",
      "runtime-debugging-skill",
      "superpowers-systematic-debugging",
      "bug-fixing",
      "nexus-error-explain"
    ],
    "skillCount": 6
  },
  {
    "id": 30,
    "slug": "tech-code-refactoring",
    "displayName": "代码重构",
    "summary": "从深度代码结构分析与DDD模式识别、技术债务评估与架构反模式检测、SOLID/整洁代码/整洁架构原则指导、经典重构模式与遗留代码改造技术，到系统架构与模块化设计、不改变行为的代码简化重构执行的完整代码重构工作流。支持复杂度分析、依赖关系可视化、设计模式应用和渐进式重构策略。",
    "scene": "tech",
    "subScene": "code-refactoring",
    "category": "tech",
    "content": "---\nscene: \"tech\"\nsub_scene: \"code-refactoring\"\nskills:\n  - \"code-analyzer\"\n  - \"agent-git-oracle\"\n  - \"uncle-bob\"\n  - \"code-refactoring\"\n  - \"system-architect\"\n  - \"simplify\"\n---\n\n# 代码重构工作流\n\n你现在要完成一次全面的代码重构任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：深度代码结构分析（获取层）\n使用 **Code Analyzer** 完成：\n- 对目标代码进行深度结构分析，识别类/函数/模块的职责分布\n- 运用 DDD（领域驱动设计）模式识别领域边界和聚合根\n- 分析代码复杂度（圈复杂度、认知复杂度）\n- 绘制依赖关系图，标记高耦合区域\n- 识别重复代码段和相似逻辑片段\n\n输出代码结构分析报告和复杂度热力图。\n\n## 步骤 2：技术债务与架构反模式检测（获取层）\n使用 **Agent Git Oracle** 完成：\n- 分析 Git 历史，识别频繁修改的热点文件\n- 检测技术债务累积区域（高变更频率 + 高复杂度）\n- 识别架构反模式（循环依赖、上帝类、过度耦合）\n- 评估代码腐化程度和重构优先级\n- 标记风险最高的模块和函数\n\n输出技术债务评估报告和重构优先级排序。\n\n## 步骤 3：整洁代码与 SOLID 原则评审（分析层）\n使用 **Uncle Bob** 完成：\n- 基于 SOLID 原则（单一职责、开闭、里氏替换、接口隔离、依赖倒置）逐项评审\n- 按整洁代码（Clean Code）标准审查命名、函数长度、注释质量\n- 运用整洁架构（Clean Architecture）评估分层是否合理\n- 识别违反设计原则的具体代码位置\n- 给出符合原则的重构方向建议\n\n输出 SOLID 合规报告和原则性重构建议。\n\n## 步骤 4：重构模式与技术选型（分析层）\n使用 **Code Refactoring** 完成：\n- 针对步骤 2-3 识别的问题，匹配经典重构模式（Extract Method、Move Field、Replace Conditional with Polymorphism 等）\n- 制定遗留代码改造策略（Strangler Fig、Branch by Abstraction）\n- 设计渐进式重构路径，确保每步可独立验证\n- 评估重构风险和回归测试需求\n- 生成重构操作清单（按优先级排序）\n\n输出重构模式匹配表和分步操作计划。\n\n## 步骤 5：系统架构与模块化设计（输出层）\n使用 **System Architect** 完成：\n- 基于重构目标设计新的模块化架构\n- 定义清晰的模块边界、接口契约和依赖方向\n- 规划组件拆分策略和通信机制\n- 设计面向扩展的架构（插件化、微服务化等）\n- 输出架构图（模块关系图、分层图、数据流图）\n\n输出目标架构设计文档和模块拆分方案。\n\n## 步骤 6：代码简化与重构执行（输出层）\n使用 **Simplify** 完成：\n- 在不改变外部行为的前提下简化代码实现\n- 消除冗余逻辑、合并重复代码、简化条件表达式\n- 优化函数签名和参数传递\n- 提升代码可读性和可维护性\n- 验证重构后代码的功能等价性\n\n输出简化后的代码和变更说明。\n\n## 最终输出\n将以上步骤的结果整合为完整的代码重构包，交付以下文件：\n1. **代码结构分析报告**：复杂度热力图、依赖关系图、重复代码清单\n2. **技术债务评估报告**：债务分布、风险等级、重构优先级排序\n3. **SOLID 合规报告**：原则违反清单和修正方向\n4. **重构操作计划**：分步重构清单、模式匹配、风险评估\n5. **目标架构设计**：模块化架构图、接口契约、迁移路径\n6. **重构后代码**：简化优化后的代码实现和变更日志",
    "skillSlugs": [
      "code-analyzer",
      "agent-git-oracle",
      "uncle-bob",
      "code-refactoring",
      "system-architect",
      "simplify"
    ],
    "skillCount": 6
  },
  {
    "id": 31,
    "slug": "tech-code-review",
    "displayName": "代码审查",
    "summary": "从GitHub PR自动接入到严格代码审查、编码规范检查与自动修复、安全漏洞扫描、整洁代码原则验证，再到生成结构化中文审查报告的完整代码审查工作流。支持Python、JavaScript、TypeScript、Go、Swift、Kotlin等多语言，覆盖Bug检测、安全漏洞、性能问题、代码风格、复杂度分析等维度。",
    "scene": "tech",
    "subScene": "code-review",
    "category": "tech",
    "content": "---\nscene: \"tech\"\nsub_scene: \"code-review\"\nskills:\n  - \"pr-reviewer\"\n  - \"critical-code-reviewer\"\n  - \"project-code-standard\"\n  - \"security-audit\"\n  - \"clean-code-review\"\n  - \"cody\"\n---\n\n# 代码审查工作流\n\n你现在要完成一次全面的代码审查任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：PR 接入与差异分析（获取层）\n使用 **pr-reviewer** 完成：\n- 自动获取 GitHub Pull Request 的代码差异（diff）\n- 集成 lint 工具进行初步静态检查\n- 分析变更文件范围、影响面和风险等级\n- 标记新增/修改/删除的代码段\n\n输出 PR 差异分析和初步 lint 结果。\n\n## 步骤 2：严格代码质量审查（分析层）\n使用 **Critical Code Reviewer** 完成：\n- 以对抗性视角严格审查代码，绝不容忍平庸\n- 检测潜在 Bug、逻辑错误和边界条件问题\n- 识别安全漏洞（注入、XSS、敏感信息泄露等）\n- 分析性能瓶颈和资源泄漏风险\n- 支持 Python、JavaScript/TypeScript、SQL 及前端代码\n\n记录所有质量问题和严重等级。\n\n## 步骤 3：编码规范检查与自动修复（分析层）\n使用 **Project Code Standard** 完成：\n- 检查代码是否符合项目/团队编码规范\n- 验证命名规范、缩进风格、导入顺序等格式要求\n- 对格式问题执行自动修复\n- 生成代码规范合规报告\n\n输出规范检查结果和自动修复建议。\n\n## 步骤 4：安全审计（分析层）\n使用 **Security Audit** 完成：\n- 扫描暴露的凭证、密钥和敏感配置\n- 检测已知 CVE 漏洞和不安全依赖\n- 审查认证授权逻辑、输入校验和加密实现\n- 评估安全风险等级并提供修复方案\n\n输出安全审计报告。\n\n## 步骤 5：整洁代码原则验证（分析层）\n使用 **Clean Code** 完成：\n- 基于 KISS/DRY/YAGNI 原则审查代码设计\n- 识别反模式（God Object、Long Method、Feature Envy 等）\n- 评估函数/类的职责是否单一\n- 检查代码可读性和可维护性\n- 提出重构建议\n\n输出整洁代码评估和重构建议。\n\n## 步骤 6：生成结构化中文审查报告（输出层）\n使用 **code-review-assistant** 完成：\n- 汇总前五步的审查结果\n- 生成结构化中文 Review 报告\n- 报告覆盖：Bug、安全漏洞、性能问题、可读性、最佳实践、类型安全、错误处理、测试覆盖\n- 按严重程度分级（致命/严重/警告/建议）\n\n## 最终输出\n将以上步骤的结果整合为完整的代码审查包，交付以下文件：\n1. **代码审查报告**：按严重等级分类的问题清单和修复建议\n2. **安全审计报告**：漏洞扫描结果和安全风险评估\n3. **规范合规报告**：编码规范检查结果和自动修复记录\n4. **重构建议清单**：基于整洁代码原则的优化方向",
    "skillSlugs": [
      "pr-reviewer",
      "critical-code-reviewer",
      "project-code-standard",
      "security-audit",
      "clean-code-review",
      "cody"
    ],
    "skillCount": 6
  },
  {
    "id": 32,
    "slug": "tech-test-automation",
    "displayName": "自动化测试",
    "summary": "从TDD方法论指导到自动生成单元测试代码、跨语言测试编写与运行、Playwright/Cypress E2E测试编排、REST/GraphQL API测试自动化，再到QA测试计划与覆盖率矩阵生成的完整自动化测试工作流。支持Python(pytest)、JavaScript(Jest/Vitest/Mocha)、Go等多语言，覆盖单元测试、集成测试、E2E测试、API测试和性能测试。",
    "scene": "tech",
    "subScene": "test-automation",
    "category": "tech",
    "content": "---\nscene: \"tech\"\nsub_scene: \"test-automation\"\nskills:\n  - \"tdd-guide\"\n  - \"test-case-generator\"\n  - \"test-patterns\"\n  - \"e2e-test-orchestrator\"\n  - \"api-test-automation\"\n  - \"afrexai-qa-test-plan\"\n---\n\n# 自动化测试工作流\n\n你现在要完成一项软件自动化测试任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：TDD 方法论与测试策略（获取层）\n使用 **Tdd Guide** 完成：\n- 制定测试驱动开发策略：红-绿-重构工作流\n- 识别需要测试的功能模块和边界条件\n- 确定测试层次：单元测试 → 集成测试 → E2E 测试\n- 生成测试固件（fixture）和模拟对象（mock）策略\n- 分析现有覆盖率缺口，制定补测计划\n\n输出测试策略文档和优先级排序。\n\n## 步骤 2：自动生成测试用例（分析层）\n使用 **Test Case Generator** 完成：\n- 分析目标代码的函数签名、逻辑分支和依赖关系\n- 自动生成完整的测试代码（含 import、mock、断言）\n- 支持 Jest、Mocha、Pytest 等主流测试框架\n- 覆盖正常路径、边界条件、异常输入等场景\n\n输出可直接运行的测试文件。\n\n## 步骤 3：跨语言测试编写与运行（分析层）\n使用 **Test Patterns** 完成：\n- 搭建测试套件并配置测试环境\n- 编写单元测试、集成测试和 E2E 测试\n- 支持 Node.js（Jest/Vitest）、Python（pytest）、Go 等多语言\n- 模拟外部依赖（数据库、API、文件系统）\n- 运行测试并测量覆盖率\n\n输出测试执行结果和覆盖率数据。\n\n## 步骤 4：E2E 测试编排与执行（分析层）\n使用 **E2e Test Orchestrator** 完成：\n- 基于 Playwright/Cypress 设计 E2E 测试用例\n- 通过源码优先定位页面元素，必要时使用截图识别兜底\n- 编排测试执行顺序和数据依赖\n- 自动修复因页面变更导致的脚本失效\n- 执行跨浏览器兼容性测试\n\n输出 E2E 测试脚本和执行报告。\n\n## 步骤 5：API 接口测试自动化（分析层）\n使用 **Api Test Automation** 完成：\n- 对 REST/GraphQL 接口进行自动化测试\n- 执行接口功能测试（CRUD、认证、权限）\n- 运行性能测试和压力测试\n- 执行契约测试，验证接口规范一致性\n- 配置 Mock 服务进行隔离测试\n\n输出 API 测试结果和性能基准数据。\n\n## 步骤 6：QA 测试计划与报告生成（输出层）\n使用 **QA Test Plan Generator** 完成：\n- 生成详细的 QA 测试计划文档\n- 构建覆盖率矩阵（功能 × 测试类型）\n- 汇总测试用例清单和缺陷严重级分类\n- 计算自动化 ROI 指标\n- 生成发布检查清单和质量仪表板\n\n## 最终输出\n将以上步骤的结果整合为完整的自动化测试包，交付以下文件：\n1. **测试策略文档**：TDD 方法论、测试层次规划、覆盖率目标\n2. **测试代码**：单元测试 + 集成测试 + E2E 测试 + API 测试\n3. **测试执行报告**：通过/失败统计、覆盖率数据、性能基准\n4. **QA 测试计划**：覆盖率矩阵、发布检查清单、质量仪表板",
    "skillSlugs": [
      "tdd-guide",
      "test-case-generator",
      "test-patterns",
      "e2e-test-orchestrator",
      "api-test-automation",
      "afrexai-qa-test-plan"
    ],
    "skillCount": 6
  },
  {
    "id": 38,
    "slug": "design-competitor-analysis",
    "displayName": "竞品分析",
    "summary": "从市场规模估算、行业细分与竞品图谱绘制，到企业市场趋势分析与用户行为洞察，再到B2B SaaS多场景竞争情报收集、SWOT分析与功能/价格对比、结构化HTML竞品分析报告输出，以及竞品网站/产品/定价自动化持续监控与预警的完整竞品分析工作流。覆盖市场调研、竞争情报、定位策略和动态追踪全链路。",
    "scene": "design",
    "subScene": "competitor-analysis",
    "category": "design",
    "content": "---\nscene: \"design\"\nsub_scene: \"competitor-analysis\"\nskills:\n  - \"market-research\"\n  - \"market-analysis-cn\"\n  - \"competitive-intelligence-market-research\"\n  - \"competitor-analysis-report\"\n  - \"competitive-product-research\"\n  - \"competitor-watch\"\n---\n\n# 竞品分析工作流\n\n你现在要完成一项全面的竞品分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：市场基础调研（获取层）\n使用 **Market Research** 完成：\n- 估算目标市场规模（TAM/SAM/SOM）\n- 绘制市场细分图谱，识别主要玩家\n- 进行价格核查，了解行业定价区间\n- 验证市场需求，识别未满足的用户痛点\n- 将模糊构想转化为数据驱动的决策依据\n\n输出市场规模报告和竞品图谱。\n\n## 步骤 2：企业市场趋势与用户洞察（获取层）\n使用 **Market Analysis CN** 完成：\n- 分析目标企业/行业的市场趋势变化\n- 洞察用户行为模式和消费偏好\n- 识别行业增长点和风险因素\n- 收集竞品的产品定位和目标客群信息\n- 补充中文市场特有的行业数据\n\n输出市场趋势分析和用户行为洞察报告。\n\n## 步骤 3：B2B 竞争情报收集（分析层）\n使用 **Competitive Intelligence & Market Research** 完成：\n- 覆盖销售、HR、金融科技、运营技术等 24 个 B2B SaaS 场景\n- 收集竞争对手的产品功能、技术栈和商业模式\n- 分析竞品的客户案例和成功路径\n- 评估竞品的市场渗透率和增长策略\n- 识别竞争差距和潜在机会\n\n输出竞争情报摘要。\n\n## 步骤 4：SWOT 分析与结构化对比（分析层）\n使用 **Competitor Analysis Report** 完成：\n- 对每个主要竞品进行 SWOT 分析（优势/劣势/机会/威胁）\n- 制作功能对比矩阵（核心功能、差异化功能、缺失功能）\n- 分析竞品定价策略和价格对比表\n- 评估各竞品的市场定位和目标客群差异\n- 给出战略建议和差异化方向\n\n输出 SWOT 分析表和功能/价格对比矩阵。\n\n## 步骤 5：竞品分析报告输出（输出层）\n使用 **competitive-product-research** 完成：\n- 基于\"双轨四层法\"整合前述分析结果\n- 生成可直接用于评审会/复盘会的 HTML 格式报告\n- 包含体验对标、数据对标、策略对标等多维度分析\n- 确保报告结构清晰、数据完整、结论可执行\n\n输出 HTML 格式竞品分析报告。\n\n## 步骤 6：竞品持续监控（输出层）\n使用 **Competitor Watch** 完成：\n- 配置核心竞品的网站/产品页/定价页监控\n- 设置检测变更、新功能发布、价格更新的自动预警\n- 配置竞品分层监控策略（核心对手深度监控 vs 宏观监控）\n- 生成竞品动态摘要报告\n\n输出竞品监控配置和动态预警规则。\n\n## 最终输出\n将以上步骤的结果整合为完整的竞品分析包，交付以下文件：\n1. **市场规模与趋势报告**：TAM/SAM/SOM、行业趋势、用户行为洞察\n2. **竞争情报摘要**：竞品产品/技术/商业模式情报\n3. **SWOT 分析与对比矩阵**：功能/价格/定位多维对比\n4. **HTML 竞品分析报告**：可直接用于评审会的完整报告\n5. **竞品监控方案**：持续追踪配置和预警规则",
    "skillSlugs": [
      "market-research",
      "market-analysis-cn",
      "competitive-intelligence-market-research",
      "competitor-analysis-report",
      "competitive-product-research",
      "competitor-watch"
    ],
    "skillCount": 6
  },
  {
    "id": 40,
    "slug": "design-prd-writing",
    "displayName": "PRD 撰写",
    "summary": "从多轮对话式需求探索与EPIC分解、RICE优先级排序与客户访谈分析框架，到结构化PRD创建（含用户故事与验收标准）、专业PRD撰写润色、PRD转设计需求文档（信息架构/交互流程/页面布局），再到PRD量化评审打分的完整PRD撰写工作流。支持需求分析、优先级排序、文档生成、设计交付和质量评审全链路。",
    "scene": "design",
    "subScene": "prd-writing",
    "category": "design",
    "content": "---\nscene: \"design\"\nsub_scene: \"prd-writing\"\nskills:\n  - \"requirements-analysis\"\n  - \"product-manager-toolkit\"\n  - \"prd\"\n  - \"prd-writer-pro\"\n  - \"prd-to-design-doc\"\n  - \"prd-reviewer\"\n---\n\n# PRD 撰写工作流\n\n你现在要完成一项产品需求文档（PRD）的撰写任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：需求探索与结构化（获取层）\n使用 **requirements-analysis** 完成：\n- 通过多轮对话将用户的简短想法转化为详细需求\n- 将 EPIC 分解为具体需求项和用户故事\n- 识别利益相关者和依赖关系\n- 定义每个需求的验收标准\n- 对多个需求进行初步分类和关联\n\n输出结构化需求清单和用户故事列表。\n\n## 步骤 2：优先级排序与框架选型（分析层）\n使用 **Product Manager Toolkit** 完成：\n- 使用 RICE 框架（Reach/Impact/Confidence/Effort）对需求排序\n- 结合客户访谈数据验证需求假设\n- 选择合适的 PRD 模板和文档结构\n- 制定产品路线图和版本规划\n- 确定 MVP 范围和迭代策略\n\n输出优先级排序表和 MVP 功能清单。\n\n## 步骤 3：PRD 结构化创建（输出层）\n使用 **Prd** 完成：\n- 创建包含完整结构的产品需求文档\n- 编写每个功能的用户故事（As a... I want... So that...）\n- 定义详细的验收标准（Given/When/Then）\n- 规划功能实现的任务拆解\n- 关联需求间的依赖关系\n\n输出结构化 PRD 框架文档。\n\n## 步骤 4：PRD 专业撰写与润色（输出层）\n使用 **PRD文档撰写助手Pro** 完成：\n- 将结构化 PRD 转化为专业、完整的需求文档\n- 补充背景描述、业务目标和成功指标\n- 完善非功能需求（性能、安全、兼容性）\n- 添加边界条件和异常处理说明\n- 确保文档语言清晰、逻辑严谨、无歧义\n\n输出专业级 PRD 文档。\n\n## 步骤 5：PRD 转设计需求文档（输出层）\n使用 **产品prd转设计文档** 完成：\n- 将 PRD 转换为设计团队可直接使用的设计需求文档\n- 输出信息架构图和页面层级关系\n- 生成交互流程图（含异常流程）\n- 定义页面布局规范和视觉要求\n- 自动生成 Mermaid 格式的交互流程图\n\n输出设计需求文档和交互流程图。\n\n## 步骤 6：PRD 量化评审（输出层）\n使用 **Prd Reviewer** 完成：\n- 对 PRD 进行 10 分制严格量化评审\n- 逐模块评分（完整性、清晰度、可执行性、一致性等）\n- 标注具体扣分项和改进建议\n- 识别逻辑漏洞、遗漏场景和模糊描述\n- 生成评审报告，指导 PRD 迭代优化\n\n输出评审评分表和改进建议。\n\n## 最终输出\n将以上步骤的结果整合为完整的 PRD 撰写包，交付以下文件：\n1. **需求清单**：结构化需求列表、用户故事和优先级排序\n2. **PRD 文档**：专业完整的产品需求文档\n3. **设计需求文档**：信息架构、交互流程、页面布局规范\n4. **评审报告**：量化评分、扣分说明和迭代优化建议",
    "skillSlugs": [
      "requirements-analysis",
      "product-manager-toolkit",
      "prd",
      "prd-writer-pro",
      "prd-to-design-doc",
      "prd-reviewer"
    ],
    "skillCount": 6
  },
  {
    "id": 43,
    "slug": "ecommerce-product-copywriting",
    "displayName": "商品文案",
    "summary": "从淘宝商品文案生成（宝贝标题/详情页文案/主图文案/促销活动/关键词挖掘/大促文案）与全平台上架文案生成（跨境电商多平台listing/短视频脚本/FAQ生成/组合套装文案/标题优化诊断/用户痛点挖掘），经电商爆款文案生成（淘宝/拼多多/抖音/京东高转化商品标题/详情页文案/卖点提炼/促销文案）与跨境电商多语种文案生成（Amazon/Shopee/Temu/TikTok商品标题/短描述/详细描述/SEO关键词优化），到SEO优化产品描述生成（关键词/功能/优势/行动号召/多平台适配）与中文电商文案人性化优化（去AI痕迹/保留销售意图/自然转化/本土化表达）的完整商品文案工作流。覆盖淘宝文案、全平台listing、爆款文案、跨境多语种、SEO优化、去AI化全链路。",
    "scene": "ecommerce",
    "subScene": "product-copywriting",
    "category": "ecommerce",
    "content": "---\nscene: \"ecommerce\"\nsub_scene: \"product-copywriting\"\nskills:\n  - \"taobao-listing\"\n  - \"product-listing-generator\"\n  - \"ecommerce-copywriter\"\n  - \"cross-border-copywriter\"\n  - \"product-description-generator\"\n  - \"ecommerce-copy-humanizer-zh\"\n---\n\n# 商品文案工作流\n\n你现在要完成一项商品文案生成任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：淘宝商品文案生成（获取层）\n使用 **Taobao Listing** 完成：\n- 生成淘宝宝贝标题，融合核心关键词和卖点\n- 撰写详情页文案，突出产品功能和使用场景\n- 创作主图文案，吸引点击和收藏\n- 设计促销活动文案和大促专属文案\n- 挖掘长尾关键词提升搜索流量\n\n输出淘宝全套商品文案和关键词清单。\n\n## 步骤 2：全平台上架文案生成（获取层）\n使用 **product-listing-generator** 完成：\n- 生成适配多个电商平台的上架文案\n- 创作跨境电商多平台 Listing 内容\n- 撰写短视频带货脚本和 FAQ\n- 设计组合套装文案和促销方案\n- 进行标题优化诊断和用户痛点挖掘\n\n输出全平台上架文案和优化建议。\n\n## 步骤 3：电商爆款文案优化（分析层）\n使用 **电商爆款文案生成器** 完成：\n- 针对淘宝/拼多多/抖音/京东生成高转化文案\n- 优化商品标题的搜索权重和点击率\n- 提炼产品核心卖点和差异化优势\n- 撰写详情页文案强化购买转化\n- 设计促销文案和活动引流话术\n\n输出各平台爆款文案和卖点提炼。\n\n## 步骤 4：跨境电商多语种文案（分析层）\n使用 **跨境电商爆款商品文案生成器** 完成：\n- 生成 Amazon/Shopee/Temu/TikTok 平台文案\n- 输出多语种商品标题和短描述\n- 撰写详细产品描述和功能亮点\n- 进行 SEO 关键词优化提升搜索排名\n- 适配不同平台的文案规范和字数限制\n\n输出跨境多语种文案和 SEO 关键词。\n\n## 步骤 5：SEO 优化产品描述（输出层）\n使用 **Product Description Generator** 完成：\n- 生成符合 SEO 标准的产品描述\n- 融入高搜索量关键词和长尾词\n- 突出产品功能、优势和使用场景\n- 设计有效的行动号召提升转化率\n- 适配亚马逊、Shopify、eBay、Etsy 等平台\n\n输出 SEO 优化产品描述和关键词布局。\n\n## 步骤 6：中文文案去 AI 味优化（输出层）\n使用 **Ecommerce Copy Humanizer Zh** 完成：\n- 优化中文电商文案使其更自然人性化\n- 消除明显的 AI 生成痕迹和模板感\n- 保留销售意图和转化导向\n- 适配本土化表达习惯和消费者心理\n- 确保文案在真实转化场景中的有效性\n\n输出人性化优化后的最终文案。\n\n## 最终输出\n将以上步骤的结果整合为完整的商品文案成果包，交付以下文件：\n1. **淘宝文案**：宝贝标题、详情页、主图文案、促销文案\n2. **全平台文案**：多平台 Listing、短视频脚本、FAQ\n3. **爆款文案**：高转化标题、卖点提炼、促销话术\n4. **跨境文案**：多语种标题、描述、SEO 关键词\n5. **SEO 描述**：优化产品描述、关键词布局、行动号召\n6. **人性化文案**：去 AI 味、本土化表达、最终定稿",
    "skillSlugs": [
      "taobao-listing",
      "product-listing-generator",
      "ecommerce-copywriter",
      "cross-border-copywriter",
      "product-description-generator",
      "ecommerce-copy-humanizer-zh"
    ],
    "skillCount": 6
  },
  {
    "id": 44,
    "slug": "ecommerce-product-selection",
    "displayName": "选品分析",
    "summary": "从多平台爆款产品筛选（淘宝/京东/拼多多/抖音/亚马逊数据分析）与淘宝产品调研采集（主图/标题/价格/销量/评价/店铺信息），经亚马逊选品市场验证（竞品分析/ASIN评估/定价指导/产品机会挖掘）与跨境电商榜单分析（利润计算/侵权排查/竞品分析/选品决策报告），到OZON平台选品货源搜索（1688货源关键词/供应商筛选/产品定价参考）与1688选品铺货（商机趋势/多平台铺货/货源搜索）的完整选品分析工作流。覆盖多平台数据采集、市场验证、竞品分析、利润评估、货源对接、铺货分发全链路。",
    "scene": "ecommerce",
    "subScene": "product-selection",
    "category": "ecommerce",
    "content": "---\nscene: \"ecommerce\"\nsub_scene: \"product-selection\"\nskills:\n  - \"ecommerce-product-selector\"\n  - \"taobao-product-research\"\n  - \"amazon-product-research-skill\"\n  - \"cross-border-ecommerce-product-analysis\"\n  - \"ozon-product-sourcing\"\n  - \"1688-shopkeeper-official\"\n---\n\n# 选品分析工作流\n\n你现在要完成一项选品分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：多平台爆款产品筛选（获取层）\n使用 **电商选品** 完成：\n- 基于数据分析进行智能选品筛选\n- 支持淘宝、京东、拼多多、抖音、亚马逊等主流平台\n- 按销量、增长率、竞争度等维度评分排序\n- 识别高潜力爆款产品和蓝海品类\n- 输出筛选结果和初步选品清单\n\n输出多平台爆款筛选清单和品类评分。\n\n## 步骤 2：淘宝产品数据采集（获取层）\n使用 **Taobao Product Research** 完成：\n- 采集淘宝商品的主图、标题、价格等核心信息\n- 获取销量、评价数、店铺等级等竞争数据\n- 支持按关键词和品类进行批量采集\n- 生成包含图片的 Excel 数据报告\n- 为市场调研和竞品分析提供原始数据\n\n输出淘宝商品数据采集报告。\n\n## 步骤 3：亚马逊市场验证与竞品分析（分析层）\n使用 **Amazon Product Research** 完成：\n- 对候选产品进行亚马逊市场验证\n- 分析竞品的定价、评分、销量排名\n- 评估 ASIN 级别的产品表现和机会\n- 提供定价指导和差异化建议\n- 挖掘产品机会和市场空白\n\n输出亚马逊市场验证报告和竞品分析。\n\n## 步骤 4：跨境电商榜单与利润分析（分析层）\n使用 **跨境电商选品分析** 完成：\n- 自动爬取跨境电商平台热销榜单\n- 智能计算产品利润率和成本结构\n- 排查知识产权侵权风险\n- 分析竞品策略和市场机会\n- 生成专业完整的选品决策报告\n\n输出选品决策报告和利润分析。\n\n## 步骤 5：海外平台选品与货源搜索（输出层）\n使用 **OZON选品货源搜索助手** 完成：\n- 为目标市场提供针对性选品建议\n- 生成 1688/拼多多货源搜索关键词\n- 制定供应商筛选标准和评估维度\n- 提供产品定价参考和利润预估\n- 匹配适合一件代发的优质货源\n\n输出货源搜索方案和供应商筛选标准。\n\n## 步骤 6：1688 选品铺货与商机趋势（输出层）\n使用 **1688-shopkeeper** 完成：\n- 在 1688 搜索商品和优质货源\n- 分析商机趋势和市场热度变化\n- 将选定商品铺货到抖音/拼多多/小红书/淘宝等平台\n- 查询下游店铺绑定和铺货状态\n- 完成从选品到上架的最后一公里\n\n输出铺货方案和商机趋势分析。\n\n## 最终输出\n将以上步骤的结果整合为完整的选品分析成果包，交付以下文件：\n1. **爆款筛选**：多平台数据、品类评分、潜力产品\n2. **数据采集**：淘宝商品信息、竞争数据、Excel报告\n3. **市场验证**：亚马逊竞品、ASIN评估、定价指导\n4. **利润分析**：榜单爬取、成本计算、侵权排查\n5. **货源方案**：供应商筛选、定价参考、代发匹配\n6. **铺货执行**：1688选品、商机趋势、多平台上架",
    "skillSlugs": [
      "ecommerce-product-selector",
      "taobao-product-research",
      "amazon-product-research-skill",
      "cross-border-ecommerce-product-analysis",
      "ozon-product-sourcing",
      "1688-shopkeeper-official"
    ],
    "skillCount": 6
  },
  {
    "id": 46,
    "slug": "education-quiz-generation",
    "displayName": "题库生成",
    "summary": "从多题型自动生成（选择题/填空题/简答题/模拟考试/难度分级/题库管理）与中国中小学试卷生成（学科试卷/考点覆盖/格式规范）、教材同步智能出题（知识点讲解/智能出题/作业批改/解题答疑）与教学大纲考点提取（考点结构化/重要等级标注/题型匹配），到K12全学段举一反三练习生成（作业批改/错题分析/练习生成）与学习材料练习测试生成（闪卡/学习计划/计时模拟）的完整题库生成工作流。覆盖多题型生成、试卷组卷、智能出题、考点分析、练习生成、测试评估全链路。",
    "scene": "education",
    "subScene": "quiz-generation",
    "category": "education",
    "content": "---\nscene: \"education\"\nsub_scene: \"quiz-generation\"\nskills:\n  - \"quiz-generator\"\n  - \"exam-generator\"\n  - \"math-edu-assistant\"\n  - \"exam-analyzer\"\n  - \"k12-smart-teacher\"\n  - \"exam\"\n---\n\n# 题库生成工作流\n\n你现在要完成一项题库生成任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：多题型批量生成（获取层）\n使用 **Quiz Generator** 完成：\n- 根据知识点批量生成选择题、填空题和简答题\n- 设定题目难度分级（基础/中等/拔高）\n- 为每道题配备标准答案和详细解析\n- 支持模拟考试模式的题目组织\n- 建立题库并进行分类管理\n\n输出分难度分题型的题库初版。\n\n## 步骤 2：学科试卷组卷（获取层）\n使用 **Exam Generator** 完成：\n- 按照中国中小学考试标准生成完整试卷\n- 确保题目覆盖教学大纲核心考点\n- 控制试卷难度梯度和分值分布\n- 规范试卷排版格式（题号/分值/答题区）\n- 生成配套的评分标准和参考答案\n\n输出规范化的学科试卷和评分标准。\n\n## 步骤 3：教材同步智能出题（分析层）\n使用 **数学教育助手** 完成：\n- 严格按照教材内容同步出题\n- 围绕每个知识点设计配套练习\n- 对题目进行解题思路讲解\n- 确保题目难度与教学进度匹配\n- 支持小学至高中全阶段教材\n\n输出教材同步练习题和解题讲解。\n\n## 步骤 4：考点结构化与题型匹配（分析层）\n使用 **考点分析专家** 完成：\n- 从教学大纲或考试说明中提取核心考点\n- 对每个考点标注重要等级（高频/中频/低频）\n- 匹配每个考点适用的题型\n- 生成考点作战地图\n- 确保题库覆盖所有必考知识点\n\n输出考点作战地图和题型匹配方案。\n\n## 步骤 5：举一反三练习生成（输出层）\n使用 **K12智能老师** 完成：\n- 基于错题和薄弱点生成举一反三练习\n- 覆盖小学/初中/高中全学段九大学科\n- 自动批改练习并给出反馈\n- 分析错误原因和知识盲区\n- 生成个性化的强化练习题组\n\n输出举一反三练习题和错题分析报告。\n\n## 步骤 6：综合测试与评估生成（输出层）\n使用 **Exam** 完成：\n- 从学习材料中自动生成综合练习测试\n- 制作知识点闪卡便于快速复习\n- 生成配套学习计划和复习安排\n- 支持计时模拟考试模式\n- 输出完整的测试评估方案\n\n输出综合测试卷、闪卡和学习计划。\n\n## 最终输出\n将以上步骤的结果整合为完整的题库生成成果包，交付以下文件：\n1. **分级题库**：多题型、难度分级、答案解析\n2. **规范试卷**：学科组卷、分值分布、评分标准\n3. **同步练习**：教材同步、知识点配套、解题讲解\n4. **考点地图**：考点提取、重要等级、题型匹配\n5. **强化练习**：举一反三、错题分析、个性化练习\n6. **综合测试**：练习测试、闪卡、学习计划、计时模拟",
    "skillSlugs": [
      "quiz-generator",
      "exam-generator",
      "math-edu-assistant",
      "exam-analyzer",
      "k12-smart-teacher",
      "exam"
    ],
    "skillCount": 6
  },
  {
    "id": 47,
    "slug": "education-student-assessment",
    "displayName": "学生评估",
    "summary": "从教师评估工具箱（评分标准Rubric设计/评估方案/学生反馈/课堂评价）与评分标准差距分析（草稿对标/差距识别/提分计划），错题归类与知识点定位（薄弱环节分析/复习建议/知识图谱）与K12全学段作业批改（错题分析/举一反三/九大学科），到学员成长档案管理（学情记录/家长反馈/咨询留档）与成绩单家长评语生成（作业评语/校园沟通/温暖得体）的完整学生评估工作流。覆盖评估设计、标准分析、错题诊断、作业批改、档案管理、评语生成全链路。",
    "scene": "education",
    "subScene": "student-assessment",
    "category": "education",
    "content": "---\nscene: \"education\"\nsub_scene: \"student-assessment\"\nskills:\n  - \"teacher-toolkit\"\n  - \"rubric-gap-analyzer\"\n  - \"error-analysis\"\n  - \"k12-smart-teacher\"\n  - \"student-growth-ops\"\n  - \"xueersi-parent-comment\"\n---\n\n# 学生评估工作流\n\n你现在要完成一项学生评估任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：评估方案与Rubric设计（获取层）\n使用 **Teacher Toolkit** 完成：\n- 设计针对教学目标的评估方案\n- 制定多维度评分标准（Rubric）\n- 规划形成性评估和终结性评估策略\n- 设计学生反馈收集机制\n- 准备评估相关的家长沟通材料\n\n输出评估方案和评分标准文档。\n\n## 步骤 2：评分差距分析与提分计划（获取层）\n使用 **rubric-gap-analyzer** 完成：\n- 读取评分标准和作业要求\n- 对照Rubric分析学生作品的差距\n- 识别各评分维度的具体不足\n- 制定针对性的提分改进计划\n- 给出优先级排序的改进建议\n\n输出差距分析报告和提分计划。\n\n## 步骤 3：错题归类与薄弱环节诊断（分析层）\n使用 **Error Analysis** 完成：\n- 对学生错题进行系统归类\n- 定位每道错题对应的知识点\n- 分析薄弱环节和知识盲区\n- 绘制知识点掌握度图谱\n- 给出针对薄弱环节的复习建议\n\n输出错题分析报告和知识点图谱。\n\n## 步骤 4：全学段作业批改与反馈（分析层）\n使用 **K12智能老师** 完成：\n- 对作业进行智能批改和打分\n- 分析错误原因并归类错误类型\n- 生成举一反三的巩固练习\n- 覆盖小学/初中/高中九大学科\n- 输出详细的批改反馈和改进方向\n\n输出作业批改结果和个性化反馈。\n\n## 步骤 5：学员成长档案管理（输出层）\n使用 **student-growth-ops** 完成：\n- 建立学员个人学习档案\n- 记录每次学情评估结果和变化趋势\n- 整理家长反馈和沟通记录\n- 跟踪学习目标达成进度\n- 留档咨询记录和关键事件\n\n输出学员成长档案和学情追踪记录。\n\n## 步骤 6：成绩单评语生成（输出层）\n使用 **Xueersi Parent Comment Generator** 完成：\n- 根据学生表现生成温暖得体的评语\n- 适配成绩单、作业本和校园沟通等场景\n- 突出学生优点并委婉指出改进方向\n- 确保评语个性化、避免千篇一律\n- 输出可直接签字使用的家长评语\n\n输出个性化学生评语集。\n\n## 最终输出\n将以上步骤的结果整合为完整的学生评估成果包，交付以下文件：\n1. **评估方案**：Rubric评分标准、评估策略、反馈机制\n2. **差距分析**：维度对标、不足识别、提分计划\n3. **错题诊断**：错题归类、知识点图谱、复习建议\n4. **批改反馈**：作业批改、错因分析、巩固练习\n5. **成长档案**：学情记录、趋势追踪、沟通留档\n6. **评语集**：成绩单评语、作业评语、家长沟通",
    "skillSlugs": [
      "teacher-toolkit",
      "rubric-gap-analyzer",
      "error-analysis",
      "k12-smart-teacher",
      "student-growth-ops",
      "xueersi-parent-comment"
    ],
    "skillCount": 6
  },
  {
    "id": 50,
    "slug": "finance-financial-report-analysis",
    "displayName": "财报分析",
    "summary": "从A股/美股财务数据获取到三表深度解读、财务造假识别再到专业分析报告自动生成的完整财报分析工作流。支持利润表、资产负债表、现金流量表的全面解析，覆盖盈利能力、资产质量、现金流健康度等核心维度。",
    "scene": "finance",
    "subScene": "financial-report-analysis",
    "category": "finance",
    "content": "---\nscene: \"finance\"\nsub_scene: \"financial-report-analysis\"\nskills:\n  - \"tushare-finance\"\n  - \"finnhub\"\n  - \"earnings-reader\"\n  - \"financial-fraud-detection\"\n  - \"finance-report-analyzer\"\n  - \"ai-financial-report-cn\"\n---\n\n# 财报分析工作流\n\n你现在要完成一份上市公司财报的深度分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：获取A股/中国市场财务数据（获取层）\n使用 **tushare-finance** 完成：\n- 获取目标公司的三大财务报表（资产负债表、利润表、现金流量表）\n- 拉取近3-5年的历史财务数据用于趋势对比\n- 获取行业均值和宏观经济指标作为对比基准\n- 如需港股/美股数据也可通过此 Skill 获取\n\n将原始财务数据保存待用。\n\n## 步骤 2：补充美股/全球市场数据（获取层）\n使用 **Finnhub** 完成：\n- 获取美股实时股价、公司新闻和市场数据\n- 拉取目标公司的海外财务报表和交易信号\n- 收集分析师评级和市场情绪数据\n\n将全球市场数据与步骤1的数据合并。\n\n## 步骤 3：三表深度解读（分析层）\n使用 **Earnings Reader** 完成：\n- 逐表解读利润表（营收增速、毛利率、净利率趋势）\n- 逐表解读资产负债表（资产结构、负债水平、流动性）\n- 逐表解读现金流量表（经营性现金流、自由现金流）\n- 识别各科目同比环比的异常波动\n\n记录关键发现和异常指标。\n\n## 步骤 4：财务造假风险识别（分析层）\n使用 **上市公司财务造假识别工具** 完成：\n- 评估虚增资产、虚增利润的嫌疑\n- 识别收入造假、成本造假、资产美化等手法\n- 审查报表可信度，输出财务健康度评估\n- 对比行业正常水平，标记可疑科目\n\n输出风险识别报告。\n\n## 步骤 5：交互式分析报告（输出层）\n使用 **Finance Report Analyzer** 完成：\n- 分析上传的财务数据（Excel/PDF格式）\n- 生成包含迷你趋势图的交互式报告\n- 支持导出为 PDF、DOCX、Markdown 等格式\n\n## 步骤 6：三表报告自动生成（输出层）\n使用 **Ai Financial Report Cn** 完成：\n- 自动生成标准化的资产负债表、利润表、现金流量表分析报告\n- 提供智能财务分析摘要\n- 支持多格式导出和多账套管理\n\n## 最终输出\n将以上步骤的结果整合为完整的财报分析包，交付以下文件：\n1. **财报深度分析报告**：包含三表解读、关键指标分析、趋势对比\n2. **财务风险识别报告**：造假风险评估和可疑科目标注\n3. **可视化数据报告**：含趋势图的交互式报告（PDF/DOCX）",
    "skillSlugs": [
      "tushare-finance",
      "finnhub",
      "earnings-reader",
      "financial-fraud-detection",
      "finance-report-analyzer",
      "ai-financial-report-cn"
    ],
    "skillCount": 6
  },
  {
    "id": 51,
    "slug": "finance-investment-research",
    "displayName": "投研报告",
    "summary": "从A股/美股实时行情数据获取到个股基本面深度研究、行业分析、DCF估值建模，再到自动生成专业投研报告PDF的完整投研工作流。覆盖买方基金经理视角的个股分析简报、投行级行业研究报告、价值投资估值框架，支持技术分析与交易信号输出。",
    "scene": "finance",
    "subScene": "investment-research",
    "category": "finance",
    "content": "---\nscene: \"finance\"\nsub_scene: \"investment-research\"\nskills:\n  - \"ai-stock-analyst\"\n  - \"investlog-ai\"\n  - \"stock-research-engine\"\n  - \"industry-research-analyst\"\n  - \"valuation-analysis\"\n  - \"finance-research-report\"\n---\n\n# 投研报告工作流\n\n你现在要完成一份上市公司或行业的投研报告撰写任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：获取A股市场数据（获取层）\n使用 **AI Stock Analyst** 完成：\n- 通过 AkShare 获取目标A股公司的实时行情、技术指标和估值数据\n- 拉取公司近期新闻和市场动态\n- 获取评分投资建议作为初步参考\n\n将A股市场数据保存待用。\n\n## 步骤 2：获取美股/全球市场数据（获取层）\n使用 **InvestLog AI** 完成：\n- 获取美股目标公司的股价行情、财报数据和估值分析\n- 拉取分析师评级、目标价和机构持仓数据\n- 获取内部人交易、ETF持仓和技术指标信息\n- 收集财务健康评分和分红拆股记录\n\n将美股数据与步骤1的数据合并，形成完整的数据底稿。\n\n## 步骤 3：个股基本面深度研究（分析层）\n使用 **stock-research-engine** 完成：\n- 以买方基金经理视角，对目标公司进行深度投资分析\n- 输出包含市场情绪、基本面评估、管理层评估的分析简报\n- 完成业务拆解、催化剂日历梳理和风险提示\n- 覆盖A股、港股、美股的估值数据展示\n\n记录个股分析的关键发现和投资逻辑。\n\n## 步骤 4：行业深度研究（分析层）\n使用 **Industry Research Analyst 行业研究分析师** 完成：\n- 输出投行级别的行业深度研究报告\n- 覆盖行业概览、市场规模和增长逻辑分析\n- 梳理产业链上中下游关系和竞争格局\n- 分析行业驱动因素、风险评估和投资建议\n\n输出行业研究报告作为个股研究的宏观补充。\n\n## 步骤 5：估值建模分析（分析层）\n使用 **股票价值投资分析系统** 完成：\n- 基于价值投资方法论进行完整估值分析\n- 执行护城河分析（品牌、规模、网络效应、转换成本）\n- 进行财务健康检查和会计质量评估\n- 运用 DCF 模型计算内在价值，对比当前市价\n- 整合管理层评估和行业分析，形成投资决策建议\n\n输出估值分析报告和投资评级。\n\n## 步骤 6：生成专业投研报告（输出层）\n使用 **Skill Fin Report** 完成：\n- 整合前五步的数据和分析结论\n- 自动生成包含技术分析、交易信号、风险评估的专业 PDF 研报\n- 报告包含行情走势图、关键财务指标图表\n- 输出格式化的A股投研报告\n\n## 最终输出\n将以上步骤的结果整合为完整的投研报告包，交付以下文件：\n1. **个股深度研究报告**：基本面分析、估值建模、投资评级\n2. **行业研究报告**：行业概览、竞争格局、驱动因素\n3. **专业投研报告PDF**：含技术分析图表、交易信号、风险评估的完整研报",
    "skillSlugs": [
      "ai-stock-analyst",
      "investlog-ai",
      "stock-research-engine",
      "industry-research-analyst",
      "valuation-analysis",
      "finance-research-report"
    ],
    "skillCount": 6
  },
  {
    "id": 53,
    "slug": "finance-risk-assessment",
    "displayName": "风控评估",
    "summary": "覆盖信用风控建模、量化风险管理（VaR/压力测试/蒙特卡洛模拟）、财务欺诈识别、私募合规审查、仓位风控到风险仪表盘可视化的完整金融风控工作流。支持评分卡构建、决策树模型、特征工程分箱、投资组合风险分析及合规报告生成。",
    "scene": "finance",
    "subScene": "risk-assessment",
    "category": "finance",
    "content": "---\nscene: \"finance\"\nsub_scene: \"risk-assessment\"\nskills:\n  - \"fintech-risk-control\"\n  - \"riskofficer\"\n  - \"financial-fraud-detection\"\n  - \"pe-compliance-expert-pro\"\n  - \"position-risk-manager\"\n  - \"quant-risk-dashboard\"\n---\n\n# 风控评估工作流\n\n你现在要完成一项金融风险控制与评估任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 0：数据准备（获取层 — AI 原生能力）\n此环节由 AI 原生能力完成（无需额外 Skill）：\n- 读取用户上传的贷款记录、交易流水、财务报表等原始数据\n- 解析 Excel/CSV/PDF 格式的金融数据文件\n- 整理数据字段，统一格式，识别缺失值和异常值\n- 准备好结构化数据供后续分析使用\n\n## 步骤 1：信用风控建模（分析层）\n使用 **FinTech Risk Control Expert** 完成：\n- 对金融数据进行特征工程与分箱处理\n- 构建信用评分卡模型（逻辑回归/决策树）\n- 生成风控规则和策略（准入规则、额度规则、定价规则）\n- 分析信用风险指标，输出客户风险分级\n- 使用 Python 处理数据并生成可视化结果\n\n输出信用风控模型和评分卡报告。\n\n## 步骤 2：量化风险分析（分析层）\n使用 **RiskOfficer** 完成：\n- 计算投资组合的 VaR（风险价值）和 CVaR（条件风险价值）\n- 运行蒙特卡洛模拟，评估极端场景下的潜在损失\n- 执行压力测试（历史情景法、假设情景法）\n- 应用风险平价/Black-Litterman 等模型进行组合优化\n- 生成风险指标汇总（Beta、夏普比率、最大回撤）\n\n输出量化风险分析报告。\n\n## 步骤 3：财务欺诈风险识别（分析层）\n使用 **上市公司财务造假识别工具** 完成：\n- 评估虚增资产、虚增利润的嫌疑\n- 识别收入造假、成本造假、资产美化等手法\n- 审查在建工程、商誉、存货、应收账款等科目异常\n- 分析现金流异常和报表可信度\n- 输出财务舞弊风险等级评估\n\n输出欺诈风险识别报告。\n\n## 步骤 4：合规风险审查（分析层）\n使用 **私募基金合规2026** 完成：\n- 基于 AMAC 最新监管规则进行合规审查\n- 覆盖登记备案、募集、内控、信息披露等 8 大合规模块\n- 识别合规缺陷和监管风险点\n- 生成合规审查 Word 报告\n\n输出合规风险审查报告。\n\n## 步骤 5：仓位与持仓风控（分析层）\n使用 **Position Risk Manager** 完成：\n- 评估当前持仓的风险敞口\n- 提供止盈/止损建议和仓位调整方案\n- 应用核心-卫星策略、动态再平衡等模型\n- 对大幅浮盈、深度套牢、震荡洗盘等场景给出风控建议\n\n输出仓位风控调整方案。\n\n## 步骤 6：风控仪表盘可视化（输出层）\n使用 **Quant Risk Dashboard** 完成：\n- 汇总所有风险指标到统一的风控仪表盘\n- 实时展示 VaR/CVaR 计算结果\n- 可视化压力测试结果和敞口监控\n- 展示仓位限额使用情况和回撤分析图表\n\n## 最终输出\n将以上步骤的结果整合为完整的风控评估包，交付以下文件：\n1. **信用风控报告**：评分卡模型、风控规则、客户风险分级\n2. **量化风险报告**：VaR/压力测试/蒙特卡洛模拟结果\n3. **欺诈风险识别报告**：财务造假风险评估和可疑科目标注\n4. **合规审查报告**：合规缺陷清单和整改建议\n5. **风控仪表盘**：综合风险指标可视化",
    "skillSlugs": [
      "fintech-risk-control",
      "riskofficer",
      "financial-fraud-detection",
      "pe-compliance-expert-pro",
      "position-risk-manager",
      "quant-risk-dashboard"
    ],
    "skillCount": 6
  },
  {
    "id": 58,
    "slug": "hr-meeting-minutes",
    "displayName": "会议纪要",
    "summary": "从会议纪要模板化生成（行动项/待办事项/站会模板）与凌乱笔记快速转化为清晰行动项、探讨决策型会议结构化纪要（结论/共识/分歧/决策轨迹/行动项）与原始记录智能整理归档，到本地Whisper音频转写生成Word/PDF纪要与钉钉飞书会议记录自动分类归档的完整会议纪要工作流。覆盖记录生成、结构化整理、行动项提取、多平台归档全链路。",
    "scene": "hr",
    "subScene": "meeting-minutes",
    "category": "hr",
    "content": "---\nscene: \"hr\"\nsub_scene: \"meeting-minutes\"\nskills:\n  - \"meeting-minutes\"\n  - \"ai-meeting-notes\"\n  - \"meeting-note\"\n  - \"meeting-minutes-organizer\"\n  - \"meeting-notes-assistant\"\n  - \"dingtalk-minutes\"\n---\n\n# 会议纪要工作流\n\n你现在要完成一项会议纪要的生成与整理任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：会议纪要模板化生成（获取层）\n使用 **Meeting Minutes** 完成：\n- 根据会议类型选择合适模板（站会/周会/复盘会/决策会）\n- 记录会议基本信息（时间/地点/参会人/主持人）\n- 按模板结构填充会议内容和讨论要点\n- 提取行动项（Action Items）和待办事项\n- 输出标准格式的会议纪要初稿\n\n输出模板化会议纪要初稿。\n\n## 步骤 2：笔记转行动项（获取层）\n使用 **AI Meeting Notes w/ Action Items + To-Do List Tracker** 完成：\n- 将凌乱的会议笔记或转录文本快速清洗整理\n- 提取带负责人和截止日期的行动项\n- 自动生成会议摘要\n- 保存并建立可搜索的会议记录库\n- 集成待办追踪，确保跟进到位\n\n输出清洗后的会议摘要和行动项清单。\n\n## 步骤 3：决策型会议深度整理（分析层）\n使用 **Meeting Note** 完成：\n- 梳理探讨/决策型会议的完整脉络\n- 提炼结论、共识、分歧和决策轨迹\n- 识别隐含假设、风险机会\n- 提取可执行行动项并标注优先级\n- 通过 Zettelkasten 方法建立知识连接\n\n输出决策型会议的深度纪要和知识图谱。\n\n## 步骤 4：原始记录结构化整理（分析层）\n使用 **会议纪要自动整理工具** 完成：\n- 将原始会议记录智能整理为结构化纪要\n- 按议题分类归纳讨论内容\n- 提取决议事项、责任人和截止日期\n- 标注关键决策和待确认事项\n- 输出格式规范的正式会议纪要\n\n输出结构化正式会议纪要。\n\n## 步骤 5：音频转写与文档输出（输出层）\n使用 **Meeting Notes Assistant** 完成：\n- 使用本地 Whisper 进行会议录音转写（离线、隐私安全）\n- 生成结构化纪要（时间/议题/结论/待办/关键词）\n- 提取 Action Items 并分配责任人\n- 支持 Word/PDF/邮件格式输出\n- 实现会议归档与待办分发\n\n输出转写文本和多格式会议纪要文档。\n\n## 步骤 6：多平台会议记录归档（输出层）\n使用 **DingTalk Minutes** 完成：\n- 自动整理钉钉/飞书等平台的会议记录\n- 智能合并相关会议内容\n- 提取参会人、核心议题、关键决策和待办事项\n- 分类归档到指定目录\n- 建立可检索的会议知识库\n\n## 最终输出\n将以上步骤的结果整合为完整的会议纪要包，交付以下文件：\n1. **模板化纪要初稿**：标准格式、基本信息、讨论要点\n2. **行动项清单**：负责人、截止日期、待办追踪\n3. **决策型深度纪要**：结论/共识/分歧、决策轨迹、知识连接\n4. **结构化正式纪要**：议题分类、决议事项、格式规范\n5. **多格式文档**：音频转写、Word/PDF 输出、归档分发\n6. **多平台归档**：钉钉/飞书记录整理、分类归档、知识库",
    "skillSlugs": [
      "meeting-minutes",
      "ai-meeting-notes",
      "meeting-note",
      "meeting-minutes-organizer",
      "meeting-notes-assistant",
      "dingtalk-minutes"
    ],
    "skillCount": 6
  },
  {
    "id": 62,
    "slug": "legal-compliance-analysis",
    "displayName": "合规分析",
    "summary": "从中国法律合规AI技能包（50个合规技能/合同审查/法律问答/劳动合规/知识产权/数据保护）与全流程法律合规智库（实时监管检索/条款穿透分析/时效追踪/公文级报告），法律财务采购合规审计引擎（风险条款提取/法规基线映射/审计底稿/澄清模板）与中国个人信息保护法PIPL合规（合规检查/风险评估/文档模板/合规自查），到多框架合规审计报告生成（SOC2/ISO27001/GDPR/HIPAA/PCI DSS/风险评分/整改路线图）与AI合规分析（欧盟AI法案/ISO42001/NIST AI RMF/GDPR/OECD/金融监管）的完整合规分析工作流。覆盖法律合规、监管检索、审计引擎、数据保护、多框架审计、AI合规全链路。",
    "scene": "legal",
    "subScene": "compliance-analysis",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"compliance-analysis\"\nskills:\n  - \"legal-compliance-bundle\"\n  - \"nathan-legal-os-pro\"\n  - \"compliance-audit-pro\"\n  - \"pipl-compliance\"\n  - \"afrexai-compliance-audit\"\n  - \"ai-compliance\"\n---\n\n# 合规分析工作流\n\n你现在要完成一项合规分析任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：法律合规全景扫描（获取层）\n使用 **中国法律合规AI技能包** 完成：\n- 基于50个AI法律合规技能进行全景扫描\n- 覆盖合同审查、劳动合规、知识产权等领域\n- 检查数据保护和隐私合规要求\n- 提供法律问答和合规咨询\n- 识别企业运营中的合规风险点\n\n输出法律合规全景扫描报告。\n\n## 步骤 2：实时监管检索与条款穿透（获取层）\n使用 **全流程法律合规智库** 完成：\n- 实时检索最新监管法规和政策动态\n- 对关键条款进行穿透式深度分析\n- 追踪合规时效和重要截止日期\n- 支持PE/VC私募合规和劳动纠纷场景\n- 生成公文级别的合规分析报告\n\n输出监管检索结果和条款穿透分析。\n\n## 步骤 3：合规审计引擎分析（分析层）\n使用 **compliance-audit-pro** 完成：\n- 自动提取文档中的风险条款\n- 将风险条款映射至法规基线\n- 生成标准化审计底稿\n- 制作澄清问题模板\n- 覆盖法律、财务和采购场景\n\n输出审计底稿和风险映射报告。\n\n## 步骤 4：个人信息保护法合规检查（分析层）\n使用 **PIPL-Compliance（PIPL合规工具）** 完成：\n- 按PIPL要求进行合规检查\n- 评估个人信息处理的合规风险\n- 生成合规文档模板\n- 支持企业PIPL合规自查\n- 检查跨境数据传输合规性\n\n输出PIPL合规检查报告和文档模板。\n\n## 步骤 5：多框架合规审计报告（输出层）\n使用 **Compliance Audit Generator** 完成：\n- 生成SOC 2、ISO 27001等框架的合规审计\n- 覆盖GDPR、HIPAA和PCI DSS等标准\n- 提供风险优先排序的审计发现\n- 制定补救计划和整改路线图\n- 输出可交付的合规审计报告\n\n输出多框架合规审计报告和整改计划。\n\n## 步骤 6：AI合规专项分析（输出层）\n使用 **AI Compliance** 完成：\n- 分析欧盟AI法案的合规要求\n- 评估ISO 42001和NIST AI RMF合规\n- 检查GDPR和OECD相关AI规定\n- 审查金融服务监管中的AI合规\n- 输出AI合规分析和建议报告\n\n输出AI合规分析报告。\n\n## 最终输出\n将以上步骤的结果整合为完整的合规分析成果包，交付以下文件：\n1. **全景扫描**：50项合规技能、多领域风险识别\n2. **监管检索**：实时法规、条款穿透、时效追踪\n3. **审计底稿**：风险条款、法规映射、澄清模板\n4. **PIPL合规**：个人信息保护、合规自查、文档模板\n5. **多框架审计**：SOC2/ISO/GDPR/HIPAA/PCI DSS\n6. **AI合规**：欧盟AI法案、ISO42001、NIST、金融监管",
    "skillSlugs": [
      "legal-compliance-bundle",
      "nathan-legal-os-pro",
      "compliance-audit-pro",
      "pipl-compliance",
      "afrexai-compliance-audit",
      "ai-compliance"
    ],
    "skillCount": 6
  },
  {
    "id": 64,
    "slug": "legal-contract-generation",
    "displayName": "合同生成",
    "summary": "从模板参考到风险审查再到专业合同文书输出的一站式合同生成工作流，覆盖劳动合同、保密协议、租赁合同、服务协议等主流合同类型。",
    "scene": "legal",
    "subScene": "contract-generation",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"contract-generation\"\nskills:\n  - \"contract-template\"\n  - \"legal-advisor\"\n  - \"ai-contract-review-cn\"\n  - \"nathan-legal-os-pro\"\n  - \"zhang-contract-generator\"\n  - \"legal-doc-writer\"\n---\n\n# 合同生成工作流\n\n你现在要完成一份专业合同的生成任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：需求确认与法律咨询（获取层）\n使用 **Legal Advisor** 完成：\n- 确认用户需要的合同类型（劳动合同、租赁合同、保密协议、服务协议等）\n- 明确合同涉及的法律领域（合同法、劳动法、知识产权法等）\n- 咨询该类合同的法律要点和必备条款\n- 确认合同双方的主体信息和核心商务条件\n\n将法律要点和必备条款清单保存为后续起草参考。\n\n## 步骤 2：模板获取与条款参考（获取层）\n使用 **Contract Template** 完成：\n- 根据步骤 1 确定的合同类型，获取对应的标准合同模板\n- 提取模板中的核心条款结构（标的、价款、履行、违约、争议解决等）\n- 参考模板的条款措辞和法律用语规范\n\n将模板结构和关键条款作为起草基础。\n\n## 步骤 3：合同起草（输出层）\n使用 **九章合同生成器 V1.4.0** 完成：\n- 基于步骤 1 的法律要点和步骤 2 的模板结构，生成完整合同初稿\n- 填入用户提供的具体商务条件（金额、期限、标的物等）\n- 确保合同包含所有必备条款（主体、标的、价款、履行、违约责任、争议解决、生效条件）\n\n生成合同初稿文本。\n\n## 步骤 4：合规性审查（分析层）\n使用 **全流程法律合规智库** 完成：\n- 对合同初稿进行条款穿透分析，检查是否符合现行法律法规\n- 检查是否存在违反强制性法律规定的条款\n- 验证时效条款、管辖条款等程序性条款的合规性\n\n记录合规审查意见和修改建议。\n\n## 步骤 5：风险识别与条款优化（分析层）\n使用 **Ai Contract Review Cn** 完成：\n- 对合同进行全面风险识别，标记高风险条款\n- 解读模糊或不利条款，提供优化建议\n- 检查权利义务是否对等，是否存在显失公平条款\n- 根据审查结果修改优化合同文本\n\n输出风险报告和修改后的合同终稿。\n\n## 步骤 6：附属文书生成（输出层）\n使用 **Legal Doc Writer** 完成：\n- 根据需要生成合同附属文书（补充协议、法律意见书、合同签署指引等）\n- 确保附属文书与主合同条款保持一致\n- 格式化输出符合法律文书规范的最终文档\n\n## 最终输出\n将以上步骤的结果整合为完整的合同文书包，交付以下文件：\n1. **合同正文**：经过合规审查和风险优化的完整合同文本\n2. **风险报告**：合同条款的风险识别结果和应对建议\n3. **附属文书**（按需）：补充协议、签署指引或法律意见书",
    "skillSlugs": [
      "contract-template",
      "legal-advisor",
      "ai-contract-review-cn",
      "nathan-legal-os-pro",
      "zhang-contract-generator",
      "legal-doc-writer"
    ],
    "skillCount": 6
  },
  {
    "id": 65,
    "slug": "legal-contract-review",
    "displayName": "合同审查",
    "summary": "从基于CUAD数据集41个风险类别的法律合同分析（保密协议/SaaS协议/并购协议/雇佣协议/支付协议/寻源协议）与智能合同风险识别条款解读（多合同类型/法律咨询/合规管理），合同风险条款自动识别与缺失条款补充（差异条款对比/风险标注）与商业合同审查（NDA/MSA/SaaS协议/供应商合同/风险/缺失/合规），到合同关键信息提取与到期追踪（风险条款/关键信息/到期日）与合同文本提取审查（Word格式/金额条款/付款节点/违约金）的完整合同审查工作流。覆盖风险识别、条款解读、缺失补充、差异对比、关键提取、金额审查全链路。",
    "scene": "legal",
    "subScene": "contract-review",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"contract-review\"\nskills:\n  - \"contract-review\"\n  - \"ai-contract-review-cn\"\n  - \"audit-new\"\n  - \"contract-reviewer\"\n  - \"contract-guardian\"\n  - \"contract-auditor\"\n---\n\n# 合同审查工作流\n\n你现在要完成一项合同审查任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：CUAD多维风险扫描（获取层）\n使用 **Contract Review** 完成：\n- 基于CUAD数据集对合同进行41个风险类别扫描\n- 覆盖保密协议、SaaS协议、并购协议等多种合同类型\n- 自动识别高风险条款并标注风险等级\n- 分析雇佣协议、支付协议和寻源协议中的潜在问题\n- 生成多维度风险扫描报告\n\n输出41类风险扫描报告和高风险条款清单。\n\n## 步骤 2：智能风险识别与条款解读（获取层）\n使用 **Ai Contract Review Cn** 完成：\n- 对合同进行智能风险识别\n- 逐条解读关键条款的法律含义\n- 支持多种合同类型的专业分析\n- 提供法律咨询和合规管理建议\n- 标注需要重点关注的风险点\n\n输出条款解读报告和法律咨询建议。\n\n## 步骤 3：风险条款识别与缺失补充（分析层）\n使用 **合同审查Pro** 完成：\n- 自动识别合同中的风险条款\n- 补充合同中缺失的必要条款\n- 对比不同版本间的差异条款\n- 标注每个风险点的严重程度\n- 生成条款修改建议\n\n输出风险条款清单、缺失条款补充和差异对比。\n\n## 步骤 4：商业合同深度审查（分析层）\n使用 **Contract Reviewer** 完成：\n- 深度审查NDA、MSA、SaaS协议等商业合同\n- 检查合规漏洞和不利条款\n- 识别缺失的保护性条款\n- 评估整体合同风险水平\n- 生成谈判准备清单\n\n输出深度审查报告和谈判清单。\n\n## 步骤 5：关键信息提取与到期追踪（输出层）\n使用 **合同卫士 / Contract Guardian** 完成：\n- 提取合同中的关键信息和核心条款\n- 识别并汇总所有风险条款\n- 追踪合同到期日和关键时间节点\n- 建立合同关键信息索引\n- 设置到期预警和续签提醒\n\n输出关键信息提取表和到期追踪清单。\n\n## 步骤 6：金额条款与付款审查（输出层）\n使用 **Contract Auditor** 完成：\n- 自动提取合同文本内容\n- 重点审查金额条款的一致性\n- 核查付款节点和付款条件\n- 评估违约金条款的合理性\n- 输出合同审计报告\n\n输出金额审查报告和合同审计意见。\n\n## 最终输出\n将以上步骤的结果整合为完整的合同审查成果包，交付以下文件：\n1. **风险扫描**：41类CUAD风险、高风险条款清单\n2. **条款解读**：智能识别、法律含义、咨询建议\n3. **缺失补充**：风险条款、缺失条款、差异对比\n4. **深度审查**：商业合同、合规漏洞、谈判清单\n5. **关键提取**：核心条款、到期追踪、续签提醒\n6. **金额审计**：金额一致性、付款节点、违约金评估",
    "skillSlugs": [
      "contract-review",
      "ai-contract-review-cn",
      "audit-new",
      "contract-reviewer",
      "contract-guardian",
      "contract-auditor"
    ],
    "skillCount": 6
  },
  {
    "id": 66,
    "slug": "legal-legal-research",
    "displayName": "法规检索",
    "summary": "从法律知识库检索（法律法规查询/合同条款检索/司法解释查找）与财税法律案例法规检索（税务判例/行政复议/税收法规/税务实践案例），官方API法条查询与多类纠纷检索（劳动纠纷/借贷纠纷/侵权纠纷/合同纠纷/工伤认定/婚姻家事/消费维权）与法条体系定位（上位规范/并列条款/下位细化/程序衔接/竞合分析），到类案检索验证（真实案例验证/裁判分歧/高频败诉原因/法官审查重点）与不确定法律概念深挖（合理期限/重大误解/明显不当/裁判标准/边界案例）的完整法规检索工作流。覆盖法规查询、案例检索、纠纷查找、体系定位、类案验证、概念分析全链路。",
    "scene": "legal",
    "subScene": "legal-research",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"legal-research\"\nskills:\n  - \"legalkb\"\n  - \"case-research\"\n  - \"legal-hybrid-skill\"\n  - \"legal-system-mapper-mctmilk\"\n  - \"legal-case-validator-mctmilk\"\n  - \"legal-concept-deep-dive-mctmilk\"\n---\n\n# 法规检索工作流\n\n你现在要完成一项法规检索任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：法律知识库检索（获取层）\n使用 **legalkb** 完成：\n- 检索法律法规的具体条文内容\n- 查询合同相关条款的法律依据\n- 查找司法解释和适用规范\n- 支持按关键词和法条编号精确检索\n- 汇总检索结果形成法规清单\n\n输出法规检索结果和法条清单。\n\n## 步骤 2：案例与法规专项检索（获取层）\n使用 **Case Research** 完成：\n- 检索税务相关判例和行政复议决定\n- 查找税收法规政策和税务总局公告\n- 搜索各地税务实践案例\n- 查找类似税务争议案例\n- 整理案例与法规的对应关系\n\n输出案例检索报告和法规政策汇编。\n\n## 步骤 3：多类纠纷法条查询（分析层）\n使用 **智能法律助手** 完成：\n- 通过官方API进行精准法条查询\n- 覆盖劳动纠纷、借贷纠纷、侵权纠纷等查询\n- 支持合同纠纷、工伤认定、婚姻家事查询\n- 提供消费维权相关法律依据\n- 本地法条兜底确保查询完整性\n\n输出多类纠纷法条查询结果。\n\n## 步骤 4：法条体系关联定位（分析层）\n使用 **法条体系定位** 完成：\n- 以目标法条为核心构建关联网络\n- 梳理上位规范和下位细化条款\n- 识别并列条款和程序衔接关系\n- 分析法条竞合情况\n- 明确法条在法律体系中的位置\n\n输出法条关联网络图和体系定位报告。\n\n## 步骤 5：类案检索与裁判验证（输出层）\n使用 **类案检索验证** 完成：\n- 用真实案例验证法条分析结果\n- 分析司法实践中的裁判分歧\n- 识别高频败诉原因\n- 提炼法官审查重点\n- 确保法律分析与司法实践一致\n\n输出类案验证报告和裁判分析。\n\n## 步骤 6：不确定法律概念分析（输出层）\n使用 **不确定法律概念深挖** 完成：\n- 深入分析法条中的模糊概念\n- 界定\"合理期限\"\"重大误解\"\"明显不当\"等概念\n- 梳理裁判标准和适用条件\n- 收集边界案例和典型判例\n- 明确概念的内涵和外延\n\n输出法律概念分析报告和裁判标准梳理。\n\n## 最终输出\n将以上步骤的结果整合为完整的法规检索成果包，交付以下文件：\n1. **法规清单**：法条检索、司法解释、适用规范\n2. **案例汇编**：判例检索、法规政策、实践案例\n3. **纠纷法条**：多类纠纷、官方API、法条查询\n4. **体系定位**：关联网络、上下位法条、竞合分析\n5. **类案验证**：真实案例、裁判分歧、败诉原因\n6. **概念分析**：模糊概念、裁判标准、边界案例",
    "skillSlugs": [
      "legalkb",
      "case-research",
      "legal-hybrid-skill",
      "legal-system-mapper-mctmilk",
      "legal-case-validator-mctmilk",
      "legal-concept-deep-dive-mctmilk"
    ],
    "skillCount": 6
  },
  {
    "id": 67,
    "slug": "legal-litigation-strategy",
    "displayName": "诉讼策略",
    "summary": "从诉讼准备指南（流程指引/文书准备/证据整理/时间线/费用估算）与法律辅助分析（合同风险初筛/诉讼成本估算/文书骨架生成），经应诉策略制定（起诉状解析/争议焦点提炼/法律关系分析/答辩方向制定）与证据矩阵建模（七步分析法/证据矩阵/案件全景分析），到民事起诉状生成（案情要素提取/证据材料整理/规范起诉状生成）与全流程法律服务（刑事辩护/民商事诉讼/合同审查/法律AI中台）的完整诉讼策略工作流。覆盖诉讼准备、风险评估、策略制定、证据建模、文书起草、全流程支持全链路。",
    "scene": "legal",
    "subScene": "litigation-strategy",
    "category": "legal",
    "content": "---\nscene: \"legal\"\nsub_scene: \"litigation-strategy\"\nskills:\n  - \"court-prep\"\n  - \"ai-legal-assistant-pro\"\n  - \"litigation-response\"\n  - \"pro-legal-strategist-v2\"\n  - \"complaint-drafting\"\n  - \"zhang-lawyer-suite\"\n---\n\n# 诉讼策略工作流\n\n你现在要完成一项诉讼策略制定任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：诉讼准备与流程指引（获取层）\n使用 **Court Prep** 完成：\n- 梳理诉讼整体流程和关键节点\n- 准备诉讼所需的各类文书清单\n- 整理已有证据材料并建立证据目录\n- 制定诉讼时间线和里程碑\n- 估算诉讼费用和相关成本\n- 评估是否适用小额诉讼程序\n\n输出诉讼准备方案和流程指引。\n\n## 步骤 2：风险评估与成本分析（获取层）\n使用 **AI Legal Assistant Pro** 完成：\n- 对涉案合同进行风险初步筛查\n- 评估劳动纠纷等常见纠纷类型的诉讼风险\n- 估算诉讼总体成本和胜诉概率\n- 生成民事起诉状/答辩状/证据提纲的结构化骨架\n- 提供诉讼与非诉解决路径的对比分析\n\n输出风险评估报告和成本估算。\n\n## 步骤 3：争议焦点与应诉策略（分析层）\n使用 **litigation response** 完成：\n- 解析对方起诉状的核心诉求和事实主张\n- 提炼案件争议焦点和法律适用问题\n- 分析各方法律关系和权利义务\n- 制定系统化的答辩方向和应诉策略\n- 识别对方诉求中的薄弱环节\n\n输出争议焦点分析和应诉策略方案。\n\n## 步骤 4：证据矩阵与案件建模（分析层）\n使用 **顶级诉讼律师：七步证据矩阵建模专家** 完成：\n- 运用七步分析法对案件进行全景分析\n- 构建证据矩阵，映射证据与争议焦点的对应关系\n- 评估每项证据的证明力和关联性\n- 识别证据链中的薄弱环节和补强方向\n- 模拟对方可能的反驳策略\n\n输出证据矩阵和案件分析建模报告。\n\n## 步骤 5：诉讼文书起草（输出层）\n使用 **complaint drafting** 完成：\n- 从案件材料中提取关键案情要素\n- 整理和归类证据材料\n- 按照法定格式生成规范的民事起诉状\n- 确保诉讼请求、事实理由、法律依据的完整性\n- 校验文书格式和必备要件\n\n输出规范的民事起诉状和证据材料清单。\n\n## 步骤 6：全流程法律支持（输出层）\n使用 **张律师综合套装** 完成：\n- 提供刑事辩护策略和辩护要点\n- 支持民商事诉讼全流程法律服务\n- 审查涉案合同条款的法律效力\n- 综合运用法律AI中台的多维度分析能力\n- 输出最终的诉讼策略建议书\n\n输出综合法律分析报告和诉讼策略建议。\n\n## 最终输出\n将以上步骤的结果整合为完整的诉讼策略成果包，交付以下文件：\n1. **诉讼准备**：流程指引、文书清单、时间线、费用估算\n2. **风险评估**：合同风险、纠纷分析、成本估算、路径对比\n3. **应诉策略**：争议焦点、法律关系、答辩方向、薄弱环节\n4. **证据矩阵**：七步分析、证据映射、证明力评估、补强方向\n5. **诉讼文书**：民事起诉状、证据清单、法律依据\n6. **综合策略**：全流程支持、多维分析、最终策略建议",
    "skillSlugs": [
      "court-prep",
      "ai-legal-assistant-pro",
      "litigation-response",
      "pro-legal-strategist-v2",
      "complaint-drafting",
      "zhang-lawyer-suite"
    ],
    "skillCount": 6
  },
  {
    "id": 69,
    "slug": "marketing-content-deai",
    "displayName": "内容去 AI 味",
    "summary": "从AI写作特征检测（16+类模式识别与AI概率评分）、基于维基百科指南的24种AI痕迹消除规则、绕过GPTZero等主流AI检测器的通用改写，到中文专精的机械文本人性化转换、AI高频词/过度结构化/机械连接词等7项中文专项修复、中文写作深度润色与语言优化的完整内容去AI味工作流。",
    "scene": "marketing",
    "subScene": "content-deai",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"content-deai\"\nskills:\n  - \"ai-text-humanizer-zh\"\n  - \"humanizer\"\n  - \"humanize-ai-text\"\n  - \"humanize-zh\"\n  - \"unclecheng-reduce-ai-perception-v2\"\n  - \"writing-polish\"\n---\n\n# 内容去 AI 味工作流\n\n你现在要完成一项内容去 AI 味（人性化改写）任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：AI 特征检测与诊断（获取层）\n使用 **AI 文本净化器（中文版）** 完成：\n- 扫描文本中的 16+ 类 AI 写作特征\n- 标注 AI 高频词汇、夸大表述、机器人痕迹\n- 输出 AI 概率评分和详细检测报告\n- 清理 Markdown 格式残留和 chatbot 痕迹\n- 定位需要重点改写的段落和句子\n\n输出 AI 特征检测报告和改写优先级标注。\n\n## 步骤 2：通用 AI 痕迹消除（分析层）\n使用 **Humanizer** 完成：\n- 基于维基百科 AI 写作特征指南执行 24 种规则改写\n- 消除夸张象征、宣传用语、模糊归因等模式\n- 修正破折号滥用、三项排比、AI 高频词汇\n- 去除负面平行结构和冗长连接词\n- 确保改写后文本自然、真实\n\n输出通用去 AI 化改写版本。\n\n## 步骤 3：AI 检测器绕过优化（分析层）\n使用 **Humanize AI text** 完成：\n- 针对 GPTZero、Turnitin 等主流 AI 检测器优化\n- 调整句子结构变化度和词汇多样性\n- 增加人类写作特有的不规则性（口语化/省略/语气词）\n- 保持原文核心含义不变\n- 降低 AI 检测评分至安全阈值\n\n输出可通过 AI 检测器的改写版本。\n\n## 步骤 4：中文人性化转换（输出层）\n使用 **中文去AI味** 完成：\n- 将机械化中文表述转换为自然、有人情味的风格\n- 注入真实情感和个人观点表达\n- 调整语句节奏（长短句交替、口语化穿插）\n- 去除\"总之\"\"综上所述\"等 AI 典型连接词\n- 确保中文语境下的表达地道\n\n输出人性化中文版本。\n\n## 步骤 5：中文专项修复（输出层）\n使用 **文章去AI味工具** 完成：\n- 修复 AI 高频词汇（\"值得注意的是\"\"不仅...而且\"等）\n- 消除过度结构化（机械分段、强制小标题）\n- 去除虚假客观性和机械化连接词\n- 修正完美主义陷阱和公式化结尾\n- 消除过度修饰和堆砌式表达\n\n输出专项修复后的文本。\n\n## 步骤 6：深度润色与语言优化（输出层）\n使用 **Writing Polish** 完成：\n- 整体语言流畅度优化\n- 调整文风一致性和阅读节奏\n- 优化措辞精准度和表达力\n- 确保段落间逻辑连贯自然\n- 最终通读校对，确保发布质量\n\n## 最终输出\n将以上步骤的结果整合为完整的去 AI 味改写包，交付以下文件：\n1. **AI 检测报告**：原文 AI 特征分析和概率评分\n2. **改写对照文档**：原文 vs 改写后的逐段对照\n3. **最终定稿**：经过多轮去 AI 化和润色的发布级文本\n4. **改写策略说明**：本次改写采用的具体策略和修改要点",
    "skillSlugs": [
      "ai-text-humanizer-zh",
      "humanizer",
      "humanize-ai-text",
      "humanize-zh",
      "unclecheng-reduce-ai-perception-v2",
      "writing-polish"
    ],
    "skillCount": 6
  },
  {
    "id": 71,
    "slug": "marketing-official-document-writing",
    "displayName": "公文写作",
    "summary": "从符合GB/T 9704-2012标准的党政机关公文全类型生成（通知/报告/请示/批复/函/会议纪要/公告/决定等15种）、国家标准格式规范与各类公文模板写作技巧、素材重构与主题创作双模式，到公文格式检查/语气检查/模板库、中国公文格式Word文档自动生成（标题分级/字体样式/段落格式/中文引号）、公文规范自动排版的完整公文写作工作流。",
    "scene": "marketing",
    "subScene": "official-document-writing",
    "category": "marketing",
    "content": "---\nscene: \"marketing\"\nsub_scene: \"official-document-writing\"\nskills:\n  - \"official-doc-writer\"\n  - \"official-writing\"\n  - \"govwriter-pro\"\n  - \"official-doc\"\n  - \"docx-formatter\"\n  - \"doc-format-gw\"\n---\n\n# 公文写作工作流\n\n你现在要完成一项中国党政机关公文的起草与排版任务。你已安装以下 Skill，请按步骤串联使用：\n\n## 步骤 1：公文类型确定与初稿生成（获取层）\n使用 **中文公文写作技能** 完成：\n- 根据用户需求确定公文类型（通知/报告/请示/批复/函/会议纪要等 15 种法定公文）\n- 按照 GB/T 9704-2012 国家标准生成公文初稿\n- 确保公文要素齐全（发文机关/发文字号/标题/主送机关/正文/落款/日期）\n- 生成初版 Word 格式文档\n\n输出符合国家标准的公文初稿。\n\n## 步骤 2：格式规范与模板对标（分析层）\n使用 **Official Writing 公文写作技能** 完成：\n- 对照国家标准格式规范逐项检查\n- 匹配对应公文类型的标准模板\n- 运用写作技巧优化公文语言（简洁/准确/庄重/平实）\n- 确保用语规范（公文特定表述/套语/结尾格式）\n- 检查层级结构和段落逻辑\n\n输出格式规范检查结果和优化建议。\n\n## 步骤 3：素材重构与内容优化（分析层）\n使用 **体制内写作·灵析** 完成：\n- 修改模式：基于原文素材进行深度重构和优化\n- 创作模式：根据主题 + 提纲 + 过往材料生成新稿\n- 确保公文论述逻辑严密、层次分明\n- 优化公文语言风格（体制内惯用表达）\n- 输出符合 GB/T 9704-2012 标准的 Word 文档\n\n输出优化后的公文正文。\n\n## 步骤 4：格式检查与语气审查（输出层）\n使用 **Official Doc** 完成：\n- 执行公文格式全面检查（标题/正文/附件/落款）\n- 语气检查（确保庄重、严肃、得体）\n- 从模板库中匹配最佳参考范文\n- 对照通知/报告/请示/批复/会议纪要等专项要求逐一校验\n- 标注格式偏差和修改建议\n\n输出格式检查报告和语气审查结果。\n\n## 步骤 5：Word 文档生成（输出层）\n使用 **DOCX Formatter** 完成：\n- 生成符合中国公文格式规范的 Word 文档\n- 自动设置标题样式（二号方正小标宋 / 三号黑体 / 三号楷体等）\n- 正文自动排版（三号仿宋_GB2312，每行 28 字，每页 22 行）\n- 处理中文引号配对和标点规范\n- 设置页边距、行间距、页码等格式参数\n\n输出标准格式的 Word 文档。\n\n## 步骤 6：公文排版终检（输出层）\n使用 **机关公文排版skill** 完成：\n- 对最终 Word 文档进行排版终检\n- 确认页边距（上 3.7cm / 下 3.5cm / 左 2.8cm / 右 2.6cm）\n- 验证行距、字号、字体等排版参数\n- 检查版记格式（抄送/印发/日期）\n- 确保打印输出符合发文要求\n\n## 最终输出\n将以上步骤的结果整合为完整的公文写作包，交付以下文件：\n1. **公文正稿**：符合 GB/T 9704-2012 标准的 Word 文档\n2. **格式检查报告**：格式规范性和语气审查结果\n3. **公文参考模板**：同类型公文的标准范文参考",
    "skillSlugs": [
      "official-doc-writer",
      "official-writing",
      "govwriter-pro",
      "official-doc",
      "docx-formatter",
      "doc-format-gw"
    ],
    "skillCount": 6
  }
];
