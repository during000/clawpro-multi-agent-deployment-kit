import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkFrontmatter from 'remark-frontmatter';
import rehypeRaw from 'rehype-raw';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/cjs/styles/prism';

// 覆盖 vscDarkPlus 中 code 的背景色，使每行代码背景与 pre 完全一致
const codeStyle = {
  ...vscDarkPlus,
  'code[class*="language-"]': {
    ...vscDarkPlus['code[class*="language-"]'],
    background: '#1e1e1e',
  },
  'pre[class*="language-"]': {
    ...vscDarkPlus['pre[class*="language-"]'],
    background: '#1e1e1e',
  },
};

interface MDXRendererProps {
  content: string;
}

// 自定义 Markdown 组件样式
// 标题字号整体缩小：h1 ≈ 技能名称大小(text-xl)，正文 ≈ 描述大小(text-sm)
const components = {
  h1: ({ node, ...props }: any) => (
    <h1 className="text-xl font-bold text-gray-900 mt-6 mb-3 pb-2 border-b-2 border-blue-500" {...props} />
  ),
  h2: ({ node, ...props }: any) => (
    <h2 className="text-lg font-bold text-gray-900 mt-5 mb-2" {...props} />
  ),
  h3: ({ node, ...props }: any) => (
    <h3 className="text-base font-semibold text-gray-800 mt-4 mb-2" {...props} />
  ),
  h4: ({ node, ...props }: any) => (
    <h4 className="text-sm font-semibold text-gray-800 mt-3 mb-1.5" {...props} />
  ),
  h5: ({ node, ...props }: any) => (
    <h5 className="text-sm font-semibold text-gray-700 mt-3 mb-1.5" {...props} />
  ),
  h6: ({ node, ...props }: any) => (
    <h6 className="text-xs font-semibold text-gray-700 mt-2 mb-1" {...props} />
  ),
  p: ({ node, ...props }: any) => (
    <p className="text-sm text-gray-700 leading-6 mb-3" {...props} />
  ),
  a: ({ node, ...props }: any) => (
    <a className="text-blue-600 hover:text-blue-800 underline" {...props} />
  ),
  ul: ({ node, ...props }: any) => (
    <ul className="list-disc list-inside text-sm text-gray-700 mb-3 space-y-1.5 pl-4" {...props} />
  ),
  ol: ({ node, ...props }: any) => (
    <ol className="list-decimal list-inside text-sm text-gray-700 mb-3 space-y-1.5 pl-4" {...props} />
  ),
  li: ({ node, ...props }: any) => (
    <li className="text-sm text-gray-700" {...props} />
  ),
  blockquote: ({ node, ...props }: any) => (
    <blockquote className="border-l-[3px] border-gray-200 bg-gray-50 pl-3 py-2 my-3 text-sm text-gray-600 italic rounded-r" {...props} />
  ),
  code: ({ node, className, children, ...props }: any) => {
    const match = /language-(\w+)/.exec(className || '');
    const language = match ? match[1] : '';
    
    // 判断是否为代码块：有 language class，或者父元素是 pre
    const isCodeBlock = !!language || node?.parent?.tagName === 'pre' 
      || (typeof children === 'string' && children.includes('\n'));

    if (isCodeBlock) {
      return (
        <div className="my-3 rounded-[4px] overflow-hidden">
          <SyntaxHighlighter
            style={codeStyle}
            language={language || 'text'}
            showLineNumbers={true}
            wrapLines={false}
            customStyle={{ fontSize: '0.8125rem', padding: '18px 16px', margin: 0 }}
            lineProps={{ style: { background: 'transparent' } }}
            {...props}
          >
            {String(children).replace(/\n$/, '')}
          </SyntaxHighlighter>
        </div>
      );
    }

    // 内联代码：红色文字，内嵌显示（不换行）
    return (
      <code className="bg-red-50 text-red-600 px-1.5 py-0.5 rounded text-xs font-mono border border-red-100" {...props}>
        {children}
      </code>
    );
  },
  pre: ({ node, children, ...props }: any) => {
    // code 组件已返回 <div><SyntaxHighlighter>，直接透传 children，不再套 <pre> 避免双层包裹产生色差
    const child = React.Children.toArray(children)[0] as any;
    if (child?.type === 'div' || child?.props?.className?.includes?.('rounded-[4px]')) {
      return <>{children}</>;
    }
    // 普通 pre（非代码块），保留基础样式但不设背景色
    return <pre className="rounded-[4px] overflow-x-auto mb-3 text-xs" {...props}>{children}</pre>;
  },
  table: ({ node, ...props }: any) => (
    <div className="overflow-x-auto my-3">
      <table className="w-full border-collapse border border-gray-300 text-sm" {...props} />
    </div>
  ),
  thead: ({ node, ...props }: any) => (
    <thead className="bg-gray-50 border-b border-gray-200" {...props} />
  ),
  tbody: ({ node, ...props }: any) => (
    <tbody {...props} />
  ),
  tr: ({ node, ...props }: any) => (
    <tr className="hover:bg-gray-50 transition-colors" {...props} />
  ),
  th: ({ node, ...props }: any) => (
    <th className="px-4 py-2 text-left font-semibold text-gray-900 border border-gray-300" {...props} />
  ),
  td: ({ node, ...props }: any) => (
    <td className="px-4 py-2 text-gray-700 border border-gray-300" {...props} />
  ),
  hr: ({ node, ...props }: any) => (
    <hr className="my-4 border-t border-gray-200" {...props} />
  ),
  img: ({ node, ...props }: any) => (
    <img className="max-w-full h-auto rounded-[4px] my-4 shadow-md" {...props} />
  ),
};

export default function MDXRenderer({ content }: MDXRendererProps) {
  // 移除 frontmatter（--- 开头的元数据）
  const cleanContent = content
    .replace(/^---[\s\S]*?---\n/, '')
    .trim();

  return (
    <div className="prose prose-sm max-w-none">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkFrontmatter]}
        rehypePlugins={[rehypeRaw]}
        components={components}
      >
        {cleanContent}
      </ReactMarkdown>
    </div>
  );
}
