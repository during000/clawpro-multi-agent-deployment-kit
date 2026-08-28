import { requestApi } from './requestApi';

/**
 * 通过任务ID轮询下载链接
 * @link http://hpapi.atlas.oa.com/access/new/edit?a=ExportTasks&m=yunjing&env=api_test&version=2018-02-28
 * @param currentTaskId 任务id
 */
export const DescribeExportCsv = async (currentTaskId: string) => {
  try {
    const result = await requestApi({
      cmd: 'ExportTasks',
      data: {
        TaskId: String(currentTaskId),
      },
    });
    return result || {};
  } catch (error) {
    return {
      Status: 'ERROR',
      DownloadUrl: '',
    };
  }
};
