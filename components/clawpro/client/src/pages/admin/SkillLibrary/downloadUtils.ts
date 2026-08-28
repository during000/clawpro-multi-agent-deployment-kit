/**
 * ZIP 打包下载工具函数
 * 使用 JSZip 将 Skill 文件打包为 ZIP 并触发浏览器下载
 */
import JSZip from 'jszip';
import { type Skill } from './types';

/**
 * 将 Skill 的文件打包为 ZIP 并下载
 * @param skill Skill 对象
 * @param version 可选版本号，默认使用最新版本
 */
export async function downloadSkillAsZip(skill: Skill, version?: string): Promise<void> {
  const ver = version || skill.version;
  const zip = new JSZip();

  // 将所有文件添加到 ZIP
  const files = skill.files || [];
  for (const file of files) {
    zip.file(file.name, file.content || '');
  }

  // 生成 ZIP Blob
  const blob = await zip.generateAsync({ type: 'blob' });

  // 触发下载
  const fileName = `${skill.slug}-v${ver}.zip`;
  triggerDownload(blob, fileName);
}

/**
 * 下载样例 skill-creator.zip（直接下载 public 目录下的静态文件）
 */
export function downloadSampleSkillZip(): void {
  const link = document.createElement('a');
  link.href = '/skill-creator.zip';
  link.download = 'skill-creator.zip';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/**
 * 下载样例插件 ZIP（直接下载 public 目录下的静态文件）
 */
export function downloadSamplePluginZip(): void {
  const link = document.createElement('a');
  link.href = '/system-info-plugin.zip';
  link.download = 'system-info-plugin.zip';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/**
 * 触发浏览器下载
 */
function triggerDownload(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * 比较两个 semver 版本号
 * @returns 正数表示 a > b，负数表示 a < b，0 表示相等
 */
export function compareSemver(a: string, b: string): number {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    const na = pa[i] || 0;
    const nb = pb[i] || 0;
    if (na !== nb) return na - nb;
  }
  return 0;
}

/**
 * 校验版本号格式是否符合 semver
 */
export function isValidSemver(version: string): boolean {
  return /^\d+\.\d+\.\d+$/.test(version);
}
