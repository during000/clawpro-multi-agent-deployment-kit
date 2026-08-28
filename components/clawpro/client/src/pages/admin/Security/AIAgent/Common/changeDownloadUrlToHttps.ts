// import { replaceHttpWithHttps } from '@src/utils/urlUtil';
// import { checkIsTce } from './commonUtil';

export function changeDownloadUrlToHttps(url: string) {
  // if (checkIsTce()) {
  //   return replaceHttpWithHttps(url);
  // }
  return url?.startsWith('http://') ? url.replace('http://', 'https://') : url;
}
