type Unit = "d" | "day" | "days";

const PAD_2 = (value: number) => String(value).padStart(2, "0");

function cloneDate(date: Date) {
  return new Date(date.getTime());
}

function normalizeInput(input?: string | number | Date | MiniMoment | null) {
  if (input instanceof MiniMoment) {
    return input.toDate();
  }
  if (input instanceof Date) {
    return cloneDate(input);
  }
  if (typeof input === "number") {
    return new Date(input);
  }
  if (typeof input === "string") {
    if (/^\d{4}-\d{2}-\d{2} /.test(input)) {
      return new Date(input.replace(" ", "T"));
    }
    return new Date(input);
  }
  return new Date();
}

function formatDate(date: Date, pattern: string) {
  const year = date.getFullYear();
  const month = PAD_2(date.getMonth() + 1);
  const day = PAD_2(date.getDate());
  const hour = PAD_2(date.getHours());
  const minute = PAD_2(date.getMinutes());
  const second = PAD_2(date.getSeconds());

  return pattern
    .replace(/YYYY/g, String(year))
    .replace(/MM/g, month)
    .replace(/DD/g, day)
    .replace(/HH/g, hour)
    .replace(/mm/g, minute)
    .replace(/ss/g, second);
}

class MiniMoment {
  private readonly date: Date;

  constructor(input?: string | number | Date | MiniMoment | null) {
    this.date = normalizeInput(input);
  }

  clone() {
    return new MiniMoment(this.date);
  }

  toDate() {
    return cloneDate(this.date);
  }

  valueOf() {
    return this.date.getTime();
  }

  isValid() {
    return !Number.isNaN(this.date.getTime());
  }

  format(pattern: string) {
    if (!this.isValid()) return "Invalid Date";
    return formatDate(this.date, pattern);
  }

  subtract(amount: number, unit: Unit) {
    const next = cloneDate(this.date);
    if (unit === "d" || unit === "day" || unit === "days") {
      next.setDate(next.getDate() - amount);
    }
    return new MiniMoment(next);
  }

  startOf(unit: Unit) {
    const next = cloneDate(this.date);
    if (unit === "d" || unit === "day" || unit === "days") {
      next.setHours(0, 0, 0, 0);
    }
    return new MiniMoment(next);
  }

  endOf(unit: Unit) {
    const next = cloneDate(this.date);
    if (unit === "d" || unit === "day" || unit === "days") {
      next.setHours(23, 59, 59, 999);
    }
    return new MiniMoment(next);
  }
}

export type MomentInput = string | number | Date | MiniMoment | null | undefined;

export type MomentLike = MiniMoment;

export default function moment(input?: MomentInput) {
  return new MiniMoment(input);
}
