type CookieOptions = {
  expires?: number | Date;
  path?: string;
  domain?: string;
};

function buildExpires(expires?: number | Date) {
  if (!expires) return "";
  const date =
    typeof expires === "number"
      ? new Date(Date.now() + expires * 24 * 60 * 60 * 1000)
      : expires;
  return `; expires=${date.toUTCString()}`;
}

function buildMeta(options?: CookieOptions) {
  if (!options) return "";
  const path = options.path ? `; path=${options.path}` : "";
  const domain = options.domain ? `; domain=${options.domain}` : "";
  return `${buildExpires(options.expires)}${path}${domain}`;
}

const Cookies = {
  get(name: string) {
    const prefix = `${encodeURIComponent(name)}=`;
    const parts = document.cookie ? document.cookie.split("; ") : [];
    for (const part of parts) {
      if (part.startsWith(prefix)) {
        return decodeURIComponent(part.slice(prefix.length));
      }
    }
    return undefined;
  },
  set(name: string, value: string, options?: CookieOptions) {
    document.cookie = `${encodeURIComponent(name)}=${encodeURIComponent(value)}${buildMeta(options)}`;
    return document.cookie;
  },
};

export default Cookies;
