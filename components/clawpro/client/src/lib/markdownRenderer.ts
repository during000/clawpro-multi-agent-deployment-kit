import MarkdownIt from 'markdown-it';
import hljs from 'highlight.js';
import 'highlight.js/styles/atom-one-light.css';

/**
 * 判断代码块内容是否为树状目录结构
 * 包含 ├──、└──、│ 等树状字符即判定为目录树
 */
function isTreeStructure(code: string): boolean {
  return /[├└│]/.test(code);
}

/**
 * 将树状结构代码渲染为带颜色分层 span 的 HTML
 * - 连接符（├── └── │ ─）：tree-connector（蓝灰色）
 * - 括号内注释（圆括号包裹内容）：tree-comment（浅灰斜体）
 * - 其余文件/目录名：tree-name（深色加粗）
 */
function renderTreeLine(line: string): string {
  // 先转义 HTML 特殊字符
  const escaped = line
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');

  // 提取行首的连接符部分（空格 + 树状字符 + 空格）
  // 匹配：可选空格 + 树状连接符组合 + 空格
  const connectorMatch = escaped.match(/^([ \t]*[├└│─ \t]*[├└│─][ \t]+)(.*)/);
  if (!connectorMatch) {
    // 无连接符，整行作为 tree-name（如根目录行）
    return `<span class="tree-name">${escaped}</span>`;
  }

  const connector = connectorMatch[1];
  const rest = connectorMatch[2];

  // 在 rest 中识别括号注释：（...）或 (...)
  const commentMatch = rest.match(/^(.*?)(\s*[（(][^）)]*[）)].*)?$/);
  const namePart = commentMatch ? commentMatch[1] : rest;
  const commentPart = commentMatch && commentMatch[2] ? commentMatch[2] : '';

  return (
    `<span class="tree-connector">${connector}</span>` +
    `<span class="tree-name">${namePart}</span>` +
    (commentPart ? `<span class="tree-comment">${commentPart}</span>` : '')
  );
}

/**
 * 将树状结构代码块整体渲染为带分层颜色的 HTML
 */
function renderTreeBlock(code: string): string {
  const lines = code.split('\n');
  // 去掉末尾空行
  while (lines.length > 0 && lines[lines.length - 1].trim() === '') {
    lines.pop();
  }
  const renderedLines = lines.map(renderTreeLine).join('\n');
  return `<pre class="hljs tree-block"><code>${renderedLines}</code></pre>`;
}

// 创建 markdown-it 实例
const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  highlight: (str: string, lang: string): string => {
    // 树状结构优先走专用渲染器
    if (isTreeStructure(str)) {
      return renderTreeBlock(str);
    }

    if (lang && hljs.getLanguage(lang)) {
      try {
        return (
          '<pre class="hljs"><code>' +
          hljs.highlight(str, { language: lang, ignoreIllegals: true }).value +
          '</code></pre>'
        );
      } catch (__) {}
    }
    return (
      '<pre class="hljs"><code>' +
      md.utils.escapeHtml(str) +
      '</code></pre>'
    );
  },
});

/**
 * 移除 YAML frontmatter（--- 分隔的元数据）
 */
export function removeFrontmatter(content: string): string {
  const frontmatterRegex = /^---\s*\n([\s\S]*?)\n---\s*\n/;
  return content.replace(frontmatterRegex, '');
}

/**
 * 渲染 Markdown 为 HTML
 */
export function renderMarkdown(content: string): string {
  const cleanContent = removeFrontmatter(content);
  return md.render(cleanContent);
}

export default md;
