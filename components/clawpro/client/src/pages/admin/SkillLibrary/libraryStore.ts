/**
 * libraryStore - 通用资产库共享 Store 工厂
 *
 * 背景：企业技能/插件/MCP/规范等资产原本各自在 Tab 组件内维护 mock + localStorage 缓存，
 * 无法跨模块共享。「项目资产管理」需要与工具库读写同一份数据，因此将这些数据提升为
 * 跨模块共享的 Store：以 localStorage 持久化 + window CustomEvent 通知订阅者。
 *
 * 统一接口：getAll / getById / add / update / upsert / remove / replaceAll / subscribe
 */

export interface LibraryStoreOptions<T> {
  /** localStorage 数据缓存 key */
  cacheKey: string;
  /** localStorage 版本 key */
  versionKey: string;
  /** 当前缓存版本号，变更时清空旧缓存回退到初始数据 */
  version: string;
  /** 初始 mock 数据 */
  initialData: T[];
  /** 取唯一标识 */
  getId: (item: T) => string;
  /** 事件名，用于通知订阅者数据变更 */
  eventName: string;
  /** 反序列化时恢复 Date 等特殊字段（可选） */
  reviver?: (raw: any) => T;
}

export interface LibraryStore<T> {
  getAll: () => T[];
  getById: (id: string) => T | undefined;
  add: (item: T) => void;
  update: (id: string, updater: (prev: T) => T) => void;
  upsert: (item: T) => void;
  remove: (id: string) => void;
  replaceAll: (items: T[]) => void;
  subscribe: (listener: () => void) => () => void;
  eventName: string;
}

export function createLibraryStore<T>(options: LibraryStoreOptions<T>): LibraryStore<T> {
  const { cacheKey, versionKey, version, initialData, getId, eventName, reviver } = options;

  let cache: T[] | null = null;

  const load = (): T[] => {
    try {
      const cachedVersion = localStorage.getItem(versionKey);
      if (cachedVersion !== version) {
        localStorage.removeItem(cacheKey);
        localStorage.setItem(versionKey, version);
        return [...initialData];
      }
      const raw = localStorage.getItem(cacheKey);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
          return reviver ? parsed.map(reviver) : (parsed as T[]);
        }
      }
    } catch (e) {
      console.warn(`[libraryStore] 加载缓存失败 (${cacheKey}):`, e);
    }
    return [...initialData];
  };

  const persist = (items: T[]) => {
    try {
      localStorage.setItem(cacheKey, JSON.stringify(items));
      localStorage.setItem(versionKey, version);
    } catch (e) {
      console.warn(`[libraryStore] 写入缓存失败 (${cacheKey}):`, e);
    }
  };

  const ensure = (): T[] => {
    if (cache === null) cache = load();
    return cache;
  };

  const emit = () => {
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent(eventName));
    }
  };

  const commit = (items: T[]) => {
    cache = items;
    persist(items);
    emit();
  };

  return {
    eventName,
    getAll: () => [...ensure()],
    getById: (id: string) => ensure().find((item) => getId(item) === id),
    add: (item: T) => {
      commit([...ensure(), item]);
    },
    update: (id: string, updater: (prev: T) => T) => {
      const items = ensure();
      let changed = false;
      const next = items.map((item) => {
        if (getId(item) === id) {
          changed = true;
          return updater(item);
        }
        return item;
      });
      if (changed) commit(next);
    },
    upsert: (item: T) => {
      const items = ensure();
      const id = getId(item);
      const exists = items.some((it) => getId(it) === id);
      commit(exists ? items.map((it) => (getId(it) === id ? item : it)) : [...items, item]);
    },
    remove: (id: string) => {
      const items = ensure();
      const next = items.filter((item) => getId(item) !== id);
      if (next.length !== items.length) commit(next);
    },
    replaceAll: (items: T[]) => {
      commit([...items]);
    },
    subscribe: (listener: () => void) => {
      if (typeof window === 'undefined') return () => {};
      window.addEventListener(eventName, listener);
      return () => window.removeEventListener(eventName, listener);
    },
  };
}
