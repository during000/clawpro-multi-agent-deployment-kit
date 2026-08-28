/**
 * KnowledgeBaseTab - 企业知识库列表（空白态）
 * Design: 「流动蓝图」Fluid Blueprint
 * 空状态使用 Empty 组件系列，符合 ClawPro 设计规范
 */
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription } from '@/components/ui/empty';

export default function KnowledgeBaseTab() {
  return (
    <div className="space-y-4">
      <Empty className="py-20">
        <EmptyHeader>
          <EmptyMedia />
          <EmptyTitle>暂无企业知识库</EmptyTitle>
          <EmptyDescription>
            上传企业文档、FAQ 等知识资产，构建 Agent 专属知识检索能力，提升回答准确性与业务贴合度。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  );
}
