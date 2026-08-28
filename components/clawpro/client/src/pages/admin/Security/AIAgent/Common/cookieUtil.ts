import Cookies from '@/vendor/js-cookie';

/**
 * 获取cookie
 * @param {string} cname cookie名称
 */
export const getCookie = (name: string) => Cookies.get(name);

/**
 * 设置cookie
 * @param key
 * @param value
 */
export const setCookie = (key: string, value: string, time: number | Date) => {
  // 默认30天缓存
  time = time || 30;
  return Cookies.set(key, value, { expires: time, path: '/' });
};

export const delCookie = (key: string, value: string, time: number | Date, path: string) => {
  // 默认30天缓存
  time = time || 30;
  return Cookies.set(key, value, { expires: time, path });
};

export const setCookieByDomain = (key: string, value: string, time: number, domain: string) => {
  // 默认30天缓存
  time = time || 30;
  return Cookies.set(key, value, { expires: time, path: '/', domain });
};
