function isObjectLike(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function difference<T>(array: T[] = [], values: T[] = []) {
  const valueSet = new Set(values);
  return array.filter((item) => !valueSet.has(item));
}

function isEmpty(value: unknown) {
  if (value == null) return true;
  if (Array.isArray(value) || typeof value === "string") return value.length === 0;
  if (isObjectLike(value)) return Object.keys(value).length === 0;
  return false;
}

function cloneDeep<T>(value: T): T {
  if (typeof structuredClone === "function") {
    try {
      return structuredClone(value);
    } catch {
      // 含函数等不可结构化克隆的值时回退
    }
  }
  return JSON.parse(JSON.stringify(value));
}

function uniq<T>(array: T[] = []): T[] {
  return Array.from(new Set(array));
}

const lodash = {
  difference,
  isEmpty,
  cloneDeep,
  uniq,
};

export { difference, isEmpty, cloneDeep, uniq };
export default lodash;
