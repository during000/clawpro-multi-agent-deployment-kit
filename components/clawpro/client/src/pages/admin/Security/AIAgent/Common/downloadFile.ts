import { Base64 } from '@/vendor/js-base64';

/**
 *
 * @param fileName 文件名称
 * @param base64 base64文件流
 */
export const downloadBase64File = (fileName:any, base64:any) => {
  // 解码Base64编码的文件内容
  const decodedContent = Base64.decode(base64);
  const uint8Array = new Uint8Array(decodedContent.split('').map(char => char.charCodeAt(0)));
  const blob = new Blob([uint8Array], { type: 'application/octet-stream' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  // 设置下载的文件名（没有扩展名）
  link.download = fileName;
  // 设置下载的文件类型为二进制数据流
  link.type = 'application/octet-stream';
  link.click();
  URL.revokeObjectURL(url);
};

/**
 * 处理下载链接，是否使用 base64 下载
 * @param DownloadUrl 接口返回的下载内容
 * @param fileName 文件名称
 * @param notBase64DownloadFn 如果不是 base64 下载，
 * @param afterDownloadBase64Fn base64下载之后需要执行的内容，例如关闭弹窗等
 * @returns
 */
export const handleIfUseBase64Download = (DownloadUrl:any, fileName:any, notBase64DownloadFn:any, afterDownloadBase64Fn:any) => {
  const base64Prefix = 'data:application/octet-stream;base64,';
  if (DownloadUrl.startsWith(base64Prefix)) {
    // 如果后端返回base64则使用前端下载
    const base64 = DownloadUrl.replace(base64Prefix, '');
    downloadBase64File(fileName, base64);
    afterDownloadBase64Fn?.();
    return;
  }
  notBase64DownloadFn?.();
};
