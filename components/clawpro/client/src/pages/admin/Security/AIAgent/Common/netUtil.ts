import { innerIpReg, ipV4Reg, ipV6Reg, REGEXP_MAP } from './regexp';

// 校验域名
export const validateDomain = (domain?: string) => {
  if (!domain) return false;
  if (REGEXP_MAP.DOMAIN_REG.test(domain)) {
    return true;
  }
  return false;
};

// 校验url域名
export const validateUrlDomain = (domain?: string) => {
  if (!domain) return false;
  if (REGEXP_MAP.OMAIN_URL_REG.test(domain)) {
    return true;
  }
  return false;
};

export const isURL = (url: any) => {
  if (url?.length > 2083) return false; // url过长
  const options = {
    strict: false,
    exact: true,
  };
  const protocol = `(?:(?:[a-z]+:)?//)${options.strict ? '' : '?'}`;
  const auth = '(?:\\S+(?::\\S*)?@)?';
  const ip = '(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]\\d|\\d)(?:\\.(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]\\d|\\d)){3}';
  const host = '(?:(?:[a-z\\u00a1-\\uffff0-9][-_]*)*[a-z\\u00a1-\\uffff0-9]+)';
  const domain = '(?:\\.(?:[a-z\\u00a1-\\uffff0-9]-*)*[a-z\\u00a1-\\uffff0-9]+)*';
  const port = '(?::\\d{2,5})?';
  const path = '(?:[/?#][^\\s"]*)?';
  const regex = `(?:${protocol}|www\\.)${auth}(?:localhost|${ip}|${host}${domain})${port}${path}`;

  return options.exact ? new RegExp(`(?:^${regex}$)`, 'i').test(url) : new RegExp(regex, 'ig').test(url);
};

export const isIpUrl = (url: string) => REGEXP_MAP.IPURL_REG.test(url);

// 验证ipv4
export function isValidIpv4(ip: string) {
  return REGEXP_MAP.IP_REG.test(ip);
}

export function isValidIpV4OrV6(ip: string) {
  return ipV4Reg.test(ip) || ipV6Reg.test(ip);
}

/**
 * @description: 校验资产公网IP
 * @param {string} value
 * @return {*}
 */
export function isValidIp(value: string) {
  if (!value) return false;
  const ip = `${value}`;
  const newIp = ip.replace(/(^\s*)|(\s*$)/g, '');
  const matchArr = newIp.match(/([0-9]{1,3}.){3}[0-9]{1,3}/);

  // if (isTCE() && value?.match(/^(vpc-).+(\|).+/)) {
  //   return true;
  // }

  if (!matchArr || !isValidIpv4(matchArr[0])) {
    return false;
  }
  // 过滤内网IP
  if (innerIpReg.test(matchArr[0])) {
    return false;
  }
  // 过滤cidr
  if (value.includes('/') && !value.includes('http')) {
    return false;
  }
  if (isIpUrl(newIp)) {
    return true;
  }
  return false;
}

/**
 * @description: 校验资产域名
 * @param {string} value
 * @return {*}
 */
export function isValidDomain(value: string) {
  if (!value) return false;
  const domain = `${value}`;
  const newUrl = domain.replace(/(^\s*)|(\s*$)/g, '');

  // if (isTCE() && value?.match(/^(vpc-).+(\|).+/)) {
  //   return true;
  // }

  if (isIpUrl(newUrl)) {
    return false;
  }

  if (validateDomain(domain)) {
    return true;
  }
  return validateUrlDomain(newUrl);
}

export function validateUrl(value: any) {
  return REGEXP_MAP.URL_REG.test(value);
}

/**
 * @description: 获取主机数目
 * @param {any[]} values
 * @return {number} 主机个数
 */
export function getDomainCount(values: any[]) {
  let count = 0;
  values.forEach(element => {
    if (isValidDomain(element)) {
      count += 1;
    }
  });
  return count;
}

/**
 * @description: 验证数字
 * @param {string} port
 * @return {*}
 */
export function validateNum(input: string | number, min: number, max: number) {
  const num = +input;
  return num >= min && num <= max && String(input) === String(num);
}

/**
 * @description: 验证端口
 * @param {string} port
 * @return {*}
 */
export function isValidPort(port: string | number) {
  return validateNum(port, 1, 65535);
}

export function isValidateWeakPwd(weakPwd: string) {
  return REGEXP_MAP.WEAkPWD_REG.test(weakPwd);
}

/**
 * 判断是否为ipv4范围
 * 1.2.3.1-1.2.3.9
 */
export function isValidIpv4Range(ip: string) {
  const reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])-(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
  return reg.test(ip);
}

/**
 * 判断是否为ipv6
 */
export function isValidIpv6(ip: string) {
  const reg = /^((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?$/;
  return reg.test(ip);
}

/**
 * ipv4 netmask
 * 1.2.3.4/24
 */
export function isValidIpv4Netmask(ip: string) {
  const reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/(\d|[1-2]\d|3[0-2])$/;
  return reg.test(ip);
}

/**
 * 判断是否为ipv6范围
 * FFEE::-FF00::
 */
export function isValidIpv6Range(ip: string) {
  // match1 field1-field2
  const exp1 = /^[0-9A-Fa-f:]+-[0-9A-Fa-f:]+$/;
  if (!exp1.test(ip)) return false;

  // split ipv6 range
  const ips = ip.split('-');
  if (!ips || ips.length !== 2) return false;

  // match ipv6
  return isValidIpv6(ips[0]) && isValidIpv6(ips[1]);
}

/**
 * ipv6 netmask
 */
export function isValidIpv6Netmask(ip: string) {
  const reg = /^((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\/(\d|[1-9]\d|[1][0-2][0-8])$/;
  return reg.test(ip);
}

export function isValidIpv4OrIpv6Range(ip: string) {
  return isValidIpv4Range(ip) || isValidIpv6Range(ip);
}

export function isValidIpv4OrIpv6Netmask(ip: string) {
  return isValidIpv4Netmask(ip) || isValidIpv6Netmask(ip);
}
