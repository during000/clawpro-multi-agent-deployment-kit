import { jsxs as V, jsx as g, Fragment as ss } from "react/jsx-runtime";
import ei, { useRef as Rt, useEffect as Ce, useState as z } from "react";
const Pt = ({ children: e, size: t = 24, style: a = {}, ...i }) => /* @__PURE__ */ g(
  "svg",
  {
    width: t,
    height: t,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round",
    style: { flexShrink: 0, ...a },
    ...i,
    children: e
  }
), os = (e) => /* @__PURE__ */ V(Pt, { ...e, children: [
  /* @__PURE__ */ g("path", { d: "M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" }),
  /* @__PURE__ */ g("path", { d: "m15 5 4 4" })
] }), ti = (e) => /* @__PURE__ */ V(Pt, { ...e, children: [
  /* @__PURE__ */ g("path", { d: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" }),
  /* @__PURE__ */ g("polyline", { points: "7 10 12 15 17 10" }),
  /* @__PURE__ */ g("line", { x1: "12", y1: "15", x2: "12", y2: "3" })
] }), ai = (e) => /* @__PURE__ */ V(Pt, { ...e, children: [
  /* @__PURE__ */ g("polyline", { points: "3 6 5 6 21 6" }),
  /* @__PURE__ */ g("path", { d: "M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" }),
  /* @__PURE__ */ g("line", { x1: "10", y1: "11", x2: "10", y2: "17" }),
  /* @__PURE__ */ g("line", { x1: "14", y1: "11", x2: "14", y2: "17" })
] }), ue = ({ children: e, size: t = 24, style: a = {}, bgColor: i = "#e5e5e5", ...r }) => /* @__PURE__ */ g(
  "svg",
  {
    width: t,
    height: t,
    viewBox: "0 0 24 24",
    fill: "none",
    style: { flexShrink: 0, ...a },
    ...r,
    children: e
  }
), Di = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ g(ue, { size: e, style: t, children: /* @__PURE__ */ g("path", { d: "M2 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6z", fill: t.color || "#ffb020" }) }), ns = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#0abf5b", opacity: "0.15", stroke: t.color || "#0abf5b", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("circle", { cx: "9", cy: "9", r: "2", fill: t.color || "#0abf5b" }),
  /* @__PURE__ */ g("path", { d: "M3 16l4-4 3 3 4-4 7 7v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3z", fill: t.color || "#0abf5b", opacity: "0.4" })
] }), cs = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "4", width: "18", height: "16", rx: "2", fill: t.color || "#7b61ff", opacity: "0.15", stroke: t.color || "#7b61ff", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("polygon", { points: "10,8 16,12 10,16", fill: t.color || "#7b61ff" })
] }), ls = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#e34d59", opacity: "0.15", stroke: t.color || "#e34d59", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "15", textAnchor: "middle", fontSize: "7", fontWeight: "700", fill: t.color || "#e34d59", fontFamily: "sans-serif", children: "PDF" })
] }), ds = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#3370ff", opacity: "0.15", stroke: t.color || "#3370ff", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "15", textAnchor: "middle", fontSize: "7", fontWeight: "700", fill: t.color || "#3370ff", fontFamily: "sans-serif", children: "W" })
] }), hs = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#2ba471", opacity: "0.15", stroke: t.color || "#2ba471", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "15", textAnchor: "middle", fontSize: "7", fontWeight: "700", fill: t.color || "#2ba471", fontFamily: "sans-serif", children: "X" })
] }), ps = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#ed7b2f", opacity: "0.15", stroke: t.color || "#ed7b2f", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "15", textAnchor: "middle", fontSize: "7", fontWeight: "700", fill: t.color || "#ed7b2f", fontFamily: "sans-serif", children: "P" })
] }), ys = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#a0a3b1", opacity: "0.15", stroke: t.color || "#a0a3b1", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("polyline", { points: "9,9 6,12 9,15", stroke: t.color || "#a0a3b1", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round" }),
  /* @__PURE__ */ g("polyline", { points: "15,9 18,12 15,15", stroke: t.color || "#a0a3b1", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round" })
] }), us = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z", fill: t.color || "#86909c", opacity: "0.15", stroke: t.color || "#86909c", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("polyline", { points: "14,2 14,8 20,8", stroke: t.color || "#86909c", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("line", { x1: "8", y1: "13", x2: "16", y2: "13", stroke: t.color || "#86909c", strokeWidth: "1", opacity: "0.5" }),
  /* @__PURE__ */ g("line", { x1: "8", y1: "17", x2: "13", y2: "17", stroke: t.color || "#86909c", strokeWidth: "1", opacity: "0.5" })
] }), fs = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#c9a06e", opacity: "0.15", stroke: t.color || "#c9a06e", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "15", textAnchor: "middle", fontSize: "6", fontWeight: "700", fill: t.color || "#c9a06e", fontFamily: "sans-serif", children: "ZIP" })
] }), As = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("rect", { x: "3", y: "2", width: "18", height: "20", rx: "2", fill: t.color || "#e95fbc", opacity: "0.15", stroke: t.color || "#e95fbc", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("circle", { cx: "10", cy: "15", r: "2", fill: t.color || "#e95fbc" }),
  /* @__PURE__ */ g("path", { d: "M12 15V7l5-2v8", stroke: t.color || "#e95fbc", strokeWidth: "1.5", fill: "none" })
] }), Is = ({ size: e = 24, style: t = {} }) => /* @__PURE__ */ V(ue, { size: e, style: t, children: [
  /* @__PURE__ */ g("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z", fill: t.color || "#a8b0b8", opacity: "0.15", stroke: t.color || "#a8b0b8", strokeWidth: "1.5" }),
  /* @__PURE__ */ g("text", { x: "12", y: "16", textAnchor: "middle", fontSize: "9", fontWeight: "700", fill: t.color || "#a8b0b8", fontFamily: "sans-serif", children: "?" })
] }), ms = (e) => /* @__PURE__ */ V(Pt, { ...e, children: [
  /* @__PURE__ */ g("path", { d: "M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" }),
  /* @__PURE__ */ g("polyline", { points: "10 17 15 12 10 7" }),
  /* @__PURE__ */ g("line", { x1: "15", y1: "12", x2: "3", y2: "12" })
] });
function ii({
  visible: e,
  header: t,
  children: a,
  onClose: i,
  onCancel: r,
  onConfirm: s,
  confirmBtn: o = "确定",
  cancelBtn: n = "取消",
  destroyOnClose: d = !1
}) {
  const c = Rt(null);
  if (Ce(() => {
    if (!e) return;
    const p = (y) => {
      y.key === "Escape" && (i == null || i());
    };
    return document.addEventListener("keydown", p), () => document.removeEventListener("keydown", p);
  }, [e, i]), !e && d || !e) return null;
  const l = typeof o == "object" ? o.content : o, h = typeof o == "object" ? o.loading : !1;
  return /* @__PURE__ */ g("div", { className: "smh-dialog-overlay", ref: c, onClick: (p) => {
    p.target === c.current && (i == null || i());
  }, children: /* @__PURE__ */ V("div", { className: "smh-dialog", children: [
    /* @__PURE__ */ V("div", { className: "smh-dialog__header", children: [
      /* @__PURE__ */ g("span", { className: "smh-dialog__title", children: t }),
      /* @__PURE__ */ g("button", { className: "smh-dialog__close", onClick: i, children: "✕" })
    ] }),
    /* @__PURE__ */ g("div", { className: "smh-dialog__body", children: a }),
    /* @__PURE__ */ V("div", { className: "smh-dialog__footer", children: [
      /* @__PURE__ */ g("button", { className: "smh-dialog__btn smh-dialog__btn--cancel", onClick: r || i, children: n }),
      /* @__PURE__ */ V("button", { className: "smh-dialog__btn smh-dialog__btn--confirm", onClick: s, disabled: h, children: [
        h && /* @__PURE__ */ g("span", { className: "smh-dialog__spinner" }),
        l
      ] })
    ] })
  ] }) });
}
const ri = {
  confirm({ header: e, body: t, onConfirm: a, onClose: i }) {
    const r = document.createElement("div");
    document.body.appendChild(r);
    const s = () => {
      r.remove();
    }, o = document.createElement("div");
    o.className = "smh-dialog-overlay";
    const n = document.createElement("div");
    n.className = "smh-dialog";
    const d = document.createElement("div");
    d.className = "smh-dialog__header";
    const c = document.createElement("span");
    c.className = "smh-dialog__title", c.textContent = e || "确认";
    const l = document.createElement("button");
    l.className = "smh-dialog__close", l.textContent = "✕", l.onclick = () => {
      i == null || i(), s();
    }, d.appendChild(c), d.appendChild(l);
    const h = document.createElement("div");
    h.className = "smh-dialog__body", h.textContent = t || "";
    const p = document.createElement("div");
    p.className = "smh-dialog__footer";
    const y = document.createElement("button");
    y.className = "smh-dialog__btn smh-dialog__btn--cancel", y.textContent = "取消", y.onclick = () => {
      i == null || i(), s();
    };
    const u = document.createElement("button");
    u.className = "smh-dialog__btn smh-dialog__btn--confirm smh-dialog__btn--warning", u.textContent = "删除", u.onclick = () => {
      a == null || a();
    }, p.appendChild(y), p.appendChild(u), n.appendChild(d), n.appendChild(h), n.appendChild(p), o.appendChild(n), o.addEventListener("click", (I) => {
      I.target === o && (i == null || i(), s());
    });
    const A = (I) => {
      I.key === "Escape" && (i == null || i(), s(), document.removeEventListener("keydown", A));
    };
    return document.addEventListener("keydown", A), r.appendChild(o), { destroy: s };
  }
};
var gs = Object.defineProperty, bs = (e, t) => {
  for (var a in t)
    gs(e, a, { get: t[a], enumerable: !0 });
};
function Ni(e, t) {
  return function() {
    return e.apply(t, arguments);
  };
}
var { toString: vs } = Object.prototype, { getPrototypeOf: Ba } = Object, { iterator: zt, toStringTag: Ti } = Symbol, Mt = /* @__PURE__ */ ((e) => (t) => {
  const a = vs.call(t);
  return e[a] || (e[a] = a.slice(8, -1).toLowerCase());
})(/* @__PURE__ */ Object.create(null)), Ie = (e) => (e = e.toLowerCase(), (t) => Mt(t) === e), Ht = (e) => (t) => typeof t === e, { isArray: Ke } = Array, Je = Ht("undefined");
function lt(e) {
  return e !== null && !Je(e) && e.constructor !== null && !Je(e.constructor) && ce(e.constructor.isBuffer) && e.constructor.isBuffer(e);
}
var Li = Ie("ArrayBuffer");
function Ss(e) {
  let t;
  return typeof ArrayBuffer < "u" && ArrayBuffer.isView ? t = ArrayBuffer.isView(e) : t = e && e.buffer && Li(e.buffer), t;
}
var ws = Ht("string"), ce = Ht("function"), Pi = Ht("number"), dt = (e) => e !== null && typeof e == "object", Es = (e) => e === !0 || e === !1, Ct = (e) => {
  if (Mt(e) !== "object")
    return !1;
  const t = Ba(e);
  return (t === null || t === Object.prototype || Object.getPrototypeOf(t) === null) && !(Ti in e) && !(zt in e);
}, Bs = (e) => {
  if (!dt(e) || lt(e))
    return !1;
  try {
    return Object.keys(e).length === 0 && Object.getPrototypeOf(e) === Object.prototype;
  } catch {
    return !1;
  }
}, _s = Ie("Date"), Rs = Ie("File"), Cs = (e) => !!(e && typeof e.uri < "u"), Fs = (e) => e && typeof e.getParts < "u", xs = Ie("Blob"), Os = Ie("FileList"), Us = (e) => dt(e) && ce(e.pipe);
function ks() {
  return typeof globalThis < "u" ? globalThis : typeof self < "u" ? self : typeof window < "u" ? window : typeof global < "u" ? global : {};
}
var si = ks(), oi = typeof si.FormData < "u" ? si.FormData : void 0, Vs = (e) => {
  let t;
  return e && (oi && e instanceof oi || ce(e.append) && ((t = Mt(e)) === "formdata" || // detect form-data instance
  t === "object" && ce(e.toString) && e.toString() === "[object FormData]"));
}, Qs = Ie("URLSearchParams"), [Ds, Ns, Ts, Ls] = [
  "ReadableStream",
  "Request",
  "Response",
  "Headers"
].map(Ie), Ps = (e) => e.trim ? e.trim() : e.replace(/^[\s\uFEFF\xA0]+|[\s\uFEFF\xA0]+$/g, "");
function ht(e, t, { allOwnKeys: a = !1 } = {}) {
  if (e === null || typeof e > "u")
    return;
  let i, r;
  if (typeof e != "object" && (e = [e]), Ke(e))
    for (i = 0, r = e.length; i < r; i++)
      t.call(null, e[i], i, e);
  else {
    if (lt(e))
      return;
    const s = a ? Object.getOwnPropertyNames(e) : Object.keys(e), o = s.length;
    let n;
    for (i = 0; i < o; i++)
      n = s[i], t.call(null, e[n], n, e);
  }
}
function zi(e, t) {
  if (lt(e))
    return null;
  t = t.toLowerCase();
  const a = Object.keys(e);
  let i = a.length, r;
  for (; i-- > 0; )
    if (r = a[i], t === r.toLowerCase())
      return r;
  return null;
}
var De = typeof globalThis < "u" ? globalThis : typeof self < "u" ? self : typeof window < "u" ? window : global, Mi = (e) => !Je(e) && e !== De;
function ya() {
  const { caseless: e, skipUndefined: t } = Mi(this) && this || {}, a = {}, i = (r, s) => {
    if (s === "__proto__" || s === "constructor" || s === "prototype")
      return;
    const o = e && zi(a, s) || s;
    Ct(a[o]) && Ct(r) ? a[o] = ya(a[o], r) : Ct(r) ? a[o] = ya({}, r) : Ke(r) ? a[o] = r.slice() : (!t || !Je(r)) && (a[o] = r);
  };
  for (let r = 0, s = arguments.length; r < s; r++)
    arguments[r] && ht(arguments[r], i);
  return a;
}
var zs = (e, t, a, { allOwnKeys: i } = {}) => (ht(
  t,
  (r, s) => {
    a && ce(r) ? Object.defineProperty(e, s, {
      value: Ni(r, a),
      writable: !0,
      enumerable: !0,
      configurable: !0
    }) : Object.defineProperty(e, s, {
      value: r,
      writable: !0,
      enumerable: !0,
      configurable: !0
    });
  },
  { allOwnKeys: i }
), e), Ms = (e) => (e.charCodeAt(0) === 65279 && (e = e.slice(1)), e), Hs = (e, t, a, i) => {
  e.prototype = Object.create(t.prototype, i), Object.defineProperty(e.prototype, "constructor", {
    value: e,
    writable: !0,
    enumerable: !1,
    configurable: !0
  }), Object.defineProperty(e, "super", {
    value: t.prototype
  }), a && Object.assign(e.prototype, a);
}, $s = (e, t, a, i) => {
  let r, s, o;
  const n = {};
  if (t = t || {}, e == null) return t;
  do {
    for (r = Object.getOwnPropertyNames(e), s = r.length; s-- > 0; )
      o = r[s], (!i || i(o, e, t)) && !n[o] && (t[o] = e[o], n[o] = !0);
    e = a !== !1 && Ba(e);
  } while (e && (!a || a(e, t)) && e !== Object.prototype);
  return t;
}, js = (e, t, a) => {
  e = String(e), (a === void 0 || a > e.length) && (a = e.length), a -= t.length;
  const i = e.indexOf(t, a);
  return i !== -1 && i === a;
}, Gs = (e) => {
  if (!e) return null;
  if (Ke(e)) return e;
  let t = e.length;
  if (!Pi(t)) return null;
  const a = new Array(t);
  for (; t-- > 0; )
    a[t] = e[t];
  return a;
}, Js = /* @__PURE__ */ ((e) => (t) => e && t instanceof e)(typeof Uint8Array < "u" && Ba(Uint8Array)), Ks = (e, t) => {
  const i = (e && e[zt]).call(e);
  let r;
  for (; (r = i.next()) && !r.done; ) {
    const s = r.value;
    t.call(e, s[0], s[1]);
  }
}, Xs = (e, t) => {
  let a;
  const i = [];
  for (; (a = e.exec(t)) !== null; )
    i.push(a);
  return i;
}, Ws = Ie("HTMLFormElement"), Zs = (e) => e.toLowerCase().replace(/[-_\s]([a-z\d])(\w*)/g, function(a, i, r) {
  return i.toUpperCase() + r;
}), ni = (({ hasOwnProperty: e }) => (t, a) => e.call(t, a))(Object.prototype), Ys = Ie("RegExp"), Hi = (e, t) => {
  const a = Object.getOwnPropertyDescriptors(e), i = {};
  ht(a, (r, s) => {
    let o;
    (o = t(r, s, e)) !== !1 && (i[s] = o || r);
  }), Object.defineProperties(e, i);
}, qs = (e) => {
  Hi(e, (t, a) => {
    if (ce(e) && ["arguments", "caller", "callee"].indexOf(a) !== -1)
      return !1;
    const i = e[a];
    if (ce(i)) {
      if (t.enumerable = !1, "writable" in t) {
        t.writable = !1;
        return;
      }
      t.set || (t.set = () => {
        throw Error("Can not rewrite read-only method '" + a + "'");
      });
    }
  });
}, eo = (e, t) => {
  const a = {}, i = (r) => {
    r.forEach((s) => {
      a[s] = !0;
    });
  };
  return Ke(e) ? i(e) : i(String(e).split(t)), a;
}, to = () => {
}, ao = (e, t) => e != null && Number.isFinite(e = +e) ? e : t;
function io(e) {
  return !!(e && ce(e.append) && e[Ti] === "FormData" && e[zt]);
}
var ro = (e) => {
  const t = new Array(10), a = (i, r) => {
    if (dt(i)) {
      if (t.indexOf(i) >= 0)
        return;
      if (lt(i))
        return i;
      if (!("toJSON" in i)) {
        t[r] = i;
        const s = Ke(i) ? [] : {};
        return ht(i, (o, n) => {
          const d = a(o, r + 1);
          !Je(d) && (s[n] = d);
        }), t[r] = void 0, s;
      }
    }
    return i;
  };
  return a(e, 0);
}, so = Ie("AsyncFunction"), oo = (e) => e && (dt(e) || ce(e)) && ce(e.then) && ce(e.catch), $i = ((e, t) => e ? setImmediate : t ? ((a, i) => (De.addEventListener(
  "message",
  ({ source: r, data: s }) => {
    r === De && s === a && i.length && i.shift()();
  },
  !1
), (r) => {
  i.push(r), De.postMessage(a, "*");
}))(`axios@${Math.random()}`, []) : (a) => setTimeout(a))(typeof setImmediate == "function", ce(De.postMessage)), no = typeof queueMicrotask < "u" ? queueMicrotask.bind(De) : typeof process < "u" && process.nextTick || $i, co = (e) => e != null && ce(e[zt]), v = {
  isArray: Ke,
  isArrayBuffer: Li,
  isBuffer: lt,
  isFormData: Vs,
  isArrayBufferView: Ss,
  isString: ws,
  isNumber: Pi,
  isBoolean: Es,
  isObject: dt,
  isPlainObject: Ct,
  isEmptyObject: Bs,
  isReadableStream: Ds,
  isRequest: Ns,
  isResponse: Ts,
  isHeaders: Ls,
  isUndefined: Je,
  isDate: _s,
  isFile: Rs,
  isReactNativeBlob: Cs,
  isReactNative: Fs,
  isBlob: xs,
  isRegExp: Ys,
  isFunction: ce,
  isStream: Us,
  isURLSearchParams: Qs,
  isTypedArray: Js,
  isFileList: Os,
  forEach: ht,
  merge: ya,
  extend: zs,
  trim: Ps,
  stripBOM: Ms,
  inherits: Hs,
  toFlatObject: $s,
  kindOf: Mt,
  kindOfTest: Ie,
  endsWith: js,
  toArray: Gs,
  forEachEntry: Ks,
  matchAll: Xs,
  isHTMLForm: Ws,
  hasOwnProperty: ni,
  hasOwnProp: ni,
  // an alias to avoid ESLint no-prototype-builtins detection
  reduceDescriptors: Hi,
  freezeMethods: qs,
  toObjectSet: eo,
  toCamelCase: Zs,
  noop: to,
  toFiniteNumber: ao,
  findKey: zi,
  global: De,
  isContextDefined: Mi,
  isSpecCompliantForm: io,
  toJSONObject: ro,
  isAsyncFn: so,
  isThenable: oo,
  setImmediate: $i,
  asap: no,
  isIterable: co
}, de = class ji extends Error {
  static from(t, a, i, r, s, o) {
    const n = new ji(t.message, a || t.code, i, r, s);
    return n.cause = t, n.name = t.name, t.status != null && n.status == null && (n.status = t.status), o && Object.assign(n, o), n;
  }
  /**
   * Create an Error with the specified message, config, error code, request and response.
   *
   * @param {string} message The error message.
   * @param {string} [code] The error code (for example, 'ECONNABORTED').
   * @param {Object} [config] The config.
   * @param {Object} [request] The request.
   * @param {Object} [response] The response.
   *
   * @returns {Error} The created error.
   */
  constructor(t, a, i, r, s) {
    super(t), Object.defineProperty(this, "message", {
      value: t,
      enumerable: !0,
      writable: !0,
      configurable: !0
    }), this.name = "AxiosError", this.isAxiosError = !0, a && (this.code = a), i && (this.config = i), r && (this.request = r), s && (this.response = s, this.status = s.status);
  }
  toJSON() {
    return {
      // Standard
      message: this.message,
      name: this.name,
      // Microsoft
      description: this.description,
      number: this.number,
      // Mozilla
      fileName: this.fileName,
      lineNumber: this.lineNumber,
      columnNumber: this.columnNumber,
      stack: this.stack,
      // Axios
      config: v.toJSONObject(this.config),
      code: this.code,
      status: this.status
    };
  }
};
de.ERR_BAD_OPTION_VALUE = "ERR_BAD_OPTION_VALUE";
de.ERR_BAD_OPTION = "ERR_BAD_OPTION";
de.ECONNABORTED = "ECONNABORTED";
de.ETIMEDOUT = "ETIMEDOUT";
de.ERR_NETWORK = "ERR_NETWORK";
de.ERR_FR_TOO_MANY_REDIRECTS = "ERR_FR_TOO_MANY_REDIRECTS";
de.ERR_DEPRECATED = "ERR_DEPRECATED";
de.ERR_BAD_RESPONSE = "ERR_BAD_RESPONSE";
de.ERR_BAD_REQUEST = "ERR_BAD_REQUEST";
de.ERR_CANCELED = "ERR_CANCELED";
de.ERR_NOT_SUPPORT = "ERR_NOT_SUPPORT";
de.ERR_INVALID_URL = "ERR_INVALID_URL";
var M = de, lo = null;
function ua(e) {
  return v.isPlainObject(e) || v.isArray(e);
}
function Gi(e) {
  return v.endsWith(e, "[]") ? e.slice(0, -2) : e;
}
function ta(e, t, a) {
  return e ? e.concat(t).map(function(r, s) {
    return r = Gi(r), !a && s ? "[" + r + "]" : r;
  }).join(a ? "." : "") : t;
}
function ho(e) {
  return v.isArray(e) && !e.some(ua);
}
var po = v.toFlatObject(v, {}, null, function(t) {
  return /^is[A-Z]/.test(t);
});
function yo(e, t, a) {
  if (!v.isObject(e))
    throw new TypeError("target must be an object");
  t = t || new FormData(), a = v.toFlatObject(
    a,
    {
      metaTokens: !0,
      dots: !1,
      indexes: !1
    },
    !1,
    function(A, I) {
      return !v.isUndefined(I[A]);
    }
  );
  const i = a.metaTokens, r = a.visitor || l, s = a.dots, o = a.indexes, d = (a.Blob || typeof Blob < "u" && Blob) && v.isSpecCompliantForm(t);
  if (!v.isFunction(r))
    throw new TypeError("visitor must be a function");
  function c(u) {
    if (u === null) return "";
    if (v.isDate(u))
      return u.toISOString();
    if (v.isBoolean(u))
      return u.toString();
    if (!d && v.isBlob(u))
      throw new M("Blob is not supported. Use a Buffer instead.");
    return v.isArrayBuffer(u) || v.isTypedArray(u) ? d && typeof Blob == "function" ? new Blob([u]) : Buffer.from(u) : u;
  }
  function l(u, A, I) {
    let b = u;
    if (v.isReactNative(t) && v.isReactNativeBlob(u))
      return t.append(ta(I, A, s), c(u)), !1;
    if (u && !I && typeof u == "object") {
      if (v.endsWith(A, "{}"))
        A = i ? A : A.slice(0, -2), u = JSON.stringify(u);
      else if (v.isArray(u) && ho(u) || (v.isFileList(u) || v.endsWith(A, "[]")) && (b = v.toArray(u)))
        return A = Gi(A), b.forEach(function(w, R) {
          !(v.isUndefined(w) || w === null) && t.append(
            // eslint-disable-next-line no-nested-ternary
            o === !0 ? ta([A], R, s) : o === null ? A : A + "[]",
            c(w)
          );
        }), !1;
    }
    return ua(u) ? !0 : (t.append(ta(I, A, s), c(u)), !1);
  }
  const h = [], p = Object.assign(po, {
    defaultVisitor: l,
    convertValue: c,
    isVisitable: ua
  });
  function y(u, A) {
    if (!v.isUndefined(u)) {
      if (h.indexOf(u) !== -1)
        throw Error("Circular reference detected in " + A.join("."));
      h.push(u), v.forEach(u, function(b, S) {
        (!(v.isUndefined(b) || b === null) && r.call(t, b, v.isString(S) ? S.trim() : S, A, p)) === !0 && y(b, A ? A.concat(S) : [S]);
      }), h.pop();
    }
  }
  if (!v.isObject(e))
    throw new TypeError("data must be an object");
  return y(e), t;
}
var $t = yo;
function ci(e) {
  const t = {
    "!": "%21",
    "'": "%27",
    "(": "%28",
    ")": "%29",
    "~": "%7E",
    "%20": "+",
    "%00": "\0"
  };
  return encodeURIComponent(e).replace(/[!'()~]|%20|%00/g, function(i) {
    return t[i];
  });
}
function Ji(e, t) {
  this._pairs = [], e && $t(e, this, t);
}
var Ki = Ji.prototype;
Ki.append = function(t, a) {
  this._pairs.push([t, a]);
};
Ki.toString = function(t) {
  const a = t ? function(i) {
    return t.call(this, i, ci);
  } : ci;
  return this._pairs.map(function(r) {
    return a(r[0]) + "=" + a(r[1]);
  }, "").join("&");
};
var Xi = Ji;
function uo(e) {
  return encodeURIComponent(e).replace(/%3A/gi, ":").replace(/%24/g, "$").replace(/%2C/gi, ",").replace(/%20/g, "+");
}
function Wi(e, t, a) {
  if (!t)
    return e;
  const i = a && a.encode || uo, r = v.isFunction(a) ? {
    serialize: a
  } : a, s = r && r.serialize;
  let o;
  if (s ? o = s(t, r) : o = v.isURLSearchParams(t) ? t.toString() : new Xi(t, r).toString(i), o) {
    const n = e.indexOf("#");
    n !== -1 && (e = e.slice(0, n)), e += (e.indexOf("?") === -1 ? "?" : "&") + o;
  }
  return e;
}
var fo = class {
  constructor() {
    this.handlers = [];
  }
  /**
   * Add a new interceptor to the stack
   *
   * @param {Function} fulfilled The function to handle `then` for a `Promise`
   * @param {Function} rejected The function to handle `reject` for a `Promise`
   * @param {Object} options The options for the interceptor, synchronous and runWhen
   *
   * @return {Number} An ID used to remove interceptor later
   */
  use(e, t, a) {
    return this.handlers.push({
      fulfilled: e,
      rejected: t,
      synchronous: a ? a.synchronous : !1,
      runWhen: a ? a.runWhen : null
    }), this.handlers.length - 1;
  }
  /**
   * Remove an interceptor from the stack
   *
   * @param {Number} id The ID that was returned by `use`
   *
   * @returns {void}
   */
  eject(e) {
    this.handlers[e] && (this.handlers[e] = null);
  }
  /**
   * Clear all interceptors from the stack
   *
   * @returns {void}
   */
  clear() {
    this.handlers && (this.handlers = []);
  }
  /**
   * Iterate over all the registered interceptors
   *
   * This method is particularly useful for skipping over any
   * interceptors that may have become `null` calling `eject`.
   *
   * @param {Function} fn The function to call for each interceptor
   *
   * @returns {void}
   */
  forEach(e) {
    v.forEach(this.handlers, function(a) {
      a !== null && e(a);
    });
  }
}, li = fo, _a = {
  silentJSONParsing: !0,
  forcedJSONParsing: !0,
  clarifyTimeoutError: !1,
  legacyInterceptorReqResOrdering: !0
}, Ao = typeof URLSearchParams < "u" ? URLSearchParams : Xi, Io = typeof FormData < "u" ? FormData : null, mo = typeof Blob < "u" ? Blob : null, go = {
  isBrowser: !0,
  classes: {
    URLSearchParams: Ao,
    FormData: Io,
    Blob: mo
  },
  protocols: ["http", "https", "file", "blob", "url", "data"]
}, Zi = {};
bs(Zi, {
  hasBrowserEnv: () => Ra,
  hasStandardBrowserEnv: () => bo,
  hasStandardBrowserWebWorkerEnv: () => vo,
  navigator: () => fa,
  origin: () => So
});
var Ra = typeof window < "u" && typeof document < "u", fa = typeof navigator == "object" && navigator || void 0, bo = Ra && (!fa || ["ReactNative", "NativeScript", "NS"].indexOf(fa.product) < 0), vo = typeof WorkerGlobalScope < "u" && // eslint-disable-next-line no-undef
self instanceof WorkerGlobalScope && typeof self.importScripts == "function", So = Ra && window.location.href || "http://localhost", re = {
  ...Zi,
  ...go
};
function wo(e, t) {
  return $t(e, new re.classes.URLSearchParams(), {
    visitor: function(a, i, r, s) {
      return re.isNode && v.isBuffer(a) ? (this.append(i, a.toString("base64")), !1) : s.defaultVisitor.apply(this, arguments);
    },
    ...t
  });
}
function Eo(e) {
  return v.matchAll(/\w+|\[(\w*)]/g, e).map((t) => t[0] === "[]" ? "" : t[1] || t[0]);
}
function Bo(e) {
  const t = {}, a = Object.keys(e);
  let i;
  const r = a.length;
  let s;
  for (i = 0; i < r; i++)
    s = a[i], t[s] = e[s];
  return t;
}
function _o(e) {
  function t(a, i, r, s) {
    let o = a[s++];
    if (o === "__proto__") return !0;
    const n = Number.isFinite(+o), d = s >= a.length;
    return o = !o && v.isArray(r) ? r.length : o, d ? (v.hasOwnProp(r, o) ? r[o] = [r[o], i] : r[o] = i, !n) : ((!r[o] || !v.isObject(r[o])) && (r[o] = []), t(a, i, r[o], s) && v.isArray(r[o]) && (r[o] = Bo(r[o])), !n);
  }
  if (v.isFormData(e) && v.isFunction(e.entries)) {
    const a = {};
    return v.forEachEntry(e, (i, r) => {
      t(Eo(i), r, a, 0);
    }), a;
  }
  return null;
}
var Yi = _o;
function Ro(e, t, a) {
  if (v.isString(e))
    try {
      return (t || JSON.parse)(e), v.trim(e);
    } catch (i) {
      if (i.name !== "SyntaxError")
        throw i;
    }
  return (a || JSON.stringify)(e);
}
var Ca = {
  transitional: _a,
  adapter: ["xhr", "http", "fetch"],
  transformRequest: [
    function(t, a) {
      const i = a.getContentType() || "", r = i.indexOf("application/json") > -1, s = v.isObject(t);
      if (s && v.isHTMLForm(t) && (t = new FormData(t)), v.isFormData(t))
        return r ? JSON.stringify(Yi(t)) : t;
      if (v.isArrayBuffer(t) || v.isBuffer(t) || v.isStream(t) || v.isFile(t) || v.isBlob(t) || v.isReadableStream(t))
        return t;
      if (v.isArrayBufferView(t))
        return t.buffer;
      if (v.isURLSearchParams(t))
        return a.setContentType("application/x-www-form-urlencoded;charset=utf-8", !1), t.toString();
      let n;
      if (s) {
        if (i.indexOf("application/x-www-form-urlencoded") > -1)
          return wo(t, this.formSerializer).toString();
        if ((n = v.isFileList(t)) || i.indexOf("multipart/form-data") > -1) {
          const d = this.env && this.env.FormData;
          return $t(
            n ? { "files[]": t } : t,
            d && new d(),
            this.formSerializer
          );
        }
      }
      return s || r ? (a.setContentType("application/json", !1), Ro(t)) : t;
    }
  ],
  transformResponse: [
    function(t) {
      const a = this.transitional || Ca.transitional, i = a && a.forcedJSONParsing, r = this.responseType === "json";
      if (v.isResponse(t) || v.isReadableStream(t))
        return t;
      if (t && v.isString(t) && (i && !this.responseType || r)) {
        const o = !(a && a.silentJSONParsing) && r;
        try {
          return JSON.parse(t, this.parseReviver);
        } catch (n) {
          if (o)
            throw n.name === "SyntaxError" ? M.from(n, M.ERR_BAD_RESPONSE, this, null, this.response) : n;
        }
      }
      return t;
    }
  ],
  /**
   * A timeout in milliseconds to abort a request. If set to 0 (default) a
   * timeout is not created.
   */
  timeout: 0,
  xsrfCookieName: "XSRF-TOKEN",
  xsrfHeaderName: "X-XSRF-TOKEN",
  maxContentLength: -1,
  maxBodyLength: -1,
  env: {
    FormData: re.classes.FormData,
    Blob: re.classes.Blob
  },
  validateStatus: function(t) {
    return t >= 200 && t < 300;
  },
  headers: {
    common: {
      Accept: "application/json, text/plain, */*",
      "Content-Type": void 0
    }
  }
};
v.forEach(["delete", "get", "head", "post", "put", "patch"], (e) => {
  Ca.headers[e] = {};
});
var Fa = Ca, Co = v.toObjectSet([
  "age",
  "authorization",
  "content-length",
  "content-type",
  "etag",
  "expires",
  "from",
  "host",
  "if-modified-since",
  "if-unmodified-since",
  "last-modified",
  "location",
  "max-forwards",
  "proxy-authorization",
  "referer",
  "retry-after",
  "user-agent"
]), Fo = (e) => {
  const t = {};
  let a, i, r;
  return e && e.split(`
`).forEach(function(o) {
    r = o.indexOf(":"), a = o.substring(0, r).trim().toLowerCase(), i = o.substring(r + 1).trim(), !(!a || t[a] && Co[a]) && (a === "set-cookie" ? t[a] ? t[a].push(i) : t[a] = [i] : t[a] = t[a] ? t[a] + ", " + i : i);
  }), t;
}, di = /* @__PURE__ */ Symbol("internals");
function at(e) {
  return e && String(e).trim().toLowerCase();
}
function Ft(e) {
  return e === !1 || e == null ? e : v.isArray(e) ? e.map(Ft) : String(e);
}
function xo(e) {
  const t = /* @__PURE__ */ Object.create(null), a = /([^\s,;=]+)\s*(?:=\s*([^,;]+))?/g;
  let i;
  for (; i = a.exec(e); )
    t[i[1]] = i[2];
  return t;
}
var Oo = (e) => /^[-_a-zA-Z0-9^`|~,!#$%&'*+.]+$/.test(e.trim());
function aa(e, t, a, i, r) {
  if (v.isFunction(i))
    return i.call(this, t, a);
  if (r && (t = a), !!v.isString(t)) {
    if (v.isString(i))
      return t.indexOf(i) !== -1;
    if (v.isRegExp(i))
      return i.test(t);
  }
}
function Uo(e) {
  return e.trim().toLowerCase().replace(/([a-z\d])(\w*)/g, (t, a, i) => a.toUpperCase() + i);
}
function ko(e, t) {
  const a = v.toCamelCase(" " + t);
  ["get", "set", "has"].forEach((i) => {
    Object.defineProperty(e, i + a, {
      value: function(r, s, o) {
        return this[i].call(this, t, r, s, o);
      },
      configurable: !0
    });
  });
}
var jt = class {
  constructor(e) {
    e && this.set(e);
  }
  set(e, t, a) {
    const i = this;
    function r(o, n, d) {
      const c = at(n);
      if (!c)
        throw new Error("header name must be a non-empty string");
      const l = v.findKey(i, c);
      (!l || i[l] === void 0 || d === !0 || d === void 0 && i[l] !== !1) && (i[l || n] = Ft(o));
    }
    const s = (o, n) => v.forEach(o, (d, c) => r(d, c, n));
    if (v.isPlainObject(e) || e instanceof this.constructor)
      s(e, t);
    else if (v.isString(e) && (e = e.trim()) && !Oo(e))
      s(Fo(e), t);
    else if (v.isObject(e) && v.isIterable(e)) {
      let o = {}, n, d;
      for (const c of e) {
        if (!v.isArray(c))
          throw TypeError("Object iterator must return a key-value pair");
        o[d = c[0]] = (n = o[d]) ? v.isArray(n) ? [...n, c[1]] : [n, c[1]] : c[1];
      }
      s(o, t);
    } else
      e != null && r(t, e, a);
    return this;
  }
  get(e, t) {
    if (e = at(e), e) {
      const a = v.findKey(this, e);
      if (a) {
        const i = this[a];
        if (!t)
          return i;
        if (t === !0)
          return xo(i);
        if (v.isFunction(t))
          return t.call(this, i, a);
        if (v.isRegExp(t))
          return t.exec(i);
        throw new TypeError("parser must be boolean|regexp|function");
      }
    }
  }
  has(e, t) {
    if (e = at(e), e) {
      const a = v.findKey(this, e);
      return !!(a && this[a] !== void 0 && (!t || aa(this, this[a], a, t)));
    }
    return !1;
  }
  delete(e, t) {
    const a = this;
    let i = !1;
    function r(s) {
      if (s = at(s), s) {
        const o = v.findKey(a, s);
        o && (!t || aa(a, a[o], o, t)) && (delete a[o], i = !0);
      }
    }
    return v.isArray(e) ? e.forEach(r) : r(e), i;
  }
  clear(e) {
    const t = Object.keys(this);
    let a = t.length, i = !1;
    for (; a--; ) {
      const r = t[a];
      (!e || aa(this, this[r], r, e, !0)) && (delete this[r], i = !0);
    }
    return i;
  }
  normalize(e) {
    const t = this, a = {};
    return v.forEach(this, (i, r) => {
      const s = v.findKey(a, r);
      if (s) {
        t[s] = Ft(i), delete t[r];
        return;
      }
      const o = e ? Uo(r) : String(r).trim();
      o !== r && delete t[r], t[o] = Ft(i), a[o] = !0;
    }), this;
  }
  concat(...e) {
    return this.constructor.concat(this, ...e);
  }
  toJSON(e) {
    const t = /* @__PURE__ */ Object.create(null);
    return v.forEach(this, (a, i) => {
      a != null && a !== !1 && (t[i] = e && v.isArray(a) ? a.join(", ") : a);
    }), t;
  }
  [Symbol.iterator]() {
    return Object.entries(this.toJSON())[Symbol.iterator]();
  }
  toString() {
    return Object.entries(this.toJSON()).map(([e, t]) => e + ": " + t).join(`
`);
  }
  getSetCookie() {
    return this.get("set-cookie") || [];
  }
  get [Symbol.toStringTag]() {
    return "AxiosHeaders";
  }
  static from(e) {
    return e instanceof this ? e : new this(e);
  }
  static concat(e, ...t) {
    const a = new this(e);
    return t.forEach((i) => a.set(i)), a;
  }
  static accessor(e) {
    const a = (this[di] = this[di] = {
      accessors: {}
    }).accessors, i = this.prototype;
    function r(s) {
      const o = at(s);
      a[o] || (ko(i, s), a[o] = !0);
    }
    return v.isArray(e) ? e.forEach(r) : r(e), this;
  }
};
jt.accessor([
  "Content-Type",
  "Content-Length",
  "Accept",
  "Accept-Encoding",
  "User-Agent",
  "Authorization"
]);
v.reduceDescriptors(jt.prototype, ({ value: e }, t) => {
  let a = t[0].toUpperCase() + t.slice(1);
  return {
    get: () => e,
    set(i) {
      this[a] = i;
    }
  };
});
v.freezeMethods(jt);
var Ae = jt;
function ia(e, t) {
  const a = this || Fa, i = t || a, r = Ae.from(i.headers);
  let s = i.data;
  return v.forEach(e, function(n) {
    s = n.call(a, s, r.normalize(), t ? t.status : void 0);
  }), r.normalize(), s;
}
function qi(e) {
  return !!(e && e.__CANCEL__);
}
var Vo = class extends M {
  /**
   * A `CanceledError` is an object that is thrown when an operation is canceled.
   *
   * @param {string=} message The message.
   * @param {Object=} config The config.
   * @param {Object=} request The request.
   *
   * @returns {CanceledError} The created error.
   */
  constructor(e, t, a) {
    super(e ?? "canceled", M.ERR_CANCELED, t, a), this.name = "CanceledError", this.__CANCEL__ = !0;
  }
}, pt = Vo;
function er(e, t, a) {
  const i = a.config.validateStatus;
  !a.status || !i || i(a.status) ? e(a) : t(
    new M(
      "Request failed with status code " + a.status,
      [M.ERR_BAD_REQUEST, M.ERR_BAD_RESPONSE][Math.floor(a.status / 100) - 4],
      a.config,
      a.request,
      a
    )
  );
}
function Qo(e) {
  const t = /^([-+\w]{1,25})(:?\/\/|:)/.exec(e);
  return t && t[1] || "";
}
function Do(e, t) {
  e = e || 10;
  const a = new Array(e), i = new Array(e);
  let r = 0, s = 0, o;
  return t = t !== void 0 ? t : 1e3, function(d) {
    const c = Date.now(), l = i[s];
    o || (o = c), a[r] = d, i[r] = c;
    let h = s, p = 0;
    for (; h !== r; )
      p += a[h++], h = h % e;
    if (r = (r + 1) % e, r === s && (s = (s + 1) % e), c - o < t)
      return;
    const y = l && c - l;
    return y ? Math.round(p * 1e3 / y) : void 0;
  };
}
var No = Do;
function To(e, t) {
  let a = 0, i = 1e3 / t, r, s;
  const o = (c, l = Date.now()) => {
    a = l, r = null, s && (clearTimeout(s), s = null), e(...c);
  };
  return [(...c) => {
    const l = Date.now(), h = l - a;
    h >= i ? o(c, l) : (r = c, s || (s = setTimeout(() => {
      s = null, o(r);
    }, i - h)));
  }, () => r && o(r)];
}
var Lo = To, Vt = (e, t, a = 3) => {
  let i = 0;
  const r = No(50, 250);
  return Lo((s) => {
    const o = s.loaded, n = s.lengthComputable ? s.total : void 0, d = o - i, c = r(d), l = o <= n;
    i = o;
    const h = {
      loaded: o,
      total: n,
      progress: n ? o / n : void 0,
      bytes: d,
      rate: c || void 0,
      estimated: c && n && l ? (n - o) / c : void 0,
      event: s,
      lengthComputable: n != null,
      [t ? "download" : "upload"]: !0
    };
    e(h);
  }, a);
}, hi = (e, t) => {
  const a = e != null;
  return [
    (i) => t[0]({
      lengthComputable: a,
      total: e,
      loaded: i
    }),
    t[1]
  ];
}, pi = (e) => (...t) => v.asap(() => e(...t)), Po = re.hasStandardBrowserEnv ? /* @__PURE__ */ ((e, t) => (a) => (a = new URL(a, re.origin), e.protocol === a.protocol && e.host === a.host && (t || e.port === a.port)))(
  new URL(re.origin),
  re.navigator && /(msie|trident)/i.test(re.navigator.userAgent)
) : () => !0, zo = re.hasStandardBrowserEnv ? (
  // Standard browser envs support document.cookie
  {
    write(e, t, a, i, r, s, o) {
      if (typeof document > "u") return;
      const n = [`${e}=${encodeURIComponent(t)}`];
      v.isNumber(a) && n.push(`expires=${new Date(a).toUTCString()}`), v.isString(i) && n.push(`path=${i}`), v.isString(r) && n.push(`domain=${r}`), s === !0 && n.push("secure"), v.isString(o) && n.push(`SameSite=${o}`), document.cookie = n.join("; ");
    },
    read(e) {
      if (typeof document > "u") return null;
      const t = document.cookie.match(new RegExp("(?:^|; )" + e + "=([^;]*)"));
      return t ? decodeURIComponent(t[1]) : null;
    },
    remove(e) {
      this.write(e, "", Date.now() - 864e5, "/");
    }
  }
) : (
  // Non-standard browser env (web workers, react-native) lack needed support.
  {
    write() {
    },
    read() {
      return null;
    },
    remove() {
    }
  }
);
function Mo(e) {
  return typeof e != "string" ? !1 : /^([a-z][a-z\d+\-.]*:)?\/\//i.test(e);
}
function Ho(e, t) {
  return t ? e.replace(/\/?\/$/, "") + "/" + t.replace(/^\/+/, "") : e;
}
function tr(e, t, a) {
  let i = !Mo(t);
  return e && (i || a == !1) ? Ho(e, t) : t;
}
var yi = (e) => e instanceof Ae ? { ...e } : e;
function Ne(e, t) {
  t = t || {};
  const a = {};
  function i(c, l, h, p) {
    return v.isPlainObject(c) && v.isPlainObject(l) ? v.merge.call({ caseless: p }, c, l) : v.isPlainObject(l) ? v.merge({}, l) : v.isArray(l) ? l.slice() : l;
  }
  function r(c, l, h, p) {
    if (v.isUndefined(l)) {
      if (!v.isUndefined(c))
        return i(void 0, c, h, p);
    } else return i(c, l, h, p);
  }
  function s(c, l) {
    if (!v.isUndefined(l))
      return i(void 0, l);
  }
  function o(c, l) {
    if (v.isUndefined(l)) {
      if (!v.isUndefined(c))
        return i(void 0, c);
    } else return i(void 0, l);
  }
  function n(c, l, h) {
    if (h in t)
      return i(c, l);
    if (h in e)
      return i(void 0, c);
  }
  const d = {
    url: s,
    method: s,
    data: s,
    baseURL: o,
    transformRequest: o,
    transformResponse: o,
    paramsSerializer: o,
    timeout: o,
    timeoutMessage: o,
    withCredentials: o,
    withXSRFToken: o,
    adapter: o,
    responseType: o,
    xsrfCookieName: o,
    xsrfHeaderName: o,
    onUploadProgress: o,
    onDownloadProgress: o,
    decompress: o,
    maxContentLength: o,
    maxBodyLength: o,
    beforeRedirect: o,
    transport: o,
    httpAgent: o,
    httpsAgent: o,
    cancelToken: o,
    socketPath: o,
    responseEncoding: o,
    validateStatus: n,
    headers: (c, l, h) => r(yi(c), yi(l), h, !0)
  };
  return v.forEach(Object.keys({ ...e, ...t }), function(l) {
    if (l === "__proto__" || l === "constructor" || l === "prototype") return;
    const h = v.hasOwnProp(d, l) ? d[l] : r, p = h(e[l], t[l], l);
    v.isUndefined(p) && h !== n || (a[l] = p);
  }), a;
}
var ar = (e) => {
  const t = Ne({}, e);
  let { data: a, withXSRFToken: i, xsrfHeaderName: r, xsrfCookieName: s, headers: o, auth: n } = t;
  if (t.headers = o = Ae.from(o), t.url = Wi(
    tr(t.baseURL, t.url, t.allowAbsoluteUrls),
    e.params,
    e.paramsSerializer
  ), n && o.set(
    "Authorization",
    "Basic " + btoa(
      (n.username || "") + ":" + (n.password ? unescape(encodeURIComponent(n.password)) : "")
    )
  ), v.isFormData(a)) {
    if (re.hasStandardBrowserEnv || re.hasStandardBrowserWebWorkerEnv)
      o.setContentType(void 0);
    else if (v.isFunction(a.getHeaders)) {
      const d = a.getHeaders(), c = ["content-type", "content-length"];
      Object.entries(d).forEach(([l, h]) => {
        c.includes(l.toLowerCase()) && o.set(l, h);
      });
    }
  }
  if (re.hasStandardBrowserEnv && (i && v.isFunction(i) && (i = i(t)), i || i !== !1 && Po(t.url))) {
    const d = r && s && zo.read(s);
    d && o.set(r, d);
  }
  return t;
}, $o = typeof XMLHttpRequest < "u", jo = $o && function(e) {
  return new Promise(function(a, i) {
    const r = ar(e);
    let s = r.data;
    const o = Ae.from(r.headers).normalize();
    let { responseType: n, onUploadProgress: d, onDownloadProgress: c } = r, l, h, p, y, u;
    function A() {
      y && y(), u && u(), r.cancelToken && r.cancelToken.unsubscribe(l), r.signal && r.signal.removeEventListener("abort", l);
    }
    let I = new XMLHttpRequest();
    I.open(r.method.toUpperCase(), r.url, !0), I.timeout = r.timeout;
    function b() {
      if (!I)
        return;
      const w = Ae.from(
        "getAllResponseHeaders" in I && I.getAllResponseHeaders()
      ), P = {
        data: !n || n === "text" || n === "json" ? I.responseText : I.response,
        status: I.status,
        statusText: I.statusText,
        headers: w,
        config: e,
        request: I
      };
      er(
        function(T) {
          a(T), A();
        },
        function(T) {
          i(T), A();
        },
        P
      ), I = null;
    }
    "onloadend" in I ? I.onloadend = b : I.onreadystatechange = function() {
      !I || I.readyState !== 4 || I.status === 0 && !(I.responseURL && I.responseURL.indexOf("file:") === 0) || setTimeout(b);
    }, I.onabort = function() {
      I && (i(new M("Request aborted", M.ECONNABORTED, e, I)), I = null);
    }, I.onerror = function(R) {
      const P = R && R.message ? R.message : "Network Error", B = new M(P, M.ERR_NETWORK, e, I);
      B.event = R || null, i(B), I = null;
    }, I.ontimeout = function() {
      let R = r.timeout ? "timeout of " + r.timeout + "ms exceeded" : "timeout exceeded";
      const P = r.transitional || _a;
      r.timeoutErrorMessage && (R = r.timeoutErrorMessage), i(
        new M(
          R,
          P.clarifyTimeoutError ? M.ETIMEDOUT : M.ECONNABORTED,
          e,
          I
        )
      ), I = null;
    }, s === void 0 && o.setContentType(null), "setRequestHeader" in I && v.forEach(o.toJSON(), function(R, P) {
      I.setRequestHeader(P, R);
    }), v.isUndefined(r.withCredentials) || (I.withCredentials = !!r.withCredentials), n && n !== "json" && (I.responseType = r.responseType), c && ([p, u] = Vt(c, !0), I.addEventListener("progress", p)), d && I.upload && ([h, y] = Vt(d), I.upload.addEventListener("progress", h), I.upload.addEventListener("loadend", y)), (r.cancelToken || r.signal) && (l = (w) => {
      I && (i(!w || w.type ? new pt(null, e, I) : w), I.abort(), I = null);
    }, r.cancelToken && r.cancelToken.subscribe(l), r.signal && (r.signal.aborted ? l() : r.signal.addEventListener("abort", l)));
    const S = Qo(r.url);
    if (S && re.protocols.indexOf(S) === -1) {
      i(
        new M(
          "Unsupported protocol " + S + ":",
          M.ERR_BAD_REQUEST,
          e
        )
      );
      return;
    }
    I.send(s || null);
  });
}, Go = (e, t) => {
  const { length: a } = e = e ? e.filter(Boolean) : [];
  if (t || a) {
    let i = new AbortController(), r;
    const s = function(c) {
      if (!r) {
        r = !0, n();
        const l = c instanceof Error ? c : this.reason;
        i.abort(
          l instanceof M ? l : new pt(l instanceof Error ? l.message : l)
        );
      }
    };
    let o = t && setTimeout(() => {
      o = null, s(new M(`timeout of ${t}ms exceeded`, M.ETIMEDOUT));
    }, t);
    const n = () => {
      e && (o && clearTimeout(o), o = null, e.forEach((c) => {
        c.unsubscribe ? c.unsubscribe(s) : c.removeEventListener("abort", s);
      }), e = null);
    };
    e.forEach((c) => c.addEventListener("abort", s));
    const { signal: d } = i;
    return d.unsubscribe = () => v.asap(n), d;
  }
}, Jo = Go, Ko = function* (e, t) {
  let a = e.byteLength;
  if (a < t) {
    yield e;
    return;
  }
  let i = 0, r;
  for (; i < a; )
    r = i + t, yield e.slice(i, r), i = r;
}, Xo = async function* (e, t) {
  for await (const a of Wo(e))
    yield* Ko(a, t);
}, Wo = async function* (e) {
  if (e[Symbol.asyncIterator]) {
    yield* e;
    return;
  }
  const t = e.getReader();
  try {
    for (; ; ) {
      const { done: a, value: i } = await t.read();
      if (a)
        break;
      yield i;
    }
  } finally {
    await t.cancel();
  }
}, ui = (e, t, a, i) => {
  const r = Xo(e, t);
  let s = 0, o, n = (d) => {
    o || (o = !0, i && i(d));
  };
  return new ReadableStream(
    {
      async pull(d) {
        try {
          const { done: c, value: l } = await r.next();
          if (c) {
            n(), d.close();
            return;
          }
          let h = l.byteLength;
          if (a) {
            let p = s += h;
            a(p);
          }
          d.enqueue(new Uint8Array(l));
        } catch (c) {
          throw n(c), c;
        }
      },
      cancel(d) {
        return n(d), r.return();
      }
    },
    {
      highWaterMark: 2
    }
  );
}, fi = 64 * 1024, { isFunction: bt } = v, Zo = (({ Request: e, Response: t }) => ({
  Request: e,
  Response: t
}))(v.global), { ReadableStream: Ai, TextEncoder: Ii } = v.global, mi = (e, ...t) => {
  try {
    return !!e(...t);
  } catch {
    return !1;
  }
}, Yo = (e) => {
  e = v.merge.call(
    {
      skipUndefined: !0
    },
    Zo,
    e
  );
  const { fetch: t, Request: a, Response: i } = e, r = t ? bt(t) : typeof fetch == "function", s = bt(a), o = bt(i);
  if (!r)
    return !1;
  const n = r && bt(Ai), d = r && (typeof Ii == "function" ? /* @__PURE__ */ ((u) => (A) => u.encode(A))(new Ii()) : async (u) => new Uint8Array(await new a(u).arrayBuffer())), c = s && n && mi(() => {
    let u = !1;
    const A = new a(re.origin, {
      body: new Ai(),
      method: "POST",
      get duplex() {
        return u = !0, "half";
      }
    }).headers.has("Content-Type");
    return u && !A;
  }), l = o && n && mi(() => v.isReadableStream(new i("").body)), h = {
    stream: l && ((u) => u.body)
  };
  r && ["text", "arrayBuffer", "blob", "formData", "stream"].forEach((u) => {
    !h[u] && (h[u] = (A, I) => {
      let b = A && A[u];
      if (b)
        return b.call(A);
      throw new M(
        `Response type '${u}' is not supported`,
        M.ERR_NOT_SUPPORT,
        I
      );
    });
  });
  const p = async (u) => {
    if (u == null)
      return 0;
    if (v.isBlob(u))
      return u.size;
    if (v.isSpecCompliantForm(u))
      return (await new a(re.origin, {
        method: "POST",
        body: u
      }).arrayBuffer()).byteLength;
    if (v.isArrayBufferView(u) || v.isArrayBuffer(u))
      return u.byteLength;
    if (v.isURLSearchParams(u) && (u = u + ""), v.isString(u))
      return (await d(u)).byteLength;
  }, y = async (u, A) => {
    const I = v.toFiniteNumber(u.getContentLength());
    return I ?? p(A);
  };
  return async (u) => {
    let {
      url: A,
      method: I,
      data: b,
      signal: S,
      cancelToken: w,
      timeout: R,
      onDownloadProgress: P,
      onUploadProgress: B,
      responseType: T,
      headers: j,
      withCredentials: q = "same-origin",
      fetchOptions: H
    } = ar(u), Oe = t || fetch;
    T = T ? (T + "").toLowerCase() : "text";
    let Te = Jo(
      [S, w && w.toAbortSignal()],
      R
    ), we = null;
    const Ee = Te && Te.unsubscribe && (() => {
      Te.unsubscribe();
    });
    let Ue;
    try {
      if (B && c && I !== "get" && I !== "head" && (Ue = await y(j, b)) !== 0) {
        let pe = new a(A, {
          method: "POST",
          body: b,
          duplex: "half"
        }), ke;
        if (v.isFormData(b) && (ke = pe.headers.get("content-type")) && j.setContentType(ke), pe.body) {
          const [Fe, We] = hi(
            Ue,
            Vt(pi(B))
          );
          b = ui(pe.body, fi, Fe, We);
        }
      }
      v.isString(q) || (q = q ? "include" : "omit");
      const Z = s && "credentials" in a.prototype, Zt = {
        ...H,
        signal: Te,
        method: I.toUpperCase(),
        headers: j.normalize().toJSON(),
        body: b,
        duplex: "half",
        credentials: Z ? q : void 0
      };
      we = s && new a(A, Zt);
      let fe = await (s ? Oe(we, H) : Oe(A, Zt));
      const yt = l && (T === "stream" || T === "response");
      if (l && (P || yt && Ee)) {
        const pe = {};
        ["status", "statusText", "headers"].forEach((Ze) => {
          pe[Ze] = fe[Ze];
        });
        const ke = v.toFiniteNumber(fe.headers.get("content-length")), [Fe, We] = P && hi(
          ke,
          Vt(pi(P), !0)
        ) || [];
        fe = new i(
          ui(fe.body, fi, Fe, () => {
            We && We(), Ee && Ee();
          }),
          pe
        );
      }
      T = T || "text";
      let Xe = await h[v.findKey(h, T) || "text"](
        fe,
        u
      );
      return !yt && Ee && Ee(), await new Promise((pe, ke) => {
        er(pe, ke, {
          data: Xe,
          headers: Ae.from(fe.headers),
          status: fe.status,
          statusText: fe.statusText,
          config: u,
          request: we
        });
      });
    } catch (Z) {
      throw Ee && Ee(), Z && Z.name === "TypeError" && /Load failed|fetch/i.test(Z.message) ? Object.assign(
        new M(
          "Network Error",
          M.ERR_NETWORK,
          u,
          we,
          Z && Z.response
        ),
        {
          cause: Z.cause || Z
        }
      ) : M.from(Z, Z && Z.code, u, we, Z && Z.response);
    }
  };
}, qo = /* @__PURE__ */ new Map(), ir = (e) => {
  let t = e && e.env || {};
  const { fetch: a, Request: i, Response: r } = t, s = [i, r, a];
  let o = s.length, n = o, d, c, l = qo;
  for (; n--; )
    d = s[n], c = l.get(d), c === void 0 && l.set(d, c = n ? /* @__PURE__ */ new Map() : Yo(t)), l = c;
  return c;
};
ir();
var xa = {
  http: lo,
  xhr: jo,
  fetch: {
    get: ir
  }
};
v.forEach(xa, (e, t) => {
  if (e) {
    try {
      Object.defineProperty(e, "name", { value: t });
    } catch {
    }
    Object.defineProperty(e, "adapterName", { value: t });
  }
});
var gi = (e) => `- ${e}`, en = (e) => v.isFunction(e) || e === null || e === !1;
function tn(e, t) {
  e = v.isArray(e) ? e : [e];
  const { length: a } = e;
  let i, r;
  const s = {};
  for (let o = 0; o < a; o++) {
    i = e[o];
    let n;
    if (r = i, !en(i) && (r = xa[(n = String(i)).toLowerCase()], r === void 0))
      throw new M(`Unknown adapter '${n}'`);
    if (r && (v.isFunction(r) || (r = r.get(t))))
      break;
    s[n || "#" + o] = r;
  }
  if (!r) {
    const o = Object.entries(s).map(
      ([d, c]) => `adapter ${d} ` + (c === !1 ? "is not supported by the environment" : "is not available in the build")
    );
    let n = a ? o.length > 1 ? `since :
` + o.map(gi).join(`
`) : " " + gi(o[0]) : "as no adapter specified";
    throw new M(
      "There is no suitable adapter to dispatch the request " + n,
      "ERR_NOT_SUPPORT"
    );
  }
  return r;
}
var rr = {
  /**
   * Resolve an adapter from a list of adapter names or functions.
   * @type {Function}
   */
  getAdapter: tn,
  /**
   * Exposes all known adapters
   * @type {Object<string, Function|Object>}
   */
  adapters: xa
};
function ra(e) {
  if (e.cancelToken && e.cancelToken.throwIfRequested(), e.signal && e.signal.aborted)
    throw new pt(null, e);
}
function bi(e) {
  return ra(e), e.headers = Ae.from(e.headers), e.data = ia.call(e, e.transformRequest), ["post", "put", "patch"].indexOf(e.method) !== -1 && e.headers.setContentType("application/x-www-form-urlencoded", !1), rr.getAdapter(e.adapter || Fa.adapter, e)(e).then(
    function(i) {
      return ra(e), i.data = ia.call(e, e.transformResponse, i), i.headers = Ae.from(i.headers), i;
    },
    function(i) {
      return qi(i) || (ra(e), i && i.response && (i.response.data = ia.call(
        e,
        e.transformResponse,
        i.response
      ), i.response.headers = Ae.from(i.response.headers))), Promise.reject(i);
    }
  );
}
var sr = "1.13.6", Gt = {};
["object", "boolean", "number", "function", "string", "symbol"].forEach((e, t) => {
  Gt[e] = function(i) {
    return typeof i === e || "a" + (t < 1 ? "n " : " ") + e;
  };
});
var vi = {};
Gt.transitional = function(t, a, i) {
  function r(s, o) {
    return "[Axios v" + sr + "] Transitional option '" + s + "'" + o + (i ? ". " + i : "");
  }
  return (s, o, n) => {
    if (t === !1)
      throw new M(
        r(o, " has been removed" + (a ? " in " + a : "")),
        M.ERR_DEPRECATED
      );
    return a && !vi[o] && (vi[o] = !0, console.warn(
      r(
        o,
        " has been deprecated since v" + a + " and will be removed in the near future"
      )
    )), t ? t(s, o, n) : !0;
  };
};
Gt.spelling = function(t) {
  return (a, i) => (console.warn(`${i} is likely a misspelling of ${t}`), !0);
};
function an(e, t, a) {
  if (typeof e != "object")
    throw new M("options must be an object", M.ERR_BAD_OPTION_VALUE);
  const i = Object.keys(e);
  let r = i.length;
  for (; r-- > 0; ) {
    const s = i[r], o = t[s];
    if (o) {
      const n = e[s], d = n === void 0 || o(n, s, e);
      if (d !== !0)
        throw new M(
          "option " + s + " must be " + d,
          M.ERR_BAD_OPTION_VALUE
        );
      continue;
    }
    if (a !== !0)
      throw new M("Unknown option " + s, M.ERR_BAD_OPTION);
  }
}
var xt = {
  assertOptions: an,
  validators: Gt
}, ye = xt.validators, Qt = class {
  constructor(e) {
    this.defaults = e || {}, this.interceptors = {
      request: new li(),
      response: new li()
    };
  }
  /**
   * Dispatch a request
   *
   * @param {String|Object} configOrUrl The config specific for this request (merged with this.defaults)
   * @param {?Object} config
   *
   * @returns {Promise} The Promise to be fulfilled
   */
  async request(e, t) {
    try {
      return await this._request(e, t);
    } catch (a) {
      if (a instanceof Error) {
        let i = {};
        Error.captureStackTrace ? Error.captureStackTrace(i) : i = new Error();
        const r = i.stack ? i.stack.replace(/^.+\n/, "") : "";
        try {
          a.stack ? r && !String(a.stack).endsWith(r.replace(/^.+\n.+\n/, "")) && (a.stack += `
` + r) : a.stack = r;
        } catch {
        }
      }
      throw a;
    }
  }
  _request(e, t) {
    typeof e == "string" ? (t = t || {}, t.url = e) : t = e || {}, t = Ne(this.defaults, t);
    const { transitional: a, paramsSerializer: i, headers: r } = t;
    a !== void 0 && xt.assertOptions(
      a,
      {
        silentJSONParsing: ye.transitional(ye.boolean),
        forcedJSONParsing: ye.transitional(ye.boolean),
        clarifyTimeoutError: ye.transitional(ye.boolean),
        legacyInterceptorReqResOrdering: ye.transitional(ye.boolean)
      },
      !1
    ), i != null && (v.isFunction(i) ? t.paramsSerializer = {
      serialize: i
    } : xt.assertOptions(
      i,
      {
        encode: ye.function,
        serialize: ye.function
      },
      !0
    )), t.allowAbsoluteUrls !== void 0 || (this.defaults.allowAbsoluteUrls !== void 0 ? t.allowAbsoluteUrls = this.defaults.allowAbsoluteUrls : t.allowAbsoluteUrls = !0), xt.assertOptions(
      t,
      {
        baseUrl: ye.spelling("baseURL"),
        withXsrfToken: ye.spelling("withXSRFToken")
      },
      !0
    ), t.method = (t.method || this.defaults.method || "get").toLowerCase();
    let s = r && v.merge(r.common, r[t.method]);
    r && v.forEach(["delete", "get", "head", "post", "put", "patch", "common"], (y) => {
      delete r[y];
    }), t.headers = Ae.concat(s, r);
    const o = [];
    let n = !0;
    this.interceptors.request.forEach(function(u) {
      if (typeof u.runWhen == "function" && u.runWhen(t) === !1)
        return;
      n = n && u.synchronous;
      const A = t.transitional || _a;
      A && A.legacyInterceptorReqResOrdering ? o.unshift(u.fulfilled, u.rejected) : o.push(u.fulfilled, u.rejected);
    });
    const d = [];
    this.interceptors.response.forEach(function(u) {
      d.push(u.fulfilled, u.rejected);
    });
    let c, l = 0, h;
    if (!n) {
      const y = [bi.bind(this), void 0];
      for (y.unshift(...o), y.push(...d), h = y.length, c = Promise.resolve(t); l < h; )
        c = c.then(y[l++], y[l++]);
      return c;
    }
    h = o.length;
    let p = t;
    for (; l < h; ) {
      const y = o[l++], u = o[l++];
      try {
        p = y(p);
      } catch (A) {
        u.call(this, A);
        break;
      }
    }
    try {
      c = bi.call(this, p);
    } catch (y) {
      return Promise.reject(y);
    }
    for (l = 0, h = d.length; l < h; )
      c = c.then(d[l++], d[l++]);
    return c;
  }
  getUri(e) {
    e = Ne(this.defaults, e);
    const t = tr(e.baseURL, e.url, e.allowAbsoluteUrls);
    return Wi(t, e.params, e.paramsSerializer);
  }
};
v.forEach(["delete", "get", "head", "options"], function(t) {
  Qt.prototype[t] = function(a, i) {
    return this.request(
      Ne(i || {}, {
        method: t,
        url: a,
        data: (i || {}).data
      })
    );
  };
});
v.forEach(["post", "put", "patch"], function(t) {
  function a(i) {
    return function(s, o, n) {
      return this.request(
        Ne(n || {}, {
          method: t,
          headers: i ? {
            "Content-Type": "multipart/form-data"
          } : {},
          url: s,
          data: o
        })
      );
    };
  }
  Qt.prototype[t] = a(), Qt.prototype[t + "Form"] = a(!0);
});
var Ot = Qt, rn = class or {
  constructor(t) {
    if (typeof t != "function")
      throw new TypeError("executor must be a function.");
    let a;
    this.promise = new Promise(function(s) {
      a = s;
    });
    const i = this;
    this.promise.then((r) => {
      if (!i._listeners) return;
      let s = i._listeners.length;
      for (; s-- > 0; )
        i._listeners[s](r);
      i._listeners = null;
    }), this.promise.then = (r) => {
      let s;
      const o = new Promise((n) => {
        i.subscribe(n), s = n;
      }).then(r);
      return o.cancel = function() {
        i.unsubscribe(s);
      }, o;
    }, t(function(s, o, n) {
      i.reason || (i.reason = new pt(s, o, n), a(i.reason));
    });
  }
  /**
   * Throws a `CanceledError` if cancellation has been requested.
   */
  throwIfRequested() {
    if (this.reason)
      throw this.reason;
  }
  /**
   * Subscribe to the cancel signal
   */
  subscribe(t) {
    if (this.reason) {
      t(this.reason);
      return;
    }
    this._listeners ? this._listeners.push(t) : this._listeners = [t];
  }
  /**
   * Unsubscribe from the cancel signal
   */
  unsubscribe(t) {
    if (!this._listeners)
      return;
    const a = this._listeners.indexOf(t);
    a !== -1 && this._listeners.splice(a, 1);
  }
  toAbortSignal() {
    const t = new AbortController(), a = (i) => {
      t.abort(i);
    };
    return this.subscribe(a), t.signal.unsubscribe = () => this.unsubscribe(a), t.signal;
  }
  /**
   * Returns an object that contains a new `CancelToken` and a function that, when called,
   * cancels the `CancelToken`.
   */
  static source() {
    let t;
    return {
      token: new or(function(r) {
        t = r;
      }),
      cancel: t
    };
  }
}, sn = rn;
function on(e) {
  return function(a) {
    return e.apply(null, a);
  };
}
function nn(e) {
  return v.isObject(e) && e.isAxiosError === !0;
}
var Aa = {
  Continue: 100,
  SwitchingProtocols: 101,
  Processing: 102,
  EarlyHints: 103,
  Ok: 200,
  Created: 201,
  Accepted: 202,
  NonAuthoritativeInformation: 203,
  NoContent: 204,
  ResetContent: 205,
  PartialContent: 206,
  MultiStatus: 207,
  AlreadyReported: 208,
  ImUsed: 226,
  MultipleChoices: 300,
  MovedPermanently: 301,
  Found: 302,
  SeeOther: 303,
  NotModified: 304,
  UseProxy: 305,
  Unused: 306,
  TemporaryRedirect: 307,
  PermanentRedirect: 308,
  BadRequest: 400,
  Unauthorized: 401,
  PaymentRequired: 402,
  Forbidden: 403,
  NotFound: 404,
  MethodNotAllowed: 405,
  NotAcceptable: 406,
  ProxyAuthenticationRequired: 407,
  RequestTimeout: 408,
  Conflict: 409,
  Gone: 410,
  LengthRequired: 411,
  PreconditionFailed: 412,
  PayloadTooLarge: 413,
  UriTooLong: 414,
  UnsupportedMediaType: 415,
  RangeNotSatisfiable: 416,
  ExpectationFailed: 417,
  ImATeapot: 418,
  MisdirectedRequest: 421,
  UnprocessableEntity: 422,
  Locked: 423,
  FailedDependency: 424,
  TooEarly: 425,
  UpgradeRequired: 426,
  PreconditionRequired: 428,
  TooManyRequests: 429,
  RequestHeaderFieldsTooLarge: 431,
  UnavailableForLegalReasons: 451,
  InternalServerError: 500,
  NotImplemented: 501,
  BadGateway: 502,
  ServiceUnavailable: 503,
  GatewayTimeout: 504,
  HttpVersionNotSupported: 505,
  VariantAlsoNegotiates: 506,
  InsufficientStorage: 507,
  LoopDetected: 508,
  NotExtended: 510,
  NetworkAuthenticationRequired: 511,
  WebServerIsDown: 521,
  ConnectionTimedOut: 522,
  OriginIsUnreachable: 523,
  TimeoutOccurred: 524,
  SslHandshakeFailed: 525,
  InvalidSslCertificate: 526
};
Object.entries(Aa).forEach(([e, t]) => {
  Aa[t] = e;
});
var cn = Aa;
function nr(e) {
  const t = new Ot(e), a = Ni(Ot.prototype.request, t);
  return v.extend(a, Ot.prototype, t, { allOwnKeys: !0 }), v.extend(a, t, null, { allOwnKeys: !0 }), a.create = function(r) {
    return nr(Ne(e, r));
  }, a;
}
var W = nr(Fa);
W.Axios = Ot;
W.CanceledError = pt;
W.CancelToken = sn;
W.isCancel = qi;
W.VERSION = sr;
W.toFormData = $t;
W.AxiosError = M;
W.Cancel = W.CanceledError;
W.all = function(t) {
  return Promise.all(t);
};
W.spread = on;
W.isAxiosError = nn;
W.mergeConfig = Ne;
W.AxiosHeaders = Ae;
W.formToJSON = (e) => Yi(v.isHTMLForm(e) ? new FormData(e) : e);
W.getAdapter = rr.getAdapter;
W.HttpStatusCode = cn;
W.default = W;
var _ = W, {
  Axios: Rc,
  AxiosError: Cc,
  CanceledError: Fc,
  isCancel: xc,
  CancelToken: Oc,
  VERSION: Uc,
  all: kc,
  Cancel: Vc,
  isAxiosError: Qc,
  spread: Dc,
  toFormData: Nc,
  AxiosHeaders: Tc,
  HttpStatusCode: Lc,
  formToJSON: Pc,
  getAdapter: zc,
  mergeConfig: Mc
} = _, C = "https://api.tencentsmh.cn".replace(/\/+$/, ""), he = class {
  constructor(e, t = C, a = _) {
    this.basePath = t, this.axios = a, e && (this.configuration = e, this.basePath = e.basePath ?? t);
  }
}, ln = class extends Error {
  constructor(e, t) {
    super(t), this.field = e, this.name = "RequiredError";
  }
}, F = {}, x = "https://example.com", f = function(e, t, a) {
  if (a == null)
    throw new ln(t, `Required parameter ${t} was null or undefined when calling ${e}.`);
};
function Ia(e, t, a = "") {
  t != null && (typeof t == "object" ? Array.isArray(t) ? t.forEach((i) => Ia(e, i, a)) : Object.keys(t).forEach(
    (i) => Ia(e, t[i], `${a}${a !== "" ? "." : ""}${i}`)
  ) : e.has(a) ? e.append(a, t) : e.set(a, t));
}
var O = function(e, ...t) {
  const a = new URLSearchParams(e.search);
  Ia(a, t), e.search = a.toString();
}, $ = function(e, t, a) {
  const i = typeof e != "string";
  return (i && a && a.isJsonMime ? a.isJsonMime(t.headers["Content-Type"]) : i) ? JSON.stringify(e !== void 0 ? e : {}) : e || "";
}, U = function(e) {
  return e.pathname + e.search + e.hash;
}, k = function(e, t, a, i) {
  return (r = t, s = a) => {
    const o = { ...e.options, url: (r.defaults.baseURL ? "" : (i == null ? void 0 : i.basePath) ?? s) + e.url };
    return r.request(o);
  };
}, dn = function(e) {
  return {
    /**
     * 用于批量复制目录或文件
     * @summary 批量复制目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchCopyCopyEnum} copy 开启批量复制操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchCopyRequestInner>} batchCopyRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    batchCopy: async (t, a, i, r, s, o, n = {}) => {
      f("batchCopy", "libraryId", t), f("batchCopy", "spaceId", a), f("batchCopy", "copy", i), f("batchCopy", "accessToken", r), f("batchCopy", "batchCopyRequest", s);
      const d = "/api/v1/batch/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      i !== void 0 && (y.copy = i), r !== void 0 && (y.access_token = r), o !== void 0 && (y.user_id = o), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(s, h, e), {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于批量删除目录或文件
     * @summary 批量删除目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchDeleteDeleteEnum} _delete 开启批量删除操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchDeleteRequestInner>} batchDeleteRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    batchDelete: async (t, a, i, r, s, o, n = {}) => {
      f("batchDelete", "libraryId", t), f("batchDelete", "spaceId", a), f("batchDelete", "_delete", i), f("batchDelete", "accessToken", r), f("batchDelete", "batchDeleteRequest", s);
      const d = "/api/v1/batch/{LibraryId}/{SpaceId}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      i !== void 0 && (y.delete = i), r !== void 0 && (y.access_token = r), o !== void 0 && (y.user_id = o), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(s, h, e), {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于批量重命名或移动目录或文件
     * @summary 批量重命名或移动目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchMoveMoveEnum} move 开启批量重命名或移动操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchMoveRequestInner>} batchMoveRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    batchMove: async (t, a, i, r, s, o, n = {}) => {
      f("batchMove", "libraryId", t), f("batchMove", "spaceId", a), f("batchMove", "move", i), f("batchMove", "accessToken", r), f("batchMove", "batchMoveRequest", s);
      const d = "/api/v1/batch/{LibraryId}/{SpaceId}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      i !== void 0 && (y.move = i), r !== void 0 && (y.access_token = r), o !== void 0 && (y.user_id = o), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(s, h, e), {
        url: U(c),
        options: h
      };
    }
  };
}, sa = function(e) {
  const t = dn(e);
  return {
    /**
     * 用于批量复制目录或文件
     * @summary 批量复制目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchCopyCopyEnum} copy 开启批量复制操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchCopyRequestInner>} batchCopyRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async batchCopy(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.batchCopy(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["BatchApi.batchCopy"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于批量删除目录或文件
     * @summary 批量删除目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchDeleteDeleteEnum} _delete 开启批量删除操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchDeleteRequestInner>} batchDeleteRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async batchDelete(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.batchDelete(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["BatchApi.batchDelete"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于批量重命名或移动目录或文件
     * @summary 批量重命名或移动目录或文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {BatchMoveMoveEnum} move 开启批量重命名或移动操作
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<BatchMoveRequestInner>} batchMoveRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async batchMove(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.batchMove(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["BatchApi.batchMove"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    }
  };
}, hn = class extends he {
  /**
   * 用于批量复制目录或文件
   * @summary 批量复制目录或文件
   * @param {BatchApiBatchCopyRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  batchCopy(e, t) {
    return sa(this.configuration).batchCopy(e.libraryId, e.spaceId, e.copy, e.accessToken, e.batchCopyRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于批量删除目录或文件
   * @summary 批量删除目录或文件
   * @param {BatchApiBatchDeleteRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  batchDelete(e, t) {
    return sa(this.configuration).batchDelete(e.libraryId, e.spaceId, e._delete, e.accessToken, e.batchDeleteRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于批量重命名或移动目录或文件
   * @summary 批量重命名或移动目录或文件
   * @param {BatchApiBatchMoveRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  batchMove(e, t) {
    return sa(this.configuration).batchMove(e.libraryId, e.spaceId, e.move, e.accessToken, e.batchMoveRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
}, pn = function(e) {
  return {
    /**
     * 用于检查目录或相簿状态
     * @summary 检查目录或相簿状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    checkDirectoryStatus: async (t, a, i, r, s, o = {}) => {
      f("checkDirectoryStatus", "libraryId", t), f("checkDirectoryStatus", "spaceId", a), f("checkDirectoryStatus", "filePath", i);
      const n = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "HEAD", ...c, ...o }, h = {}, p = {};
      r !== void 0 && (p.access_token = r), s !== void 0 && (p.user_id = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于复制目录或相簿。 - 自动创建中间所需的各级父目录。 
     * @summary 复制目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CopyDirectoryRequest} copyDirectoryRequest 
     * @param {CopyDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    copyDirectory: async (t, a, i, r, s, o, n, d = {}) => {
      f("copyDirectory", "libraryId", t), f("copyDirectory", "spaceId", a), f("copyDirectory", "filePath", i), f("copyDirectory", "accessToken", r), f("copyDirectory", "copyDirectoryRequest", s);
      const c = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "PUT", ...h, ...d }, y = {}, u = {};
      o !== void 0 && (u.conflict_resolution_strategy = o), r !== void 0 && (u.access_token = r), n !== void 0 && (u.user_id = n), y["Content-Type"] = "application/json", O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, p.data = $(s, p, e), {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于创建目录或相簿。 - 媒体类型媒体库可以进一步设置是否为分相簿媒体库，当设置为不分相簿时，则不允许创建目录或相簿，当设置为分相簿时，仅允许创建1层目录或相簿；文件类型媒体库不限制目录层数； - 自动创建中间所需的各级父目录； - 即使 ConflictResolutionStrategy 为 rename，如果路径中的某一父级实际为文件，则依然会返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码。 
     * @summary 创建目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CreateDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createDirectory: async (t, a, i, r, s, o, n, d = {}) => {
      f("createDirectory", "libraryId", t), f("createDirectory", "spaceId", a), f("createDirectory", "filePath", i), f("createDirectory", "accessToken", r);
      const c = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "PUT", ...h, ...d }, y = {}, u = {};
      s !== void 0 && (u.conflict_resolution_strategy = s), r !== void 0 && (u.access_token = r), o !== void 0 && (u.user_id = o), n !== void 0 && (u.with_inode = n), O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于删除目录或相簿。如果媒体库启用回收站功能，则该接口不会永久删除目录或相簿，而是将目录或相簿以及其下的文件移入回收站，可通过相关接口永久删除或恢复回收站内的目录或相簿，或直接清空回收站；
     * @summary 删除目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteDirectoryPermanentEnum} [permanent] 当媒体库开启回收站时，则该参数指定将文件移入回收站还是永久删除文件，1: 永久删除，0: 移入回收站，默认为 0
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteDirectory: async (t, a, i, r, s, o, n = {}) => {
      f("deleteDirectory", "libraryId", t), f("deleteDirectory", "spaceId", a), f("deleteDirectory", "filePath", i), f("deleteDirectory", "accessToken", r);
      const d = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "DELETE", ...l, ...n }, p = {}, y = {};
      s !== void 0 && (y.permanent = s), r !== void 0 && (y.access_token = r), o !== void 0 && (y.user_id = o), O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, {
        url: U(c),
        options: h
      };
    },
    /**
     * 此接口可同时用于查看文件或文件夹详情，路径如果为文件，则返回文件详情，如果为文件夹，则返回文件夹详情。 
     * @summary 查看文件、目录或相簿详情
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {InfoFileOrDirectoryInfoEnum} info 固定为 1
     * @param {InfoFileOrDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {InfoFileOrDirectoryWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {InfoFileOrDirectoryWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    infoFileOrDirectory: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("infoFileOrDirectory", "libraryId", t), f("infoFileOrDirectory", "spaceId", a), f("infoFileOrDirectory", "filePath", i), f("infoFileOrDirectory", "info", r);
      const l = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "GET", ...p, ...c }, u = {}, A = {};
      r !== void 0 && (A.info = r), s !== void 0 && (A.with_inode = s), o !== void 0 && (A.access_token = o), n !== void 0 && (A.with_favorite_status = n), d !== void 0 && (A.with_content_cas = d), O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, {
        url: U(h),
        options: y
      };
    },
    /**
     * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 
     * @summary 列出目录或相簿内容
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ListDirectoryByMarkerEnum} byMarker 固定传 1，表示使用 marker 方式分页
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，不传默认值20，最大返回100
     * @param {ListDirectoryOrderByEnum} [orderBy] 排序字段
     * @param {ListDirectoryOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {ListDirectoryFilterEnum} [filter] 筛选方式，不传返回全部，onlyDir 只返回文件夹，onlyFile 只返回文件
     * @param {ListDirectorySortTypeEnum} [sortType] 排序方式，不传则文件和文件夹单独排序，先返回文件夹，后返回文件。union 文件和文件夹拉通排序
     * @param {ListDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {ListDirectoryWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {ListDirectoryWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listDirectory: async (t, a, i, r, s, o, n, d, c, l, h, p, y, u, A, I = {}) => {
      f("listDirectory", "libraryId", t), f("listDirectory", "spaceId", a), f("listDirectory", "filePath", i), f("listDirectory", "byMarker", r);
      const b = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), S = new URL(b, x);
      let w;
      e && (w = e.baseOptions);
      const R = { method: "GET", ...w, ...I }, P = {}, B = {};
      r !== void 0 && (B["by-marker"] = r), s !== void 0 && (B.marker = s), o !== void 0 && (B.limit = o), n !== void 0 && (B.order_by = n), d !== void 0 && (B.order_by_type = d), c !== void 0 && (B.filter = c), l !== void 0 && (B.sort_type = l), h !== void 0 && (B.with_inode = h), p !== void 0 && (B.with_favorite_status = p), y !== void 0 && (B.access_token = y), u !== void 0 && (B.user_id = u), A !== void 0 && (B.with_content_cas = A), O(S, B);
      let T = w && w.headers ? w.headers : {};
      return R.headers = { ...P, ...T, ...I.headers }, {
        url: U(S),
        options: R
      };
    },
    /**
     * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 page翻页的深度会有限制，强烈建议业务方改用marker翻页的形式。 
     * @summary 列出目录或相簿内容（传统分页）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ListDirectoryByPageByPageEnum} byPage 固定传 1，表示使用 page 方式分页
     * @param {number} [page] 分页码，默认第一页，最大翻页的条目数（Page*PageSize的大小）是1万
     * @param {number} [pageSize] 分页大小，默认 20，最大翻页的条目数（Page*PageSize的大小）是1万
     * @param {ListDirectoryByPageOrderByEnum} [orderBy] 排序字段
     * @param {ListDirectoryByPageOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {ListDirectoryByPageFilterEnum} [filter] 筛选方式，不传返回全部，onlyDir 只返回文件夹，onlyFile 只返回文件
     * @param {ListDirectoryByPageSortTypeEnum} [sortType] 排序方式，不传则文件和文件夹单独排序，先返回文件夹，后返回文件。union 文件和文件夹拉通排序
     * @param {ListDirectoryByPageWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {ListDirectoryByPageWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {ListDirectoryByPageWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listDirectoryByPage: async (t, a, i, r, s, o, n, d, c, l, h, p, y, u, A, I = {}) => {
      f("listDirectoryByPage", "libraryId", t), f("listDirectoryByPage", "spaceId", a), f("listDirectoryByPage", "filePath", i), f("listDirectoryByPage", "byPage", r);
      const b = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), S = new URL(b, x);
      let w;
      e && (w = e.baseOptions);
      const R = { method: "GET", ...w, ...I }, P = {}, B = {};
      r !== void 0 && (B["by-page"] = r), s !== void 0 && (B.page = s), o !== void 0 && (B.page_size = o), n !== void 0 && (B.order_by = n), d !== void 0 && (B.order_by_type = d), c !== void 0 && (B.filter = c), l !== void 0 && (B.sort_type = l), h !== void 0 && (B.with_inode = h), p !== void 0 && (B.with_favorite_status = p), y !== void 0 && (B.access_token = y), u !== void 0 && (B.user_id = u), A !== void 0 && (B.with_content_cas = A), O(S, B);
      let T = w && w.headers ? w.headers : {};
      return R.headers = { ...P, ...T, ...I.headers }, {
        url: U(S),
        options: R
      };
    },
    /**
     * 用于重命名或移动目录或相簿。 要求权限： admin、space_admin 或 move_directory。 该接口的源和目标均需要指定完整的目录路径或相簿名；对于文件类型媒体库，源与目标可以跨越多层级多目录，来实现将目录移动到任意其他父目录下的功能，且支持同时修改目录名； 自动创建中间所需的各级父目录。 
     * @summary 重命名或移动目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MoveDirectoryRequest} moveDirectoryRequest 
     * @param {MoveDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    moveDirectory: async (t, a, i, r, s, o, n, d = {}) => {
      f("moveDirectory", "libraryId", t), f("moveDirectory", "spaceId", a), f("moveDirectory", "filePath", i), f("moveDirectory", "accessToken", r), f("moveDirectory", "moveDirectoryRequest", s);
      const c = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "PUT", ...h, ...d }, y = {}, u = {};
      o !== void 0 && (u.conflict_resolution_strategy = o), r !== void 0 && (u.access_token = r), n !== void 0 && (u.user_id = n), y["Content-Type"] = "application/json", O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, p.data = $(s, p, e), {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于更新目录自定义标签。需要 admin 权限或 spaceAdmin 权限
     * @summary 更新目录自定义标签
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {UpdateDirectoryLabelsUpdateEnum} update 固定为 1
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {UpdateDirectoryLabelsRequest} [updateDirectoryLabelsRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    updateDirectoryLabels: async (t, a, i, r, s, o, n = {}) => {
      f("updateDirectoryLabels", "libraryId", t), f("updateDirectoryLabels", "spaceId", a), f("updateDirectoryLabels", "filePath", i), f("updateDirectoryLabels", "update", r);
      const d = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      r !== void 0 && (y.update = r), s !== void 0 && (y.access_token = s), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(o, h, e), {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于更新文件的标签（Labels）或分类（Category）。 需要 admin 权限或 spaceAdmin 权限。 
     * @summary 更新文件标签或分类
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {UpdateFileLabelsUpdateEnum} update 固定为 1
     * @param {UpdateFileLabelsRequest} updateFileLabelsRequest 
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    updateFileLabels: async (t, a, i, r, s, o, n = {}) => {
      f("updateFileLabels", "libraryId", t), f("updateFileLabels", "spaceId", a), f("updateFileLabels", "filePath", i), f("updateFileLabels", "update", r), f("updateFileLabels", "updateFileLabelsRequest", s);
      const d = "/api/v1/directory/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      r !== void 0 && (y.update = r), o !== void 0 && (y.access_token = o), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(s, h, e), {
        url: U(c),
        options: h
      };
    }
  };
}, ge = function(e) {
  const t = pn(e);
  return {
    /**
     * 用于检查目录或相簿状态
     * @summary 检查目录或相簿状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async checkDirectoryStatus(a, i, r, s, o, n) {
      var h, p;
      const d = await t.checkDirectoryStatus(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["DirectoryApi.checkDirectoryStatus"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于复制目录或相簿。 - 自动创建中间所需的各级父目录。 
     * @summary 复制目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CopyDirectoryRequest} copyDirectoryRequest 
     * @param {CopyDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async copyDirectory(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.copyDirectory(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["DirectoryApi.copyDirectory"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于创建目录或相簿。 - 媒体类型媒体库可以进一步设置是否为分相簿媒体库，当设置为不分相簿时，则不允许创建目录或相簿，当设置为分相簿时，仅允许创建1层目录或相簿；文件类型媒体库不限制目录层数； - 自动创建中间所需的各级父目录； - 即使 ConflictResolutionStrategy 为 rename，如果路径中的某一父级实际为文件，则依然会返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码。 
     * @summary 创建目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CreateDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createDirectory(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.createDirectory(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["DirectoryApi.createDirectory"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于删除目录或相簿。如果媒体库启用回收站功能，则该接口不会永久删除目录或相簿，而是将目录或相簿以及其下的文件移入回收站，可通过相关接口永久删除或恢复回收站内的目录或相簿，或直接清空回收站；
     * @summary 删除目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteDirectoryPermanentEnum} [permanent] 当媒体库开启回收站时，则该参数指定将文件移入回收站还是永久删除文件，1: 永久删除，0: 移入回收站，默认为 0
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteDirectory(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.deleteDirectory(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["DirectoryApi.deleteDirectory"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 此接口可同时用于查看文件或文件夹详情，路径如果为文件，则返回文件详情，如果为文件夹，则返回文件夹详情。 
     * @summary 查看文件、目录或相簿详情
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {InfoFileOrDirectoryInfoEnum} info 固定为 1
     * @param {InfoFileOrDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {InfoFileOrDirectoryWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {InfoFileOrDirectoryWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async infoFileOrDirectory(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.infoFileOrDirectory(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["DirectoryApi.infoFileOrDirectory"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    },
    /**
     * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 
     * @summary 列出目录或相簿内容
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ListDirectoryByMarkerEnum} byMarker 固定传 1，表示使用 marker 方式分页
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，不传默认值20，最大返回100
     * @param {ListDirectoryOrderByEnum} [orderBy] 排序字段
     * @param {ListDirectoryOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {ListDirectoryFilterEnum} [filter] 筛选方式，不传返回全部，onlyDir 只返回文件夹，onlyFile 只返回文件
     * @param {ListDirectorySortTypeEnum} [sortType] 排序方式，不传则文件和文件夹单独排序，先返回文件夹，后返回文件。union 文件和文件夹拉通排序
     * @param {ListDirectoryWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {ListDirectoryWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {ListDirectoryWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listDirectory(a, i, r, s, o, n, d, c, l, h, p, y, u, A, I, b) {
      var P, B;
      const S = await t.listDirectory(a, i, r, s, o, n, d, c, l, h, p, y, u, A, I, b), w = (e == null ? void 0 : e.serverIndex) ?? 0, R = (B = (P = F["DirectoryApi.listDirectory"]) == null ? void 0 : P[w]) == null ? void 0 : B.url;
      return (T, j) => k(S, _, C, e)(T, R || j);
    },
    /**
     * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 page翻页的深度会有限制，强烈建议业务方改用marker翻页的形式。 
     * @summary 列出目录或相簿内容（传统分页）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ListDirectoryByPageByPageEnum} byPage 固定传 1，表示使用 page 方式分页
     * @param {number} [page] 分页码，默认第一页，最大翻页的条目数（Page*PageSize的大小）是1万
     * @param {number} [pageSize] 分页大小，默认 20，最大翻页的条目数（Page*PageSize的大小）是1万
     * @param {ListDirectoryByPageOrderByEnum} [orderBy] 排序字段
     * @param {ListDirectoryByPageOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {ListDirectoryByPageFilterEnum} [filter] 筛选方式，不传返回全部，onlyDir 只返回文件夹，onlyFile 只返回文件
     * @param {ListDirectoryByPageSortTypeEnum} [sortType] 排序方式，不传则文件和文件夹单独排序，先返回文件夹，后返回文件。union 文件和文件夹拉通排序
     * @param {ListDirectoryByPageWithInodeEnum} [withInode] 是否返回 inode，即文件目录 ID，0 或 1，默认不返回
     * @param {ListDirectoryByPageWithFavoriteStatusEnum} [withFavoriteStatus] 是否返回收藏状态，0 或 1，默认不返回
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {ListDirectoryByPageWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listDirectoryByPage(a, i, r, s, o, n, d, c, l, h, p, y, u, A, I, b) {
      var P, B;
      const S = await t.listDirectoryByPage(a, i, r, s, o, n, d, c, l, h, p, y, u, A, I, b), w = (e == null ? void 0 : e.serverIndex) ?? 0, R = (B = (P = F["DirectoryApi.listDirectoryByPage"]) == null ? void 0 : P[w]) == null ? void 0 : B.url;
      return (T, j) => k(S, _, C, e)(T, R || j);
    },
    /**
     * 用于重命名或移动目录或相簿。 要求权限： admin、space_admin 或 move_directory。 该接口的源和目标均需要指定完整的目录路径或相簿名；对于文件类型媒体库，源与目标可以跨越多层级多目录，来实现将目录移动到任意其他父目录下的功能，且支持同时修改目录名； 自动创建中间所需的各级父目录。 
     * @summary 重命名或移动目录或相簿
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MoveDirectoryRequest} moveDirectoryRequest 
     * @param {MoveDirectoryConflictResolutionStrategyEnum} [conflictResolutionStrategy] 最后一级目录冲突时的处理方式，ask冲突时返回 HTTP 409，rename冲突时自动重命名最后一级目录，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async moveDirectory(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.moveDirectory(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["DirectoryApi.moveDirectory"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于更新目录自定义标签。需要 admin 权限或 spaceAdmin 权限
     * @summary 更新目录自定义标签
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {UpdateDirectoryLabelsUpdateEnum} update 固定为 1
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {UpdateDirectoryLabelsRequest} [updateDirectoryLabelsRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async updateDirectoryLabels(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.updateDirectoryLabels(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["DirectoryApi.updateDirectoryLabels"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于更新文件的标签（Labels）或分类（Category）。 需要 admin 权限或 spaceAdmin 权限。 
     * @summary 更新文件标签或分类
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {UpdateFileLabelsUpdateEnum} update 固定为 1
     * @param {UpdateFileLabelsRequest} updateFileLabelsRequest 
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async updateFileLabels(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.updateFileLabels(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["DirectoryApi.updateFileLabels"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    }
  };
}, yn = class extends he {
  /**
   * 用于检查目录或相簿状态
   * @summary 检查目录或相簿状态
   * @param {DirectoryApiCheckDirectoryStatusRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  checkDirectoryStatus(e, t) {
    return ge(this.configuration).checkDirectoryStatus(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于复制目录或相簿。 - 自动创建中间所需的各级父目录。 
   * @summary 复制目录或相簿
   * @param {DirectoryApiCopyDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  copyDirectory(e, t) {
    return ge(this.configuration).copyDirectory(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.copyDirectoryRequest, e.conflictResolutionStrategy, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于创建目录或相簿。 - 媒体类型媒体库可以进一步设置是否为分相簿媒体库，当设置为不分相簿时，则不允许创建目录或相簿，当设置为分相簿时，仅允许创建1层目录或相簿；文件类型媒体库不限制目录层数； - 自动创建中间所需的各级父目录； - 即使 ConflictResolutionStrategy 为 rename，如果路径中的某一父级实际为文件，则依然会返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码。 
   * @summary 创建目录或相簿
   * @param {DirectoryApiCreateDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createDirectory(e, t) {
    return ge(this.configuration).createDirectory(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.conflictResolutionStrategy, e.userId, e.withInode, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于删除目录或相簿。如果媒体库启用回收站功能，则该接口不会永久删除目录或相簿，而是将目录或相簿以及其下的文件移入回收站，可通过相关接口永久删除或恢复回收站内的目录或相簿，或直接清空回收站；
   * @summary 删除目录或相簿
   * @param {DirectoryApiDeleteDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteDirectory(e, t) {
    return ge(this.configuration).deleteDirectory(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.permanent, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 此接口可同时用于查看文件或文件夹详情，路径如果为文件，则返回文件详情，如果为文件夹，则返回文件夹详情。 
   * @summary 查看文件、目录或相簿详情
   * @param {DirectoryApiInfoFileOrDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  infoFileOrDirectory(e, t) {
    return ge(this.configuration).infoFileOrDirectory(e.libraryId, e.spaceId, e.filePath, e.info, e.withInode, e.accessToken, e.withFavoriteStatus, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 
   * @summary 列出目录或相簿内容
   * @param {DirectoryApiListDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listDirectory(e, t) {
    return ge(this.configuration).listDirectory(e.libraryId, e.spaceId, e.filePath, e.byMarker, e.marker, e.limit, e.orderBy, e.orderByType, e.filter, e.sortType, e.withInode, e.withFavoriteStatus, e.accessToken, e.userId, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出目录或相簿内容。 目录内容的列出顺序为：首先按照字典序列出子目录，随后根据上传时间列出媒体库中的媒体资源，或根据文件名列出文件库中的文件资源。 page翻页的深度会有限制，强烈建议业务方改用marker翻页的形式。 
   * @summary 列出目录或相簿内容（传统分页）
   * @param {DirectoryApiListDirectoryByPageRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listDirectoryByPage(e, t) {
    return ge(this.configuration).listDirectoryByPage(e.libraryId, e.spaceId, e.filePath, e.byPage, e.page, e.pageSize, e.orderBy, e.orderByType, e.filter, e.sortType, e.withInode, e.withFavoriteStatus, e.accessToken, e.userId, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于重命名或移动目录或相簿。 要求权限： admin、space_admin 或 move_directory。 该接口的源和目标均需要指定完整的目录路径或相簿名；对于文件类型媒体库，源与目标可以跨越多层级多目录，来实现将目录移动到任意其他父目录下的功能，且支持同时修改目录名； 自动创建中间所需的各级父目录。 
   * @summary 重命名或移动目录或相簿
   * @param {DirectoryApiMoveDirectoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  moveDirectory(e, t) {
    return ge(this.configuration).moveDirectory(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.moveDirectoryRequest, e.conflictResolutionStrategy, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于更新目录自定义标签。需要 admin 权限或 spaceAdmin 权限
   * @summary 更新目录自定义标签
   * @param {DirectoryApiUpdateDirectoryLabelsRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  updateDirectoryLabels(e, t) {
    return ge(this.configuration).updateDirectoryLabels(e.libraryId, e.spaceId, e.filePath, e.update, e.accessToken, e.updateDirectoryLabelsRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于更新文件的标签（Labels）或分类（Category）。 需要 admin 权限或 spaceAdmin 权限。 
   * @summary 更新文件标签或分类
   * @param {DirectoryApiUpdateFileLabelsRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  updateFileLabels(e, t) {
    return ge(this.configuration).updateFileLabels(e.libraryId, e.spaceId, e.filePath, e.update, e.updateFileLabelsRequest, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
}, un = function(e) {
  return {
    /**
     * 收藏文件目录。需要提供路径或inode，二者二选一；如果同时提供，以inode为准。 
     * @summary 收藏指定空间文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateFavoriteRequest} createFavoriteRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createFavorite: async (t, a, i, r, s = {}) => {
      f("createFavorite", "libraryId", t), f("createFavorite", "spaceId", a), f("createFavorite", "accessToken", i), f("createFavorite", "createFavoriteRequest", r);
      const o = "/api/v1/favorite/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(r, c, e), {
        url: U(n),
        options: c
      };
    },
    /**
     * 取消收藏。根据路径或inode取消收藏，二者二选一；如果同时提供，以inode为准。 
     * @summary 取消收藏指定空间文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteFavoriteCancelEnum} cancel 取消收藏标志，传递该参数表示执行取消收藏操作
     * @param {CreateFavoriteRequest} deleteFavoriteRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteFavorite: async (t, a, i, r, s, o = {}) => {
      f("deleteFavorite", "libraryId", t), f("deleteFavorite", "spaceId", a), f("deleteFavorite", "accessToken", i), f("deleteFavorite", "cancel", r), f("deleteFavorite", "deleteFavoriteRequest", s);
      const n = "/api/v1/favorite/{LibraryId}/{SpaceId}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "POST", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.access_token = i), r !== void 0 && (p.cancel = r), h["Content-Type"] = "application/json", O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, l.data = $(s, l, e), {
        url: U(d),
        options: l
      };
    },
    /**
     * 查看指定空间收藏列表，支持分页和排序
     * @summary 查看指定空间收藏列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [marker] 用于顺序列出分页的标识，可选参数
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，默认为20，可选参数
     * @param {number} [page] 分页码，默认第一页，可选参数，不能与marker和limit参数同时使用
     * @param {number} [pageSize] 分页大小，默认20，可选参数，不能与marker和limit参数同时使用
     * @param {ListFavoriteOrderByEnum} [orderBy] 排序字段，按收藏时间排序为favoriteTime（默认），目前仅支持按收藏时间排序，可选参数
     * @param {ListFavoriteOrderByTypeEnum} [orderByType] 排序方式，升序为asc，降序为desc（默认），可选参数
     * @param {boolean} [withPath] 是否返回path，返回为true，不返回为false（默认），可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listFavorite: async (t, a, i, r, s, o, n, d, c, l, h = {}) => {
      f("listFavorite", "libraryId", t), f("listFavorite", "spaceId", a), f("listFavorite", "accessToken", i);
      const p = "/api/v1/favorite/{LibraryId}/{SpaceId}/list".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), y = new URL(p, x);
      let u;
      e && (u = e.baseOptions);
      const A = { method: "GET", ...u, ...h }, I = {}, b = {};
      r !== void 0 && (b.marker = r), s !== void 0 && (b.limit = s), o !== void 0 && (b.page = o), n !== void 0 && (b.page_size = n), d !== void 0 && (b.order_by = d), c !== void 0 && (b.order_by_type = c), l !== void 0 && (b.with_path = l), i !== void 0 && (b.access_token = i), O(y, b);
      let S = u && u.headers ? u.headers : {};
      return A.headers = { ...I, ...S, ...h.headers }, {
        url: U(y),
        options: A
      };
    }
  };
}, oa = function(e) {
  const t = un(e);
  return {
    /**
     * 收藏文件目录。需要提供路径或inode，二者二选一；如果同时提供，以inode为准。 
     * @summary 收藏指定空间文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateFavoriteRequest} createFavoriteRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createFavorite(a, i, r, s, o) {
      var l, h;
      const n = await t.createFavorite(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["FavoriteApi.createFavorite"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 取消收藏。根据路径或inode取消收藏，二者二选一；如果同时提供，以inode为准。 
     * @summary 取消收藏指定空间文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteFavoriteCancelEnum} cancel 取消收藏标志，传递该参数表示执行取消收藏操作
     * @param {CreateFavoriteRequest} deleteFavoriteRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteFavorite(a, i, r, s, o, n) {
      var h, p;
      const d = await t.deleteFavorite(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["FavoriteApi.deleteFavorite"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 查看指定空间收藏列表，支持分页和排序
     * @summary 查看指定空间收藏列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [marker] 用于顺序列出分页的标识，可选参数
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，默认为20，可选参数
     * @param {number} [page] 分页码，默认第一页，可选参数，不能与marker和limit参数同时使用
     * @param {number} [pageSize] 分页大小，默认20，可选参数，不能与marker和limit参数同时使用
     * @param {ListFavoriteOrderByEnum} [orderBy] 排序字段，按收藏时间排序为favoriteTime（默认），目前仅支持按收藏时间排序，可选参数
     * @param {ListFavoriteOrderByTypeEnum} [orderByType] 排序方式，升序为asc，降序为desc（默认），可选参数
     * @param {boolean} [withPath] 是否返回path，返回为true，不返回为false（默认），可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listFavorite(a, i, r, s, o, n, d, c, l, h, p) {
      var I, b;
      const y = await t.listFavorite(a, i, r, s, o, n, d, c, l, h, p), u = (e == null ? void 0 : e.serverIndex) ?? 0, A = (b = (I = F["FavoriteApi.listFavorite"]) == null ? void 0 : I[u]) == null ? void 0 : b.url;
      return (S, w) => k(y, _, C, e)(S, A || w);
    }
  };
}, fn = class extends he {
  /**
   * 收藏文件目录。需要提供路径或inode，二者二选一；如果同时提供，以inode为准。 
   * @summary 收藏指定空间文件
   * @param {FavoriteApiCreateFavoriteRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createFavorite(e, t) {
    return oa(this.configuration).createFavorite(e.libraryId, e.spaceId, e.accessToken, e.createFavoriteRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 取消收藏。根据路径或inode取消收藏，二者二选一；如果同时提供，以inode为准。 
   * @summary 取消收藏指定空间文件
   * @param {FavoriteApiDeleteFavoriteRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteFavorite(e, t) {
    return oa(this.configuration).deleteFavorite(e.libraryId, e.spaceId, e.accessToken, e.cancel, e.deleteFavoriteRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 查看指定空间收藏列表，支持分页和排序
   * @summary 查看指定空间收藏列表
   * @param {FavoriteApiListFavoriteRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listFavorite(e, t) {
    return oa(this.configuration).listFavorite(e.libraryId, e.spaceId, e.accessToken, e.marker, e.limit, e.page, e.pageSize, e.orderBy, e.orderByType, e.withPath, t).then((a) => a(this.axios, this.basePath));
  }
}, An = function(e) {
  return {
    /**
     * 用于取消上传任务。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为删除接口，但因为删除的是上传任务信息，故仍需上传文件的相关权限） 如果上传任务为分块上传任务，那么该请求将同时放弃 COS 中的分块上传任务。 
     * @summary 取消上传任务
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数
     * @param {AbortFileUploadUploadEnum} upload 上传任务标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    abortFileUpload: async (t, a, i, r, s, o, n = {}) => {
      f("abortFileUpload", "libraryId", t), f("abortFileUpload", "spaceId", a), f("abortFileUpload", "confirmKey", i), f("abortFileUpload", "upload", r), f("abortFileUpload", "accessToken", s);
      const d = "/api/v1/file/{LibraryId}/{SpaceId}/{ConfirmKey}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{ConfirmKey}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "DELETE", ...l, ...n }, p = {}, y = {};
      r !== void 0 && (y.upload = r), s !== void 0 && (y.access_token = s), o !== void 0 && (y.user_id = o), O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于查询文件删除的原因，可能是用户主动删除或者 quota 超限删除。 要求权限：admin 或 space_admin 
     * @summary 查询文件删除原因
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} inode 文件的 Inode
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    checkFileDeletion: async (t, a, i, r, s = {}) => {
      f("checkFileDeletion", "libraryId", t), f("checkFileDeletion", "spaceId", a), f("checkFileDeletion", "inode", i), f("checkFileDeletion", "accessToken", r);
      const o = "/api/v1/file-deletion-check/{LibraryId}/{SpaceId}/{Inode}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{Inode}", encodeURIComponent(String(i))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      r !== void 0 && (h.access_token = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于检查文件状态
     * @summary 检查文件状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    checkFileStatus: async (t, a, i, r, s, o, n = {}) => {
      f("checkFileStatus", "libraryId", t), f("checkFileStatus", "spaceId", a), f("checkFileStatus", "filePath", i);
      const d = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "HEAD", ...l, ...n }, p = {}, y = {};
      r !== void 0 && (y.history_id = r), s !== void 0 && (y.access_token = s), o !== void 0 && (y.user_id = o), O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于完成上传文件。 要求权限：admin、space_admin、upload_file、upload_file_force 或 confirm_upload。 在文件上传完成后，请务必及时调用该接口，否则文件将不能被正确存储；如果调用该接口时实际并未完成文件上传，将返回错误信息。 
     * @summary 完成上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数，指定为开始上传文件时响应体中的 confirmKey 字段的值
     * @param {CompleteFileUploadConfirmEnum} confirm 完成上传标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CompleteFileUploadConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件；不传则沿用开始上传时的设置
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CompleteFileUploadWithInodeEnum} [withInode] 是否返回 inode（文件目录 ID），0 或 1，默认 0
     * @param {CompleteFileUploadWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {CompleteFileUploadRequest} [completeFileUploadRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    completeFileUpload: async (t, a, i, r, s, o, n, d, c, l, h, p = {}) => {
      f("completeFileUpload", "libraryId", t), f("completeFileUpload", "spaceId", a), f("completeFileUpload", "confirmKey", i), f("completeFileUpload", "confirm", r), f("completeFileUpload", "accessToken", s);
      const y = "/api/v1/file/{LibraryId}/{SpaceId}/{ConfirmKey}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{ConfirmKey}", encodeURIComponent(String(i))), u = new URL(y, x);
      let A;
      e && (A = e.baseOptions);
      const I = { method: "POST", ...A, ...p }, b = {}, S = {};
      r !== void 0 && (S.confirm = r), o !== void 0 && (S.conflict_resolution_strategy = o), n !== void 0 && (S.content_cas = n), s !== void 0 && (S.access_token = s), d !== void 0 && (S.user_id = d), c !== void 0 && (S.with_inode = c), l !== void 0 && (S.with_content_cas = l), b["Content-Type"] = "application/json", O(u, S);
      let w = A && A.headers ? A.headers : {};
      return I.headers = { ...b, ...w, ...p.headers }, I.data = $(h, I, e), {
        url: U(u),
        options: I
      };
    },
    /**
     * 用于转换文档格式，当前仅支持 doc/docx 转 pdf。 要求权限： 非 acl 鉴权：admin、space_admin acl 鉴权：canDownload（当前文件夹可下载）& canUpload（目标文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 文档转码
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ConvertFileConvertEnum} convert 文档转码操作标识，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {ConvertFileRequest} convertFileRequest 
     * @param {ConvertFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    convertFile: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("convertFile", "libraryId", t), f("convertFile", "spaceId", a), f("convertFile", "filePath", i), f("convertFile", "convert", r), f("convertFile", "accessToken", s), f("convertFile", "convertFileRequest", o);
      const l = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "PUT", ...p, ...c }, u = {}, A = {};
      r !== void 0 && (A.convert = r), n !== void 0 && (A.conflict_resolution_strategy = n), s !== void 0 && (A.access_token = s), d !== void 0 && (A.user_id = d), u["Content-Type"] = "application/json", O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, y.data = $(o, y, e), {
        url: U(h),
        options: y
      };
    },
    /**
     * 用于复制文件。 要求权限： admin、space_admin 或 copy_file/copy_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件复制到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 复制文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CopyFileRequest} copyFileRequest 
     * @param {CopyFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CopyFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    copyFile: async (t, a, i, r, s, o, n, d, c, l = {}) => {
      f("copyFile", "libraryId", t), f("copyFile", "spaceId", a), f("copyFile", "filePath", i), f("copyFile", "accessToken", r), f("copyFile", "copyFileRequest", s);
      const h = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#3".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), p = new URL(h, x);
      let y;
      e && (y = e.baseOptions);
      const u = { method: "PUT", ...y, ...l }, A = {}, I = {};
      o !== void 0 && (I.conflict_resolution_strategy = o), n !== void 0 && (I.content_cas = n), r !== void 0 && (I.access_token = r), d !== void 0 && (I.user_id = d), c !== void 0 && (I.with_content_cas = c), A["Content-Type"] = "application/json", O(p, I);
      let b = y && y.headers ? y.headers : {};
      return u.headers = { ...A, ...b, ...l.headers }, u.data = $(s, u, e), {
        url: U(p),
        options: u
      };
    },
    /**
     * 用于创建符号链接。 要求权限： 非 acl 鉴权：admin、space_admin 或 upload_file/upload_file_force/create_symlink/create_symlink_force acl 鉴权：canUpload（当前文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 符号链接本身与文件的概念一致，可以通过删除文件、重命名或移动文件、复制文件等接口删除、重命名或移动或复制符号链接本身，而不会影响符号链接所指向的文件； 与标准文件系统略有不同，符号链接所指向的文件，不会因为重命名或移动而丢失指向； 当符号链接指向的文件被覆盖上传时，该符号链接将指向新上传的文件。 
     * @summary 创建符号链接
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateSymlinkRequest} createSymlinkRequest 
     * @param {CreateSymlinkConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite覆盖已有文件，默认为 rename
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createSymlink: async (t, a, i, r, s, o, n, d = {}) => {
      f("createSymlink", "libraryId", t), f("createSymlink", "spaceId", a), f("createSymlink", "filePath", i), f("createSymlink", "accessToken", r), f("createSymlink", "createSymlinkRequest", s);
      const c = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "PUT", ...h, ...d }, y = {}, u = {};
      o !== void 0 && (u.conflict_resolution_strategy = o), r !== void 0 && (u.access_token = r), n !== void 0 && (u.user_id = n), y["Content-Type"] = "application/json", O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, p.data = $(s, p, e), {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于删除文件。 要求权限： admin、space_admin 或 delete_file（未开启回收站或 Permanent 为 0）/delete_file_permanent（开启回收站且 Permanent 为 1） 如果媒体库启用回收站功能，则该接口不会永久删除文件，而是将文件移入回收站，可通过相关接口永久删除或恢复回收站内的文件，或直接清空回收站。 
     * @summary 删除文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteFilePermanentEnum} [permanent] 当媒体库开启回收站时，则该参数指定将文件移入回收站还是永久删除文件，1: 永久删除，0: 移入回收站，默认为 0
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteFile: async (t, a, i, r, s, o, n, d = {}) => {
      f("deleteFile", "libraryId", t), f("deleteFile", "spaceId", a), f("deleteFile", "filePath", i), f("deleteFile", "accessToken", r);
      const c = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "DELETE", ...h, ...d }, y = {}, u = {};
      s !== void 0 && (u.permanent = s), r !== void 0 && (u.access_token = r), o !== void 0 && (u.user_id = o), n !== void 0 && (u.content_cas = n), O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于下载文件。 可以直接在使用文件的参数中指定该 URL，例如对于图片文件可直接在小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等中使用，该接口将自动 302 跳转到真实的图片 URL；视频和文件同理； 
     * @summary 下载文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {DownloadFileContentDispositionEnum} [contentDisposition] 用于设置Content-Disposition响应头，支持 inline 或者 attachment，可选参数，不传默认为inline
     * @param {DownloadFilePurposeEnum} [purpose] 用途，可选参数，可以设置为download或者preview，用于决定是否将该文件加入最近使用文件列表中，如果设置为preview，则会将该文件加入最近使用文件列表中，否则不会加入
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {DownloadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    downloadFile: async (t, a, i, r, s, o, n, d, c, l, h, p = {}) => {
      f("downloadFile", "libraryId", t), f("downloadFile", "spaceId", a), f("downloadFile", "filePath", i);
      const y = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), u = new URL(y, x);
      let A;
      e && (A = e.baseOptions);
      const I = { method: "GET", ...A, ...p }, b = {}, S = {};
      r !== void 0 && (S.history_id = r), s !== void 0 && (S.content_disposition = s), o !== void 0 && (S.purpose = o), n !== void 0 && (S.access_token = n), d !== void 0 && (S.user_id = d), c !== void 0 && (S.traffic_limit = c), l !== void 0 && (S.content_cas = l), h !== void 0 && (S.with_content_cas = h), O(u, S);
      let w = A && A.headers ? A.headers : {};
      return I.headers = { ...b, ...w, ...p.headers }, {
        url: U(u),
        options: I
      };
    },
    /**
     * 用于开始表单上传文件（multipart/form-data）。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 调用该接口将返回一系列用于 form 表单上传（multipart/form-data 格式）和确认上传完成的参数，上传的目标 URL 为 https://{Domain}/，其中 Domain 为响应体中的 domain 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/； form 表单上传时还需要指定一系列额外的信息字段，这些字段的名和值包含在响应体中的 form 字段中，可以在 HTML form 表单中通过隐藏域或通过 JS 相关库、小程序 wx.uploadFile 等指定这些字段； form 表单中的文件字段，其表单字段名固定为 file，且必须作为表单中的最后一项； 在完成实际上传后，上传的目标 URL 将返回 HTTP 204 No Content，由于可能的跨域限制，建议直接通过相关接口的回调来判断是否上传完成，并且在上传完成后及时调用完成上传文件接口，确认上传结果； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始表单上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {FormUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {FormUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {FormUploadFileRequest} [formUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    formUploadFile: async (t, a, i, r, s, o, n, d, c, l, h, p, y = {}) => {
      f("formUploadFile", "libraryId", t), f("formUploadFile", "spaceId", a), f("formUploadFile", "filePath", i), f("formUploadFile", "accessToken", r);
      const u = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), A = new URL(u, x);
      let I;
      e && (I = e.baseOptions);
      const b = { method: "POST", ...I, ...y }, S = {}, w = {};
      s !== void 0 && (w.conflict_resolution_strategy = s), o !== void 0 && (w.filesize = o), r !== void 0 && (w.access_token = r), n !== void 0 && (w.user_id = n), c !== void 0 && (w.traffic_limit = c), l !== void 0 && (w.prefer_same_origin = l), h !== void 0 && (w.with_content_cas = h), S["Content-Type"] = "application/json", d != null && (S["x-smh-meta-*"] = String(d)), O(A, w);
      let R = I && I.headers ? I.headers : {};
      return b.headers = { ...S, ...R, ...y.headers }, b.data = $(p, b, e), {
        url: U(A),
        options: b
      };
    },
    /**
     * 用于获取照片/视频封面缩略图。 视频封面使用该视频的首帧图片； 针对照片或视频封面，优先使用人脸识别智能缩放裁剪为 {Size}px × {Size}px 大小，如果未识别到人脸则居中缩放裁剪为 {Size}px × {Size}px 大小，如果未指定 {Size} 参数则使用照片或视频封面原图，最后 302 跳转到对应的图片的 URL； 可以直接在使用图片的参数中指定该 URL，例如小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等，该接口将自动 302 跳转到真实的图片 URL； 如果文件不属于可预览的媒体类型，则会跳转至文件的下载链接。 
     * @summary 获取照片/视频封面缩略图
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {number} preview 预览标识，固定值为1
     * @param {number} [size] 缩放大小，优先使用人脸识别智能缩放裁剪为 size×size，未识别到人脸则居中缩放裁剪为 size×size；不传则使用原图
     * @param {number} [scale] 等比例缩放百分比（1-100），当未传 size 时生效
     * @param {number} [widthSize] 缩放宽度，当未传 size 和 scale 时生效；未传高度时，高度按等比例缩放
     * @param {number} [heightSize] 缩放高度，当未传 size 和 scale 时生效；未传宽度时，宽度按等比例缩放
     * @param {number} [frameNumber] 帧数，针对 gif 的降帧处理
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getCover: async (t, a, i, r, s, o, n, d, c, l, h, p = {}) => {
      f("getCover", "libraryId", t), f("getCover", "spaceId", a), f("getCover", "filePath", i), f("getCover", "preview", r);
      const y = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), u = new URL(y, x);
      let A;
      e && (A = e.baseOptions);
      const I = { method: "GET", ...A, ...p }, b = {}, S = {};
      r !== void 0 && (S.preview = r), s !== void 0 && (S.size = s), o !== void 0 && (S.scale = o), n !== void 0 && (S.width_size = n), d !== void 0 && (S.height_size = d), c !== void 0 && (S.frame_number = c), l !== void 0 && (S.access_token = l), h !== void 0 && (S.user_id = h), O(u, S);
      let w = A && A.headers ? A.headers : {};
      return I.headers = { ...b, ...w, ...p.headers }, {
        url: U(u),
        options: I
      };
    },
    /**
     * 根据文件 ID 查询文件信息
     * @summary 根据文件ID查询文件信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} inode 文件ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {GetFileInfoByInodeWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getFileInfoByInode: async (t, a, i, r, s, o = {}) => {
      f("getFileInfoByInode", "libraryId", t), f("getFileInfoByInode", "spaceId", a), f("getFileInfoByInode", "inode", i), f("getFileInfoByInode", "accessToken", r);
      const n = "/api/v1/inode/{LibraryId}/{SpaceId}/{Inode}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{Inode}", encodeURIComponent(String(i))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "GET", ...c, ...o }, h = {}, p = {};
      r !== void 0 && (p.access_token = r), s !== void 0 && (p.with_content_cas = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于获取文件上传任务状态。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为读接口，但因为读取的是上传任务信息，故仍需上传文件的相关权限） 
     * @summary 获取文件上传任务状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数
     * @param {GetFileUploadUploadEnum} upload 上传任务标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getFileUpload: async (t, a, i, r, s, o, n = {}) => {
      f("getFileUpload", "libraryId", t), f("getFileUpload", "spaceId", a), f("getFileUpload", "confirmKey", i), f("getFileUpload", "upload", r), f("getFileUpload", "accessToken", s);
      const d = "/api/v1/file/{LibraryId}/{SpaceId}/{ConfirmKey}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{ConfirmKey}", encodeURIComponent(String(i))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "GET", ...l, ...n }, p = {}, y = {};
      r !== void 0 && (y.upload = r), s !== void 0 && (y.access_token = s), o !== void 0 && (y.user_id = o), O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于获取文件下载链接和信息。 要求权限：无 
     * @summary 获取文件下载链接和信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {InfoFileInfoEnum} info 获取文件信息标识
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {InfoFileContentDispositionEnum} [contentDisposition] 用于设置Content-Disposition响应头，支持 inline 或者 attachment，可选参数，不传默认为inline
     * @param {InfoFilePurposeEnum} [purpose] 用途，可选参数，可以设置为download或者preview，用于决定是否将该文件加入最近使用文件列表中，如果设置为preview，则会将该文件加入最近使用文件列表中，否则不会加入
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {InfoFilePreCheckEnum} [preCheck] 是否只用于校验文件是否可预览和下载，设置该参数后返回结果中不包含cosUrl
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {InfoFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    infoFile: async (t, a, i, r, s, o, n, d, c, l, h, p, y, u = {}) => {
      f("infoFile", "libraryId", t), f("infoFile", "spaceId", a), f("infoFile", "filePath", i), f("infoFile", "info", r);
      const A = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), I = new URL(A, x);
      let b;
      e && (b = e.baseOptions);
      const S = { method: "GET", ...b, ...u }, w = {}, R = {};
      r !== void 0 && (R.info = r), s !== void 0 && (R.history_id = s), o !== void 0 && (R.content_disposition = o), n !== void 0 && (R.purpose = n), d !== void 0 && (R.access_token = d), c !== void 0 && (R.user_id = c), l !== void 0 && (R.traffic_limit = l), h !== void 0 && (R.pre_check = h), p !== void 0 && (R.content_cas = p), y !== void 0 && (R.with_content_cas = y), O(I, R);
      let P = b && b.headers ? b.headers : {};
      return S.headers = { ...w, ...P, ...u.headers }, {
        url: U(I),
        options: S
      };
    },
    /**
     * 用于重命名或移动文件。 要求权限： admin、space_admin 或 move_file/move_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 重命名或移动文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MoveFileRequest} moveFileRequest 
     * @param {MoveFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {MoveFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    moveFile: async (t, a, i, r, s, o, n, d, c, l = {}) => {
      f("moveFile", "libraryId", t), f("moveFile", "spaceId", a), f("moveFile", "filePath", i), f("moveFile", "accessToken", r), f("moveFile", "moveFileRequest", s);
      const h = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#4".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), p = new URL(h, x);
      let y;
      e && (y = e.baseOptions);
      const u = { method: "PUT", ...y, ...l }, A = {}, I = {};
      o !== void 0 && (I.conflict_resolution_strategy = o), n !== void 0 && (I.content_cas = n), r !== void 0 && (I.access_token = r), d !== void 0 && (I.user_id = d), c !== void 0 && (I.with_content_cas = c), A["Content-Type"] = "application/json", O(p, I);
      let b = y && y.headers ? y.headers : {};
      return u.headers = { ...A, ...b, ...l.headers }, u.data = $(s, u, e), {
        url: U(p),
        options: u
      };
    },
    /**
     * 用于开始分块上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 分块上传指使用通过 HTTP PUT 请求上传一个文件的分块，通过多次上传完成整个文件的上传，每次请求的请求体为文件内容的单个分块； 调用该接口将返回一系列用于分块上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}?uploadId={UploadId}&partNumber={PartNumber}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，UploadId 为响应体中的 uploadId 字段，PartNumber 为从 1 开始的分块顺序，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4?uploadId=xxx&partNumber=1； 上传每个分块时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 与对象存储 COS 的分块上传不同，SMH 的分块上传无需记录 ETag，也无需在完成上传时传入这些 ETag，只需保证上传分块的连续即可，SMH 将在完成上传时自动执行这些操作； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始分块上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {MultipartUploadFileMultipartEnum} multipart 是否为分块上传标识，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MultipartUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {MultipartUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {MultipartUploadFileRequest} [multipartUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    multipartUploadFile: async (t, a, i, r, s, o, n, d, c, l, h, p, y, u = {}) => {
      f("multipartUploadFile", "libraryId", t), f("multipartUploadFile", "spaceId", a), f("multipartUploadFile", "filePath", i), f("multipartUploadFile", "multipart", r), f("multipartUploadFile", "accessToken", s);
      const A = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), I = new URL(A, x);
      let b;
      e && (b = e.baseOptions);
      const S = { method: "POST", ...b, ...u }, w = {}, R = {};
      r !== void 0 && (R.multipart = r), o !== void 0 && (R.conflict_resolution_strategy = o), n !== void 0 && (R.filesize = n), s !== void 0 && (R.access_token = s), d !== void 0 && (R.user_id = d), l !== void 0 && (R.traffic_limit = l), h !== void 0 && (R.prefer_same_origin = h), p !== void 0 && (R.with_content_cas = p), w["Content-Type"] = "application/json", c != null && (w["x-smh-meta-*"] = String(c)), O(I, R);
      let P = b && b.headers ? b.headers : {};
      return S.headers = { ...w, ...P, ...u.headers }, S.data = $(y, S, e), {
        url: U(I),
        options: S
      };
    },
    /**
     * 用于获取 HTML 格式文档预览。 返回HTML或JPG格式的文档用于预览； 如果文件不属于可预览的文档类型，则会跳转至文件的下载链接。 
     * @summary 获取 HTML 格式文档预览
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {PreviewFilePreviewEnum} preview 文档预览标识，固定值为1
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {string} [type] 文档预览方式，如果设置为 pic 则以 jpg 格式预览文档首页，否则以 html 格式预览文档
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    previewFile: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("previewFile", "libraryId", t), f("previewFile", "spaceId", a), f("previewFile", "filePath", i), f("previewFile", "preview", r);
      const l = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#3".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "GET", ...p, ...c }, u = {}, A = {};
      r !== void 0 && (A.preview = r), s !== void 0 && (A.history_id = s), o !== void 0 && (A.type = o), n !== void 0 && (A.access_token = n), d !== void 0 && (A.user_id = d), O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, {
        url: U(h),
        options: y
      };
    },
    /**
     * 用于分块上传任务续期。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 仅支持分块上传任务的续期。 
     * @summary 分块上传任务续期
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数，指定为开始上传文件时响应体中的 confirmKey 字段的值
     * @param {RenewMultipartUploadRenewEnum} renew 续期标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    renewMultipartUpload: async (t, a, i, r, s, o, n, d = {}) => {
      f("renewMultipartUpload", "libraryId", t), f("renewMultipartUpload", "spaceId", a), f("renewMultipartUpload", "confirmKey", i), f("renewMultipartUpload", "renew", r), f("renewMultipartUpload", "accessToken", s);
      const c = "/api/v1/file/{LibraryId}/{SpaceId}/{ConfirmKey}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{ConfirmKey}", encodeURIComponent(String(i))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "POST", ...h, ...d }, y = {}, u = {};
      r !== void 0 && (u.renew = r), s !== void 0 && (u.access_token = s), o !== void 0 && (u.user_id = o), n !== void 0 && (u.traffic_limit = n), O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于开始简单上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 PUT 简单上传指使用 HTTP PUT 请求上传一个文件，请求体即为文件的内容； 调用该接口将返回一系列用于 PUT 简单上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4； PUT 简单上传时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始简单上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SimpleUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {SimpleUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {SimpleUploadFileRequest} [simpleUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    simpleUploadFile: async (t, a, i, r, s, o, n, d, c, l, h, p, y = {}) => {
      f("simpleUploadFile", "libraryId", t), f("simpleUploadFile", "spaceId", a), f("simpleUploadFile", "filePath", i), f("simpleUploadFile", "accessToken", r);
      const u = "/api/v1/file/{LibraryId}/{SpaceId}/{FilePath}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), A = new URL(u, x);
      let I;
      e && (I = e.baseOptions);
      const b = { method: "PUT", ...I, ...y }, S = {}, w = {};
      s !== void 0 && (w.conflict_resolution_strategy = s), o !== void 0 && (w.filesize = o), r !== void 0 && (w.access_token = r), n !== void 0 && (w.user_id = n), c !== void 0 && (w.traffic_limit = c), l !== void 0 && (w.prefer_same_origin = l), h !== void 0 && (w.with_content_cas = h), S["Content-Type"] = "application/json", d != null && (S["x-smh-meta-*"] = String(d)), O(A, w);
      let R = I && I.headers ? I.headers : {};
      return b.headers = { ...S, ...R, ...y.headers }, b.data = $(p, b, e), {
        url: U(A),
        options: b
      };
    }
  };
}, Y = function(e) {
  const t = An(e);
  return {
    /**
     * 用于取消上传任务。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为删除接口，但因为删除的是上传任务信息，故仍需上传文件的相关权限） 如果上传任务为分块上传任务，那么该请求将同时放弃 COS 中的分块上传任务。 
     * @summary 取消上传任务
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数
     * @param {AbortFileUploadUploadEnum} upload 上传任务标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async abortFileUpload(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.abortFileUpload(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["FileApi.abortFileUpload"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于查询文件删除的原因，可能是用户主动删除或者 quota 超限删除。 要求权限：admin 或 space_admin 
     * @summary 查询文件删除原因
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} inode 文件的 Inode
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async checkFileDeletion(a, i, r, s, o) {
      var l, h;
      const n = await t.checkFileDeletion(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["FileApi.checkFileDeletion"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于检查文件状态
     * @summary 检查文件状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async checkFileStatus(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.checkFileStatus(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["FileApi.checkFileStatus"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于完成上传文件。 要求权限：admin、space_admin、upload_file、upload_file_force 或 confirm_upload。 在文件上传完成后，请务必及时调用该接口，否则文件将不能被正确存储；如果调用该接口时实际并未完成文件上传，将返回错误信息。 
     * @summary 完成上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数，指定为开始上传文件时响应体中的 confirmKey 字段的值
     * @param {CompleteFileUploadConfirmEnum} confirm 完成上传标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CompleteFileUploadConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件；不传则沿用开始上传时的设置
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CompleteFileUploadWithInodeEnum} [withInode] 是否返回 inode（文件目录 ID），0 或 1，默认 0
     * @param {CompleteFileUploadWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {CompleteFileUploadRequest} [completeFileUploadRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async completeFileUpload(a, i, r, s, o, n, d, c, l, h, p, y) {
      var b, S;
      const u = await t.completeFileUpload(a, i, r, s, o, n, d, c, l, h, p, y), A = (e == null ? void 0 : e.serverIndex) ?? 0, I = (S = (b = F["FileApi.completeFileUpload"]) == null ? void 0 : b[A]) == null ? void 0 : S.url;
      return (w, R) => k(u, _, C, e)(w, I || R);
    },
    /**
     * 用于转换文档格式，当前仅支持 doc/docx 转 pdf。 要求权限： 非 acl 鉴权：admin、space_admin acl 鉴权：canDownload（当前文件夹可下载）& canUpload（目标文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 文档转码
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {ConvertFileConvertEnum} convert 文档转码操作标识，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {ConvertFileRequest} convertFileRequest 
     * @param {ConvertFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async convertFile(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.convertFile(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["FileApi.convertFile"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    },
    /**
     * 用于复制文件。 要求权限： admin、space_admin 或 copy_file/copy_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件复制到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 复制文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CopyFileRequest} copyFileRequest 
     * @param {CopyFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CopyFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async copyFile(a, i, r, s, o, n, d, c, l, h) {
      var A, I;
      const p = await t.copyFile(a, i, r, s, o, n, d, c, l, h), y = (e == null ? void 0 : e.serverIndex) ?? 0, u = (I = (A = F["FileApi.copyFile"]) == null ? void 0 : A[y]) == null ? void 0 : I.url;
      return (b, S) => k(p, _, C, e)(b, u || S);
    },
    /**
     * 用于创建符号链接。 要求权限： 非 acl 鉴权：admin、space_admin 或 upload_file/upload_file_force/create_symlink/create_symlink_force acl 鉴权：canUpload（当前文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 符号链接本身与文件的概念一致，可以通过删除文件、重命名或移动文件、复制文件等接口删除、重命名或移动或复制符号链接本身，而不会影响符号链接所指向的文件； 与标准文件系统略有不同，符号链接所指向的文件，不会因为重命名或移动而丢失指向； 当符号链接指向的文件被覆盖上传时，该符号链接将指向新上传的文件。 
     * @summary 创建符号链接
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateSymlinkRequest} createSymlinkRequest 
     * @param {CreateSymlinkConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite覆盖已有文件，默认为 rename
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createSymlink(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.createSymlink(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["FileApi.createSymlink"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于删除文件。 要求权限： admin、space_admin 或 delete_file（未开启回收站或 Permanent 为 0）/delete_file_permanent（开启回收站且 Permanent 为 1） 如果媒体库启用回收站功能，则该接口不会永久删除文件，而是将文件移入回收站，可通过相关接口永久删除或恢复回收站内的文件，或直接清空回收站。 
     * @summary 删除文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {DeleteFilePermanentEnum} [permanent] 当媒体库开启回收站时，则该参数指定将文件移入回收站还是永久删除文件，1: 永久删除，0: 移入回收站，默认为 0
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteFile(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.deleteFile(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["FileApi.deleteFile"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于下载文件。 可以直接在使用文件的参数中指定该 URL，例如对于图片文件可直接在小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等中使用，该接口将自动 302 跳转到真实的图片 URL；视频和文件同理； 
     * @summary 下载文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {DownloadFileContentDispositionEnum} [contentDisposition] 用于设置Content-Disposition响应头，支持 inline 或者 attachment，可选参数，不传默认为inline
     * @param {DownloadFilePurposeEnum} [purpose] 用途，可选参数，可以设置为download或者preview，用于决定是否将该文件加入最近使用文件列表中，如果设置为preview，则会将该文件加入最近使用文件列表中，否则不会加入
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {DownloadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async downloadFile(a, i, r, s, o, n, d, c, l, h, p, y) {
      var b, S;
      const u = await t.downloadFile(a, i, r, s, o, n, d, c, l, h, p, y), A = (e == null ? void 0 : e.serverIndex) ?? 0, I = (S = (b = F["FileApi.downloadFile"]) == null ? void 0 : b[A]) == null ? void 0 : S.url;
      return (w, R) => k(u, _, C, e)(w, I || R);
    },
    /**
     * 用于开始表单上传文件（multipart/form-data）。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 调用该接口将返回一系列用于 form 表单上传（multipart/form-data 格式）和确认上传完成的参数，上传的目标 URL 为 https://{Domain}/，其中 Domain 为响应体中的 domain 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/； form 表单上传时还需要指定一系列额外的信息字段，这些字段的名和值包含在响应体中的 form 字段中，可以在 HTML form 表单中通过隐藏域或通过 JS 相关库、小程序 wx.uploadFile 等指定这些字段； form 表单中的文件字段，其表单字段名固定为 file，且必须作为表单中的最后一项； 在完成实际上传后，上传的目标 URL 将返回 HTTP 204 No Content，由于可能的跨域限制，建议直接通过相关接口的回调来判断是否上传完成，并且在上传完成后及时调用完成上传文件接口，确认上传结果； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始表单上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {FormUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {FormUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {FormUploadFileRequest} [formUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async formUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u) {
      var S, w;
      const A = await t.formUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u), I = (e == null ? void 0 : e.serverIndex) ?? 0, b = (w = (S = F["FileApi.formUploadFile"]) == null ? void 0 : S[I]) == null ? void 0 : w.url;
      return (R, P) => k(A, _, C, e)(R, b || P);
    },
    /**
     * 用于获取照片/视频封面缩略图。 视频封面使用该视频的首帧图片； 针对照片或视频封面，优先使用人脸识别智能缩放裁剪为 {Size}px × {Size}px 大小，如果未识别到人脸则居中缩放裁剪为 {Size}px × {Size}px 大小，如果未指定 {Size} 参数则使用照片或视频封面原图，最后 302 跳转到对应的图片的 URL； 可以直接在使用图片的参数中指定该 URL，例如小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等，该接口将自动 302 跳转到真实的图片 URL； 如果文件不属于可预览的媒体类型，则会跳转至文件的下载链接。 
     * @summary 获取照片/视频封面缩略图
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {number} preview 预览标识，固定值为1
     * @param {number} [size] 缩放大小，优先使用人脸识别智能缩放裁剪为 size×size，未识别到人脸则居中缩放裁剪为 size×size；不传则使用原图
     * @param {number} [scale] 等比例缩放百分比（1-100），当未传 size 时生效
     * @param {number} [widthSize] 缩放宽度，当未传 size 和 scale 时生效；未传高度时，高度按等比例缩放
     * @param {number} [heightSize] 缩放高度，当未传 size 和 scale 时生效；未传宽度时，宽度按等比例缩放
     * @param {number} [frameNumber] 帧数，针对 gif 的降帧处理
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getCover(a, i, r, s, o, n, d, c, l, h, p, y) {
      var b, S;
      const u = await t.getCover(a, i, r, s, o, n, d, c, l, h, p, y), A = (e == null ? void 0 : e.serverIndex) ?? 0, I = (S = (b = F["FileApi.getCover"]) == null ? void 0 : b[A]) == null ? void 0 : S.url;
      return (w, R) => k(u, _, C, e)(w, I || R);
    },
    /**
     * 根据文件 ID 查询文件信息
     * @summary 根据文件ID查询文件信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} inode 文件ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {GetFileInfoByInodeWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getFileInfoByInode(a, i, r, s, o, n) {
      var h, p;
      const d = await t.getFileInfoByInode(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["FileApi.getFileInfoByInode"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于获取文件上传任务状态。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为读接口，但因为读取的是上传任务信息，故仍需上传文件的相关权限） 
     * @summary 获取文件上传任务状态
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数
     * @param {GetFileUploadUploadEnum} upload 上传任务标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getFileUpload(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.getFileUpload(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["FileApi.getFileUpload"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于获取文件下载链接和信息。 要求权限：无 
     * @summary 获取文件下载链接和信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {InfoFileInfoEnum} info 获取文件信息标识
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {InfoFileContentDispositionEnum} [contentDisposition] 用于设置Content-Disposition响应头，支持 inline 或者 attachment，可选参数，不传默认为inline
     * @param {InfoFilePurposeEnum} [purpose] 用途，可选参数，可以设置为download或者preview，用于决定是否将该文件加入最近使用文件列表中，如果设置为preview，则会将该文件加入最近使用文件列表中，否则不会加入
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {InfoFilePreCheckEnum} [preCheck] 是否只用于校验文件是否可预览和下载，设置该参数后返回结果中不包含cosUrl
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {InfoFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async infoFile(a, i, r, s, o, n, d, c, l, h, p, y, u, A) {
      var w, R;
      const I = await t.infoFile(a, i, r, s, o, n, d, c, l, h, p, y, u, A), b = (e == null ? void 0 : e.serverIndex) ?? 0, S = (R = (w = F["FileApi.infoFile"]) == null ? void 0 : w[b]) == null ? void 0 : R.url;
      return (P, B) => k(I, _, C, e)(P, S || B);
    },
    /**
     * 用于重命名或移动文件。 要求权限： admin、space_admin 或 move_file/move_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 重命名或移动文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MoveFileRequest} moveFileRequest 
     * @param {MoveFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {string} [contentCas] 文件内容的Cas标识，可选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {MoveFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async moveFile(a, i, r, s, o, n, d, c, l, h) {
      var A, I;
      const p = await t.moveFile(a, i, r, s, o, n, d, c, l, h), y = (e == null ? void 0 : e.serverIndex) ?? 0, u = (I = (A = F["FileApi.moveFile"]) == null ? void 0 : A[y]) == null ? void 0 : I.url;
      return (b, S) => k(p, _, C, e)(b, u || S);
    },
    /**
     * 用于开始分块上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 分块上传指使用通过 HTTP PUT 请求上传一个文件的分块，通过多次上传完成整个文件的上传，每次请求的请求体为文件内容的单个分块； 调用该接口将返回一系列用于分块上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}?uploadId={UploadId}&partNumber={PartNumber}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，UploadId 为响应体中的 uploadId 字段，PartNumber 为从 1 开始的分块顺序，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4?uploadId=xxx&partNumber=1； 上传每个分块时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 与对象存储 COS 的分块上传不同，SMH 的分块上传无需记录 ETag，也无需在完成上传时传入这些 ETag，只需保证上传分块的连续即可，SMH 将在完成上传时自动执行这些操作； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始分块上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {MultipartUploadFileMultipartEnum} multipart 是否为分块上传标识，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {MultipartUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {MultipartUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {MultipartUploadFileRequest} [multipartUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async multipartUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u, A) {
      var w, R;
      const I = await t.multipartUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u, A), b = (e == null ? void 0 : e.serverIndex) ?? 0, S = (R = (w = F["FileApi.multipartUploadFile"]) == null ? void 0 : w[b]) == null ? void 0 : R.url;
      return (P, B) => k(I, _, C, e)(P, S || B);
    },
    /**
     * 用于获取 HTML 格式文档预览。 返回HTML或JPG格式的文档用于预览； 如果文件不属于可预览的文档类型，则会跳转至文件的下载链接。 
     * @summary 获取 HTML 格式文档预览
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {PreviewFilePreviewEnum} preview 文档预览标识，固定值为1
     * @param {string} [historyId] 历史版本 ID，用于获取不同版本的文件内容，可选参数，不传默认为最新版
     * @param {string} [type] 文档预览方式，如果设置为 pic 则以 jpg 格式预览文档首页，否则以 html 格式预览文档
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async previewFile(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.previewFile(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["FileApi.previewFile"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    },
    /**
     * 用于分块上传任务续期。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 仅支持分块上传任务的续期。 
     * @summary 分块上传任务续期
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} confirmKey 确认参数，指定为开始上传文件时响应体中的 confirmKey 字段的值
     * @param {RenewMultipartUploadRenewEnum} renew 续期标识
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async renewMultipartUpload(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.renewMultipartUpload(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["FileApi.renewMultipartUpload"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于开始简单上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 PUT 简单上传指使用 HTTP PUT 请求上传一个文件，请求体即为文件的内容； 调用该接口将返回一系列用于 PUT 简单上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4； PUT 简单上传时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
     * @summary 开始简单上传文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径｜目录路径，对于多级文件路径，使用斜杠(/)分隔，例如 foo/bar/file.txt；对于根目录，该参数留空
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SimpleUploadFileConflictResolutionStrategyEnum} [conflictResolutionStrategy] 文件名冲突时的处理方式，ask冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename冲突时自动重命名文件，overwrite如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 rename
     * @param {number} [filesize] 上传文件大小，单位为字节（Byte），用于判断剩余空间是否足够
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [xSmhMeta] 自定义元数据，名称以 x-smh-meta- 开头的扩展头，值为字符串
     * @param {number} [trafficLimit] 单链接下载限速，范围100KB/s-100MB/s，单位B
     * @param {boolean} [preferSameOrigin] 是否倾向于保持相同域名，可选参数，可能的值为 true 或 false。此参数仅当上传文件的路径存在同名文件，且 ConflictResolutionStrategy 设置为 rename 或 overwrite 时生效。当设置此参数时，后台会尽量保证新上传的文件与原文件使用相同的域名进行上传或下载，但在特殊情况下仍有可能使用不同域名，因此不应过于依赖此参数。
     * @param {SimpleUploadFileWithContentCasEnum} [withContentCas] 0 或 1，是否返回文件内容的Cas标识，可选，默认不返回
     * @param {SimpleUploadFileRequest} [simpleUploadFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async simpleUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u) {
      var S, w;
      const A = await t.simpleUploadFile(a, i, r, s, o, n, d, c, l, h, p, y, u), I = (e == null ? void 0 : e.serverIndex) ?? 0, b = (w = (S = F["FileApi.simpleUploadFile"]) == null ? void 0 : S[I]) == null ? void 0 : w.url;
      return (R, P) => k(A, _, C, e)(R, b || P);
    }
  };
}, Dt = class extends he {
  /**
   * 用于取消上传任务。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为删除接口，但因为删除的是上传任务信息，故仍需上传文件的相关权限） 如果上传任务为分块上传任务，那么该请求将同时放弃 COS 中的分块上传任务。 
   * @summary 取消上传任务
   * @param {FileApiAbortFileUploadRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  abortFileUpload(e, t) {
    return Y(this.configuration).abortFileUpload(e.libraryId, e.spaceId, e.confirmKey, e.upload, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询文件删除的原因，可能是用户主动删除或者 quota 超限删除。 要求权限：admin 或 space_admin 
   * @summary 查询文件删除原因
   * @param {FileApiCheckFileDeletionRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  checkFileDeletion(e, t) {
    return Y(this.configuration).checkFileDeletion(e.libraryId, e.spaceId, e.inode, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于检查文件状态
   * @summary 检查文件状态
   * @param {FileApiCheckFileStatusRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  checkFileStatus(e, t) {
    return Y(this.configuration).checkFileStatus(e.libraryId, e.spaceId, e.filePath, e.historyId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于完成上传文件。 要求权限：admin、space_admin、upload_file、upload_file_force 或 confirm_upload。 在文件上传完成后，请务必及时调用该接口，否则文件将不能被正确存储；如果调用该接口时实际并未完成文件上传，将返回错误信息。 
   * @summary 完成上传文件
   * @param {FileApiCompleteFileUploadRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  completeFileUpload(e, t) {
    return Y(this.configuration).completeFileUpload(e.libraryId, e.spaceId, e.confirmKey, e.confirm, e.accessToken, e.conflictResolutionStrategy, e.contentCas, e.userId, e.withInode, e.withContentCas, e.completeFileUploadRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于转换文档格式，当前仅支持 doc/docx 转 pdf。 要求权限： 非 acl 鉴权：admin、space_admin acl 鉴权：canDownload（当前文件夹可下载）& canUpload（目标文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 文档转码
   * @param {FileApiConvertFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  convertFile(e, t) {
    return Y(this.configuration).convertFile(e.libraryId, e.spaceId, e.filePath, e.convert, e.accessToken, e.convertFileRequest, e.conflictResolutionStrategy, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于复制文件。 要求权限： admin、space_admin 或 copy_file/copy_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件复制到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 复制文件
   * @param {FileApiCopyFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  copyFile(e, t) {
    return Y(this.configuration).copyFile(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.copyFileRequest, e.conflictResolutionStrategy, e.contentCas, e.userId, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于创建符号链接。 要求权限： 非 acl 鉴权：admin、space_admin 或 upload_file/upload_file_force/create_symlink/create_symlink_force acl 鉴权：canUpload（当前文件夹可上传） 非 acl 鉴权是指当前用户对所有文件的操作权限，详情可参考生成访问令牌接口； acl 鉴权是通过共享授权接口给指定用户，以文件夹为单位授予的权限，详情可参考角色授权模块； 符号链接本身与文件的概念一致，可以通过删除文件、重命名或移动文件、复制文件等接口删除、重命名或移动或复制符号链接本身，而不会影响符号链接所指向的文件； 与标准文件系统略有不同，符号链接所指向的文件，不会因为重命名或移动而丢失指向； 当符号链接指向的文件被覆盖上传时，该符号链接将指向新上传的文件。 
   * @summary 创建符号链接
   * @param {FileApiCreateSymlinkRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createSymlink(e, t) {
    return Y(this.configuration).createSymlink(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.createSymlinkRequest, e.conflictResolutionStrategy, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于删除文件。 要求权限： admin、space_admin 或 delete_file（未开启回收站或 Permanent 为 0）/delete_file_permanent（开启回收站且 Permanent 为 1） 如果媒体库启用回收站功能，则该接口不会永久删除文件，而是将文件移入回收站，可通过相关接口永久删除或恢复回收站内的文件，或直接清空回收站。 
   * @summary 删除文件
   * @param {FileApiDeleteFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteFile(e, t) {
    return Y(this.configuration).deleteFile(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.permanent, e.userId, e.contentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于下载文件。 可以直接在使用文件的参数中指定该 URL，例如对于图片文件可直接在小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等中使用，该接口将自动 302 跳转到真实的图片 URL；视频和文件同理； 
   * @summary 下载文件
   * @param {FileApiDownloadFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  downloadFile(e, t) {
    return Y(this.configuration).downloadFile(e.libraryId, e.spaceId, e.filePath, e.historyId, e.contentDisposition, e.purpose, e.accessToken, e.userId, e.trafficLimit, e.contentCas, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于开始表单上传文件（multipart/form-data）。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 调用该接口将返回一系列用于 form 表单上传（multipart/form-data 格式）和确认上传完成的参数，上传的目标 URL 为 https://{Domain}/，其中 Domain 为响应体中的 domain 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/； form 表单上传时还需要指定一系列额外的信息字段，这些字段的名和值包含在响应体中的 form 字段中，可以在 HTML form 表单中通过隐藏域或通过 JS 相关库、小程序 wx.uploadFile 等指定这些字段； form 表单中的文件字段，其表单字段名固定为 file，且必须作为表单中的最后一项； 在完成实际上传后，上传的目标 URL 将返回 HTTP 204 No Content，由于可能的跨域限制，建议直接通过相关接口的回调来判断是否上传完成，并且在上传完成后及时调用完成上传文件接口，确认上传结果； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 开始表单上传文件
   * @param {FileApiFormUploadFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  formUploadFile(e, t) {
    return Y(this.configuration).formUploadFile(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.conflictResolutionStrategy, e.filesize, e.userId, e.xSmhMeta, e.trafficLimit, e.preferSameOrigin, e.withContentCas, e.formUploadFileRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取照片/视频封面缩略图。 视频封面使用该视频的首帧图片； 针对照片或视频封面，优先使用人脸识别智能缩放裁剪为 {Size}px × {Size}px 大小，如果未识别到人脸则居中缩放裁剪为 {Size}px × {Size}px 大小，如果未指定 {Size} 参数则使用照片或视频封面原图，最后 302 跳转到对应的图片的 URL； 可以直接在使用图片的参数中指定该 URL，例如小程序 <image> 标签、 HTML <img> 标签或小程序 wx.previewImage 接口等，该接口将自动 302 跳转到真实的图片 URL； 如果文件不属于可预览的媒体类型，则会跳转至文件的下载链接。 
   * @summary 获取照片/视频封面缩略图
   * @param {FileApiGetCoverRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getCover(e, t) {
    return Y(this.configuration).getCover(e.libraryId, e.spaceId, e.filePath, e.preview, e.size, e.scale, e.widthSize, e.heightSize, e.frameNumber, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 根据文件 ID 查询文件信息
   * @summary 根据文件ID查询文件信息
   * @param {FileApiGetFileInfoByInodeRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getFileInfoByInode(e, t) {
    return Y(this.configuration).getFileInfoByInode(e.libraryId, e.spaceId, e.inode, e.accessToken, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取文件上传任务状态。 要求权限： admin、space_admin、upload_file、upload_file_force、begin_upload 或 begin_upload_force（注意：虽然本接口为读接口，但因为读取的是上传任务信息，故仍需上传文件的相关权限） 
   * @summary 获取文件上传任务状态
   * @param {FileApiGetFileUploadRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getFileUpload(e, t) {
    return Y(this.configuration).getFileUpload(e.libraryId, e.spaceId, e.confirmKey, e.upload, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取文件下载链接和信息。 要求权限：无 
   * @summary 获取文件下载链接和信息
   * @param {FileApiInfoFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  infoFile(e, t) {
    return Y(this.configuration).infoFile(e.libraryId, e.spaceId, e.filePath, e.info, e.historyId, e.contentDisposition, e.purpose, e.accessToken, e.userId, e.trafficLimit, e.preCheck, e.contentCas, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于重命名或移动文件。 要求权限： admin、space_admin 或 move_file/move_file_force。 该接口的源和目标均需要指定完整的文件路径，源与目标可以跨越目录，来实现将文件移动到任意其他目录下的功能，且支持同时修改文件名； 不会自动创建中间所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 重命名或移动文件
   * @param {FileApiMoveFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  moveFile(e, t) {
    return Y(this.configuration).moveFile(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.moveFileRequest, e.conflictResolutionStrategy, e.contentCas, e.userId, e.withContentCas, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于开始分块上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 分块上传指使用通过 HTTP PUT 请求上传一个文件的分块，通过多次上传完成整个文件的上传，每次请求的请求体为文件内容的单个分块； 调用该接口将返回一系列用于分块上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}?uploadId={UploadId}&partNumber={PartNumber}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，UploadId 为响应体中的 uploadId 字段，PartNumber 为从 1 开始的分块顺序，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4?uploadId=xxx&partNumber=1； 上传每个分块时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 与对象存储 COS 的分块上传不同，SMH 的分块上传无需记录 ETag，也无需在完成上传时传入这些 ETag，只需保证上传分块的连续即可，SMH 将在完成上传时自动执行这些操作； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 开始分块上传文件
   * @param {FileApiMultipartUploadFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  multipartUploadFile(e, t) {
    return Y(this.configuration).multipartUploadFile(e.libraryId, e.spaceId, e.filePath, e.multipart, e.accessToken, e.conflictResolutionStrategy, e.filesize, e.userId, e.xSmhMeta, e.trafficLimit, e.preferSameOrigin, e.withContentCas, e.multipartUploadFileRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取 HTML 格式文档预览。 返回HTML或JPG格式的文档用于预览； 如果文件不属于可预览的文档类型，则会跳转至文件的下载链接。 
   * @summary 获取 HTML 格式文档预览
   * @param {FileApiPreviewFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  previewFile(e, t) {
    return Y(this.configuration).previewFile(e.libraryId, e.spaceId, e.filePath, e.preview, e.historyId, e.type, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于分块上传任务续期。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 仅支持分块上传任务的续期。 
   * @summary 分块上传任务续期
   * @param {FileApiRenewMultipartUploadRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  renewMultipartUpload(e, t) {
    return Y(this.configuration).renewMultipartUpload(e.libraryId, e.spaceId, e.confirmKey, e.renew, e.accessToken, e.userId, e.trafficLimit, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于开始简单上传文件。 要求权限：admin、space_admin 或 upload_file/upload_file_force/begin_upload/begin_upload_force。 PUT 简单上传指使用 HTTP PUT 请求上传一个文件，请求体即为文件的内容； 调用该接口将返回一系列用于 PUT 简单上传请求和确认上传完成的参数，上传的目标 URL 为 https://{Domain}{Path}，其中 Domain 为响应体中的 domain 字段，Path 为响应体中的 path 字段，例如 https://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/smhxxx/xxx.mp4； PUT 简单上传时还需要指定一系列额外的请求头部字段，这些字段的名和值包含在响应体中的 headers 字段中； 当在浏览器使用 JS 上传文件时，需要提前在绑定的 COS 存储桶中设置跨域访问 CORS 设置； 在完成实际上传后，上传的目标 URL 将返回 HTTP 200 OK； 默认情况下同名文件将自动修改文件名，可在完成上传文件接口中获取最终的文件路径； 不会自动创建所需的各级父目录，所以必须保证路径的各级目录存在。 
   * @summary 开始简单上传文件
   * @param {FileApiSimpleUploadFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  simpleUploadFile(e, t) {
    return Y(this.configuration).simpleUploadFile(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.conflictResolutionStrategy, e.filesize, e.userId, e.xSmhMeta, e.trafficLimit, e.preferSameOrigin, e.withContentCas, e.simpleUploadFileRequest, t).then((a) => a(this.axios, this.basePath));
  }
}, In = function(e) {
  return {
    /**
     * 用于删除特定历史版本。权限要求：delete_history权限、admin权限或space_admin权限
     * @summary 删除历史版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<string>} requestBody 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteHistory: async (t, a, i, r, s = {}) => {
      f("deleteHistory", "libraryId", t), f("deleteHistory", "spaceId", a), f("deleteHistory", "accessToken", i), f("deleteHistory", "requestBody", r);
      const o = "/api/v1/directory-history/{LibraryId}/{SpaceId}/delete".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(r, c, e), {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于清空整个library的历史版本，请求此接口时，需要先关闭历史版本。注意：此接口会清空整个library全部文件的历史版本，相应的空间会释放，不可找回数据，请谨慎操作！此接口有频控限制，每分钟最多调用1次，请勿频繁调用。权限要求：admin权限
     * @summary 清空历史版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    emptyHistory: async (t, a, i = {}) => {
      f("emptyHistory", "libraryId", t), f("emptyHistory", "accessToken", a);
      const r = "/api/v1/directory-history/{LibraryId}/library-history".replace("{LibraryId}", encodeURIComponent(String(t))), s = new URL(r, x);
      let o;
      e && (o = e.baseOptions);
      const n = { method: "DELETE", ...o, ...i }, d = {}, c = {};
      a !== void 0 && (c.access_token = a), O(s, c);
      let l = o && o.headers ? o.headers : {};
      return n.headers = { ...d, ...l, ...i.headers }, {
        url: U(s),
        options: n
      };
    },
    /**
     * 用于查询历史版本配置信息。权限要求：admin权限
     * @summary 查询历史版本配置信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getHistoryConfig: async (t, a, i = {}) => {
      f("getHistoryConfig", "libraryId", t), f("getHistoryConfig", "accessToken", a);
      const r = "/api/v1/directory-history/{LibraryId}/library-history".replace("{LibraryId}", encodeURIComponent(String(t))), s = new URL(r, x);
      let o;
      e && (o = e.baseOptions);
      const n = { method: "GET", ...o, ...i }, d = {}, c = {};
      a !== void 0 && (c.access_token = a), O(s, c);
      let l = o && o.headers ? o.headers : {};
      return n.headers = { ...d, ...l, ...i.headers }, {
        url: U(s),
        options: n
      };
    },
    /**
     * 用于查看历史版本列表。
     * @summary 查看历史版本列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径，对于多级目录，使用斜杠(/)分隔，例如 foo/bar.txt
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，默认为 20；若不指定任何翻页参数，默认采用（marker，limit）参数翻页；若与（page，page_size）参数同时使用，默认采用（page，page_size）参数翻页
     * @param {number} [page] 分页码，默认第一页
     * @param {number} [pageSize] 分页大小，默认 20；若与（marker，limit）参数同时使用，默认采用（page，page_size）参数翻页
     * @param {ListHistoryOrderByEnum} [orderBy] 排序字段，按文件 id 排序为 id，按创建时间排序为 creationTime，默认为 id，最新版本排序始终在首位
     * @param {ListHistoryOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc，默认为 desc
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listHistory: async (t, a, i, r, s, o, n, d, c, l, h = {}) => {
      f("listHistory", "libraryId", t), f("listHistory", "spaceId", a), f("listHistory", "filePath", i), f("listHistory", "accessToken", r);
      const p = "/api/v1/directory-history/{LibraryId}/{SpaceId}/history-list/{FilePath}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{FilePath}", encodeURIComponent(String(i))), y = new URL(p, x);
      let u;
      e && (u = e.baseOptions);
      const A = { method: "GET", ...u, ...h }, I = {}, b = {};
      s !== void 0 && (b.marker = s), o !== void 0 && (b.limit = o), n !== void 0 && (b.page = n), d !== void 0 && (b.page_size = d), c !== void 0 && (b.order_by = c), l !== void 0 && (b.order_by_type = l), r !== void 0 && (b.access_token = r), O(y, b);
      let S = u && u.headers ? u.headers : {};
      return A.headers = { ...I, ...S, ...h.headers }, {
        url: U(y),
        options: A
      };
    },
    /**
     * 用于设置历史版本配置信息。权限要求：admin权限。多次调用接口会覆盖之前设置，以最后一次调用为准。更新时，可以设置部分字段；未传入字段，其值保持不变。配置设置生效可能有 1 分钟左右延迟。
     * @summary 设置历史版本配置信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SetHistoryConfigRequest} setHistoryConfigRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    setHistoryConfig: async (t, a, i, r = {}) => {
      f("setHistoryConfig", "libraryId", t), f("setHistoryConfig", "accessToken", a), f("setHistoryConfig", "setHistoryConfigRequest", i);
      const s = "/api/v1/directory-history/{LibraryId}/library-history".replace("{LibraryId}", encodeURIComponent(String(t))), o = new URL(s, x);
      let n;
      e && (n = e.baseOptions);
      const d = { method: "POST", ...n, ...r }, c = {}, l = {};
      a !== void 0 && (l.access_token = a), c["Content-Type"] = "application/json", O(o, l);
      let h = n && n.headers ? n.headers : {};
      return d.headers = { ...c, ...h, ...r.headers }, d.data = $(i, d, e), {
        url: U(o),
        options: d
      };
    },
    /**
     * 用于设置历史版本为最新版本。权限要求：admin权限、space_admin权限或set_history_latest权限
     * @summary 设置历史版本为最新版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} historyId 历史版本 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    setHistoryLatest: async (t, a, i, r, s = {}) => {
      f("setHistoryLatest", "libraryId", t), f("setHistoryLatest", "spaceId", a), f("setHistoryLatest", "historyId", i), f("setHistoryLatest", "accessToken", r);
      const o = "/api/v1/directory-history/{LibraryId}/{SpaceId}/latest-version/{HistoryId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{HistoryId}", encodeURIComponent(String(i))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      r !== void 0 && (h.access_token = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    }
  };
}, He = function(e) {
  const t = In(e);
  return {
    /**
     * 用于删除特定历史版本。权限要求：delete_history权限、admin权限或space_admin权限
     * @summary 删除历史版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<string>} requestBody 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteHistory(a, i, r, s, o) {
      var l, h;
      const n = await t.deleteHistory(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["HistoryApi.deleteHistory"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于清空整个library的历史版本，请求此接口时，需要先关闭历史版本。注意：此接口会清空整个library全部文件的历史版本，相应的空间会释放，不可找回数据，请谨慎操作！此接口有频控限制，每分钟最多调用1次，请勿频繁调用。权限要求：admin权限
     * @summary 清空历史版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async emptyHistory(a, i, r) {
      var d, c;
      const s = await t.emptyHistory(a, i, r), o = (e == null ? void 0 : e.serverIndex) ?? 0, n = (c = (d = F["HistoryApi.emptyHistory"]) == null ? void 0 : d[o]) == null ? void 0 : c.url;
      return (l, h) => k(s, _, C, e)(l, n || h);
    },
    /**
     * 用于查询历史版本配置信息。权限要求：admin权限
     * @summary 查询历史版本配置信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getHistoryConfig(a, i, r) {
      var d, c;
      const s = await t.getHistoryConfig(a, i, r), o = (e == null ? void 0 : e.serverIndex) ?? 0, n = (c = (d = F["HistoryApi.getHistoryConfig"]) == null ? void 0 : d[o]) == null ? void 0 : c.url;
      return (l, h) => k(s, _, C, e)(l, n || h);
    },
    /**
     * 用于查看历史版本列表。
     * @summary 查看历史版本列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} filePath 文件路径，对于多级目录，使用斜杠(/)分隔，例如 foo/bar.txt
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，默认为 20；若不指定任何翻页参数，默认采用（marker，limit）参数翻页；若与（page，page_size）参数同时使用，默认采用（page，page_size）参数翻页
     * @param {number} [page] 分页码，默认第一页
     * @param {number} [pageSize] 分页大小，默认 20；若与（marker，limit）参数同时使用，默认采用（page，page_size）参数翻页
     * @param {ListHistoryOrderByEnum} [orderBy] 排序字段，按文件 id 排序为 id，按创建时间排序为 creationTime，默认为 id，最新版本排序始终在首位
     * @param {ListHistoryOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc，默认为 desc
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listHistory(a, i, r, s, o, n, d, c, l, h, p) {
      var I, b;
      const y = await t.listHistory(a, i, r, s, o, n, d, c, l, h, p), u = (e == null ? void 0 : e.serverIndex) ?? 0, A = (b = (I = F["HistoryApi.listHistory"]) == null ? void 0 : I[u]) == null ? void 0 : b.url;
      return (S, w) => k(y, _, C, e)(S, A || w);
    },
    /**
     * 用于设置历史版本配置信息。权限要求：admin权限。多次调用接口会覆盖之前设置，以最后一次调用为准。更新时，可以设置部分字段；未传入字段，其值保持不变。配置设置生效可能有 1 分钟左右延迟。
     * @summary 设置历史版本配置信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SetHistoryConfigRequest} setHistoryConfigRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async setHistoryConfig(a, i, r, s) {
      var c, l;
      const o = await t.setHistoryConfig(a, i, r, s), n = (e == null ? void 0 : e.serverIndex) ?? 0, d = (l = (c = F["HistoryApi.setHistoryConfig"]) == null ? void 0 : c[n]) == null ? void 0 : l.url;
      return (h, p) => k(o, _, C, e)(h, d || p);
    },
    /**
     * 用于设置历史版本为最新版本。权限要求：admin权限、space_admin权限或set_history_latest权限
     * @summary 设置历史版本为最新版本
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} historyId 历史版本 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async setHistoryLatest(a, i, r, s, o) {
      var l, h;
      const n = await t.setHistoryLatest(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["HistoryApi.setHistoryLatest"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    }
  };
}, mn = class extends he {
  /**
   * 用于删除特定历史版本。权限要求：delete_history权限、admin权限或space_admin权限
   * @summary 删除历史版本
   * @param {HistoryApiDeleteHistoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteHistory(e, t) {
    return He(this.configuration).deleteHistory(e.libraryId, e.spaceId, e.accessToken, e.requestBody, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于清空整个library的历史版本，请求此接口时，需要先关闭历史版本。注意：此接口会清空整个library全部文件的历史版本，相应的空间会释放，不可找回数据，请谨慎操作！此接口有频控限制，每分钟最多调用1次，请勿频繁调用。权限要求：admin权限
   * @summary 清空历史版本
   * @param {HistoryApiEmptyHistoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  emptyHistory(e, t) {
    return He(this.configuration).emptyHistory(e.libraryId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询历史版本配置信息。权限要求：admin权限
   * @summary 查询历史版本配置信息
   * @param {HistoryApiGetHistoryConfigRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getHistoryConfig(e, t) {
    return He(this.configuration).getHistoryConfig(e.libraryId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查看历史版本列表。
   * @summary 查看历史版本列表
   * @param {HistoryApiListHistoryRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listHistory(e, t) {
    return He(this.configuration).listHistory(e.libraryId, e.spaceId, e.filePath, e.accessToken, e.marker, e.limit, e.page, e.pageSize, e.orderBy, e.orderByType, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于设置历史版本配置信息。权限要求：admin权限。多次调用接口会覆盖之前设置，以最后一次调用为准。更新时，可以设置部分字段；未传入字段，其值保持不变。配置设置生效可能有 1 分钟左右延迟。
   * @summary 设置历史版本配置信息
   * @param {HistoryApiSetHistoryConfigRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  setHistoryConfig(e, t) {
    return He(this.configuration).setHistoryConfig(e.libraryId, e.accessToken, e.setHistoryConfigRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于设置历史版本为最新版本。权限要求：admin权限、space_admin权限或set_history_latest权限
   * @summary 设置历史版本为最新版本
   * @param {HistoryApiSetHistoryLatestRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  setHistoryLatest(e, t) {
    return He(this.configuration).setHistoryLatest(e.libraryId, e.spaceId, e.historyId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
}, gn = function(e) {
  return {
    /**
     * 用于创建配额。当在配置了配额的租户空间中上传即将超过配额的文件时，会返回 QuotaLimitReached 错误码；租户空间的剩余空间非实时更新，当系统负荷较高时可能会有比较大的更新延时，进而可能导致意外超出配额，如果配置了超额自动删除选项，可能进一步导致旧文件被删除；配额与租户空间是一对多的关系，即多个租户空间可以共享同一个配额，但每个租户空间只能设置一个配额。
     * @summary 创建配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateQuotaRequest} createQuotaRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createQuota: async (t, a, i, r, s = {}) => {
      f("createQuota", "libraryId", t), f("createQuota", "accessToken", a), f("createQuota", "createQuotaRequest", i);
      const o = "/api/v1/quota/{LibraryId}".replace("{LibraryId}", encodeURIComponent(String(t))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      a !== void 0 && (h.access_token = a), r !== void 0 && (h.user_id = r), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(i, c, e), {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于获取租户空间配额
     * @summary 获取租户空间配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getQuota: async (t, a, i, r, s = {}) => {
      f("getQuota", "libraryId", t), f("getQuota", "spaceId", a), f("getQuota", "accessToken", i);
      const o = "/api/v1/quota/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于获取租户配额信息
     * @summary 获取租户配额信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} quotaId 配额 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getQuotaInfo: async (t, a, i, r, s = {}) => {
      f("getQuotaInfo", "libraryId", t), f("getQuotaInfo", "quotaId", a), f("getQuotaInfo", "accessToken", i);
      const o = "/api/v1/quota/{LibraryId}/{QuotaId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{QuotaId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于修改配额
     * @summary 修改配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {UpdateQuotaRequest} updateQuotaRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    updateQuota: async (t, a, i, r, s, o = {}) => {
      f("updateQuota", "libraryId", t), f("updateQuota", "spaceId", a), f("updateQuota", "accessToken", i), f("updateQuota", "updateQuotaRequest", r);
      const n = "/api/v1/quota/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "PUT", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.access_token = i), s !== void 0 && (p.user_id = s), h["Content-Type"] = "application/json", O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, l.data = $(r, l, e), {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于根据配额 ID 修改配额
     * @summary 修改配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} quotaId 配额 ID，创建配额时会返回，也可以通过【获取租户空间配额】接口查询指定租户空间所在的配额 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {UpdateQuotaByIdRequest} updateQuotaByIdRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    updateQuotaById: async (t, a, i, r, s, o = {}) => {
      f("updateQuotaById", "libraryId", t), f("updateQuotaById", "quotaId", a), f("updateQuotaById", "accessToken", i), f("updateQuotaById", "updateQuotaByIdRequest", r);
      const n = "/api/v1/quota/{LibraryId}/{QuotaId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{QuotaId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "PUT", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.access_token = i), s !== void 0 && (p.user_id = s), h["Content-Type"] = "application/json", O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, l.data = $(r, l, e), {
        url: U(d),
        options: l
      };
    }
  };
}, it = function(e) {
  const t = gn(e);
  return {
    /**
     * 用于创建配额。当在配置了配额的租户空间中上传即将超过配额的文件时，会返回 QuotaLimitReached 错误码；租户空间的剩余空间非实时更新，当系统负荷较高时可能会有比较大的更新延时，进而可能导致意外超出配额，如果配置了超额自动删除选项，可能进一步导致旧文件被删除；配额与租户空间是一对多的关系，即多个租户空间可以共享同一个配额，但每个租户空间只能设置一个配额。
     * @summary 创建配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {CreateQuotaRequest} createQuotaRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createQuota(a, i, r, s, o) {
      var l, h;
      const n = await t.createQuota(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["QuotaApi.createQuota"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于获取租户空间配额
     * @summary 获取租户空间配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getQuota(a, i, r, s, o) {
      var l, h;
      const n = await t.getQuota(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["QuotaApi.getQuota"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于获取租户配额信息
     * @summary 获取租户配额信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} quotaId 配额 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getQuotaInfo(a, i, r, s, o) {
      var l, h;
      const n = await t.getQuotaInfo(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["QuotaApi.getQuotaInfo"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于修改配额
     * @summary 修改配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {UpdateQuotaRequest} updateQuotaRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async updateQuota(a, i, r, s, o, n) {
      var h, p;
      const d = await t.updateQuota(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["QuotaApi.updateQuota"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于根据配额 ID 修改配额
     * @summary 修改配额
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} quotaId 配额 ID，创建配额时会返回，也可以通过【获取租户空间配额】接口查询指定租户空间所在的配额 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {UpdateQuotaByIdRequest} updateQuotaByIdRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async updateQuotaById(a, i, r, s, o, n) {
      var h, p;
      const d = await t.updateQuotaById(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["QuotaApi.updateQuotaById"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    }
  };
}, bn = class extends he {
  /**
   * 用于创建配额。当在配置了配额的租户空间中上传即将超过配额的文件时，会返回 QuotaLimitReached 错误码；租户空间的剩余空间非实时更新，当系统负荷较高时可能会有比较大的更新延时，进而可能导致意外超出配额，如果配置了超额自动删除选项，可能进一步导致旧文件被删除；配额与租户空间是一对多的关系，即多个租户空间可以共享同一个配额，但每个租户空间只能设置一个配额。
   * @summary 创建配额
   * @param {QuotaApiCreateQuotaRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createQuota(e, t) {
    return it(this.configuration).createQuota(e.libraryId, e.accessToken, e.createQuotaRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取租户空间配额
   * @summary 获取租户空间配额
   * @param {QuotaApiGetQuotaRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getQuota(e, t) {
    return it(this.configuration).getQuota(e.libraryId, e.spaceId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于获取租户配额信息
   * @summary 获取租户配额信息
   * @param {QuotaApiGetQuotaInfoRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getQuotaInfo(e, t) {
    return it(this.configuration).getQuotaInfo(e.libraryId, e.quotaId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于修改配额
   * @summary 修改配额
   * @param {QuotaApiUpdateQuotaRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  updateQuota(e, t) {
    return it(this.configuration).updateQuota(e.libraryId, e.spaceId, e.accessToken, e.updateQuotaRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于根据配额 ID 修改配额
   * @summary 修改配额
   * @param {QuotaApiUpdateQuotaByIdRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  updateQuotaById(e, t) {
    return it(this.configuration).updateQuotaById(e.libraryId, e.quotaId, e.accessToken, e.updateQuotaByIdRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
}, vn = function(e) {
  return {
    /**
     * 用于查看最近使用文件列表，仅文件预览及文件编辑操作会被记录到最近使用文件列表中，返回的文件列表按照操作时间进行倒序排列
     * @summary 查看最近使用文件列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {ListRecentlyUsedFileRequest} [listRecentlyUsedFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listRecentlyUsedFile: async (t, a, i, r, s = {}) => {
      f("listRecentlyUsedFile", "libraryId", t), f("listRecentlyUsedFile", "spaceId", a);
      const o = "/api/v1/recent/{LibraryId}/{SpaceId}/recently-used-file".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(r, c, e), {
        url: U(n),
        options: c
      };
    }
  };
}, Sn = function(e) {
  const t = vn(e);
  return {
    /**
     * 用于查看最近使用文件列表，仅文件预览及文件编辑操作会被记录到最近使用文件列表中，返回的文件列表按照操作时间进行倒序排列
     * @summary 查看最近使用文件列表
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {ListRecentlyUsedFileRequest} [listRecentlyUsedFileRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listRecentlyUsedFile(a, i, r, s, o) {
      var l, h;
      const n = await t.listRecentlyUsedFile(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["RecentApi.listRecentlyUsedFile"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    }
  };
}, wn = class extends he {
  /**
   * 用于查看最近使用文件列表，仅文件预览及文件编辑操作会被记录到最近使用文件列表中，返回的文件列表按照操作时间进行倒序排列
   * @summary 查看最近使用文件列表
   * @param {RecentApiListRecentlyUsedFileRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listRecentlyUsedFile(e, t) {
    return Sn(this.configuration).listRecentlyUsedFile(e.libraryId, e.spaceId, e.accessToken, e.listRecentlyUsedFileRequest, t).then((a) => a(this.axios, this.basePath));
  }
}, En = function(e) {
  return {
    /**
     * 用于清空回收站。要求权限：admin、space_admin 或 delete_recycled。调用清空回收站接口时，回收站内的文件将首先在回收站内不可见，删除和释放空间的操作将异步执行。
     * @summary 清空回收站
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleEmpty: async (t, a, i, r, s = {}) => {
      f("recycleEmpty", "libraryId", t), f("recycleEmpty", "spaceId", a), f("recycleEmpty", "accessToken", i);
      const o = "/api/v1/recycled/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "DELETE", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于查看回收站文件详情，以便进行预览
     * @summary 查看回收站文件详情
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站 ID
     * @param {number} info 获取文件详情，固定值为1
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleInfo: async (t, a, i, r, s, o = {}) => {
      f("recycleInfo", "libraryId", t), f("recycleInfo", "spaceId", a), f("recycleInfo", "recycledItemId", i), f("recycleInfo", "info", r);
      const n = "/api/v1/recycled/{LibraryId}/{SpaceId}/{RecycledItemId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{RecycledItemId}", encodeURIComponent(String(i))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "GET", ...c, ...o }, h = {}, p = {};
      r !== void 0 && (p.info = r), s !== void 0 && (p.access_token = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 
     * @summary 列出回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {RecycleListByMarkerEnum} byMarker 固定传 1，表示使用 marker 方式分页
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，不传默认值 20，最大返回 100
     * @param {RecycleListOrderByEnum} [orderBy] 排序字段，按名称排序为 name，按修改时间排序为 modificationTime，按文件大小排序为 size，按删除时间排序为 removalTime，按剩余时间排序为 remainingTime
     * @param {RecycleListOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleList: async (t, a, i, r, s, o, n, d, c, l = {}) => {
      f("recycleList", "libraryId", t), f("recycleList", "spaceId", a), f("recycleList", "byMarker", i);
      const h = "/api/v1/recycled/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), p = new URL(h, x);
      let y;
      e && (y = e.baseOptions);
      const u = { method: "GET", ...y, ...l }, A = {}, I = {};
      i !== void 0 && (I["by-marker"] = i), r !== void 0 && (I.marker = r), s !== void 0 && (I.limit = s), o !== void 0 && (I.order_by = o), n !== void 0 && (I.order_by_type = n), d !== void 0 && (I.access_token = d), c !== void 0 && (I.user_id = c), O(p, I);
      let b = y && y.headers ? y.headers : {};
      return u.headers = { ...A, ...b, ...l.headers }, {
        url: U(p),
        options: u
      };
    },
    /**
     * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 page 翻页的深度会有限制，强烈建议业务方改用 marker 翻页的形式。 
     * @summary 列出回收站项目（by-page）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {RecycleListByPageByPageEnum} byPage 固定传 1，表示使用 page 方式分页
     * @param {number} [page] 分页码，默认第一页，最大翻页的条目数（Page*PageSize 的大小）是 1 万
     * @param {number} [pageSize] 分页大小，默认 20，最大翻页的条目数（Page*PageSize 的大小）是 1 万
     * @param {RecycleListByPageOrderByEnum} [orderBy] 排序字段，按名称排序为 name，按修改时间排序为 modificationTime，按文件大小排序为 size，按删除时间排序为 removalTime，按剩余时间排序为 remainingTime
     * @param {RecycleListByPageOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleListByPage: async (t, a, i, r, s, o, n, d, c, l = {}) => {
      f("recycleListByPage", "libraryId", t), f("recycleListByPage", "spaceId", a), f("recycleListByPage", "byPage", i);
      const h = "/api/v1/recycled/{LibraryId}/{SpaceId}#3".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), p = new URL(h, x);
      let y;
      e && (y = e.baseOptions);
      const u = { method: "GET", ...y, ...l }, A = {}, I = {};
      i !== void 0 && (I["by-page"] = i), r !== void 0 && (I.page = r), s !== void 0 && (I.page_size = s), o !== void 0 && (I.order_by = o), n !== void 0 && (I.order_by_type = n), d !== void 0 && (I.access_token = d), c !== void 0 && (I.user_id = c), O(p, I);
      let b = y && y.headers ? y.headers : {};
      return u.headers = { ...A, ...b, ...l.headers }, {
        url: U(p),
        options: u
      };
    },
    /**
     * 可用于预览文档、图片、视频等文件类型；文档类型可返回HTML或JPG格式；视频返回首帧图片；照片或视频封面支持智能裁剪为指定大小，未识别到人脸时居中缩放裁剪；当未指定 size 参数时使用原图；接口返回302并跳转到可直接用于展示或下载的文件URL。
     * @summary 预览回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站 ID
     * @param {number} preview 预览标志，固定值为1
     * @param {string} [type] 文档类型文件的预览方式，设置为 pic 时以JPG格式预览文档首页，否则以HTML格式预览文档
     * @param {number} [size] 图片或视频封面的缩放大小，优先使用人脸识别智能缩放裁剪为 size×size 大小
     * @param {number} [scale] 图片或视频封面的等比例缩放百分比，不传 size 时生效
     * @param {number} [widthSize] 图片或视频封面的缩放宽度，不传高度时按等比例缩放，不传 size 和 scale 时生效
     * @param {number} [heightSize] 图片或视频封面的缩放高度，不传宽度时按等比例缩放，不传 size 和 scale 时生效
     * @param {number} [frameNumber] gif 文件降帧的帧数，仅在预览 gif 类型文件时生效
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recyclePreview: async (t, a, i, r, s, o, n, d, c, l, h, p = {}) => {
      f("recyclePreview", "libraryId", t), f("recyclePreview", "spaceId", a), f("recyclePreview", "recycledItemId", i), f("recyclePreview", "preview", r);
      const y = "/api/v1/recycled/{LibraryId}/{SpaceId}/{RecycledItemId}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{RecycledItemId}", encodeURIComponent(String(i))), u = new URL(y, x);
      let A;
      e && (A = e.baseOptions);
      const I = { method: "GET", ...A, ...p }, b = {}, S = {};
      r !== void 0 && (S.preview = r), s !== void 0 && (S.type = s), o !== void 0 && (S.size = o), n !== void 0 && (S.scale = n), d !== void 0 && (S.width_size = d), c !== void 0 && (S.height_size = c), l !== void 0 && (S.frame_number = l), h !== void 0 && (S.access_token = h), O(u, S);
      let w = A && A.headers ? A.headers : {};
      return I.headers = { ...b, ...w, ...p.headers }, {
        url: U(u),
        options: I
      };
    },
    /**
     * 用于永久删除指定回收站项目。要求权限：admin、space_admin 或 delete_recycled。
     * @summary 永久删除指定回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站项目 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recyclePurge: async (t, a, i, r, s, o = {}) => {
      f("recyclePurge", "libraryId", t), f("recyclePurge", "spaceId", a), f("recyclePurge", "recycledItemId", i), f("recyclePurge", "accessToken", r);
      const n = "/api/v1/recycled/{LibraryId}/{SpaceId}/{RecycledItemId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{RecycledItemId}", encodeURIComponent(String(i))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "DELETE", ...c, ...o }, h = {}, p = {};
      r !== void 0 && (p.access_token = r), s !== void 0 && (p.user_id = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于永久删除指定回收站项目（批量）。要求权限：admin、space_admin 或 delete_recycled。
     * @summary 永久删除指定回收站项目（批量）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} _delete 永久删除标志，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<number>} recyclePurgeBatchRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recyclePurgeBatch: async (t, a, i, r, s, o, n = {}) => {
      f("recyclePurgeBatch", "libraryId", t), f("recyclePurgeBatch", "spaceId", a), f("recyclePurgeBatch", "_delete", i), f("recyclePurgeBatch", "accessToken", r), f("recyclePurgeBatch", "recyclePurgeBatchRequest", s);
      const d = "/api/v1/recycled/{LibraryId}/{SpaceId}#1".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), c = new URL(d, x);
      let l;
      e && (l = e.baseOptions);
      const h = { method: "POST", ...l, ...n }, p = {}, y = {};
      i !== void 0 && (y.delete = i), r !== void 0 && (y.access_token = r), o !== void 0 && (y.user_id = o), p["Content-Type"] = "application/json", O(c, y);
      let u = l && l.headers ? l.headers : {};
      return h.headers = { ...p, ...u, ...n.headers }, h.data = $(s, h, e), {
        url: U(c),
        options: h
      };
    },
    /**
     * 用于恢复指定回收站项目。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在。
     * @summary 恢复指定回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站项目 ID
     * @param {RecycleRestoreRestoreEnum} restore 固定为 1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {RecycleRestoreConflictResolutionStrategyEnum} [conflictResolutionStrategy] 路径冲突时的处理方式，ask: 冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename: 冲突时自动重命名文件，overwrite: 如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {RecycleRestoreRestorePathStrategyEnum} [restorePathStrategy] 恢复项目源路径的处理方式，originalPath:恢复到原始路径，原始路径不存在则报错; fallbackToRoot:恢复到原始路径，原始路径不存在则恢复到根目录，默认为 originalPath
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleRestore: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("recycleRestore", "libraryId", t), f("recycleRestore", "spaceId", a), f("recycleRestore", "recycledItemId", i), f("recycleRestore", "restore", r), f("recycleRestore", "accessToken", s);
      const l = "/api/v1/recycled/{LibraryId}/{SpaceId}/{RecycledItemId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{RecycledItemId}", encodeURIComponent(String(i))), h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "POST", ...p, ...c }, u = {}, A = {};
      r !== void 0 && (A.restore = r), o !== void 0 && (A.conflict_resolution_strategy = o), s !== void 0 && (A.access_token = s), n !== void 0 && (A.user_id = n), d !== void 0 && (A.restore_path_strategy = d), O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, {
        url: U(h),
        options: y
      };
    },
    /**
     * 用于恢复指定回收站项目（批量）。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在；恢复项目时如果有同名文件存在，则默认重命名文件。
     * @summary 批量恢复回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} restore 恢复，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<number>} recycleRestoreBatchRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {RecycleRestoreBatchRestorePathStrategyEnum} [restorePathStrategy] 恢复项目源路径的处理方式，originalPath:恢复到原始路径，原始路径不存在则报错; fallbackToRoot:恢复到原始路径，原始路径不存在则恢复到根目录，默认为 originalPath
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleRestoreBatch: async (t, a, i, r, s, o, n, d = {}) => {
      f("recycleRestoreBatch", "libraryId", t), f("recycleRestoreBatch", "spaceId", a), f("recycleRestoreBatch", "restore", i), f("recycleRestoreBatch", "accessToken", r), f("recycleRestoreBatch", "recycleRestoreBatchRequest", s);
      const c = "/api/v1/recycled/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), l = new URL(c, x);
      let h;
      e && (h = e.baseOptions);
      const p = { method: "POST", ...h, ...d }, y = {}, u = {};
      i !== void 0 && (u.restore = i), r !== void 0 && (u.access_token = r), o !== void 0 && (u.user_id = o), n !== void 0 && (u.restore_path_strategy = n), y["Content-Type"] = "application/json", O(l, u);
      let A = h && h.headers ? h.headers : {};
      return p.headers = { ...y, ...A, ...d.headers }, p.data = $(s, p, e), {
        url: U(l),
        options: p
      };
    },
    /**
     * 用于设置回收站生命周期。未对租户空间设置时，采用媒体库默认值；当延长保留天数时，已有文件同步使用新值；当缩短保留天数时，已有文件沿用旧值，新删除文件使用新值。
     * @summary 设置回收站生命周期
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} lifecycle 设置回收站生命周期标志，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {RecycleSetLifecycleRequest} recycleSetLifecycleRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    recycleSetLifecycle: async (t, a, i, r, s, o = {}) => {
      f("recycleSetLifecycle", "libraryId", t), f("recycleSetLifecycle", "spaceId", a), f("recycleSetLifecycle", "lifecycle", i), f("recycleSetLifecycle", "accessToken", r), f("recycleSetLifecycle", "recycleSetLifecycleRequest", s);
      const n = "/api/v1/recycled/{LibraryId}/{SpaceId}#2".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "POST", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.lifecycle = i), r !== void 0 && (p.access_token = r), h["Content-Type"] = "application/json", O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, l.data = $(s, l, e), {
        url: U(d),
        options: l
      };
    }
  };
}, be = function(e) {
  const t = En(e);
  return {
    /**
     * 用于清空回收站。要求权限：admin、space_admin 或 delete_recycled。调用清空回收站接口时，回收站内的文件将首先在回收站内不可见，删除和释放空间的操作将异步执行。
     * @summary 清空回收站
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleEmpty(a, i, r, s, o) {
      var l, h;
      const n = await t.recycleEmpty(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["RecycledApi.recycleEmpty"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于查看回收站文件详情，以便进行预览
     * @summary 查看回收站文件详情
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站 ID
     * @param {number} info 获取文件详情，固定值为1
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleInfo(a, i, r, s, o, n) {
      var h, p;
      const d = await t.recycleInfo(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["RecycledApi.recycleInfo"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 
     * @summary 列出回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {RecycleListByMarkerEnum} byMarker 固定传 1，表示使用 marker 方式分页
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，不传默认值 20，最大返回 100
     * @param {RecycleListOrderByEnum} [orderBy] 排序字段，按名称排序为 name，按修改时间排序为 modificationTime，按文件大小排序为 size，按删除时间排序为 removalTime，按剩余时间排序为 remainingTime
     * @param {RecycleListOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleList(a, i, r, s, o, n, d, c, l, h) {
      var A, I;
      const p = await t.recycleList(a, i, r, s, o, n, d, c, l, h), y = (e == null ? void 0 : e.serverIndex) ?? 0, u = (I = (A = F["RecycledApi.recycleList"]) == null ? void 0 : A[y]) == null ? void 0 : I.url;
      return (b, S) => k(p, _, C, e)(b, u || S);
    },
    /**
     * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 page 翻页的深度会有限制，强烈建议业务方改用 marker 翻页的形式。 
     * @summary 列出回收站项目（by-page）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {RecycleListByPageByPageEnum} byPage 固定传 1，表示使用 page 方式分页
     * @param {number} [page] 分页码，默认第一页，最大翻页的条目数（Page*PageSize 的大小）是 1 万
     * @param {number} [pageSize] 分页大小，默认 20，最大翻页的条目数（Page*PageSize 的大小）是 1 万
     * @param {RecycleListByPageOrderByEnum} [orderBy] 排序字段，按名称排序为 name，按修改时间排序为 modificationTime，按文件大小排序为 size，按删除时间排序为 removalTime，按剩余时间排序为 remainingTime
     * @param {RecycleListByPageOrderByTypeEnum} [orderByType] 排序方式，升序为 asc，降序为 desc
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleListByPage(a, i, r, s, o, n, d, c, l, h) {
      var A, I;
      const p = await t.recycleListByPage(a, i, r, s, o, n, d, c, l, h), y = (e == null ? void 0 : e.serverIndex) ?? 0, u = (I = (A = F["RecycledApi.recycleListByPage"]) == null ? void 0 : A[y]) == null ? void 0 : I.url;
      return (b, S) => k(p, _, C, e)(b, u || S);
    },
    /**
     * 可用于预览文档、图片、视频等文件类型；文档类型可返回HTML或JPG格式；视频返回首帧图片；照片或视频封面支持智能裁剪为指定大小，未识别到人脸时居中缩放裁剪；当未指定 size 参数时使用原图；接口返回302并跳转到可直接用于展示或下载的文件URL。
     * @summary 预览回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站 ID
     * @param {number} preview 预览标志，固定值为1
     * @param {string} [type] 文档类型文件的预览方式，设置为 pic 时以JPG格式预览文档首页，否则以HTML格式预览文档
     * @param {number} [size] 图片或视频封面的缩放大小，优先使用人脸识别智能缩放裁剪为 size×size 大小
     * @param {number} [scale] 图片或视频封面的等比例缩放百分比，不传 size 时生效
     * @param {number} [widthSize] 图片或视频封面的缩放宽度，不传高度时按等比例缩放，不传 size 和 scale 时生效
     * @param {number} [heightSize] 图片或视频封面的缩放高度，不传宽度时按等比例缩放，不传 size 和 scale 时生效
     * @param {number} [frameNumber] gif 文件降帧的帧数，仅在预览 gif 类型文件时生效
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recyclePreview(a, i, r, s, o, n, d, c, l, h, p, y) {
      var b, S;
      const u = await t.recyclePreview(a, i, r, s, o, n, d, c, l, h, p, y), A = (e == null ? void 0 : e.serverIndex) ?? 0, I = (S = (b = F["RecycledApi.recyclePreview"]) == null ? void 0 : b[A]) == null ? void 0 : S.url;
      return (w, R) => k(u, _, C, e)(w, I || R);
    },
    /**
     * 用于永久删除指定回收站项目。要求权限：admin、space_admin 或 delete_recycled。
     * @summary 永久删除指定回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站项目 ID
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recyclePurge(a, i, r, s, o, n) {
      var h, p;
      const d = await t.recyclePurge(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["RecycledApi.recyclePurge"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于永久删除指定回收站项目（批量）。要求权限：admin、space_admin 或 delete_recycled。
     * @summary 永久删除指定回收站项目（批量）
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} _delete 永久删除标志，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<number>} recyclePurgeBatchRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recyclePurgeBatch(a, i, r, s, o, n, d) {
      var p, y;
      const c = await t.recyclePurgeBatch(a, i, r, s, o, n, d), l = (e == null ? void 0 : e.serverIndex) ?? 0, h = (y = (p = F["RecycledApi.recyclePurgeBatch"]) == null ? void 0 : p[l]) == null ? void 0 : y.url;
      return (u, A) => k(c, _, C, e)(u, h || A);
    },
    /**
     * 用于恢复指定回收站项目。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在。
     * @summary 恢复指定回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} recycledItemId 回收站项目 ID
     * @param {RecycleRestoreRestoreEnum} restore 固定为 1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {RecycleRestoreConflictResolutionStrategyEnum} [conflictResolutionStrategy] 路径冲突时的处理方式，ask: 冲突时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，rename: 冲突时自动重命名文件，overwrite: 如果冲突目标为目录时返回 HTTP 409 Conflict 及 SameNameDirectoryOrFileExists 错误码，否则覆盖已有文件，默认为 ask
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {RecycleRestoreRestorePathStrategyEnum} [restorePathStrategy] 恢复项目源路径的处理方式，originalPath:恢复到原始路径，原始路径不存在则报错; fallbackToRoot:恢复到原始路径，原始路径不存在则恢复到根目录，默认为 originalPath
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleRestore(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.recycleRestore(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["RecycledApi.recycleRestore"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    },
    /**
     * 用于恢复指定回收站项目（批量）。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在；恢复项目时如果有同名文件存在，则默认重命名文件。
     * @summary 批量恢复回收站项目
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} restore 恢复，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {Array<number>} recycleRestoreBatchRequest 
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {RecycleRestoreBatchRestorePathStrategyEnum} [restorePathStrategy] 恢复项目源路径的处理方式，originalPath:恢复到原始路径，原始路径不存在则报错; fallbackToRoot:恢复到原始路径，原始路径不存在则恢复到根目录，默认为 originalPath
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleRestoreBatch(a, i, r, s, o, n, d, c) {
      var y, u;
      const l = await t.recycleRestoreBatch(a, i, r, s, o, n, d, c), h = (e == null ? void 0 : e.serverIndex) ?? 0, p = (u = (y = F["RecycledApi.recycleRestoreBatch"]) == null ? void 0 : y[h]) == null ? void 0 : u.url;
      return (A, I) => k(l, _, C, e)(A, p || I);
    },
    /**
     * 用于设置回收站生命周期。未对租户空间设置时，采用媒体库默认值；当延长保留天数时，已有文件同步使用新值；当缩短保留天数时，已有文件沿用旧值，新删除文件使用新值。
     * @summary 设置回收站生命周期
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {number} lifecycle 设置回收站生命周期标志，固定值为1
     * @param {string} accessToken 访问令牌，必选参数
     * @param {RecycleSetLifecycleRequest} recycleSetLifecycleRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async recycleSetLifecycle(a, i, r, s, o, n) {
      var h, p;
      const d = await t.recycleSetLifecycle(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["RecycledApi.recycleSetLifecycle"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    }
  };
}, Bn = class extends he {
  /**
   * 用于清空回收站。要求权限：admin、space_admin 或 delete_recycled。调用清空回收站接口时，回收站内的文件将首先在回收站内不可见，删除和释放空间的操作将异步执行。
   * @summary 清空回收站
   * @param {RecycledApiRecycleEmptyRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleEmpty(e, t) {
    return be(this.configuration).recycleEmpty(e.libraryId, e.spaceId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查看回收站文件详情，以便进行预览
   * @summary 查看回收站文件详情
   * @param {RecycledApiRecycleInfoRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleInfo(e, t) {
    return be(this.configuration).recycleInfo(e.libraryId, e.spaceId, e.recycledItemId, e.info, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 
   * @summary 列出回收站项目
   * @param {RecycledApiRecycleListRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleList(e, t) {
    return be(this.configuration).recycleList(e.libraryId, e.spaceId, e.byMarker, e.marker, e.limit, e.orderBy, e.orderByType, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出回收站项目。 目录内容的列出顺序为：默认无排序，根据传入参数 orderBy 和 orderByType 来决定排列顺序。 page 翻页的深度会有限制，强烈建议业务方改用 marker 翻页的形式。 
   * @summary 列出回收站项目（by-page）
   * @param {RecycledApiRecycleListByPageRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleListByPage(e, t) {
    return be(this.configuration).recycleListByPage(e.libraryId, e.spaceId, e.byPage, e.page, e.pageSize, e.orderBy, e.orderByType, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 可用于预览文档、图片、视频等文件类型；文档类型可返回HTML或JPG格式；视频返回首帧图片；照片或视频封面支持智能裁剪为指定大小，未识别到人脸时居中缩放裁剪；当未指定 size 参数时使用原图；接口返回302并跳转到可直接用于展示或下载的文件URL。
   * @summary 预览回收站项目
   * @param {RecycledApiRecyclePreviewRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recyclePreview(e, t) {
    return be(this.configuration).recyclePreview(e.libraryId, e.spaceId, e.recycledItemId, e.preview, e.type, e.size, e.scale, e.widthSize, e.heightSize, e.frameNumber, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于永久删除指定回收站项目。要求权限：admin、space_admin 或 delete_recycled。
   * @summary 永久删除指定回收站项目
   * @param {RecycledApiRecyclePurgeRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recyclePurge(e, t) {
    return be(this.configuration).recyclePurge(e.libraryId, e.spaceId, e.recycledItemId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于永久删除指定回收站项目（批量）。要求权限：admin、space_admin 或 delete_recycled。
   * @summary 永久删除指定回收站项目（批量）
   * @param {RecycledApiRecyclePurgeBatchRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recyclePurgeBatch(e, t) {
    return be(this.configuration).recyclePurgeBatch(e.libraryId, e.spaceId, e._delete, e.accessToken, e.recyclePurgeBatchRequest, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于恢复指定回收站项目。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在。
   * @summary 恢复指定回收站项目
   * @param {RecycledApiRecycleRestoreRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleRestore(e, t) {
    return be(this.configuration).recycleRestore(e.libraryId, e.spaceId, e.recycledItemId, e.restore, e.accessToken, e.conflictResolutionStrategy, e.userId, e.restorePathStrategy, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于恢复指定回收站项目（批量）。要求权限：admin、space_admin 或 restore_recycled。恢复项目时需保证该项目所在的目录存在；恢复项目时如果有同名文件存在，则默认重命名文件。
   * @summary 批量恢复回收站项目
   * @param {RecycledApiRecycleRestoreBatchRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleRestoreBatch(e, t) {
    return be(this.configuration).recycleRestoreBatch(e.libraryId, e.spaceId, e.restore, e.accessToken, e.recycleRestoreBatchRequest, e.userId, e.restorePathStrategy, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于设置回收站生命周期。未对租户空间设置时，采用媒体库默认值；当延长保留天数时，已有文件同步使用新值；当缩短保留天数时，已有文件沿用旧值，新删除文件使用新值。
   * @summary 设置回收站生命周期
   * @param {RecycledApiRecycleSetLifecycleRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  recycleSetLifecycle(e, t) {
    return be(this.configuration).recycleSetLifecycle(e.libraryId, e.spaceId, e.lifecycle, e.accessToken, e.recycleSetLifecycleRequest, t).then((a) => a(this.axios, this.basePath));
  }
}, _n = function(e) {
  return {
    /**
     * 用于搜索目录与文件。 使用本接口搜索时，如果在返回时有部分或全部搜索结果，则返回已搜索出的结果的第一页（每页 20 个），如果暂未搜索到结果则返回空数组，因此该接口实际返回的 contents 数量可能为 0 到 20 之间不等，且是否还有更多搜索结果，不应参考 contents 的数量，而应参考 nextMarker 字段； 当需要获取后续页时，传入marker参数进行翻页； 本接口QPS使用上限为10，此接口不可用于业务的高频操作页面，比如空间首页列表的查询等，如有更大QPS的需求请提工单联系智能媒资托管团队； 
     * @summary 搜索目录与文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [marker] 用于顺序列出分页的标识，可选参数，建议将marker放入请求体中传入
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，可选参数，取值范围[1,100]
     * @param {SearchFsWithFavoriteStatusEnum} [withFavoriteStatus] 0 或 1，是否返回收藏状态，可选，默认不返回
     * @param {SearchFsRequest} [searchFsRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    searchFs: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("searchFs", "libraryId", t), f("searchFs", "spaceId", a), f("searchFs", "accessToken", i);
      const l = "/api/v1/search/{LibraryId}/{SpaceId}/search-fs".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "POST", ...p, ...c }, u = {}, A = {};
      i !== void 0 && (A.access_token = i), r !== void 0 && (A.user_id = r), s !== void 0 && (A.marker = s), o !== void 0 && (A.limit = o), n !== void 0 && (A.with_favorite_status = n), u["Content-Type"] = "application/json", O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, y.data = $(d, y, e), {
        url: U(h),
        options: y
      };
    }
  };
}, Rn = function(e) {
  const t = _n(e);
  return {
    /**
     * 用于搜索目录与文件。 使用本接口搜索时，如果在返回时有部分或全部搜索结果，则返回已搜索出的结果的第一页（每页 20 个），如果暂未搜索到结果则返回空数组，因此该接口实际返回的 contents 数量可能为 0 到 20 之间不等，且是否还有更多搜索结果，不应参考 contents 的数量，而应参考 nextMarker 字段； 当需要获取后续页时，传入marker参数进行翻页； 本接口QPS使用上限为10，此接口不可用于业务的高频操作页面，比如空间首页列表的查询等，如有更大QPS的需求请提工单联系智能媒资托管团队； 
     * @summary 搜索目录与文件
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [marker] 用于顺序列出分页的标识，可选参数，建议将marker放入请求体中传入
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制，可选参数，取值范围[1,100]
     * @param {SearchFsWithFavoriteStatusEnum} [withFavoriteStatus] 0 或 1，是否返回收藏状态，可选，默认不返回
     * @param {SearchFsRequest} [searchFsRequest] 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async searchFs(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.searchFs(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["SearchApi.searchFs"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    }
  };
}, Cn = class extends he {
  /**
   * 用于搜索目录与文件。 使用本接口搜索时，如果在返回时有部分或全部搜索结果，则返回已搜索出的结果的第一页（每页 20 个），如果暂未搜索到结果则返回空数组，因此该接口实际返回的 contents 数量可能为 0 到 20 之间不等，且是否还有更多搜索结果，不应参考 contents 的数量，而应参考 nextMarker 字段； 当需要获取后续页时，传入marker参数进行翻页； 本接口QPS使用上限为10，此接口不可用于业务的高频操作页面，比如空间首页列表的查询等，如有更大QPS的需求请提工单联系智能媒资托管团队； 
   * @summary 搜索目录与文件
   * @param {SearchApiSearchFsRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  searchFs(e, t) {
    return Rn(this.configuration).searchFs(e.libraryId, e.spaceId, e.accessToken, e.userId, e.marker, e.limit, e.withFavoriteStatus, e.searchFsRequest, t).then((a) => a(this.axios, this.basePath));
  }
}, Fn = function(e) {
  return {
    /**
     * 用于创建租户空间。需要 admin 或 create_space 权限，有关权限详情请参见生成访问令牌接口。
     * @summary 创建租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CreateSpaceRequest} [createSpaceRequest] 租户空间的扩展属性
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createSpace: async (t, a, i, r, s = {}) => {
      f("createSpace", "libraryId", t), f("createSpace", "accessToken", a);
      const o = "/api/v1/space/{LibraryId}".replace("{LibraryId}", encodeURIComponent(String(t))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      a !== void 0 && (h.access_token = a), i !== void 0 && (h.user_id = i), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(r, c, e), {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于删除租户空间。 要求权限：admin 或 delete_space 
     * @summary 删除租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {DeleteSpaceForceEnum} [force] 是否强制删除，1:强制删除，不判断space是否为空; 0:非强制删除，不允许删除非空的space
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteSpace: async (t, a, i, r, s, o = {}) => {
      f("deleteSpace", "libraryId", t), f("deleteSpace", "spaceId", a), f("deleteSpace", "accessToken", i);
      const n = "/api/v1/space/{LibraryId}/{SpaceId}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "DELETE", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.access_token = i), r !== void 0 && (p.user_id = r), s !== void 0 && (p.force = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于列出空间首页内容，会忽略目录的层级关系，列出空间下所有文件。 要求权限：read_only 或 space_admin 或 admin 
     * @summary 列出空间首页内容
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {GetContentsViewFilterEnum} filter 筛选方式
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制
     * @param {GetContentsViewOrderByEnum} [orderBy] 排序字段
     * @param {GetContentsViewOrderByTypeEnum} [orderByType] 排序方式
     * @param {boolean} [withPath] 是否返回 path
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [category] 文件自定义的分类
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getContentsView: async (t, a, i, r, s, o, n, d, c, l, h, p = {}) => {
      f("getContentsView", "libraryId", t), f("getContentsView", "spaceId", a), f("getContentsView", "filter", i);
      const y = "/api/v1/space/{LibraryId}/{SpaceId}/contents-view".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), u = new URL(y, x);
      let A;
      e && (A = e.baseOptions);
      const I = { method: "GET", ...A, ...p }, b = {}, S = {};
      r !== void 0 && (S.marker = r), s !== void 0 && (S.limit = s), o !== void 0 && (S.order_by = o), n !== void 0 && (S.order_by_type = n), i !== void 0 && (S.filter = i), d !== void 0 && (S.with_path = d), c !== void 0 && (S.access_token = c), l !== void 0 && (S.user_id = l), h !== void 0 && (S.category = h), O(u, S);
      let w = A && A.headers ? A.headers : {};
      return I.headers = { ...b, ...w, ...p.headers }, {
        url: U(u),
        options: I
      };
    },
    /**
     * 用于空间文件数量统计。 需要拥有 admin 或 space_admin 权限。 
     * @summary 空间文件数量统计
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getFileCountInSpace: async (t, a, i, r = {}) => {
      f("getFileCountInSpace", "libraryId", t), f("getFileCountInSpace", "spaceId", a), f("getFileCountInSpace", "accessToken", i);
      const s = "/api/v1/space/{LibraryId}/{SpaceId}/file-count".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), o = new URL(s, x);
      let n;
      e && (n = e.baseOptions);
      const d = { method: "GET", ...n, ...r }, c = {}, l = {};
      i !== void 0 && (l.access_token = i), O(o, l);
      let h = n && n.headers ? n.headers : {};
      return d.headers = { ...c, ...h, ...r.headers }, {
        url: U(o),
        options: d
      };
    },
    /**
     * 用于查询媒体库中的租户空间数量
     * @summary 查询媒体库租户空间数量
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getLibrarySpaceCount: async (t, a, i, r = {}) => {
      f("getLibrarySpaceCount", "libraryId", t), f("getLibrarySpaceCount", "accessToken", a);
      const s = "/api/v1/space/{LibraryId}/count".replace("{LibraryId}", encodeURIComponent(String(t))), o = new URL(s, x);
      let n;
      e && (n = e.baseOptions);
      const d = { method: "GET", ...n, ...r }, c = {}, l = {};
      a !== void 0 && (l.access_token = a), i !== void 0 && (l.user_id = i), O(o, l);
      let h = n && n.headers ? n.headers : {};
      return d.headers = { ...c, ...h, ...r.headers }, {
        url: U(o),
        options: d
      };
    },
    /**
     * 用于查询租户空间的扩展属性
     * @summary 查询租户空间属性
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getSpaceExtension: async (t, a, i, r, s = {}) => {
      f("getSpaceExtension", "libraryId", t), f("getSpaceExtension", "spaceId", a), f("getSpaceExtension", "accessToken", i);
      const o = "/api/v1/space/{LibraryId}/{SpaceId}/extension".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于查询租户空间大小
     * @summary 查询租户空间大小
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getSpaceSize: async (t, a, i, r, s = {}) => {
      f("getSpaceSize", "libraryId", t), f("getSpaceSize", "spaceId", a), f("getSpaceSize", "accessToken", i);
      const o = "/api/v1/space/{LibraryId}/{SpaceId}/size".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于列出租户空间列表信息。如需列出所有租户空间，需要 admin 或 space_admin 权限，否则仅列出当前访问令牌所代表的用户所创建的租户空间。
     * @summary 列出租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [marker] 用于顺序列出分页的标识。
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制。
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listSpace: async (t, a, i, r, s, o = {}) => {
      f("listSpace", "libraryId", t);
      const n = "/api/v1/space/{LibraryId}/list".replace("{LibraryId}", encodeURIComponent(String(t))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "GET", ...c, ...o }, h = {}, p = {};
      a !== void 0 && (p.access_token = a), i !== void 0 && (p.user_id = i), r !== void 0 && (p.marker = r), s !== void 0 && (p.limit = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于设置租户空间的下载限速，要求权限：admin或space_admin
     * @summary 设置租户空间限速
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SetSpaceTrafficLimitRequest} setSpaceTrafficLimitRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    setSpaceTrafficLimit: async (t, a, i, r, s = {}) => {
      f("setSpaceTrafficLimit", "libraryId", t), f("setSpaceTrafficLimit", "spaceId", a), f("setSpaceTrafficLimit", "accessToken", i), f("setSpaceTrafficLimit", "setSpaceTrafficLimitRequest", r);
      const o = "/api/v1/space/{LibraryId}/{SpaceId}/traffic-limit".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "POST", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), l["Content-Type"] = "application/json", O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, c.data = $(r, c, e), {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于修改租户空间属性。 要求权限：非 acl 鉴权：admin 或 space_admin； acl 鉴权：无权限 
     * @summary 修改租户空间属性
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {UpdateSpaceExtensionRequest} [updateSpaceExtensionRequest] 租户空间的扩展属性
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    updateSpaceExtension: async (t, a, i, r, s, o = {}) => {
      f("updateSpaceExtension", "libraryId", t), f("updateSpaceExtension", "spaceId", a), f("updateSpaceExtension", "accessToken", i);
      const n = "/api/v1/space/{LibraryId}/{SpaceId}/extension".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "POST", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.access_token = i), r !== void 0 && (p.user_id = r), h["Content-Type"] = "application/json", O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, l.data = $(s, l, e), {
        url: U(d),
        options: l
      };
    }
  };
}, ve = function(e) {
  const t = Fn(e);
  return {
    /**
     * 用于创建租户空间。需要 admin 或 create_space 权限，有关权限详情请参见生成访问令牌接口。
     * @summary 创建租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {CreateSpaceRequest} [createSpaceRequest] 租户空间的扩展属性
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createSpace(a, i, r, s, o) {
      var l, h;
      const n = await t.createSpace(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["SpaceApi.createSpace"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于删除租户空间。 要求权限：admin 或 delete_space 
     * @summary 删除租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {DeleteSpaceForceEnum} [force] 是否强制删除，1:强制删除，不判断space是否为空; 0:非强制删除，不允许删除非空的space
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteSpace(a, i, r, s, o, n) {
      var h, p;
      const d = await t.deleteSpace(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["SpaceApi.deleteSpace"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于列出空间首页内容，会忽略目录的层级关系，列出空间下所有文件。 要求权限：read_only 或 space_admin 或 admin 
     * @summary 列出空间首页内容
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {GetContentsViewFilterEnum} filter 筛选方式
     * @param {string} [marker] 用于顺序列出分页的标识
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制
     * @param {GetContentsViewOrderByEnum} [orderBy] 排序字段
     * @param {GetContentsViewOrderByTypeEnum} [orderByType] 排序方式
     * @param {boolean} [withPath] 是否返回 path
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [category] 文件自定义的分类
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getContentsView(a, i, r, s, o, n, d, c, l, h, p, y) {
      var b, S;
      const u = await t.getContentsView(a, i, r, s, o, n, d, c, l, h, p, y), A = (e == null ? void 0 : e.serverIndex) ?? 0, I = (S = (b = F["SpaceApi.getContentsView"]) == null ? void 0 : b[A]) == null ? void 0 : S.url;
      return (w, R) => k(u, _, C, e)(w, I || R);
    },
    /**
     * 用于空间文件数量统计。 需要拥有 admin 或 space_admin 权限。 
     * @summary 空间文件数量统计
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getFileCountInSpace(a, i, r, s) {
      var c, l;
      const o = await t.getFileCountInSpace(a, i, r, s), n = (e == null ? void 0 : e.serverIndex) ?? 0, d = (l = (c = F["SpaceApi.getFileCountInSpace"]) == null ? void 0 : c[n]) == null ? void 0 : l.url;
      return (h, p) => k(o, _, C, e)(h, d || p);
    },
    /**
     * 用于查询媒体库中的租户空间数量
     * @summary 查询媒体库租户空间数量
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getLibrarySpaceCount(a, i, r, s) {
      var c, l;
      const o = await t.getLibrarySpaceCount(a, i, r, s), n = (e == null ? void 0 : e.serverIndex) ?? 0, d = (l = (c = F["SpaceApi.getLibrarySpaceCount"]) == null ? void 0 : c[n]) == null ? void 0 : l.url;
      return (h, p) => k(o, _, C, e)(h, d || p);
    },
    /**
     * 用于查询租户空间的扩展属性
     * @summary 查询租户空间属性
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getSpaceExtension(a, i, r, s, o) {
      var l, h;
      const n = await t.getSpaceExtension(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["SpaceApi.getSpaceExtension"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于查询租户空间大小
     * @summary 查询租户空间大小
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getSpaceSize(a, i, r, s, o) {
      var l, h;
      const n = await t.getSpaceSize(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["SpaceApi.getSpaceSize"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于列出租户空间列表信息。如需列出所有租户空间，需要 admin 或 space_admin 权限，否则仅列出当前访问令牌所代表的用户所创建的租户空间。
     * @summary 列出租户空间
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [marker] 用于顺序列出分页的标识。
     * @param {number} [limit] 用于顺序列出分页时本地列出的项目数限制。
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async listSpace(a, i, r, s, o, n) {
      var h, p;
      const d = await t.listSpace(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["SpaceApi.listSpace"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于设置租户空间的下载限速，要求权限：admin或space_admin
     * @summary 设置租户空间限速
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {SetSpaceTrafficLimitRequest} setSpaceTrafficLimitRequest 
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async setSpaceTrafficLimit(a, i, r, s, o) {
      var l, h;
      const n = await t.setSpaceTrafficLimit(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["SpaceApi.setSpaceTrafficLimit"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于修改租户空间属性。 要求权限：非 acl 鉴权：admin 或 space_admin； acl 鉴权：无权限 
     * @summary 修改租户空间属性
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {UpdateSpaceExtensionRequest} [updateSpaceExtensionRequest] 租户空间的扩展属性
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async updateSpaceExtension(a, i, r, s, o, n) {
      var h, p;
      const d = await t.updateSpaceExtension(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["SpaceApi.updateSpaceExtension"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    }
  };
}, xn = class extends he {
  /**
   * 用于创建租户空间。需要 admin 或 create_space 权限，有关权限详情请参见生成访问令牌接口。
   * @summary 创建租户空间
   * @param {SpaceApiCreateSpaceRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createSpace(e, t) {
    return ve(this.configuration).createSpace(e.libraryId, e.accessToken, e.userId, e.createSpaceRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于删除租户空间。 要求权限：admin 或 delete_space 
   * @summary 删除租户空间
   * @param {SpaceApiDeleteSpaceRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteSpace(e, t) {
    return ve(this.configuration).deleteSpace(e.libraryId, e.spaceId, e.accessToken, e.userId, e.force, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出空间首页内容，会忽略目录的层级关系，列出空间下所有文件。 要求权限：read_only 或 space_admin 或 admin 
   * @summary 列出空间首页内容
   * @param {SpaceApiGetContentsViewRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getContentsView(e, t) {
    return ve(this.configuration).getContentsView(e.libraryId, e.spaceId, e.filter, e.marker, e.limit, e.orderBy, e.orderByType, e.withPath, e.accessToken, e.userId, e.category, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于空间文件数量统计。 需要拥有 admin 或 space_admin 权限。 
   * @summary 空间文件数量统计
   * @param {SpaceApiGetFileCountInSpaceRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getFileCountInSpace(e, t) {
    return ve(this.configuration).getFileCountInSpace(e.libraryId, e.spaceId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询媒体库中的租户空间数量
   * @summary 查询媒体库租户空间数量
   * @param {SpaceApiGetLibrarySpaceCountRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getLibrarySpaceCount(e, t) {
    return ve(this.configuration).getLibrarySpaceCount(e.libraryId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询租户空间的扩展属性
   * @summary 查询租户空间属性
   * @param {SpaceApiGetSpaceExtensionRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getSpaceExtension(e, t) {
    return ve(this.configuration).getSpaceExtension(e.libraryId, e.spaceId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询租户空间大小
   * @summary 查询租户空间大小
   * @param {SpaceApiGetSpaceSizeRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getSpaceSize(e, t) {
    return ve(this.configuration).getSpaceSize(e.libraryId, e.spaceId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于列出租户空间列表信息。如需列出所有租户空间，需要 admin 或 space_admin 权限，否则仅列出当前访问令牌所代表的用户所创建的租户空间。
   * @summary 列出租户空间
   * @param {SpaceApiListSpaceRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  listSpace(e, t) {
    return ve(this.configuration).listSpace(e.libraryId, e.accessToken, e.userId, e.marker, e.limit, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于设置租户空间的下载限速，要求权限：admin或space_admin
   * @summary 设置租户空间限速
   * @param {SpaceApiSetSpaceTrafficLimitRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  setSpaceTrafficLimit(e, t) {
    return ve(this.configuration).setSpaceTrafficLimit(e.libraryId, e.spaceId, e.accessToken, e.setSpaceTrafficLimitRequest, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于修改租户空间属性。 要求权限：非 acl 鉴权：admin 或 space_admin； acl 鉴权：无权限 
   * @summary 修改租户空间属性
   * @param {SpaceApiUpdateSpaceExtensionRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  updateSpaceExtension(e, t) {
    return ve(this.configuration).updateSpaceExtension(e.libraryId, e.spaceId, e.accessToken, e.userId, e.updateSpaceExtensionRequest, t).then((a) => a(this.axios, this.basePath));
  }
}, On = function(e) {
  return {
    /**
     * 用于查询媒体库级别耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
     * @summary 查询媒体库任务接口
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} taskIdList 任务 ID 列表，用逗号分隔，例如 10 或 10,12,13
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    queryLibraryTask: async (t, a, i, r, s = {}) => {
      f("queryLibraryTask", "libraryId", t), f("queryLibraryTask", "taskIdList", a), f("queryLibraryTask", "accessToken", i);
      const o = "/api/v1/task/{LibraryId}/{TaskIdList}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{TaskIdList}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    },
    /**
     * 用于查询耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
     * @summary 查询任务接口
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} taskIdList 任务 ID 列表，用逗号分隔，例如 10 或 10,12,13
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    queryTask: async (t, a, i, r, s, o = {}) => {
      f("queryTask", "libraryId", t), f("queryTask", "spaceId", a), f("queryTask", "taskIdList", i), f("queryTask", "accessToken", r);
      const n = "/api/v1/task/{LibraryId}/{SpaceId}/{TaskIdList}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceId}", encodeURIComponent(String(a))).replace("{TaskIdList}", encodeURIComponent(String(i))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "GET", ...c, ...o }, h = {}, p = {};
      r !== void 0 && (p.access_token = r), s !== void 0 && (p.user_id = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    }
  };
}, Si = function(e) {
  const t = On(e);
  return {
    /**
     * 用于查询媒体库级别耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
     * @summary 查询媒体库任务接口
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} taskIdList 任务 ID 列表，用逗号分隔，例如 10 或 10,12,13
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async queryLibraryTask(a, i, r, s, o) {
      var l, h;
      const n = await t.queryLibraryTask(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["TaskApi.queryLibraryTask"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    },
    /**
     * 用于查询耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
     * @summary 查询任务接口
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceId 空间 ID，如果媒体库为单租户模式，则该参数固定为连字符(-)；如果媒体库为多租户模式，则必须指定该参数
     * @param {string} taskIdList 任务 ID 列表，用逗号分隔，例如 10 或 10,12,13
     * @param {string} accessToken 访问令牌，必选参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async queryTask(a, i, r, s, o, n) {
      var h, p;
      const d = await t.queryTask(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["TaskApi.queryTask"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    }
  };
}, Un = class extends he {
  /**
   * 用于查询媒体库级别耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
   * @summary 查询媒体库任务接口
   * @param {TaskApiQueryLibraryTaskRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  queryLibraryTask(e, t) {
    return Si(this.configuration).queryLibraryTask(e.libraryId, e.taskIdList, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于查询耗时任务执行情况。任务的具体返回请参阅会产生异步任务的接口说明，部分接口会根据任务耗时情况返回同步或异步结果，此时异步结果通常与同步结果的格式保持一致；仅能查询到任务结束时间在最近30天的任务，更早期的任务无法查询；
   * @summary 查询任务接口
   * @param {TaskApiQueryTaskRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  queryTask(e, t) {
    return Si(this.configuration).queryTask(e.libraryId, e.spaceId, e.taskIdList, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
}, kn = function(e) {
  return {
    /**
     * 用于生成调用智能媒资托管服务的访问令牌（Access Token）。
     * @summary 生成访问令牌
     * @param {string} libraryId 媒体库ID，在媒体托管控制台创建媒体库后获取。
     * @param {string} librarySecret 媒体库密钥，在媒体托管控制台创建媒体库后获取。
     * @param {string} [spaceId] 空间ID，可同时指定多个空间ID，使用英文逗号（,）分隔。
     * @param {string} [userId] 用户身份识别，由业务后台自行控制。
     * @param {string} [clientId] 客户端识别，由业务后台自行控制。
     * @param {string} [sessionId] SessionId，由业务后台自行控制。
     * @param {number} [period] 令牌有效时长及每次使用令牌后自动续期的有效时长，单位为秒。
     * @param {CreateTokenGrantEnum} [grant] 授予的权限，如为空则只授予读取权限。
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createToken: async (t, a, i, r, s, o, n, d, c = {}) => {
      f("createToken", "libraryId", t), f("createToken", "librarySecret", a);
      const l = "/api/v1/token", h = new URL(l, x);
      let p;
      e && (p = e.baseOptions);
      const y = { method: "GET", ...p, ...c }, u = {}, A = {};
      t !== void 0 && (A.library_id = t), a !== void 0 && (A.library_secret = a), i !== void 0 && (A.space_id = i), r !== void 0 && (A.user_id = r), s !== void 0 && (A.client_id = s), o !== void 0 && (A.session_id = o), n !== void 0 && (A.period = n), d !== void 0 && (A.grant = d), O(h, A);
      let I = p && p.headers ? p.headers : {};
      return y.headers = { ...u, ...I, ...c.headers }, {
        url: U(h),
        options: y
      };
    },
    /**
     * 用于删除指定的访问令牌（Access Token）。删除指定访问令牌无需校验媒体库密钥，故可在客户端调用该接口。
     * @summary 删除访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteToken: async (t, a, i = {}) => {
      f("deleteToken", "libraryId", t), f("deleteToken", "accessToken", a);
      const r = "/api/v1/token/{LibraryId}/{AccessToken}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{AccessToken}", encodeURIComponent(String(a))), s = new URL(r, x);
      let o;
      e && (o = e.baseOptions);
      const n = { method: "DELETE", ...o, ...i }, d = {};
      O(s, {});
      let l = o && o.headers ? o.headers : {};
      return n.headers = { ...d, ...l, ...i.headers }, {
        url: U(s),
        options: n
      };
    },
    /**
     * 用于删除特定用户的所有访问令牌（Access Token）。调用该接口需要用到媒体库密钥，所以必须在后端调用该接口以保证密钥安全；必须指定 UserId 参数，因此在创建访问令牌时，如果后续计划主动删除对应的访问令牌，则在创建时也需要指定 UserId；
     * @summary 删除特定用户的所有访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} librarySecret 媒体库密钥
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [clientId] 客户端识别，多个 ClientId 用英文逗号分隔，一次最多不超过 100 个
     * @param {string} [sessionId] 会话识别，多个 SessionId 用英文逗号分隔，一次最多不超过 100 个
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteUserTokens: async (t, a, i, r, s, o = {}) => {
      f("deleteUserTokens", "libraryId", t), f("deleteUserTokens", "librarySecret", a);
      const n = "/api/v1/token/{LibraryId}".replace("{LibraryId}", encodeURIComponent(String(t))), d = new URL(n, x);
      let c;
      e && (c = e.baseOptions);
      const l = { method: "DELETE", ...c, ...o }, h = {}, p = {};
      i !== void 0 && (p.user_id = i), a !== void 0 && (p.library_secret = a), r !== void 0 && (p.client_id = r), s !== void 0 && (p.session_id = s), O(d, p);
      let y = c && c.headers ? c.headers : {};
      return l.headers = { ...h, ...y, ...o.headers }, {
        url: U(d),
        options: l
      };
    },
    /**
     * 用于续期访问令牌（Access Token）。续期时不支持指定新的有效时长，仅按照获取令牌时指定的有效时长续期。
     * @summary 续期访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    renewToken: async (t, a, i = {}) => {
      f("renewToken", "libraryId", t), f("renewToken", "accessToken", a);
      const r = "/api/v1/token/{LibraryId}/{AccessToken}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{AccessToken}", encodeURIComponent(String(a))), s = new URL(r, x);
      let o;
      e && (o = e.baseOptions);
      const n = { method: "POST", ...o, ...i }, d = {};
      O(s, {});
      let l = o && o.headers ? o.headers : {};
      return n.headers = { ...d, ...l, ...i.headers }, {
        url: U(s),
        options: n
      };
    }
  };
}, vt = function(e) {
  const t = kn(e);
  return {
    /**
     * 用于生成调用智能媒资托管服务的访问令牌（Access Token）。
     * @summary 生成访问令牌
     * @param {string} libraryId 媒体库ID，在媒体托管控制台创建媒体库后获取。
     * @param {string} librarySecret 媒体库密钥，在媒体托管控制台创建媒体库后获取。
     * @param {string} [spaceId] 空间ID，可同时指定多个空间ID，使用英文逗号（,）分隔。
     * @param {string} [userId] 用户身份识别，由业务后台自行控制。
     * @param {string} [clientId] 客户端识别，由业务后台自行控制。
     * @param {string} [sessionId] SessionId，由业务后台自行控制。
     * @param {number} [period] 令牌有效时长及每次使用令牌后自动续期的有效时长，单位为秒。
     * @param {CreateTokenGrantEnum} [grant] 授予的权限，如为空则只授予读取权限。
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async createToken(a, i, r, s, o, n, d, c, l) {
      var u, A;
      const h = await t.createToken(a, i, r, s, o, n, d, c, l), p = (e == null ? void 0 : e.serverIndex) ?? 0, y = (A = (u = F["TokenApi.createToken"]) == null ? void 0 : u[p]) == null ? void 0 : A.url;
      return (I, b) => k(h, _, C, e)(I, y || b);
    },
    /**
     * 用于删除指定的访问令牌（Access Token）。删除指定访问令牌无需校验媒体库密钥，故可在客户端调用该接口。
     * @summary 删除访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteToken(a, i, r) {
      var d, c;
      const s = await t.deleteToken(a, i, r), o = (e == null ? void 0 : e.serverIndex) ?? 0, n = (c = (d = F["TokenApi.deleteToken"]) == null ? void 0 : d[o]) == null ? void 0 : c.url;
      return (l, h) => k(s, _, C, e)(l, n || h);
    },
    /**
     * 用于删除特定用户的所有访问令牌（Access Token）。调用该接口需要用到媒体库密钥，所以必须在后端调用该接口以保证密钥安全；必须指定 UserId 参数，因此在创建访问令牌时，如果后续计划主动删除对应的访问令牌，则在创建时也需要指定 UserId；
     * @summary 删除特定用户的所有访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} librarySecret 媒体库密钥
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {string} [clientId] 客户端识别，多个 ClientId 用英文逗号分隔，一次最多不超过 100 个
     * @param {string} [sessionId] 会话识别，多个 SessionId 用英文逗号分隔，一次最多不超过 100 个
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async deleteUserTokens(a, i, r, s, o, n) {
      var h, p;
      const d = await t.deleteUserTokens(a, i, r, s, o, n), c = (e == null ? void 0 : e.serverIndex) ?? 0, l = (p = (h = F["TokenApi.deleteUserTokens"]) == null ? void 0 : h[c]) == null ? void 0 : p.url;
      return (y, u) => k(d, _, C, e)(y, l || u);
    },
    /**
     * 用于续期访问令牌（Access Token）。续期时不支持指定新的有效时长，仅按照获取令牌时指定的有效时长续期。
     * @summary 续期访问令牌
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} accessToken 访问令牌
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async renewToken(a, i, r) {
      var d, c;
      const s = await t.renewToken(a, i, r), o = (e == null ? void 0 : e.serverIndex) ?? 0, n = (c = (d = F["TokenApi.renewToken"]) == null ? void 0 : d[o]) == null ? void 0 : c.url;
      return (l, h) => k(s, _, C, e)(l, n || h);
    }
  };
}, Vn = class extends he {
  /**
   * 用于生成调用智能媒资托管服务的访问令牌（Access Token）。
   * @summary 生成访问令牌
   * @param {TokenApiCreateTokenRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  createToken(e, t) {
    return vt(this.configuration).createToken(e.libraryId, e.librarySecret, e.spaceId, e.userId, e.clientId, e.sessionId, e.period, e.grant, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于删除指定的访问令牌（Access Token）。删除指定访问令牌无需校验媒体库密钥，故可在客户端调用该接口。
   * @summary 删除访问令牌
   * @param {TokenApiDeleteTokenRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteToken(e, t) {
    return vt(this.configuration).deleteToken(e.libraryId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于删除特定用户的所有访问令牌（Access Token）。调用该接口需要用到媒体库密钥，所以必须在后端调用该接口以保证密钥安全；必须指定 UserId 参数，因此在创建访问令牌时，如果后续计划主动删除对应的访问令牌，则在创建时也需要指定 UserId；
   * @summary 删除特定用户的所有访问令牌
   * @param {TokenApiDeleteUserTokensRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  deleteUserTokens(e, t) {
    return vt(this.configuration).deleteUserTokens(e.libraryId, e.librarySecret, e.userId, e.clientId, e.sessionId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于续期访问令牌（Access Token）。续期时不支持指定新的有效时长，仅按照获取令牌时指定的有效时长续期。
   * @summary 续期访问令牌
   * @param {TokenApiRenewTokenRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  renewToken(e, t) {
    return vt(this.configuration).renewToken(e.libraryId, e.accessToken, t).then((a) => a(this.axios, this.basePath));
  }
}, Qn = function(e) {
  return {
    /**
     * 用于查询媒体库级别的容量信息。 要求权限：admin 
     * @summary 查询媒体库容量信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getLibraryUsage: async (t, a, i, r = {}) => {
      f("getLibraryUsage", "libraryId", t);
      const s = "/api/v1/usage/{LibraryId}".replace("{LibraryId}", encodeURIComponent(String(t))), o = new URL(s, x);
      let n;
      e && (n = e.baseOptions);
      const d = { method: "GET", ...n, ...r }, c = {}, l = {};
      a !== void 0 && (l.access_token = a), i !== void 0 && (l.user_id = i), O(o, l);
      let h = n && n.headers ? n.headers : {};
      return d.headers = { ...c, ...h, ...r.headers }, {
        url: U(o),
        options: d
      };
    },
    /**
     * 用于批量查询列出租户空间容量信息。 要求权限：admin 或 space_admin 如果要查询任意空间的容量信息则需要 admin 权限，如果是 space_admin 权限，则只能查询访问令牌指定的租户空间的容量信息 
     * @summary 批量查询列出租户空间容量信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceIds 空间列表，以逗号分隔，如 space1,space2
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getUsage: async (t, a, i, r, s = {}) => {
      f("getUsage", "libraryId", t), f("getUsage", "spaceIds", a);
      const o = "/api/v1/usage/{LibraryId}/{SpaceIds}".replace("{LibraryId}", encodeURIComponent(String(t))).replace("{SpaceIds}", encodeURIComponent(String(a))), n = new URL(o, x);
      let d;
      e && (d = e.baseOptions);
      const c = { method: "GET", ...d, ...s }, l = {}, h = {};
      i !== void 0 && (h.access_token = i), r !== void 0 && (h.user_id = r), O(n, h);
      let p = d && d.headers ? d.headers : {};
      return c.headers = { ...l, ...p, ...s.headers }, {
        url: U(n),
        options: c
      };
    }
  };
}, wi = function(e) {
  const t = Qn(e);
  return {
    /**
     * 用于查询媒体库级别的容量信息。 要求权限：admin 
     * @summary 查询媒体库容量信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getLibraryUsage(a, i, r, s) {
      var c, l;
      const o = await t.getLibraryUsage(a, i, r, s), n = (e == null ? void 0 : e.serverIndex) ?? 0, d = (l = (c = F["UsageApi.getLibraryUsage"]) == null ? void 0 : c[n]) == null ? void 0 : l.url;
      return (h, p) => k(o, _, C, e)(h, d || p);
    },
    /**
     * 用于批量查询列出租户空间容量信息。 要求权限：admin 或 space_admin 如果要查询任意空间的容量信息则需要 admin 权限，如果是 space_admin 权限，则只能查询访问令牌指定的租户空间的容量信息 
     * @summary 批量查询列出租户空间容量信息
     * @param {string} libraryId 媒体库 ID，必选参数
     * @param {string} spaceIds 空间列表，以逗号分隔，如 space1,space2
     * @param {string} [accessToken] 访问令牌，对于公有读媒体库或租户空间，可不指定该参数，否则必须指定该参数
     * @param {string} [userId] 用户身份识别，当访问令牌对应的权限为管理员权限且申请访问令牌时的用户身份识别为空时用来临时指定用户身份，详情请参阅生成访问令牌接口，可选参数
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    async getUsage(a, i, r, s, o) {
      var l, h;
      const n = await t.getUsage(a, i, r, s, o), d = (e == null ? void 0 : e.serverIndex) ?? 0, c = (h = (l = F["UsageApi.getUsage"]) == null ? void 0 : l[d]) == null ? void 0 : h.url;
      return (p, y) => k(n, _, C, e)(p, c || y);
    }
  };
}, Dn = class extends he {
  /**
   * 用于查询媒体库级别的容量信息。 要求权限：admin 
   * @summary 查询媒体库容量信息
   * @param {UsageApiGetLibraryUsageRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getLibraryUsage(e, t) {
    return wi(this.configuration).getLibraryUsage(e.libraryId, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
  /**
   * 用于批量查询列出租户空间容量信息。 要求权限：admin 或 space_admin 如果要查询任意空间的容量信息则需要 admin 权限，如果是 space_admin 权限，则只能查询访问令牌指定的租户空间的容量信息 
   * @summary 批量查询列出租户空间容量信息
   * @param {UsageApiGetUsageRequest} requestParameters Request parameters.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   */
  getUsage(e, t) {
    return wi(this.configuration).getUsage(e.libraryId, e.spaceIds, e.accessToken, e.userId, t).then((a) => a(this.axios, this.basePath));
  }
}, Nn = class {
  constructor(e = {}) {
    var t;
    this.apiKey = e.apiKey, this.username = e.username, this.password = e.password, this.accessToken = e.accessToken, this.awsv4 = e.awsv4, this.basePath = e.basePath, this.serverIndex = e.serverIndex, this.baseOptions = {
      ...e.baseOptions,
      headers: {
        ...(t = e.baseOptions) == null ? void 0 : t.headers
      }
    }, this.formDataCtor = e.formDataCtor;
  }
  /**
   * Check if the given MIME is a JSON MIME.
   * JSON MIME examples:
   *   application/json
   *   application/json; charset=UTF8
   *   APPLICATION/JSON
   *   application/vnd.company+json
   * @param mime - MIME (Multipurpose Internet Mail Extensions)
   * @return True if the given MIME is JSON, false otherwise.
   */
  isJsonMime(e) {
    const t = new RegExp("^(application/json|[^;/ 	]+/[^;/ 	]+[+]json)[ 	]*(;.*)?$", "i");
    return e !== null && (t.test(e) || e.toLowerCase() === "application/json-patch+json");
  }
}, Tn = "1.0.5", Ln = Tn, Pn = "smh-js-sdk", zn = () => `${Pn}/${Ln}`;
function le(e) {
  if (e === 0) return "0 B";
  const t = 1024, a = ["B", "KB", "MB", "GB", "TB"], i = Math.floor(Math.log(e) / Math.log(t));
  return (e / Math.pow(t, i)).toFixed(2) + " " + a[i];
}
function cr(e) {
  const t = Math.floor(e / 1e3), a = Math.floor(t / 60), i = Math.floor(a / 60);
  return i > 0 ? `${i}h ${a % 60}m ${t % 60}s` : a > 0 ? `${a}m ${t % 60}s` : `${t}s`;
}
var St = /* @__PURE__ */ Symbol("getList"), Mn = class {
  constructor() {
    this.listeners = {};
  }
  [St](e) {
    return this.listeners[e] || (this.listeners[e] = []), this.listeners[e];
  }
  on(e, t) {
    return this[St](e).push(t), this;
  }
  once(e, t) {
    if (!t) return this;
    const a = t;
    return a.once = !0, this.on(e, a), this;
  }
  off(e, t) {
    const a = this[St](e);
    if (t === "*")
      for (let i = a.length - 1; i >= 0; i -= 1)
        a.splice(i, 1);
    else
      for (let i = a.length - 1; i >= 0; i -= 1)
        t === a[i] && a.splice(i, 1);
    return this;
  }
  emit(e, t) {
    const a = this[St](e).map((i) => i);
    for (let i = 0; i < a.length; i += 1) {
      const r = a[i];
      r(t), a[i].once && this.off(e, a[i]);
    }
    return this;
  }
}, Hn = Mn;
function Nt(e) {
  var i, r;
  const t = ((i = e == null ? void 0 : e.response) == null ? void 0 : i.status) || (e == null ? void 0 : e.status) || (e == null ? void 0 : e.statusCode), a = t === 403 || ((r = e == null ? void 0 : e.message) == null ? void 0 : r.includes("Request has expired"));
  return {
    statusCode: t,
    isExpired: a
  };
}
var ma = /* @__PURE__ */ ((e) => (e.FILE_NOT_FOUND = "FileNotFound", e.FILE_MODIFIED = "FileModified", e.FILE_SIZE_MISMATCH = "FileSizeMismatch", e.FILE_CRC64_MISMATCH = "FileCrc64Mismatch", e.FILE_TOO_LARGE = "FileTooLarge", e.INVALID_FILE = "InvalidFile", e.UPLOAD_FAILED = "UploadFailed", e.UPLOAD_CANCELED = "UploadCanceled", e.UPLOAD_PAUSED = "UploadPaused", e.PART_UPLOAD_FAILED = "PartUploadFailed", e.RENEW_UPLOAD_FAILED = "RenewUploadFailed", e.DOWNLOAD_FAILED = "DownloadFailed", e.DOWNLOAD_CANCELED = "DownloadCanceled", e.DOWNLOAD_PAUSED = "DownloadPaused", e.INVALID_PARAMETER = "InvalidParameter", e.NETWORK_ERROR = "NetworkError", e.REQUEST_TIMEOUT = "RequestTimeout", e.OPERATION_FAILED = "OperationFailed", e))(ma || {}), Jt = class ga extends Error {
  constructor(t, a, i, r) {
    let s;
    typeof t == "string" ? Object.values(ma).includes(t) ? s = a || t : s = t : t instanceof Error || typeof t == "object" && "message" in t ? s = a || t.message : s = a || "Unknown error", super(s), this.name = "SMHError", this.response = {}, this.name = "SMHError", Object.setPrototypeOf(this, ga.prototype), this.timestamp = Date.now(), typeof t == "string" && Object.values(ma).includes(t) ? (this.code = t, this.cause = i, r && Object.assign(this.response, r)) : t instanceof ga ? (this.code = t.code, this.status = t.status, this.reqId = t.reqId, this.cause = t.cause, Object.assign(this.response, t.response)) : t instanceof Error ? (this.code = "OperationFailed", this.cause = t) : typeof t == "object" && "code" in t ? (this.code = t.code, this.status = t.status, this.reqId = t.reqId, this.cause = t.cause, t.response && Object.assign(this.response, t.response)) : this.code = "OperationFailed", r && Object.assign(this.response, r), this.cause && this.cause.stack && (this.stack = `${this.stack}
Caused by: ${this.cause.stack}`);
  }
  /**
   * 转换为日志字符串
   */
  toLogString() {
    const t = [];
    if (t.push(`[${this.code}] ${this.message}`), this.status && t.push(`Status: ${this.status}`), this.reqId && t.push(`ReqId: ${this.reqId}`), Object.keys(this.response).length > 0) {
      t.push("Response:");
      for (const [a, i] of Object.entries(this.response))
        typeof i == "object" ? t.push(`  ${a}: ${JSON.stringify(i)}`) : t.push(`  ${a}: ${i}`);
    }
    return this.cause && t.push(`Caused by: ${this.cause.message}`), t.join(`
`);
  }
  /**
   * 转换为 JSON 格式
   */
  toJSON() {
    return {
      name: this.name,
      code: this.code,
      message: this.message,
      status: this.status,
      reqId: this.reqId,
      response: this.response,
      timestamp: this.timestamp,
      cause: this.cause ? {
        name: this.cause.name,
        message: this.cause.message
      } : void 0
    };
  }
};
function X(e, t, a, i) {
  return new Jt(e, t, a, i);
}
async function lr(e, t, a, i) {
  const r = new Array(e.length);
  let s = 0, o = 0, n = null;
  const d = (i == null ? void 0 : i.shouldStop) || (() => !1);
  return new Promise((c, l) => {
    let h = 0;
    const p = () => {
      for (; o < t && s < e.length && !n && !d(); ) {
        const y = s++, u = e[y];
        o++, (async () => {
          try {
            if (d() || n)
              return;
            r[y] = await a(u);
          } catch (A) {
            n || (n = A);
          } finally {
            o--, h++, n ? o === 0 && l(n) : h === e.length ? c(r) : d() ? o === 0 && c(r) : p();
          }
        })();
      }
      (e.length === 0 || o === 0 && s === 0 && d()) && c(r);
    };
    p();
  });
}
function Ut(e, t, a, i) {
  function r(s) {
    return s instanceof a ? s : new a(function(o) {
      o(s);
    });
  }
  return new (a || (a = Promise))(function(s, o) {
    function n(l) {
      try {
        c(i.next(l));
      } catch (h) {
        o(h);
      }
    }
    function d(l) {
      try {
        c(i.throw(l));
      } catch (h) {
        o(h);
      }
    }
    function c(l) {
      l.done ? s(l.value) : r(l.value).then(n, d);
    }
    c((i = i.apply(e, [])).next());
  });
}
var J = class {
  constructor() {
    this.mutex = Promise.resolve();
  }
  lock() {
    let e = () => {
    };
    return this.mutex = this.mutex.then(() => new Promise(e)), new Promise((t) => {
      e = t;
    });
  }
  dispatch(e) {
    return Ut(this, void 0, void 0, function* () {
      const t = yield this.lock();
      try {
        return yield Promise.resolve(e());
      } finally {
        t();
      }
    });
  }
}, na;
function $n() {
  return typeof globalThis < "u" ? globalThis : typeof self < "u" ? self : typeof window < "u" ? window : global;
}
var ba = $n(), ca = (na = ba.Buffer) !== null && na !== void 0 ? na : null, jn = ba.TextEncoder ? new ba.TextEncoder() : null;
function dr(e, t) {
  return (e & 15) + (e >> 6 | e >> 3 & 8) << 4 | (t & 15) + (t >> 6 | t >> 3 & 8);
}
function Gn(e, t) {
  const a = t.length >> 1;
  for (let i = 0; i < a; i++) {
    const r = i << 1;
    e[i] = dr(t.charCodeAt(r), t.charCodeAt(r + 1));
  }
}
function Jn(e, t) {
  if (e.length !== t.length * 2)
    return !1;
  for (let a = 0; a < t.length; a++) {
    const i = a << 1;
    if (t[a] !== dr(e.charCodeAt(i), e.charCodeAt(i + 1)))
      return !1;
  }
  return !0;
}
var Ei = 87, Bi = 48;
function _i(e, t, a) {
  let i = 0;
  for (let r = 0; r < a; r++) {
    let s = t[r] >>> 4;
    e[i++] = s > 9 ? s + Ei : s + Bi, s = t[r] & 15, e[i++] = s > 9 ? s + Ei : s + Bi;
  }
  return String.fromCharCode.apply(null, e);
}
var Ri = ca !== null ? (e) => {
  if (typeof e == "string") {
    const t = ca.from(e, "utf8");
    return new Uint8Array(t.buffer, t.byteOffset, t.length);
  }
  if (ca.isBuffer(e))
    return new Uint8Array(e.buffer, e.byteOffset, e.length);
  if (ArrayBuffer.isView(e))
    return new Uint8Array(e.buffer, e.byteOffset, e.byteLength);
  throw new Error("Invalid data type!");
} : (e) => {
  if (typeof e == "string")
    return jn.encode(e);
  if (ArrayBuffer.isView(e))
    return new Uint8Array(e.buffer, e.byteOffset, e.byteLength);
  throw new Error("Invalid data type!");
}, Ci = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/", ot = new Uint8Array(256);
for (let e = 0; e < Ci.length; e++)
  ot[Ci.charCodeAt(e)] = e;
function Kn(e) {
  let t = Math.floor(e.length * 0.75);
  const a = e.length;
  return e[a - 1] === "=" && (t -= 1, e[a - 2] === "=" && (t -= 1)), t;
}
function Xn(e) {
  const t = Kn(e), a = e.length, i = new Uint8Array(t);
  let r = 0;
  for (let s = 0; s < a; s += 4) {
    const o = ot[e.charCodeAt(s)], n = ot[e.charCodeAt(s + 1)], d = ot[e.charCodeAt(s + 2)], c = ot[e.charCodeAt(s + 3)];
    i[r] = o << 2 | n >> 4, r += 1, i[r] = (n & 15) << 4 | d >> 2, r += 1, i[r] = (d & 3) << 6 | c & 63, r += 1;
  }
  return i;
}
var wt = 16 * 1024, rt = 4, Wn = new J(), la = /* @__PURE__ */ new Map();
function Zn(e, t) {
  return Ut(this, void 0, void 0, function* () {
    let a = null, i = null, r = !1;
    if (typeof WebAssembly > "u")
      throw new Error("WebAssembly is not supported in this environment!");
    const s = (B, T = 0) => {
      i.set(B, T);
    }, o = () => i, n = () => a.exports, d = (B) => {
      a.exports.Hash_SetMemorySize(B);
      const T = a.exports.Hash_GetBuffer(), j = a.exports.memory.buffer;
      i = new Uint8Array(j, T, B);
    }, c = () => new DataView(a.exports.memory.buffer).getUint32(a.exports.STATE_SIZE, !0), l = Wn.dispatch(() => Ut(this, void 0, void 0, function* () {
      if (!la.has(e.name)) {
        const T = Xn(e.data), j = WebAssembly.compile(T);
        la.set(e.name, j);
      }
      const B = yield la.get(e.name);
      a = yield WebAssembly.instantiate(B, {
        // env: {
        //   emscripten_memcpy_big: (dest, src, num) => {
        //     const memoryBuffer = wasmInstance.exports.memory.buffer;
        //     const memView = new Uint8Array(memoryBuffer, 0);
        //     memView.set(memView.subarray(src, src + num), dest);
        //   },
        //   print_memory: (offset, len) => {
        //     const memoryBuffer = wasmInstance.exports.memory.buffer;
        //     const memView = new Uint8Array(memoryBuffer, 0);
        //     console.log('print_int32', memView.subarray(offset, offset + len));
        //   },
        // },
      });
    })), h = () => Ut(this, void 0, void 0, function* () {
      a || (yield l);
      const B = a.exports.Hash_GetBuffer(), T = a.exports.memory.buffer;
      i = new Uint8Array(T, B, wt);
    }), p = (B = null) => {
      r = !0, a.exports.Hash_Init(B);
    }, y = (B) => {
      let T = 0;
      for (; T < B.length; ) {
        const j = B.subarray(T, T + wt);
        T += j.length, i.set(j), a.exports.Hash_Update(j.length);
      }
    }, u = (B) => {
      if (!r)
        throw new Error("update() called before init()");
      const T = Ri(B);
      y(T);
    }, A = new Uint8Array(t * 2), I = (B, T = null) => {
      if (!r)
        throw new Error("digest() called before init()");
      return r = !1, a.exports.Hash_Final(T), B === "binary" ? i.slice(0, t) : _i(A, i, t);
    }, b = () => {
      if (!r)
        throw new Error("save() can only be called after init() and before digest()");
      const B = a.exports.Hash_GetState(), T = c(), j = a.exports.memory.buffer, q = new Uint8Array(j, B, T), H = new Uint8Array(rt + T);
      return Gn(H, e.hash), H.set(q, rt), H;
    }, S = (B) => {
      if (!(B instanceof Uint8Array))
        throw new Error("load() expects an Uint8Array generated by save()");
      const T = a.exports.Hash_GetState(), j = c(), q = rt + j, H = a.exports.memory.buffer;
      if (B.length !== q)
        throw new Error(`Bad state length (expected ${q} bytes, got ${B.length})`);
      if (!Jn(e.hash, B.subarray(0, rt)))
        throw new Error("This state was written by an incompatible hash implementation");
      const Oe = B.subarray(rt);
      new Uint8Array(H, T, j).set(Oe), r = !0;
    }, w = (B) => typeof B == "string" ? B.length < wt / 4 : B.byteLength < wt;
    let R = w;
    switch (e.name) {
      case "argon2":
      case "scrypt":
        R = () => !0;
        break;
      case "blake2b":
      case "blake2s":
        R = (B, T) => T <= 512 && w(B);
        break;
      case "blake3":
        R = (B, T) => T === 0 && w(B);
        break;
      case "xxhash64":
      case "xxhash3":
      case "xxhash128":
      case "crc64":
        R = () => !1;
        break;
    }
    const P = (B, T = null, j = null) => {
      if (!R(B, T))
        return p(T), u(B), I("hex", j);
      const q = Ri(B);
      return i.set(q), a.exports.Hash_Calculate(q.length, T, j), _i(A, i, t);
    };
    return yield h(), {
      getMemory: o,
      writeMemory: s,
      getExports: n,
      setMemorySize: d,
      init: p,
      update: u,
      digest: I,
      save: b,
      load: S,
      calculate: P,
      hashLength: t
    };
  });
}
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
var Yn = "sha256", qn = "AGFzbQEAAAABEQRgAAF/YAF/AGAAAGACf38AAwgHAAEBAQIAAwUEAQECAgYOAn8BQfCJBQt/AEGACAsHcAgGbWVtb3J5AgAOSGFzaF9HZXRCdWZmZXIAAAlIYXNoX0luaXQAAQtIYXNoX1VwZGF0ZQACCkhhc2hfRmluYWwABA1IYXNoX0dldFN0YXRlAAUOSGFzaF9DYWxjdWxhdGUABgpTVEFURV9TSVpFAwEKnEoHBQBBgAkLnQEAQQBCADcDwIkBQQBBHEEgIABB4AFGIgAbNgLoiQFBAEKnn+anxvST/b5/Qquzj/yRo7Pw2wAgABs3A+CJAUEAQrGWgP6fooWs6ABC/6S5iMWR2oKbfyAAGzcD2IkBQQBCl7rDg5Onlod3QvLmu+Ojp/2npX8gABs3A9CJAUEAQti9loj8oLW+NkLnzKfQ1tDrs7t/IAAbNwPIiQEL7wICAX4Gf0EAQQApA8CJASIBIACtfDcDwIkBAkACQAJAIAGnQT9xIgINAEGACSEDDAELAkBBwAAgAmsiBCAAIAQgAEkbIgNFDQAgA0EDcSEFIAJBgIkBaiEGQQAhAgJAIANBBEkNACADQfwAcSEHQQAhAgNAIAYgAmoiAyACQYAJai0AADoAACADQQFqIAJBgQlqLQAAOgAAIANBAmogAkGCCWotAAA6AAAgA0EDaiACQYMJai0AADoAACAHIAJBBGoiAkcNAAsLIAVFDQADQCAGIAJqIAJBgAlqLQAAOgAAIAJBAWohAiAFQX9qIgUNAAsLIAAgBEkNAUGAiQEQAyAAIARrIQAgBEGACWohAwsCQCAAQcAASQ0AA0AgAxADIANBwABqIQMgAEFAaiIAQT9LDQALCyAARQ0AQQAhAkEAIQUDQCACQYCJAWogAyACai0AADoAACACQQFqIQIgACAFQQFqIgVB/wFxSw0ACwsLoz4BRX9BACAAKAI8IgFBGHQgAUGA/gNxQQh0ciABQQh2QYD+A3EgAUEYdnJyIgFBGXcgAUEOd3MgAUEDdnMgACgCOCICQRh0IAJBgP4DcUEIdHIgAkEIdkGA/gNxIAJBGHZyciICaiAAKAIgIgNBGHQgA0GA/gNxQQh0ciADQQh2QYD+A3EgA0EYdnJyIgRBGXcgBEEOd3MgBEEDdnMgACgCHCIDQRh0IANBgP4DcUEIdHIgA0EIdkGA/gNxIANBGHZyciIFaiAAKAIEIgNBGHQgA0GA/gNxQQh0ciADQQh2QYD+A3EgA0EYdnJyIgZBGXcgBkEOd3MgBkEDdnMgACgCACIDQRh0IANBgP4DcUEIdHIgA0EIdkGA/gNxIANBGHZyciIHaiAAKAIkIgNBGHQgA0GA/gNxQQh0ciADQQh2QYD+A3EgA0EYdnJyIghqIAJBD3cgAkENd3MgAkEKdnNqIgNqIAAoAhgiCUEYdCAJQYD+A3FBCHRyIAlBCHZBgP4DcSAJQRh2cnIiCkEZdyAKQQ53cyAKQQN2cyAAKAIUIglBGHQgCUGA/gNxQQh0ciAJQQh2QYD+A3EgCUEYdnJyIgtqIAJqIAAoAhAiCUEYdCAJQYD+A3FBCHRyIAlBCHZBgP4DcSAJQRh2cnIiDEEZdyAMQQ53cyAMQQN2cyAAKAIMIglBGHQgCUGA/gNxQQh0ciAJQQh2QYD+A3EgCUEYdnJyIg1qIAAoAjAiCUEYdCAJQYD+A3FBCHRyIAlBCHZBgP4DcSAJQRh2cnIiDmogACgCCCIJQRh0IAlBgP4DcUEIdHIgCUEIdkGA/gNxIAlBGHZyciIPQRl3IA9BDndzIA9BA3ZzIAZqIAAoAigiCUEYdCAJQYD+A3FBCHRyIAlBCHZBgP4DcSAJQRh2cnIiEGogAUEPdyABQQ13cyABQQp2c2oiCUEPdyAJQQ13cyAJQQp2c2oiEUEPdyARQQ13cyARQQp2c2oiEkEPdyASQQ13cyASQQp2c2oiE2ogACgCNCIUQRh0IBRBgP4DcUEIdHIgFEEIdkGA/gNxIBRBGHZyciIVQRl3IBVBDndzIBVBA3ZzIA5qIBJqIAAoAiwiAEEYdCAAQYD+A3FBCHRyIABBCHZBgP4DcSAAQRh2cnIiFkEZdyAWQQ53cyAWQQN2cyAQaiARaiAIQRl3IAhBDndzIAhBA3ZzIARqIAlqIAVBGXcgBUEOd3MgBUEDdnMgCmogAWogC0EZdyALQQ53cyALQQN2cyAMaiAVaiANQRl3IA1BDndzIA1BA3ZzIA9qIBZqIANBD3cgA0ENd3MgA0EKdnNqIhRBD3cgFEENd3MgFEEKdnNqIhdBD3cgF0ENd3MgF0EKdnNqIhhBD3cgGEENd3MgGEEKdnNqIhlBD3cgGUENd3MgGUEKdnNqIhpBD3cgGkENd3MgGkEKdnNqIhtBD3cgG0ENd3MgG0EKdnNqIhxBGXcgHEEOd3MgHEEDdnMgAkEZdyACQQ53cyACQQN2cyAVaiAYaiAOQRl3IA5BDndzIA5BA3ZzIBZqIBdqIBBBGXcgEEEOd3MgEEEDdnMgCGogFGogE0EPdyATQQ13cyATQQp2c2oiHUEPdyAdQQ13cyAdQQp2c2oiHkEPdyAeQQ13cyAeQQp2c2oiH2ogE0EZdyATQQ53cyATQQN2cyAYaiADQRl3IANBDndzIANBA3ZzIAFqIBlqIB9BD3cgH0ENd3MgH0EKdnNqIiBqIBJBGXcgEkEOd3MgEkEDdnMgF2ogH2ogEUEZdyARQQ53cyARQQN2cyAUaiAeaiAJQRl3IAlBDndzIAlBA3ZzIANqIB1qIBxBD3cgHEENd3MgHEEKdnNqIiFBD3cgIUENd3MgIUEKdnNqIiJBD3cgIkENd3MgIkEKdnNqIiNBD3cgI0ENd3MgI0EKdnNqIiRqIBtBGXcgG0EOd3MgG0EDdnMgHmogI2ogGkEZdyAaQQ53cyAaQQN2cyAdaiAiaiAZQRl3IBlBDndzIBlBA3ZzIBNqICFqIBhBGXcgGEEOd3MgGEEDdnMgEmogHGogF0EZdyAXQQ53cyAXQQN2cyARaiAbaiAUQRl3IBRBDndzIBRBA3ZzIAlqIBpqICBBD3cgIEENd3MgIEEKdnNqIiVBD3cgJUENd3MgJUEKdnNqIiZBD3cgJkENd3MgJkEKdnNqIidBD3cgJ0ENd3MgJ0EKdnNqIihBD3cgKEENd3MgKEEKdnNqIilBD3cgKUENd3MgKUEKdnNqIipBD3cgKkENd3MgKkEKdnNqIitBGXcgK0EOd3MgK0EDdnMgH0EZdyAfQQ53cyAfQQN2cyAbaiAnaiAeQRl3IB5BDndzIB5BA3ZzIBpqICZqIB1BGXcgHUEOd3MgHUEDdnMgGWogJWogJEEPdyAkQQ13cyAkQQp2c2oiLEEPdyAsQQ13cyAsQQp2c2oiLUEPdyAtQQ13cyAtQQp2c2oiLmogJEEZdyAkQQ53cyAkQQN2cyAnaiAgQRl3ICBBDndzICBBA3ZzIBxqIChqIC5BD3cgLkENd3MgLkEKdnNqIi9qICNBGXcgI0EOd3MgI0EDdnMgJmogLmogIkEZdyAiQQ53cyAiQQN2cyAlaiAtaiAhQRl3ICFBDndzICFBA3ZzICBqICxqICtBD3cgK0ENd3MgK0EKdnNqIjBBD3cgMEENd3MgMEEKdnNqIjFBD3cgMUENd3MgMUEKdnNqIjJBD3cgMkENd3MgMkEKdnNqIjNqICpBGXcgKkEOd3MgKkEDdnMgLWogMmogKUEZdyApQQ53cyApQQN2cyAsaiAxaiAoQRl3IChBDndzIChBA3ZzICRqIDBqICdBGXcgJ0EOd3MgJ0EDdnMgI2ogK2ogJkEZdyAmQQ53cyAmQQN2cyAiaiAqaiAlQRl3ICVBDndzICVBA3ZzICFqIClqIC9BD3cgL0ENd3MgL0EKdnNqIjRBD3cgNEENd3MgNEEKdnNqIjVBD3cgNUENd3MgNUEKdnNqIjZBD3cgNkENd3MgNkEKdnNqIjdBD3cgN0ENd3MgN0EKdnNqIjhBD3cgOEENd3MgOEEKdnNqIjlBD3cgOUENd3MgOUEKdnNqIjogOCA0IC4gLCAhIBsgGSADIA4gBEEAKALYiQEiO0EadyA7QRV3cyA7QQd3c0EAKALkiQEiPGpBACgC4IkBIj1BACgC3IkBIj5zIDtxID1zaiAHakGY36iUBGoiB0EAKALUiQEiP2oiACAMaiA7IA1qID4gD2ogPSAGaiAAID4gO3NxID5zaiAAQRp3IABBFXdzIABBB3dzakGRid2JB2oiQEEAKALQiQEiQWoiDCAAIDtzcSA7c2ogDEEadyAMQRV3cyAMQQd3c2pBz/eDrntqIkJBACgCzIkBIkNqIg0gDCAAc3EgAHNqIA1BGncgDUEVd3MgDUEHd3NqQaW3181+aiJEQQAoAsiJASIAaiIPIA0gDHNxIAxzaiAPQRp3IA9BFXdzIA9BB3dzakHbhNvKA2oiRSBBIEMgAHNxIEMgAHFzIABBHncgAEETd3MgAEEKd3NqIAdqIgZqIgdqIAUgD2ogCiANaiALIAxqIAcgDyANc3EgDXNqIAdBGncgB0EVd3MgB0EHd3NqQfGjxM8FaiIKIAYgAHMgQ3EgBiAAcXMgBkEedyAGQRN3cyAGQQp3c2ogQGoiDGoiBCAHIA9zcSAPc2ogBEEadyAEQRV3cyAEQQd3c2pBpIX+kXlqIgsgDCAGcyAAcSAMIAZxcyAMQR53IAxBE3dzIAxBCndzaiBCaiINaiIPIAQgB3NxIAdzaiAPQRp3IA9BFXdzIA9BB3dzakHVvfHYemoiQCANIAxzIAZxIA0gDHFzIA1BHncgDUETd3MgDUEKd3NqIERqIgZqIgcgDyAEc3EgBHNqIAdBGncgB0EVd3MgB0EHd3NqQZjVnsB9aiJCIAYgDXMgDHEgBiANcXMgBkEedyAGQRN3cyAGQQp3c2ogRWoiDGoiBWogFiAHaiAQIA9qIAggBGogBSAHIA9zcSAPc2ogBUEadyAFQRV3cyAFQQd3c2pBgbaNlAFqIgggDCAGcyANcSAMIAZxcyAMQR53IAxBE3dzIAxBCndzaiAKaiINaiIPIAUgB3NxIAdzaiAPQRp3IA9BFXdzIA9BB3dzakG+i8ahAmoiDiANIAxzIAZxIA0gDHFzIA1BHncgDUETd3MgDUEKd3NqIAtqIgZqIgcgDyAFc3EgBXNqIAdBGncgB0EVd3MgB0EHd3NqQcP7sagFaiIQIAYgDXMgDHEgBiANcXMgBkEedyAGQRN3cyAGQQp3c2ogQGoiDGoiBCAHIA9zcSAPc2ogBEEadyAEQRV3cyAEQQd3c2pB9Lr5lQdqIhYgDCAGcyANcSAMIAZxcyAMQR53IAxBE3dzIAxBCndzaiBCaiINaiIFaiABIARqIAIgB2ogFSAPaiAFIAQgB3NxIAdzaiAFQRp3IAVBFXdzIAVBB3dzakH+4/qGeGoiByANIAxzIAZxIA0gDHFzIA1BHncgDUETd3MgDUEKd3NqIAhqIgFqIgYgBSAEc3EgBHNqIAZBGncgBkEVd3MgBkEHd3NqQaeN8N55aiIEIAEgDXMgDHEgASANcXMgAUEedyABQRN3cyABQQp3c2ogDmoiAmoiDCAGIAVzcSAFc2ogDEEadyAMQRV3cyAMQQd3c2pB9OLvjHxqIgUgAiABcyANcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAQaiIDaiINIAwgBnNxIAZzaiANQRp3IA1BFXdzIA1BB3dzakHB0+2kfmoiCCADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBZqIgFqIg8gF2ogESANaiAUIAxqIAkgBmogDyANIAxzcSAMc2ogD0EadyAPQRV3cyAPQQd3c2pBho/5/X5qIgYgASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAHaiICaiIJIA8gDXNxIA1zaiAJQRp3IAlBFXdzIAlBB3dzakHGu4b+AGoiDCACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIARqIgNqIhEgCSAPc3EgD3NqIBFBGncgEUEVd3MgEUEHd3NqQczDsqACaiINIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogBWoiAWoiFCARIAlzcSAJc2ogFEEadyAUQRV3cyAUQQd3c2pB79ik7wJqIg8gASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAIaiICaiIXaiATIBRqIBggEWogEiAJaiAXIBQgEXNxIBFzaiAXQRp3IBdBFXdzIBdBB3dzakGqidLTBGoiGCACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIAZqIgNqIgkgFyAUc3EgFHNqIAlBGncgCUEVd3MgCUEHd3NqQdzTwuUFaiIUIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogDGoiAWoiESAJIBdzcSAXc2ogEUEadyARQRV3cyARQQd3c2pB2pHmtwdqIhcgASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiANaiICaiISIBEgCXNxIAlzaiASQRp3IBJBFXdzIBJBB3dzakHSovnBeWoiGSACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIA9qIgNqIhNqIB4gEmogGiARaiAdIAlqIBMgEiARc3EgEXNqIBNBGncgE0EVd3MgE0EHd3NqQe2Mx8F6aiIaIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogGGoiAWoiCSATIBJzcSASc2ogCUEadyAJQRV3cyAJQQd3c2pByM+MgHtqIhggASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAUaiICaiIRIAkgE3NxIBNzaiARQRp3IBFBFXdzIBFBB3dzakHH/+X6e2oiFCACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIBdqIgNqIhIgESAJc3EgCXNqIBJBGncgEkEVd3MgEkEHd3NqQfOXgLd8aiIXIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogGWoiAWoiE2ogICASaiAcIBFqIB8gCWogEyASIBFzcSARc2ogE0EadyATQRV3cyATQQd3c2pBx6KerX1qIhkgASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAaaiICaiIJIBMgEnNxIBJzaiAJQRp3IAlBFXdzIAlBB3dzakHRxqk2aiIaIAIgAXMgA3EgAiABcXMgAkEedyACQRN3cyACQQp3c2ogGGoiA2oiESAJIBNzcSATc2ogEUEadyARQRV3cyARQQd3c2pB59KkoQFqIhggAyACcyABcSADIAJxcyADQR53IANBE3dzIANBCndzaiAUaiIBaiISIBEgCXNxIAlzaiASQRp3IBJBFXdzIBJBB3dzakGFldy9AmoiFCABIANzIAJxIAEgA3FzIAFBHncgAUETd3MgAUEKd3NqIBdqIgJqIhMgI2ogJiASaiAiIBFqICUgCWogEyASIBFzcSARc2ogE0EadyATQRV3cyATQQd3c2pBuMLs8AJqIhcgAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAZaiIDaiIJIBMgEnNxIBJzaiAJQRp3IAlBFXdzIAlBB3dzakH827HpBGoiGSADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBpqIgFqIhEgCSATc3EgE3NqIBFBGncgEUEVd3MgEUEHd3NqQZOa4JkFaiIaIAEgA3MgAnEgASADcXMgAUEedyABQRN3cyABQQp3c2ogGGoiAmoiEiARIAlzcSAJc2ogEkEadyASQRV3cyASQQd3c2pB1OapqAZqIhggAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAUaiIDaiITaiAoIBJqICQgEWogJyAJaiATIBIgEXNxIBFzaiATQRp3IBNBFXdzIBNBB3dzakG7laizB2oiFCADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBdqIgFqIgkgEyASc3EgEnNqIAlBGncgCUEVd3MgCUEHd3NqQa6Si454aiIXIAEgA3MgAnEgASADcXMgAUEedyABQRN3cyABQQp3c2ogGWoiAmoiESAJIBNzcSATc2ogEUEadyARQRV3cyARQQd3c2pBhdnIk3lqIhkgAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAaaiIDaiISIBEgCXNxIAlzaiASQRp3IBJBFXdzIBJBB3dzakGh0f+VemoiGiADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBhqIgFqIhNqICogEmogLSARaiApIAlqIBMgEiARc3EgEXNqIBNBGncgE0EVd3MgE0EHd3NqQcvM6cB6aiIYIAEgA3MgAnEgASADcXMgAUEedyABQRN3cyABQQp3c2ogFGoiAmoiCSATIBJzcSASc2ogCUEadyAJQRV3cyAJQQd3c2pB8JauknxqIhQgAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAXaiIDaiIRIAkgE3NxIBNzaiARQRp3IBFBFXdzIBFBB3dzakGjo7G7fGoiFyADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBlqIgFqIhIgESAJc3EgCXNqIBJBGncgEkEVd3MgEkEHd3NqQZnQy4x9aiIZIAEgA3MgAnEgASADcXMgAUEedyABQRN3cyABQQp3c2ogGmoiAmoiE2ogMCASaiAvIBFqICsgCWogEyASIBFzcSARc2ogE0EadyATQRV3cyATQQd3c2pBpIzktH1qIhogAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAYaiIDaiIJIBMgEnNxIBJzaiAJQRp3IAlBFXdzIAlBB3dzakGF67igf2oiGCADIAJzIAFxIAMgAnFzIANBHncgA0ETd3MgA0EKd3NqIBRqIgFqIhEgCSATc3EgE3NqIBFBGncgEUEVd3MgEUEHd3NqQfDAqoMBaiIUIAEgA3MgAnEgASADcXMgAUEedyABQRN3cyABQQp3c2ogF2oiAmoiEiARIAlzcSAJc2ogEkEadyASQRV3cyASQQd3c2pBloKTzQFqIhcgAiABcyADcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiAZaiIDaiITIDZqIDIgEmogNSARaiAxIAlqIBMgEiARc3EgEXNqIBNBGncgE0EVd3MgE0EHd3NqQYjY3fEBaiIZIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogGmoiAWoiCSATIBJzcSASc2ogCUEadyAJQRV3cyAJQQd3c2pBzO6hugJqIhogASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAYaiICaiIRIAkgE3NxIBNzaiARQRp3IBFBFXdzIBFBB3dzakG1+cKlA2oiGCACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIBRqIgNqIhIgESAJc3EgCXNqIBJBGncgEkEVd3MgEkEHd3NqQbOZ8MgDaiIUIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogF2oiAWoiE2ogLEEZdyAsQQ53cyAsQQN2cyAoaiA0aiAzQQ93IDNBDXdzIDNBCnZzaiIXIBJqIDcgEWogMyAJaiATIBIgEXNxIBFzaiATQRp3IBNBFXdzIBNBB3dzakHK1OL2BGoiGyABIANzIAJxIAEgA3FzIAFBHncgAUETd3MgAUEKd3NqIBlqIgJqIgkgEyASc3EgEnNqIAlBGncgCUEVd3MgCUEHd3NqQc+U89wFaiIZIAIgAXMgA3EgAiABcXMgAkEedyACQRN3cyACQQp3c2ogGmoiA2oiESAJIBNzcSATc2ogEUEadyARQRV3cyARQQd3c2pB89+5wQZqIhogAyACcyABcSADIAJxcyADQR53IANBE3dzIANBCndzaiAYaiIBaiISIBEgCXNxIAlzaiASQRp3IBJBFXdzIBJBB3dzakHuhb6kB2oiHCABIANzIAJxIAEgA3FzIAFBHncgAUETd3MgAUEKd3NqIBRqIgJqIhNqIC5BGXcgLkEOd3MgLkEDdnMgKmogNmogLUEZdyAtQQ53cyAtQQN2cyApaiA1aiAXQQ93IBdBDXdzIBdBCnZzaiIUQQ93IBRBDXdzIBRBCnZzaiIYIBJqIDkgEWogFCAJaiATIBIgEXNxIBFzaiATQRp3IBNBFXdzIBNBB3dzakHvxpXFB2oiCSACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIBtqIgNqIhEgEyASc3EgEnNqIBFBGncgEUEVd3MgEUEHd3NqQZTwoaZ4aiIbIAMgAnMgAXEgAyACcXMgA0EedyADQRN3cyADQQp3c2ogGWoiAWoiEiARIBNzcSATc2ogEkEadyASQRV3cyASQQd3c2pBiISc5nhqIhkgASADcyACcSABIANxcyABQR53IAFBE3dzIAFBCndzaiAaaiICaiITIBIgEXNxIBFzaiATQRp3IBNBFXdzIBNBB3dzakH6//uFeWoiGiACIAFzIANxIAIgAXFzIAJBHncgAkETd3MgAkEKd3NqIBxqIgNqIhQgPGo2AuSJAUEAID8gAyACcyABcSADIAJxcyADQR53IANBE3dzIANBCndzaiAJaiIBIANzIAJxIAEgA3FzIAFBHncgAUETd3MgAUEKd3NqIBtqIgIgAXMgA3EgAiABcXMgAkEedyACQRN3cyACQQp3c2ogGWoiAyACcyABcSADIAJxcyADQR53IANBE3dzIANBCndzaiAaaiIJajYC1IkBQQAgPSAvQRl3IC9BDndzIC9BA3ZzICtqIDdqIBhBD3cgGEENd3MgGEEKdnNqIhggEWogFCATIBJzcSASc2ogFEEadyAUQRV3cyAUQQd3c2pB69nBonpqIhkgAWoiEWo2AuCJAUEAIEEgCSADcyACcSAJIANxcyAJQR53IAlBE3dzIAlBCndzaiAZaiIBajYC0IkBQQAgPiAwQRl3IDBBDndzIDBBA3ZzIC9qIBdqIDpBD3cgOkENd3MgOkEKdnNqIBJqIBEgFCATc3EgE3NqIBFBGncgEUEVd3MgEUEHd3NqQffH5vd7aiIXIAJqIhJqNgLciQFBACBDIAEgCXMgA3EgASAJcXMgAUEedyABQRN3cyABQQp3c2ogF2oiAmo2AsyJAUEAIDsgNEEZdyA0QQ53cyA0QQN2cyAwaiA4aiAYQQ93IBhBDXdzIBhBCnZzaiATaiASIBEgFHNxIBRzaiASQRp3IBJBFXdzIBJBB3dzakHy8cWzfGoiESADamo2AtiJAUEAIAAgAiABcyAJcSACIAFxcyACQR53IAJBE3dzIAJBCndzaiARamo2AsiJAQuyBgIEfwF+QQAoAsCJASIAQQJ2QQ9xIgFBAnRBgIkBaiICIAIoAgBBfyAAQQN0IgB0QX9zcUGAASAAdHM2AgACQAJAAkAgAUEOSQ0AAkAgAUEORw0AQQBBADYCvIkBC0GAiQEQA0EAIQIMAQsgAUENRg0BIAFBAWohAgsgAiEDAkBBBiACa0EHcSIARQ0AIAIgAGohAyACQQJ0QYCJAWohAQNAIAFBADYCACABQQRqIQEgAEF/aiIADQALCyACQXlqQQdJDQAgA0ECdCEBA0AgAUGYiQFqQgA3AgAgAUGQiQFqQgA3AgAgAUGIiQFqQgA3AgAgAUGAiQFqQgA3AgAgAUEgaiIBQThHDQALC0EAIQFBAEEAKQPAiQEiBKciAEEbdCAAQQt0QYCA/AdxciAAQQV2QYD+A3EgAEEDdEEYdnJyNgK8iQFBACAEQh2IpyIAQRh0IABBgP4DcUEIdHIgAEEIdkGA/gNxIABBGHZycjYCuIkBQYCJARADQQBBACgC5IkBIgBBGHQgAEGA/gNxQQh0ciAAQQh2QYD+A3EgAEEYdnJyNgLkiQFBAEEAKALgiQEiAEEYdCAAQYD+A3FBCHRyIABBCHZBgP4DcSAAQRh2cnI2AuCJAUEAQQAoAtyJASIAQRh0IABBgP4DcUEIdHIgAEEIdkGA/gNxIABBGHZycjYC3IkBQQBBACgC2IkBIgBBGHQgAEGA/gNxQQh0ciAAQQh2QYD+A3EgAEEYdnJyNgLYiQFBAEEAKALUiQEiAEEYdCAAQYD+A3FBCHRyIABBCHZBgP4DcSAAQRh2cnI2AtSJAUEAQQAoAtCJASIAQRh0IABBgP4DcUEIdHIgAEEIdkGA/gNxIABBGHZycjYC0IkBQQBBACgCzIkBIgBBGHQgAEGA/gNxQQh0ciAAQQh2QYD+A3EgAEEYdnJyNgLMiQFBAEEAKALIiQEiAEEYdCAAQYD+A3FBCHRyIABBCHZBgP4DcSAAQRh2cnI2AsiJAQJAQQAoAuiJASICRQ0AQQAhAANAIAFBgAlqIAFByIkBai0AADoAACABQQFqIQEgAiAAQQFqIgBB/wFxSw0ACwsLBgBBgIkBC6MBAEEAQgA3A8CJAUEAQRxBICABQeABRiIBGzYC6IkBQQBCp5/mp8b0k/2+f0Krs4/8kaOz8NsAIAEbNwPgiQFBAEKxloD+n6KFrOgAQv+kuYjFkdqCm38gARs3A9iJAUEAQpe6w4OTp5aHd0Ly5rvjo6f9p6V/IAEbNwPQiQFBAELYvZaI/KC1vjZC58yn0NbQ67O7fyABGzcDyIkBIAAQAhAECwsLAQBBgAgLBHAAAAA=", ec = "8c18dd94", tc = {
  name: Yn,
  data: qn,
  hash: ec
};
new J();
new J();
function va() {
  return Zn(tc, 32).then((e) => {
    e.init(256);
    const t = {
      init: () => (e.init(256), t),
      update: (a) => (e.update(a), t),
      // biome-ignore lint/suspicious/noExplicitAny: Conflict with IHasher type
      digest: (a) => e.digest(a),
      save: () => e.save(),
      load: (a) => (e.load(a), t),
      blockSize: 64,
      digestSize: 32
    };
    return t;
  });
}
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
new J();
var hr = 1 * 1024 * 1024;
async function pr(e) {
  return e.arrayBuffer();
}
async function ac(e, t, a) {
  const i = Math.min(t, hr), r = e.slice(0, i), s = await pr(r), o = await va();
  return o.init(), o.update(new Uint8Array(s)), o.digest("hex");
}
async function Fi(e, t, a) {
  const i = hr, r = Math.ceil(t / i);
  let s = await va();
  s.init();
  let o = "";
  for (let n = 0; n < r; n++) {
    const d = n * i, c = Math.min(d + i, t), l = e.slice(d, c), h = await pr(l);
    if (s.update(new Uint8Array(h)), o = s.digest("hex"), n < r - 1 && (s = await va(), s.init(), s.update(ic(o))), a) {
      const p = Math.min(100, (n + 1) / r * 100);
      a(p);
    }
  }
  return o;
}
function ic(e) {
  const t = new Uint8Array(e.length / 2);
  for (let a = 0; a < e.length; a += 2)
    t[a / 2] = parseInt(e.substr(a, 2), 16);
  return t;
}
var yr = BigInt("0xC96C5795D7870F42"), Kt = BigInt("0xFFFFFFFFFFFFFFFF"), rc = BigInt(255), da = BigInt(1), sc = BigInt(8), je = Kt, oc = Kt, Ge = null;
function ur() {
  if (!Ge) {
    Ge = new Array(256);
    for (let e = 0; e < 256; e++) {
      let t = BigInt(e);
      for (let a = 0; a < 8; a++)
        t & da ? t = t >> da ^ yr : t = t >> da;
      Ge[e] = t;
    }
  }
}
function Sa(e, t) {
  let a = BigInt(0), i = 0;
  for (; t; )
    t & BigInt(1) && (a ^= e[i]), t >>= BigInt(1), i++;
  return a;
}
function Et(e, t) {
  for (let a = 0; a < 64; a++)
    e[a] = Sa(t, t[a]);
}
function wa(e, t) {
  Ge || ur();
  const a = t instanceof ArrayBuffer ? new Uint8Array(t) : t;
  for (let i = 0; i < a.length; i++) {
    const r = Number((e ^ BigInt(a[i])) & rc);
    e = e >> sc ^ Ge[r];
  }
  return e;
}
function Tt(e) {
  return ((e ^ oc) & Kt).toString(10);
}
function nc(e, t, a) {
  if (a === 0)
    return e;
  Ge || ur();
  let i = BigInt(e);
  const r = new Array(64), s = new Array(64);
  s[0] = yr;
  let o = BigInt(1);
  for (let d = 1; d < 64; d++)
    s[d] = o, o <<= BigInt(1);
  Et(r, s), Et(s, r);
  let n = a;
  for (; n > 0 && (Et(r, s), n & 1 && (i = Sa(r, i)), n >>= 1, n !== 0); )
    Et(s, r), n & 1 && (i = Sa(s, i)), n >>= 1;
  return i ^= BigInt(t), (i & Kt).toString(10);
}
function fr(e) {
  if (!e || e.length === 0)
    return Tt(je);
  if (e.length === 1)
    return e[0].crc64;
  let t = e[0].crc64;
  for (let a = 1; a < e.length; a++) {
    const i = e[a];
    t = nc(t, i.crc64, i.size);
  }
  return t;
}
async function ha(e, t) {
  let i = je, r = 0;
  for (; r < e.size; ) {
    const s = Math.min(r + 1048576, e.size), n = await e.slice(r, s).arrayBuffer();
    i = wa(i, n), r = s;
  }
  return Tt(i);
}
function Bt(e) {
  const t = e.match(/^(.+)\.cos\.([^.]+)\.myqcloud\.com$/);
  return t ? {
    bucket: t[1],
    region: t[2]
  } : {
    bucket: "",
    region: ""
  };
}
var kt = /* @__PURE__ */ ((e) => (e.WAITING = "waiting", e.START = "start", e.COMPUTING_HASH = "computing_hash", e.CREATED = "created", e.PREPARING = "preparing", e.RUNNING = "running", e.PAUSED = "paused", e.COMPLETE = "complete", e.CONFIRMING = "confirming", e.SUCCESS = "success", e.RAPID_SUCCESS = "rapid_success", e.ERROR = "error", e.CANCELED = "canceled", e))(kt || {}), Ar = class extends Hn {
  // 最大重试次数
  constructor(e, t) {
    super(), this.verbose = !1, this.state = "waiting", this.message = "", this.progress = 0, this.loaded = 0, this.speed = 0, this.left_time = 0, this.startSize = 0, this.lastEmittedProgress = 0, this.lastProgressLoaded = 0, this.start_time = 0, this.end_time = 0, this.used_avg_speed = 0, this.used_time_len = 0, this.avg_speed = 0, this.pauseFlag = !1, this.cancelFlag = !1, this.abortController = new AbortController(), this.speedList = [], this.speed_0_count = 0, this.task_start_time = 0, this.start_done_part_loaded = 0, this.PROGRESS_EMIT_STEP = 0.2, this.MAX_SPEED_0_COUNT = 10, this.MAX_RETRY_TIMES = 3, this.file = e, this.verbose = (t == null ? void 0 : t.verbose) || !1, this.id = (t == null ? void 0 : t.id) || this.generateTaskId();
  }
  /**
   * 日志输出
   */
  logInfo(...e) {
    if (this.verbose) {
      const t = this.getTaskType().charAt(0).toUpperCase();
      console.info(`[${t}]`, ...e);
    }
  }
  logWarn(...e) {
    if (this.verbose) {
      const t = this.getTaskType().charAt(0).toUpperCase();
      console.warn(`[${t}]`, ...e);
    }
  }
  logError(...e) {
    if (this.verbose) {
      const t = this.getTaskType().charAt(0).toUpperCase();
      console.error(`[${t}]`, ...e);
    }
  }
  /**
   * 生成唯一任务ID
   */
  generateTaskId() {
    return `${this.getTaskType()}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  /**
   * 等待开始
   */
  async wait() {
    this.state !== "waiting" && (this.error = void 0, this.pauseCalcSpeed(), this.pauseFlag = !1, this.cancelFlag = !1, this.state === "error" && (this.end_time = 0, this.message = ""), await this.changeState(
      "waiting"
      /* WAITING */
    ));
  }
  /**
   * 取消所有正在进行的 HTTP 请求，并重建控制器供后续使用（恢复场景）
   */
  abortRequest() {
    this.abortController.abort(), this.abortController = new AbortController();
  }
  /**
   * 获取当前的 AbortSignal，供 fetch/axios 请求使用
   */
  get abortSignal() {
    return this.abortController.signal;
  }
  /**
   * 停止（暂停）任务
   */
  async pause() {
    ["paused", "success", "error", "canceled"].includes(this.state) || (this.pauseFlag = !0, this.abortRequest(), this.pauseCalcSpeed(), this.calcTotalAvgSpeed(), await this.changeState(
      "paused"
      /* PAUSED */
    ));
  }
  /**
   * 取消任务
   */
  async cancel() {
    this.cancelFlag || this.state === "canceled" || (this.cancelFlag = !0, this.abortRequest(), this.pauseCalcSpeed(), this.calcTotalAvgSpeed(), await this.changeState(
      "canceled"
      /* CANCELED */
    ));
  }
  /**
   * 开始计算速度
   */
  startCalcSpeed() {
    this.left_time = 0, this.speed = 0, this.lastProgressLoaded = this.loaded, this.speedList = [], this.tid_speed && clearInterval(this.tid_speed), this.tid_speed = setInterval(() => {
      const e = Math.max(0, this.loaded - this.lastProgressLoaded);
      this.speedList.push(e), this.speedList.length > 10 && this.speedList.shift(), this.speed = this.calcSmoothSpeed(this.speedList), this.left_time = this.speed === 0 ? 24 * 3600 : (this.file.size - this.loaded) / this.speed, this.lastProgressLoaded = this.loaded, this.checkTimeout();
    }, 1e3);
  }
  /**
   * 停止计算速度
   */
  pauseCalcSpeed() {
    this.tid_speed && (clearInterval(this.tid_speed), this.tid_speed = void 0), this.speed = 0;
  }
  /**
   * 计算平滑速度（滑动平均）
   */
  calcSmoothSpeed(e) {
    return e.length === 0 ? 0 : e.reduce((a, i) => a + i, 0) / e.length;
  }
  /**
   * 计算总平均速度
   */
  calcTotalAvgSpeed() {
    const e = Date.now() - this.task_start_time, t = this.loaded - (this.start_done_part_loaded || 0);
    this.used_time_len && this.used_avg_speed ? this.avg_speed = (this.used_time_len / 1e3 * this.used_avg_speed + t) / (this.used_time_len + e) * 1e3 : this.avg_speed = e > 0 ? t / e * 1e3 : 0, this.used_time_len += e, this.used_avg_speed = this.avg_speed;
  }
  /**
   * 检查超时
   */
  async checkTimeout() {
    this.speed_0_count == null && (this.speed_0_count = 0), this.speed === 0 ? this.speed_0_count++ : this.speed_0_count = 0;
  }
  /**
   * 更新进度
   * @param loaded 当前已处理的总字节数
   * @param options 选项
   */
  updateProgress(e, t) {
    if (t != null && t.init) {
      this.startSize = e, this.loaded = e, this.lastProgressLoaded = e, this.progress = this.file.size > 0 ? e / this.file.size * 100 : 0, this.lastEmittedProgress = this.progress, this.notifyProgress("running", this.progress);
      return;
    }
    if (this.loaded = e, this.progress = this.file.size > 0 ? e / this.file.size * 100 : 100, this.speed > 0) {
      const i = this.file.size > e ? this.file.size - e : 0;
      this.left_time = i / this.speed;
    }
    const a = Math.abs(this.progress - this.lastEmittedProgress);
    a > 0 && (t != null && t.immediately || a >= this.PROGRESS_EMIT_STEP) && (this.lastEmittedProgress = this.progress, this.notifyProgress("running", this.progress));
  }
  /**
   * 改变状态
   */
  async changeState(e, t) {
    this.state = e;
    const a = this.getCheckpoint();
    this.emit("statechange", { checkpoint: a, state: e, error: t });
  }
  /**
   * 通知进度
   */
  notifyProgress(e, t) {
    const a = {
      loaded: this.loaded,
      total: this.file.size,
      progress: t,
      speed: this.speed,
      leftTime: this.left_time
    };
    this.emit("progress", a);
  }
  /**
   * 处理错误
   */
  async handleError(e) {
    const t = e instanceof Jt ? e : X("OperationFailed", e.message, e);
    return this.cancelFlag ? (await this.changeState("error", t), t) : e.message === "paused" ? (await this.pause(), t) : (this.message = t.message, this.error = t, this.end_time = Date.now(), this.pauseCalcSpeed(), this.calcTotalAvgSpeed(), await this.changeState("error", t), t);
  }
}, cc = class extends Ar {
  // 浏览器默认并发数
  constructor(e, t) {
    const a = {
      name: e.file.name,
      size: e.file.size,
      type: e.file.type
    };
    if (!e.file || !e.file.name || isNaN(e.file.size))
      throw X(
        "InvalidFile",
        "Invalid file: file must have name and size",
        void 0,
        { file: a }
      );
    super(a, { verbose: e.verbose }), this.part_info_list = [], this.rapid_upload = !1, this.DEFAULT_PARALLEL = 2, this.options = e, this.fileApi = new Dt(t);
    const i = e.partFileSize || 32, r = 1, s = 5 * 1024;
    if (i < r || i > s)
      throw X(
        "InvalidParameter",
        `partFileSize must be between ${r}MB and ${s}MB`,
        void 0,
        { partFileSize: i }
      );
    this.CHUNK_FILE_SIZE = i * 1024 * 1024, this.MIN_SIZE_FOR_HASH = 1 * 1024 * 1024, this.chunk_size = (e.chunkSize || 5) * 1024 * 1024, e.checkpoint && this.restoreCheckpoint(e.checkpoint);
  }
  getTaskType() {
    return "upload";
  }
  /**
   * 检查任务是否被停止
   */
  throwIfStopped(e) {
    if (this.pauseFlag || this.cancelFlag)
      throw X(
        this.pauseFlag ? "UploadPaused" : "UploadCanceled",
        `Upload stopped ${e}`,
        void 0,
        { fileName: this.file.name }
      );
  }
  /**
   * 恢复checkpoint
   */
  restoreCheckpoint(e) {
    this.state = e.state, this.progress = e.progress, this.loaded = e.loaded, this.startSize = e.loaded, this.lastProgressLoaded = e.loaded, this.upload_id = e.upload_id, this.confirm_key = e.confirm_key, this.bucket = e.bucket, this.region = e.region, this.key = e.key, this.chunk_size = e.chunk_size, this.part_info_list = e.part_info_list || [], this.rapid_upload = e.rapid_upload || !1, this.crc64 = e.crc64, this.start_time = e.start_time || 0, this.end_time = e.end_time || 0, this.used_avg_speed = e.used_avg_speed || 0, this.used_time_len = e.used_time_len || 0;
  }
  /**
   * 获取checkpoint信息
   */
  getCheckpoint() {
    return {
      id: this.id,
      file: {
        name: this.file.name,
        size: this.file.size,
        type: this.file.type
      },
      state: this.state,
      progress: this.progress,
      loaded: this.loaded,
      upload_id: this.upload_id,
      confirm_key: this.confirm_key,
      bucket: this.bucket,
      region: this.region,
      key: this.key,
      chunk_size: this.chunk_size,
      part_info_list: this.part_info_list.map((e) => ({
        part_number: e.part_number,
        chunk_size: e.chunk_size,
        etag: e.etag,
        crc64: e.crc64,
        from: e.from,
        to: e.to,
        start_time: e.start_time,
        end_time: e.end_time
      })),
      crc64: this.crc64,
      rapid_upload: this.rapid_upload,
      start_time: this.start_time,
      end_time: this.end_time,
      used_avg_speed: this.used_avg_speed,
      used_time_len: this.used_time_len
    };
  }
  /**
   * 开始任务
   */
  async start() {
    ["waiting", "error", "paused", "canceled"].includes(this.state) && (await this.changeState(
      "start"
      /* START */
    ), await this.doStart());
  }
  /**
   * 执行开始
   */
  async doStart() {
    this.pauseFlag = !1, this.cancelFlag = !1;
    try {
      await this.run();
    } catch (e) {
      if (this.pauseFlag || this.cancelFlag || _.isCancel(e))
        return;
      await this.handleError(e);
    }
  }
  /**
   * 暂停任务
   */
  async pause() {
    this.pauseFlag = !0, this.clearRenewalTimer(), this.logInfo(`Task paused: ${this.file.name}, progress: ${this.progress.toFixed(2)}%`), await super.pause();
  }
  /**
   * 取消任务
   */
  async cancel() {
    if (!this.cancelFlag) {
      if (this.clearRenewalTimer(), this.confirm_key) {
        try {
          await this.fileApi.abortFileUpload({
            libraryId: this.options.libraryId,
            spaceId: this.options.spaceId,
            confirmKey: this.confirm_key,
            upload: 1,
            accessToken: this.options.accessToken,
            userId: this.options.userId
          });
        } catch {
        }
        this.confirm_key = void 0, this.upload_id = void 0;
      }
      this.logInfo(`Task canceled: ${this.file.name}`), await super.cancel();
    }
  }
  /**
   * 主运行流程
   */
  async run() {
    if (this.start_time || (this.start_time = Date.now()), this.rapid_upload)
      return this.end_time = Date.now(), await this.changeState(
        "success"
        /* SUCCESS */
      );
    if (await this.executeUpload(), this.rapid_upload) {
      this.end_time = Date.now();
      return;
    }
    this.end_time = Date.now(), await this.changeState(
      "success"
      /* SUCCESS */
    );
    const e = this.end_time - this.start_time;
    this.logInfo(`Upload success: ${this.file.name}, size: ${le(this.file.size)}, time: ${cr(e)}, speed: ${le(this.used_avg_speed || 0)}/s`);
  }
  /**
   * 执行上传流程
   */
  async executeUpload() {
    const e = this.file.size, t = this.CHUNK_FILE_SIZE, a = this.options.enableInstantUpload !== !1, i = e > t;
    this.logInfo(`Upload strategy: fileSize=${le(e)}, threshold=${le(t)}, useMultipart=${i}, enableInstantUpload=${a}`);
    let r;
    a && e >= this.MIN_SIZE_FOR_HASH && !this.confirm_key && !this.rapid_upload && (await this.changeState(
      "computing_hash"
      /* COMPUTING_HASH */
    ), r = await ac(this.options.file, e), this.throwIfStopped("during beginning hash calculation")), i ? await this.executeMultipartUpload(r) : await this.executeSimpleUpload(r);
  }
  /**
   * 执行简单上传
   */
  async executeSimpleUpload(e) {
    var n;
    const t = e ? { beginningHash: e, size: String(this.file.size) } : {};
    await this.changeState(
      "created"
      /* CREATED */
    );
    let a = await this.fileApi.simpleUploadFile({
      libraryId: this.options.libraryId,
      spaceId: this.options.spaceId,
      filePath: this.options.filePath,
      filesize: this.file.size,
      accessToken: this.options.accessToken,
      userId: this.options.userId,
      trafficLimit: this.options.trafficLimit,
      simpleUploadFileRequest: t,
      ...this.options.conflictResolutionStrategy && {
        conflictResolutionStrategy: this.options.conflictResolutionStrategy
      }
    }), i = a.status;
    if (i === 202) {
      await this.changeState(
        "computing_hash"
        /* COMPUTING_HASH */
      );
      const d = await Fi(this.options.file, this.file.size, (c) => {
        this.notifyProgress("computing_hash", c);
      });
      if (this.throwIfStopped("during hash calculation"), a = await this.fileApi.simpleUploadFile({
        libraryId: this.options.libraryId,
        spaceId: this.options.spaceId,
        filePath: this.options.filePath,
        filesize: this.file.size,
        accessToken: this.options.accessToken,
        userId: this.options.userId,
        trafficLimit: this.options.trafficLimit,
        simpleUploadFileRequest: {
          fullHash: d,
          beginningHash: e,
          size: String(this.file.size)
        },
        ...this.options.conflictResolutionStrategy && {
          conflictResolutionStrategy: this.options.conflictResolutionStrategy
        }
      }), i = a.status, i === 200) {
        if (this.rapid_upload = !0, this.loaded = this.file.size, this.progress = 100, this.updateProgress(this.file.size, { immediately: !0 }), this.start_time) {
          const c = Date.now() - this.start_time;
          this.used_avg_speed = c > 0 ? this.file.size / c * 1e3 : 0, this.speed = this.used_avg_speed;
        }
        await this.changeState(
          "rapid_success"
          /* RAPID_SUCCESS */
        ), this.logInfo(`Rapid upload success: ${this.file.name}`);
        return;
      }
    }
    const r = a.data, { bucket: s, region: o } = Bt(r.domain);
    this.confirm_key = r.confirmKey, this.bucket = s, this.region = o, this.key = ((n = r.path) == null ? void 0 : n.replace(/^\//, "")) || "", await this.changeState(
      "running"
      /* RUNNING */
    ), this.task_start_time = Date.now(), this.startCalcSpeed(), await this.simpleUploadWithRetry(r), this.pauseCalcSpeed(), this.calcTotalAvgSpeed(), this.throwIfStopped("after upload completion"), await this.changeState(
      "confirming"
      /* CONFIRMING */
    ), await this.confirmUpload();
  }
  /**
   * 带重试的简单上传
   */
  async simpleUploadWithRetry(e, t = 0) {
    var a;
    try {
      await this.simpleUpload(e);
    } catch (i) {
      if (this.pauseFlag || this.cancelFlag)
        throw i;
      const { isExpired: r } = Nt(i);
      if (t < this.MAX_RETRY_TIMES)
        if (this.loaded = 0, this.startSize = 0, this.lastProgressLoaded = 0, this.updateProgress(0, { immediately: !0 }), r) {
          const o = (await this.fileApi.simpleUploadFile({
            libraryId: this.options.libraryId,
            spaceId: this.options.spaceId,
            filePath: this.options.filePath,
            filesize: this.file.size,
            accessToken: this.options.accessToken,
            userId: this.options.userId,
            trafficLimit: this.options.trafficLimit,
            simpleUploadFileRequest: {},
            ...this.options.conflictResolutionStrategy && {
              conflictResolutionStrategy: this.options.conflictResolutionStrategy
            }
          })).data, { bucket: n, region: d } = Bt(o.domain);
          return this.bucket = n, this.region = d, this.key = ((a = o.path) == null ? void 0 : a.replace(/^\//, "")) || "", this.simpleUploadWithRetry(o, t + 1);
        } else
          return this.logWarn(`Simple upload retry ${t + 1}/${this.MAX_RETRY_TIMES}: ${(i == null ? void 0 : i.message) || i}`), this.simpleUploadWithRetry(e, t + 1);
      else
        throw X(
          "UploadFailed",
          "Simple upload failed after retries",
          i,
          { fileName: this.file.name, fileSize: this.file.size, retryCount: t }
        );
    }
  }
  /**
   * 简单上传（使用 axios）
   */
  async simpleUpload(e) {
    const t = e.headers || {}, a = `https://${e.domain}${e.path || ""}`;
    this.updateProgress(0, { immediately: !0 }), this.crc64 = await ha(this.options.file);
    const i = await this.toUploadData(this.options.file);
    await _.put(a, i, {
      headers: {
        ...t
      },
      maxContentLength: 1 / 0,
      maxBodyLength: 1 / 0,
      timeout: Math.max(5 * 60 * 1e3, Math.ceil(this.file.size / (100 * 1024)) * 1e3),
      signal: this.abortSignal,
      onUploadProgress: (r) => {
        r.loaded && this.updateProgress(r.loaded);
      }
    }), this.updateProgress(this.file.size, { immediately: !0 });
  }
  /**
   * 执行分块上传
   */
  async executeMultipartUpload(e) {
    var a;
    (!this.part_info_list || this.part_info_list.length === 0) && this.initChunks();
    let t;
    if (this.upload_id && this.confirm_key) {
      const i = await this.renewUploadTask();
      t = {
        domain: i.domain,
        path: i.path || `/${this.key}`,
        uploadId: this.upload_id,
        confirmKey: this.confirm_key,
        expiration: i.expiration,
        headers: i.headers
      };
    } else {
      const i = e ? { beginningHash: e, size: String(this.file.size) } : {};
      await this.changeState(
        "created"
        /* CREATED */
      );
      let r = await this.fileApi.multipartUploadFile({
        libraryId: this.options.libraryId,
        spaceId: this.options.spaceId,
        filePath: this.options.filePath,
        multipart: 1,
        filesize: this.file.size,
        accessToken: this.options.accessToken,
        userId: this.options.userId,
        trafficLimit: this.options.trafficLimit,
        multipartUploadFileRequest: i,
        ...this.options.conflictResolutionStrategy && {
          conflictResolutionStrategy: this.options.conflictResolutionStrategy
        }
      }), s = r.status;
      if (s === 202) {
        await this.changeState(
          "computing_hash"
          /* COMPUTING_HASH */
        );
        const d = await Fi(this.options.file, this.file.size, (c) => {
          this.notifyProgress("computing_hash", c);
        });
        if (this.throwIfStopped("during full hash calculation"), r = await this.fileApi.multipartUploadFile({
          libraryId: this.options.libraryId,
          spaceId: this.options.spaceId,
          filePath: this.options.filePath,
          multipart: 1,
          filesize: this.file.size,
          accessToken: this.options.accessToken,
          userId: this.options.userId,
          trafficLimit: this.options.trafficLimit,
          multipartUploadFileRequest: {
            fullHash: d,
            beginningHash: e,
            size: String(this.file.size)
          },
          ...this.options.conflictResolutionStrategy && {
            conflictResolutionStrategy: this.options.conflictResolutionStrategy
          }
        }), s = r.status, s === 200) {
          if (this.rapid_upload = !0, this.loaded = this.file.size, this.progress = 100, this.updateProgress(this.file.size, { immediately: !0 }), this.start_time) {
            const c = Date.now() - this.start_time;
            this.used_avg_speed = c > 0 ? this.file.size / c * 1e3 : 0, this.speed = this.used_avg_speed;
          }
          await this.changeState(
            "rapid_success"
            /* RAPID_SUCCESS */
          ), this.logInfo(`Rapid upload success: ${this.file.name}`);
          return;
        }
      }
      t = r.data;
      const { bucket: o, region: n } = Bt(t.domain);
      this.confirm_key = t.confirmKey, this.upload_id = t.uploadId, this.bucket = o, this.region = n, this.key = ((a = t.path) == null ? void 0 : a.replace(/^\//, "")) || "", t.expiration && this.scheduleRenewal(t.expiration);
    }
    await this.changeState(
      "running"
      /* RUNNING */
    ), this.task_start_time = Date.now(), this.startCalcSpeed(), await this.multipartUpload(t), this.pauseCalcSpeed(), this.calcTotalAvgSpeed(), this.crc64 || (await this.changeState(
      "computing_hash"
      /* COMPUTING_HASH */
    ), this.crc64 = await this.calculateMultipartCRC64(), this.throwIfStopped("after CRC64 calculation")), await this.changeState(
      "confirming"
      /* CONFIRMING */
    ), await this.confirmUpload();
  }
  /**
   * 上传单个分片
   */
  async uploadSinglePart(e, t, a, i = 0) {
    if (this.throwIfStopped("during part upload"), e.etag)
      return;
    e.start_time = Date.now();
    const r = `https://${t.domain}${t.path || ""}?partNumber=${e.part_number}&uploadId=${this.upload_id}`, s = this.options.file.slice(e.from, e.to), o = await this.toUploadData(s);
    try {
      const n = await _.put(r, o, {
        headers: {
          ...a
        },
        maxContentLength: 1 / 0,
        maxBodyLength: 1 / 0,
        timeout: Math.max(3e5, Math.ceil(e.chunk_size / 102400) * 1e3),
        signal: this.abortSignal
      });
      e.etag = n.headers.etag || n.headers.ETag || "", e.end_time = Date.now(), e.crc64 || (e.crc64 = await ha(s)), this.loaded += e.chunk_size, this.updateProgress(this.loaded, { immediately: !0 }), this.notifyPartCompleted(e), this.logInfo(`Part ${e.part_number}/${this.part_info_list.length} uploaded, size: ${le(e.chunk_size)}`);
    } catch (n) {
      if (this.pauseFlag || this.cancelFlag || _.isCancel(n))
        throw n;
      if (i < this.MAX_RETRY_TIMES)
        return this.logWarn(`Part ${e.part_number} upload retry ${i + 1}/${this.MAX_RETRY_TIMES}`), await new Promise((d) => setTimeout(d, Math.min(1e3 * Math.pow(2, i), 1e4))), this.uploadSinglePart(e, t, a, i + 1);
      throw n;
    }
  }
  /**
   * 分块上传
   */
  async multipartUpload(e, t = 0) {
    const a = e.headers || {}, i = this.options.parallel || this.DEFAULT_PARALLEL;
    let r = 0;
    this.part_info_list && this.part_info_list.length > 0 && this.part_info_list.forEach((s) => {
      s.etag && (r += s.chunk_size);
    }), this.loaded = r, r > 0 ? this.updateProgress(r, { immediately: !0, init: !0 }) : this.updateProgress(0, { immediately: !0 });
    try {
      const s = this.part_info_list.filter((o) => !o.etag);
      await lr(
        s,
        i,
        async (o) => {
          await this.uploadSinglePart(o, e, a);
        },
        { shouldStop: () => this.pauseFlag || this.cancelFlag }
      ), this.updateProgress(this.file.size, { immediately: !0 });
    } catch (s) {
      if (this.pauseFlag || this.cancelFlag)
        throw s;
      const { isExpired: o } = Nt(s);
      if (o && t < this.MAX_RETRY_TIMES)
        try {
          const n = await this.renewUploadTask();
          return this.logWarn(`Multipart upload retry ${t + 1}/${this.MAX_RETRY_TIMES}: signature expired`), this.multipartUpload({
            ...n,
            headers: n.headers
          }, t + 1);
        } catch (n) {
          throw X(
            "RenewUploadFailed",
            "Failed to renew multipart upload",
            n,
            { fileName: this.file.name, confirmKey: this.confirm_key }
          );
        }
      else
        throw X(
          "PartUploadFailed",
          "Multipart upload failed after retries",
          s,
          { fileName: this.file.name, fileSize: this.file.size, retryCount: t }
        );
    }
  }
  /**
   * 动态计算分块大小
   */
  calcAutoChunkSize(e, t) {
    const a = [1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 5120], i = 1e4;
    let r = 1024 * 1024;
    for (let s = 0; s < a.length && (r = a[s] * 1024 * 1024, !(e / r <= i)); s++)
      ;
    return Math.max(t, r);
  }
  /**
   * 初始化分片信息
   */
  initChunks() {
    const e = this.file.size, t = 1e4, a = 1 * 1024 * 1024, i = 5 * 1024 * 1024 * 1024, r = i * t;
    if (e > r)
      throw X(
        "FileTooLarge",
        `File size ${le(e)} exceeds maximum supported size ${le(r)}`,
        void 0,
        { fileSize: e, maxFileSize: r }
      );
    let s = this.calcAutoChunkSize(e, this.chunk_size);
    if (s < a && (s = a), s > i)
      throw X(
        "InvalidParameter",
        `Required chunk size ${le(s)} exceeds maximum chunk size ${le(i)}`,
        void 0,
        { chunkSize: s, maxChunkSize: i }
      );
    let o = Math.ceil(e / s);
    if (o > t)
      throw X(
        "InvalidParameter",
        `File size ${le(e)} requires ${o} parts, exceeds maximum ${t} parts`,
        void 0,
        { fileSize: e, partCount: o, maxPartCount: t }
      );
    this.chunk_size = s, this.part_info_list = [];
    for (let n = 0; n < o; n++) {
      const d = n * s, c = Math.min((n + 1) * s, e);
      this.part_info_list.push({
        part_number: n + 1,
        chunk_size: c - d,
        from: d,
        to: c
      });
    }
  }
  /**
   * 计算分块上传的 CRC64
   */
  async calculateMultipartCRC64() {
    if (!this.part_info_list || this.part_info_list.length === 0)
      throw X(
        "OperationFailed",
        "No part info available for CRC64 calculation",
        void 0,
        { fileName: this.file.name }
      );
    for (const e of this.part_info_list)
      if (!e.crc64) {
        const t = this.options.file.slice(e.from, e.to);
        e.crc64 = await ha(t);
      }
    return fr(
      this.part_info_list.map((e) => ({ crc64: e.crc64, size: e.chunk_size }))
    );
  }
  /**
   * 确认上传
   */
  async confirmUpload() {
    await this.fileApi.completeFileUpload({
      libraryId: this.options.libraryId,
      spaceId: this.options.spaceId,
      confirmKey: this.confirm_key,
      confirm: 1,
      accessToken: this.options.accessToken,
      userId: this.options.userId,
      completeFileUploadRequest: {
        crc64: this.crc64
      },
      ...this.options.conflictResolutionStrategy && {
        conflictResolutionStrategy: this.options.conflictResolutionStrategy
      }
    });
  }
  /**
   * 续期上传任务
   */
  async renewUploadTask() {
    if (!this.confirm_key)
      throw X(
        "RenewUploadFailed",
        "Cannot renew upload task: confirm_key is missing",
        void 0,
        { fileName: this.file.name }
      );
    try {
      const t = (await this.fileApi.renewMultipartUpload({
        libraryId: this.options.libraryId,
        spaceId: this.options.spaceId,
        confirmKey: this.confirm_key,
        renew: 1,
        trafficLimit: this.options.trafficLimit,
        accessToken: this.options.accessToken,
        userId: this.options.userId
      })).data;
      if (t.domain) {
        const { bucket: a, region: i } = Bt(t.domain);
        this.bucket = a, this.region = i, t.expiration && this.scheduleRenewal(t.expiration);
      }
      return t;
    } catch (e) {
      const t = X(
        "RenewUploadFailed",
        "Failed to renew upload task",
        e,
        { confirmKey: this.confirm_key }
      );
      throw await this.handleError(t), t;
    }
  }
  /**
   * 安排续期定时器
   */
  scheduleRenewal(e) {
    this.clearRenewalTimer();
    const t = new Date(e).getTime(), a = Date.now(), i = (t - a) / 1e3, r = t - a - 5 * 60 * 1e3;
    if (i < 5 * 60) {
      this.renewUploadTask();
      return;
    }
    r > 0 && this.state === "running" && (this.renewTimer = setTimeout(() => {
      this.renewUploadTask();
    }, r));
  }
  /**
   * 清除续期定时器
   */
  clearRenewalTimer() {
    this.renewTimer && (clearTimeout(this.renewTimer), this.renewTimer = void 0);
  }
  async toUploadData(e) {
    if (e == null || typeof Buffer < "u" && Buffer.isBuffer(e) || typeof e.pipe == "function" || e instanceof ArrayBuffer || typeof Uint8Array < "u" && e instanceof Uint8Array || this.isNativeBlob(e)) return e;
    if (typeof e.arrayBuffer == "function" && typeof e.size == "number") {
      const t = await e.arrayBuffer();
      return typeof Buffer < "u" ? Buffer.from(t) : t;
    }
    return e;
  }
  isNativeBlob(e) {
    const t = e == null ? void 0 : e[Symbol.toStringTag];
    return t === "Blob" || t === "File" || typeof Blob < "u" && e instanceof Blob;
  }
  /**
   * 改变状态
   */
  async changeState(e, t) {
    await super.changeState(e, t);
    const a = this.getCheckpoint();
    if (typeof this.options.onStateChange == "function")
      try {
        await this.options.onStateChange(a, e, t);
      } catch {
      }
  }
  /**
   * 通知进度
   */
  notifyProgress(e, t) {
    super.notifyProgress(e, t), typeof this.options.onProgress == "function" && this.options.onProgress({
      loaded: this.loaded,
      total: this.file.size,
      progress: t,
      speed: this.speed,
      leftTime: this.left_time
    });
  }
  /**
   * 通知分片完成
   */
  notifyPartCompleted(e) {
    const t = this.getCheckpoint();
    typeof this.options.onPartComplete == "function" && this.options.onPartComplete(t, e), this.emit("partialcomplete", { checkpoint: t, partInfo: e });
  }
  /**
   * 处理错误
   */
  async handleError(e) {
    let t;
    return e instanceof Jt ? t = e : t = X(
      "UploadFailed",
      e.message || "Upload failed",
      e,
      {
        fileName: this.file.name,
        fileSize: this.file.size,
        elapsedTime: (this.end_time || Date.now()) - this.start_time
      }
    ), this.logError(`Upload failed: ${this.file.name}, error: ${t.message}`), super.handleError(t);
  }
}, xi = class extends Ar {
  constructor(e, t, a) {
    if (!e || !e.path)
      throw X(
        "InvalidParameter",
        "Invalid remote file: file and file.path are required",
        void 0,
        { file: e }
      );
    const i = {
      name: e.name,
      size: e.size || 0,
      type: e.type
    };
    super(i, { verbose: t.verbose }), this.part_info_list = [], this.is_multipart = !1, this.local_crc64 = je, this.DEFAULT_PARALLEL = 2, this.options = t, this.fileApi = new Dt(a), this.MULTIPART_THRESHOLD = (t.partFileSize || 32) * 1024 * 1024, this.chunk_size = (t.chunkSize || 5) * 1024 * 1024, t.checkpoint && this.restoreCheckpoint(t.checkpoint);
  }
  /**
   * 通过浏览器 URL 方式下载文件（推荐用于 Web 端）
   * 获取 cosUrl 后通过 <a> 标签触发浏览器原生下载，
   * 不会将文件内容加载到内存中，适合任意大小的文件。
   * 
   * @param options - URL 下载选项
   * @param configuration - SDK 配置
   * 
   */
  static async downloadByUrl(e, t) {
    const r = (await new Dt(t).infoFile({
      libraryId: e.libraryId,
      spaceId: e.spaceId,
      filePath: e.filePath,
      info: 1,
      contentDisposition: "attachment",
      accessToken: e.accessToken,
      userId: e.userId,
      trafficLimit: e.trafficLimit,
      purpose: "download"
    })).data, s = r == null ? void 0 : r.cosUrl;
    if (!s)
      throw X(
        "OperationFailed",
        "Failed to get download URL: cosUrl not found in response",
        void 0,
        { filePath: e.filePath }
      );
    const o = e.fileName || e.filePath.split("/").pop() || "download", n = document.createElement("a");
    n.href = s, n.download = o, n.style.display = "none", document.body.appendChild(n), n.click(), document.body.removeChild(n);
  }
  getTaskType() {
    return "download";
  }
  /**
   * 恢复 checkpoint
   */
  restoreCheckpoint(e) {
    this.state = e.state, this.progress = e.progress, this.loaded = e.loaded, this.startSize = e.loaded, this.lastProgressLoaded = e.loaded, this.download_url = e.download_url, this.chunk_size = e.chunk_size, this.part_info_list = e.part_info_list || [], this.remote_crc64 = e.remote_crc64, this.is_multipart = e.is_multipart || !1, this.start_time = e.start_time || 0, this.end_time = e.end_time || 0, this.used_avg_speed = e.used_avg_speed || 0, this.used_time_len = e.used_time_len || 0;
  }
  /**
   * 获取 checkpoint 信息
   */
  getCheckpoint() {
    return {
      id: this.id,
      file: {
        name: this.file.name,
        size: this.file.size,
        type: this.file.type
      },
      state: this.state,
      progress: this.progress,
      loaded: this.loaded,
      download_url: this.download_url,
      chunk_size: this.chunk_size,
      part_info_list: this.part_info_list.map((e) => ({
        part_number: e.part_number,
        start: e.start,
        end: e.end,
        size: e.size,
        done: e.done,
        crc64: e.crc64
      })),
      remote_crc64: this.remote_crc64,
      is_multipart: this.is_multipart,
      start_time: this.start_time,
      end_time: this.end_time,
      used_avg_speed: this.used_avg_speed,
      used_time_len: this.used_time_len
    };
  }
  /**
   * 开始下载
   * @returns 下载完成后返回 Blob
   */
  async start() {
    ["waiting", "error", "paused", "canceled"].includes(this.state) && (await this.changeState(
      "start"
      /* START */
    ), await this.doStart());
  }
  /**
   * 开始下载并返回结果
   * @returns 下载完成后返回 Blob
   */
  async startAndGetBlob() {
    if (!["waiting", "error", "paused", "canceled"].includes(this.state)) {
      if (this.resultBlob)
        return this.resultBlob;
      throw X("OperationFailed", "Download already in progress");
    }
    return await this.changeState(
      "start"
      /* START */
    ), await this.doStartAndGetBlob();
  }
  /**
   * 执行开始
   */
  async doStart() {
    this.pauseFlag = !1, this.cancelFlag = !1;
    try {
      await this.run();
    } catch (e) {
      if (this.pauseFlag || this.cancelFlag)
        return;
      const t = e;
      if ((t == null ? void 0 : t.name) === "AbortError")
        return;
      await this.handleError(t);
    }
  }
  /**
   * 执行开始并返回结果
   */
  async doStartAndGetBlob() {
    this.pauseFlag = !1, this.cancelFlag = !1;
    try {
      return await this.run();
    } catch (e) {
      if (this.pauseFlag || this.cancelFlag)
        throw e;
      const t = e;
      throw (t == null ? void 0 : t.name) === "AbortError" || await this.handleError(t), e;
    }
  }
  /**
   * 暂停下载
   */
  async pause() {
    this.is_multipart || (this.local_crc64 = je), this.logWarn(`Task paused: ${this.file.name}, progress: ${this.progress.toFixed(2)}%`), await super.pause();
  }
  /**
   * 取消下载
   */
  async cancel() {
    this.is_multipart && this.part_info_list && this.part_info_list.length > 0 && this.part_info_list.forEach((e) => {
      e.blob = void 0, e.done = !1;
    }), this.loaded = 0, this.progress = 0, this.resultBlob = void 0, this.logWarn(`Task canceled: ${this.file.name}`), await super.cancel();
  }
  /**
   * 获取下载结果
   */
  getResult() {
    return this.resultBlob;
  }
  /**
   * 主运行流程
   */
  async run() {
    this.start_time || (this.start_time = Date.now()), await this.getDownloadUrl();
    const e = (this.file.size || 0) > this.MULTIPART_THRESHOLD;
    this.is_multipart = e, await this.changeState(
      "preparing"
      /* PREPARING */
    ), e ? ((!this.part_info_list || this.part_info_list.length === 0) && this.initChunks(), this.updateProgress(this.loaded, { immediately: !0, init: !0 }), await this.changeState(
      "running"
      /* RUNNING */
    ), this.task_start_time = Date.now(), this.startCalcSpeed(), await this.multipartDownload(), this.pauseCalcSpeed(), this.calcTotalAvgSpeed()) : (await this.changeState(
      "running"
      /* RUNNING */
    ), this.task_start_time = Date.now(), this.startCalcSpeed(), await this.simpleDownloadWithRetry(), this.pauseCalcSpeed(), this.calcTotalAvgSpeed());
    const t = await this.finalizeDownload();
    this.end_time = Date.now(), await this.changeState(
      "success"
      /* SUCCESS */
    );
    const a = this.end_time - this.start_time;
    return this.logInfo(`Download success: ${this.file.name}, size: ${le(this.file.size)}, time: ${cr(a)}, speed: ${le(this.used_avg_speed || 0)}/s`), t;
  }
  /**
   * 检查是否被停止
   */
  throwIfStopped(e) {
    if (this.pauseFlag || this.cancelFlag)
      throw X(
        this.pauseFlag ? "DownloadPaused" : "DownloadCanceled",
        `Download stopped ${e}`,
        void 0,
        { fileName: this.file.name }
      );
  }
  /**
   * 获取下载 URL
   */
  async getDownloadUrl() {
    try {
      const t = (await this.fileApi.infoFile({
        libraryId: this.options.libraryId,
        spaceId: this.options.spaceId,
        filePath: this.options.filePath,
        info: 1,
        accessToken: this.options.accessToken,
        userId: this.options.userId,
        trafficLimit: this.options.trafficLimit,
        purpose: "download"
      })).data, a = t == null ? void 0 : t.cosUrl;
      t != null && t.size && (this.file.size = Number(t.size)), t != null && t.crc64 && !this.remote_crc64 && (this.remote_crc64 = t.crc64), this.download_url = a;
    } catch (e) {
      throw e;
    }
  }
  /**
   * 初始化分片列表
   */
  initChunks() {
    const e = this.file.size || 0, t = this.chunk_size, a = Math.ceil(e / t);
    this.part_info_list = [];
    for (let i = 0; i < a; i++) {
      const r = i * t, s = Math.min((i + 1) * t, e) - 1;
      this.part_info_list.push({
        part_number: i + 1,
        start: r,
        end: s,
        size: s - r + 1,
        done: !1
      });
    }
  }
  /**
   * 带重试的简单下载
   */
  async simpleDownloadWithRetry(e = 0) {
    try {
      await this.simpleDownload();
    } catch (t) {
      if (this.pauseFlag || this.cancelFlag)
        throw t;
      const a = t;
      if ((a == null ? void 0 : a.name) === "AbortError")
        throw t;
      const { isExpired: i } = Nt(t);
      if (e < this.MAX_RETRY_TIMES)
        return this.logWarn(`Simple download retry ${e + 1}/${this.MAX_RETRY_TIMES}`), await new Promise((r) => setTimeout(r, Math.min(1e3 * Math.pow(2, e), 1e4))), i && await this.getDownloadUrl(), this.simpleDownloadWithRetry(e + 1);
      throw this.logError(`Simple download failed: ${this.file.name}`), t;
    }
  }
  /**
   * 简单下载
   */
  async simpleDownload() {
    var o;
    this.updateProgress(0, { immediately: !0 }), this.local_crc64 = je;
    const e = await fetch(this.download_url, {
      method: "GET",
      signal: this.abortSignal
    });
    if (!e.ok)
      throw new Error(`Download failed with status ${e.status}`);
    const t = e.headers.get("content-length"), a = t ? Number(t) : this.file.size || 0;
    a > 0 && this.file.size !== a && (this.file.size = a, this.updateProgress(0, { immediately: !0 }));
    const i = (o = e.body) == null ? void 0 : o.getReader();
    if (!i)
      throw new Error("Failed to get response reader");
    const r = [];
    let s = 0;
    try {
      for (; ; ) {
        const { done: n, value: d } = await i.read();
        if (n) break;
        d && (r.push(d.buffer.slice(d.byteOffset, d.byteOffset + d.byteLength)), s += d.length, this.local_crc64 = wa(this.local_crc64, d), this.updateProgress(s));
      }
    } finally {
      i.releaseLock();
    }
    this.resultBlob = new Blob(r, { type: this.file.type || "application/octet-stream" });
  }
  /**
   * 分片下载
   */
  async multipartDownload() {
    const e = this.options.parallel || this.DEFAULT_PARALLEL;
    let t = 0;
    this.part_info_list.forEach((r) => {
      r.done && r.blob && (t += r.size);
    }), this.loaded = t, t > 0 ? this.updateProgress(t, { immediately: !0, init: !0 }) : this.updateProgress(0, { immediately: !0 });
    const a = async (r, s = 0) => {
      var o;
      if (this.throwIfStopped("during multipart download"), !(r.done && r.blob))
        try {
          const n = {
            Range: `bytes=${r.start}-${r.end}`
          }, d = await fetch(this.download_url, {
            method: "GET",
            headers: n,
            signal: this.abortSignal
          });
          if (!d.ok && d.status !== 206)
            throw new Error(`Part download failed with status ${d.status}`);
          const c = (o = d.body) == null ? void 0 : o.getReader();
          if (!c)
            throw new Error("Failed to get response reader");
          const l = [];
          let h = je;
          try {
            for (; ; ) {
              const { done: p, value: y } = await c.read();
              if (p) break;
              y && (l.push(y.buffer.slice(y.byteOffset, y.byteOffset + y.byteLength)), h = wa(h, y));
            }
          } finally {
            c.releaseLock();
          }
          r.blob = new Blob(l), r.crc64 = Tt(h), r.done = !0, this.loaded += r.size, this.updateProgress(this.loaded, { immediately: !0 }), this.notifyPartCompleted(r), this.logInfo(`Part ${r.part_number}/${this.part_info_list.length} downloaded, size: ${le(r.size)}, crc64: ${r.crc64}`);
        } catch (n) {
          if (this.pauseFlag || this.cancelFlag)
            throw n;
          const d = n;
          if ((d == null ? void 0 : d.name) === "AbortError")
            throw n;
          const { isExpired: c } = Nt(n);
          if (s < this.MAX_RETRY_TIMES)
            return this.logWarn(`Part ${r.part_number} download retry ${s + 1}/${this.MAX_RETRY_TIMES}`), c && await this.getDownloadUrl(), await new Promise((l) => setTimeout(l, Math.min(1e3 * Math.pow(2, s), 1e4))), a(r, s + 1);
          throw n;
        }
    }, i = this.part_info_list.filter((r) => !r.done || !r.blob);
    await lr(
      i,
      e,
      async (r) => {
        await a(r);
      },
      { shouldStop: () => this.pauseFlag || this.cancelFlag }
    );
  }
  /**
   * 完成下载，验证并返回结果
   */
  async finalizeDownload() {
    var e;
    if (this.file.size) {
      const t = this.is_multipart ? this.part_info_list.reduce((a, i) => {
        var r;
        return a + (((r = i.blob) == null ? void 0 : r.size) || 0);
      }, 0) : ((e = this.resultBlob) == null ? void 0 : e.size) || 0;
      if (t !== this.file.size)
        throw X(
          "FileSizeMismatch",
          `Download size mismatch: expected ${this.file.size}, got ${t}`,
          void 0,
          { expectedSize: this.file.size, actualSize: t }
        );
    }
    if (this.remote_crc64) {
      let t;
      if (this.is_multipart ? t = this.combinePartCrc64() : t = Tt(this.local_crc64), t !== this.remote_crc64)
        throw X(
          "FileCrc64Mismatch",
          `Download CRC64 mismatch: expected ${this.remote_crc64}, got ${t}`
        );
    }
    if (this.is_multipart) {
      const t = this.part_info_list.sort((i, r) => i.part_number - r.part_number), a = t.map((i) => i.blob);
      this.resultBlob = new Blob(a, { type: this.file.type || "application/octet-stream" }), t.forEach((i) => {
        i.blob = void 0;
      });
    }
    return this.resultBlob;
  }
  /**
   * 合并分片 CRC64
   */
  combinePartCrc64() {
    const e = this.part_info_list.filter((t) => t.done && t.crc64).sort((t, a) => t.part_number - a.part_number).map((t) => ({ crc64: t.crc64, size: t.size }));
    return fr(e);
  }
  /**
   * 改变状态
   */
  async changeState(e, t) {
    await super.changeState(e, t);
    const a = this.getCheckpoint();
    if (typeof this.options.onStateChange == "function")
      try {
        await this.options.onStateChange(a, e, t);
      } catch {
      }
  }
  /**
   * 通知进度
   */
  notifyProgress(e, t) {
    super.notifyProgress(e, t), typeof this.options.onProgress == "function" && this.options.onProgress({
      loaded: this.loaded,
      total: this.file.size,
      progress: t,
      speed: this.speed,
      leftTime: this.left_time
    });
  }
  /**
   * 通知分片完成
   */
  notifyPartCompleted(e) {
    const t = this.getCheckpoint();
    typeof this.options.onPartComplete == "function" && this.options.onPartComplete(t, e), this.emit("partialcomplete", { checkpoint: t, partInfo: e });
  }
  /**
   * 处理错误
   */
  async handleError(e) {
    let t;
    return e instanceof Jt ? t = e : t = X(
      "DownloadFailed",
      e.message || "Download failed",
      e,
      {
        fileName: this.file.name,
        fileSize: this.file.size,
        elapsedTime: (this.end_time || Date.now()) - this.start_time
      }
    ), this.is_multipart || (this.resultBlob = void 0), this.logError(`Download failed: ${this.file.name}, error: ${t.message}`), super.handleError(t);
  }
}, Ir = class {
  constructor(e = {}) {
    var a;
    this.defaultLibraryId = e.libraryId, this.defaultSpaceId = e.spaceId, this.defaultAccessToken = e.accessToken, this.axiosInstance = _.create({
      timeout: e.timeout || 3e4,
      ...e.baseOptions,
      headers: {
        // TODO：暂定Client-Version，后面需改成X-SMH-SDK-Version
        "Client-Version": zn(),
        ...(a = e.baseOptions) == null ? void 0 : a.headers
      }
    }), this.setupRetryInterceptor(e.maxRetries || 3, e.retryDelay || 1e3), this.configuration = new Nn({
      basePath: e.basePath,
      baseOptions: e.baseOptions
    });
    const t = this.configuration.basePath;
    this._batch = new hn(this.configuration, t, this.axiosInstance), this._directory = new yn(this.configuration, t, this.axiosInstance), this._favorite = new fn(this.configuration, t, this.axiosInstance), this._file = new Dt(this.configuration, t, this.axiosInstance), this._history = new mn(this.configuration, t, this.axiosInstance), this._quota = new bn(this.configuration, t, this.axiosInstance), this._recent = new wn(this.configuration, t, this.axiosInstance), this._recycled = new Bn(this.configuration, t, this.axiosInstance), this._search = new Cn(this.configuration, t, this.axiosInstance), this._space = new xn(this.configuration, t, this.axiosInstance), this._task = new Un(this.configuration, t, this.axiosInstance), this._token = new Vn(this.configuration, t, this.axiosInstance), this._usage = new Dn(this.configuration, t, this.axiosInstance), this.batch = this.wrapApi(this._batch), this.directory = this.wrapApi(this._directory), this.favorite = this.wrapApi(this._favorite), this.file = this.wrapApi(this._file), this.history = this.wrapApi(this._history), this.quota = this.wrapApi(this._quota), this.recent = this.wrapApi(this._recent), this.recycled = this.wrapApi(this._recycled), this.search = this.wrapApi(this._search), this.space = this.wrapApi(this._space), this.task = this.wrapApi(this._task), this.token = this.wrapApi(this._token), this.usage = this.wrapApi(this._usage);
  }
  /**
   * 设置重试拦截器
   */
  setupRetryInterceptor(e, t) {
    this.axiosInstance.interceptors.response.use(void 0, async (a) => {
      const i = a.config;
      if (!i || (i._retryCount ?? 0) >= e)
        return Promise.reject(a);
      if (a.code === "ECONNABORTED" || a.code === "ETIMEDOUT" || a.response && a.response.status >= 500) {
        i._retryCount = (i._retryCount || 0) + 1;
        const r = t * Math.pow(2, i._retryCount - 1);
        return new Promise((s) => {
          setTimeout(() => {
            s(this.axiosInstance.request(i));
          }, r);
        });
      }
      return Promise.reject(a);
    });
  }
  /**
   * 更新配置
   */
  updateConfig(e) {
    e.basePath && (this.configuration.basePath = e.basePath), e.baseOptions && (this.configuration.baseOptions = {
      ...this.configuration.baseOptions,
      ...e.baseOptions
    });
  }
  /**
   * 获取当前配置
   */
  getConfig() {
    return this.configuration;
  }
  /**
   * 设置访问令牌
   */
  setAccessToken(e) {
    this.configuration.accessToken = e;
  }
  /**
   * 清除访问令牌
   */
  clearAccessToken() {
    this.configuration.accessToken = void 0;
  }
  /**
   * 更新默认的 libraryId
   */
  setDefaultLibraryId(e) {
    this.defaultLibraryId = e;
  }
  /**
   * 更新默认的 spaceId
   */
  setDefaultSpaceId(e) {
    this.defaultSpaceId = e;
  }
  /**
   * 更新默认的 accessToken
   */
  setDefaultAccessToken(e) {
    this.defaultAccessToken = e;
  }
  /**
   * 获取默认的 libraryId
   */
  getDefaultLibraryId() {
    return this.defaultLibraryId;
  }
  /**
   * 获取默认的 spaceId
   */
  getDefaultSpaceId() {
    return this.defaultSpaceId;
  }
  /**
   * 获取默认的 accessToken
   */
  getDefaultAccessToken() {
    return this.defaultAccessToken;
  }
  /**
   * 包装API实例，自动注入 libraryId 和 accessToken
   */
  wrapApi(e) {
    return new Proxy(e, {
      get: (t, a) => {
        const i = t[a];
        return typeof i != "function" ? i : async (...r) => {
          if (r.length > 0 && typeof r[0] == "object" && r[0] !== null) {
            const o = { ...r[0] };
            a !== "createToken" && a !== "renewToken" && !o.libraryId && this.defaultLibraryId && (o.libraryId = this.defaultLibraryId), !o.spaceId && this.defaultSpaceId && (o.spaceId = this.defaultSpaceId), !o.accessToken && this.defaultAccessToken && (o.accessToken = this.defaultAccessToken), r[0] = o;
          }
          const s = await i.apply(t, r);
          if (a === "deleteToken" && r.length > 0 && typeof r[0] == "object" && r[0] !== null) {
            const o = r[0];
            o.accessToken && o.accessToken === this.defaultAccessToken && (this.defaultAccessToken = void 0);
          }
          return s;
        };
      }
    });
  }
  /**
   * 创建上传任务
   * 自动注入 libraryId、spaceId、accessToken 和 configuration
   * @returns Uploader 实例
   */
  createUploadTask(e) {
    const t = {
      ...e,
      libraryId: e.libraryId || this.defaultLibraryId || "",
      spaceId: e.spaceId || this.defaultSpaceId || "",
      accessToken: e.accessToken || this.defaultAccessToken || ""
    };
    return new cc(t, this.configuration);
  }
  /**
   * 创建下载任务
   * 自动注入 libraryId、spaceId、accessToken 和 configuration
   * @returns Downloader 实例
   */
  createDownloadTask(e) {
    const t = {
      ...e,
      libraryId: e.libraryId || this.defaultLibraryId || "",
      spaceId: e.spaceId || this.defaultSpaceId || "",
      accessToken: e.accessToken || this.defaultAccessToken || ""
    }, a = {
      name: e.filePath.split("/").pop() || "unknown",
      path: e.filePath,
      size: void 0,
      type: void 0
    };
    return new xi(a, t, this.configuration);
  }
  /**
   * 通过浏览器 URL 方式下载文件（推荐用于 Web 端）
   * 获取 cosUrl 后通过 <a> 标签触发浏览器原生下载，
   * 不需要将文件内容加载到内存中，适合任意大小的文件。
   * 
   * @example
   * ```typescript
   * await client.downloadByUrl({
   *   filePath: 'docs/file.pdf',
   *   fileName: 'my-file.pdf'  // 可选，自定义保存文件名
   * });
   * ```
   */
  async downloadByUrl(e) {
    const t = {
      ...e,
      libraryId: e.libraryId || this.defaultLibraryId || "",
      spaceId: e.spaceId || this.defaultSpaceId || "",
      accessToken: e.accessToken || this.defaultAccessToken || ""
    };
    return xi.downloadByUrl(t, this.configuration);
  }
};
/*! Bundled license information:

hash-wasm/dist/index.esm.js:
  (*!
   * hash-wasm (https://www.npmjs.com/package/hash-wasm)
   * (c) Dani Biro
   * @license MIT
   *)
*/
let oe = "", Se = 0, ct = null, xe = null, Oa = "", Lt = "", Ua = "";
function lc(e) {
  e.spaceId && e.spaceId !== Lt && (oe = "", Se = 0), e.libraryId && (Oa = e.libraryId), e.spaceId && (Lt = e.spaceId), e.basePath && (Ua = e.basePath.replace(/\/+$/, "")), typeof e.getAccessToken == "function" && (ct = e.getAccessToken), typeof e.onError == "function" && (xe = e.onError);
}
function dc() {
  return oe;
}
function ka() {
  return Oa;
}
function Xt() {
  return Lt;
}
function Va() {
  return Ua;
}
function Oi() {
  return {
    expiresAt: Se
  };
}
function Ea() {
  return Se ? Date.now() > Se - 5 * 60 * 1e3 : !0;
}
function hc() {
  return Se ? Date.now() > Se : !0;
}
let pa = !1, st = null;
async function Wt() {
  if (oe && !Ea())
    return oe;
  if (!ct) {
    if (oe && !hc()) {
      if (Ea())
        try {
          const { renewTokenViaSdk: e } = await Promise.resolve().then(() => Ui), t = await e();
          if (t && t.accessToken)
            return oe = t.accessToken, t.expiresAt && (Se = t.expiresAt), console.log("[SMH] 通过 SDK renewToken 续期成功"), oe;
        } catch (e) {
          return console.warn("[SMH] SDK renewToken 续期失败，使用当前 token:", e.message), oe;
        }
      return oe;
    }
    if (oe)
      try {
        const { renewTokenViaSdk: e } = await Promise.resolve().then(() => Ui), t = await e();
        if (t && t.accessToken)
          return oe = t.accessToken, t.expiresAt && (Se = t.expiresAt), console.log("[SMH] 通过 SDK renewToken 续期成功（token 已过期）"), oe;
      } catch (e) {
        console.error("[SMH] SDK renewToken 续期失败:", e.message);
      }
    throw xe == null || xe({ type: "error", message: "SMH 访问令牌已过期，请刷新页面" }), new Error("SMH accessToken 已过期且未提供 getAccessToken 函数");
  }
  return pa && st || (pa = !0, st = (async () => {
    try {
      const e = await ct();
      if (e && e.accessToken)
        return oe = e.accessToken, e.expiresAt && (Se = e.expiresAt), oe;
      throw new Error("getAccessToken 未返回有效的 accessToken");
    } catch (e) {
      throw xe == null || xe({ type: "error", message: e.message || "SMH 令牌获取失败" }), e;
    } finally {
      pa = !1, st = null;
    }
  })()), st;
}
async function pc() {
  if (!ct)
    throw new Error("未提供 getAccessToken 函数");
  await Wt();
}
function Hc() {
  oe = "", Se = 0, ct = null, xe = null, Oa = "", Lt = "", Ua = "";
}
let Re = null;
async function se() {
  const e = await Wt();
  return Re ? Re.setDefaultAccessToken(e) : Re = new Ir({
    basePath: Va(),
    libraryId: ka(),
    spaceId: Xt(),
    accessToken: e
  }), Re;
}
function Qa() {
  Re = null;
}
async function mr() {
  const e = dc(), t = ka();
  if (!e || !t)
    throw new Error("无法续期：缺少 accessToken 或 libraryId");
  Re || (Re = new Ir({
    basePath: Va(),
    libraryId: t,
    spaceId: Xt(),
    accessToken: e
  }));
  const a = await Re.token.renewToken({
    libraryId: t,
    accessToken: e
  }), i = a.data.accessToken, r = a.data.expiresAt ? new Date(a.data.expiresAt).getTime() : Date.now() + 7200 * 1e3;
  return Re.setDefaultAccessToken(i), { accessToken: i, expiresAt: r };
}
async function gr(e = "", { page: t = 1, pageSize: a = 100 } = {}) {
  return (await (await se()).directory.listDirectoryByPage({
    filePath: e,
    byPage: 1,
    page: t,
    pageSize: a
  })).data;
}
async function br(e = "") {
  Array.isArray(e) && (e = e.join("/"));
  try {
    return await (await se()).file.deleteFile({
      filePath: e,
      permanent: 1
    }), !0;
  } catch {
    return !1;
  }
}
async function vr(e = "") {
  Array.isArray(e) && (e = e.join("/"));
  try {
    return await (await se()).directory.deleteDirectory({
      filePath: e,
      permanent: 1
    }), !0;
  } catch {
    return !1;
  }
}
async function Sr(e = "") {
  return Array.isArray(e) && (e = e.join("/")), (await (await se()).directory.infoFileOrDirectory({
    filePath: e,
    info: 1
  })).data;
}
async function wr(e = "") {
  return Array.isArray(e) && (e = e.join("/")), (await (await se()).directory.createDirectory({
    filePath: e,
    conflictResolutionStrategy: "rename"
  })).data;
}
async function Er(e, t) {
  return Array.isArray(e) && (e = e.join("/")), Array.isArray(t) && (t = t.join("/")), (await (await se()).file.moveFile({
    filePath: t,
    moveFileRequest: { from: e },
    conflictResolutionStrategy: "rename"
  })).data;
}
async function Br(e, t) {
  return Array.isArray(e) && (e = e.join("/")), Array.isArray(t) && (t = t.join("/")), (await (await se()).directory.moveDirectory({
    filePath: t,
    moveDirectoryRequest: { from: e },
    conflictResolutionStrategy: "rename"
  })).data;
}
async function _r(e, t) {
  return Array.isArray(e) && (e = e.join("/")), Array.isArray(t) && (t = t.join("/")), (await (await se()).file.moveFile({
    filePath: t,
    moveFileRequest: { from: e },
    conflictResolutionStrategy: "ask"
  })).data;
}
async function Rr(e, t) {
  return Array.isArray(e) && (e = e.join("/")), Array.isArray(t) && (t = t.join("/")), (await (await se()).directory.moveDirectory({
    filePath: t,
    moveDirectoryRequest: { from: e },
    conflictResolutionStrategy: "ask"
  })).data;
}
async function nt(e = "", t = !1) {
  if (Array.isArray(e) && (e = e.join("/")), t) {
    const c = (await (await se()).file.infoFile({
      filePath: e,
      info: 1,
      purpose: "preview"
    })).data;
    return c.cosUrl ? (await fetch(c.cosUrl)).text() : c;
  }
  const a = await Wt(), i = Va(), r = ka(), s = Xt(), o = e.split("/").map(encodeURIComponent).join("/");
  return `${i}/api/v1/file/${r}/${s}/${o}?access_token=${encodeURIComponent(a)}`;
}
async function Cr(e = "") {
  return Array.isArray(e) && (e = e.join("/")), (await (await se()).file.infoFile({
    filePath: e,
    info: 1,
    purpose: "preview"
  })).data.cosUrl || "";
}
async function Fr(e = "", t) {
  Array.isArray(e) && (e = e.join("/")), await (await se()).downloadByUrl({
    filePath: e,
    fileName: t
  });
}
async function xr(e, t = "", a = {}) {
  const i = e, r = yc(t);
  if (r)
    try {
      await (await se()).directory.checkDirectoryStatus({ filePath: r });
    } catch {
      await (await se()).directory.createDirectory({
        filePath: r,
        conflictResolutionStrategy: "rename"
      });
    }
  const s = await se();
  return new Promise((o, n) => {
    s.createUploadTask({
      filePath: t,
      file: i,
      conflictResolutionStrategy: "overwrite",
      onStateChange: (c, l, h) => {
        var p, y, u, A;
        console.log("onStateChange", c, l, h), a.onStateChangeCallback && a.onStateChangeCallback(l), l === kt.SUCCESS || l === kt.RAPID_SUCCESS ? Da({ name: i.name, path: t.split("/") }).then((I) => {
          a.onSuccessCallback && a.onSuccessCallback({ id: I, name: i.name }), o(c);
        }).catch(() => {
          a.onSuccessCallback && a.onSuccessCallback({ id: "", name: i.name }), o(c);
        }) : l === kt.ERROR && (a.onErrorCallback && a.onErrorCallback(((y = (p = h.cause) == null ? void 0 : p.response) == null ? void 0 : y.data) || h), n(((A = (u = h.cause) == null ? void 0 : u.response) == null ? void 0 : A.data) || h));
      },
      onProgress: (c) => {
        const l = Math.floor(c.progress), h = c.speed || 0;
        console.log(`进度: ${l}%, 速度: ${h} B/s`), a.onProgressCallback && a.onProgressCallback(l, h);
      }
    }).start();
  });
}
function yc(e) {
  if (!e || e === "/" || !e.includes("/")) return "";
  if (e.endsWith("/")) return e.slice(0, -1);
  const t = e.lastIndexOf("/");
  return t === 0 ? "/" : e.substring(0, t);
}
async function Da(e) {
  var r;
  const t = e.name || e.filename || "", a = t.split(".").pop().toLowerCase(), i = ((r = e.path) == null ? void 0 : r.join("/")) || t;
  return ["jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "avif", "mp4", "avi", "mov", "wmv", "flv", "mkv", "webm"].includes(a) ? await nt(i) : ["json", "txt", "md", "log", "docx"].includes(a) ? await nt(i, !0) : await nt(i);
}
async function Na() {
  try {
    const e = await se(), t = Xt(), i = (await e.usage.getUsage({
      spaceIds: t
    })).data, r = Array.isArray(i) ? i.find((s) => s.spaceId === t) || i[0] : i;
    return r ? {
      used: parseInt(r.size, 10) || 0,
      total: parseInt(r.capacity, 10) || 0
    } : null;
  } catch (e) {
    return console.error("获取空间使用量失败:", e.message), null;
  }
}
const K = {
  getFileList: gr,
  uploadFile: xr,
  getPreview: nt,
  getDocPreviewUrl: Cr,
  downloadFile: Fr,
  getFileInfo: Sr,
  getFilePreviewUrlOrContent: Da,
  delFile: br,
  delDirectory: vr,
  createDirectory: wr,
  moveFile: Er,
  moveDirectory: Br,
  renameFile: _r,
  renameDirectory: Rr,
  resetClient: Qa,
  renewTokenViaSdk: mr,
  getSpaceUsage: Na
}, Ui = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  createDirectory: wr,
  default: K,
  delDirectory: vr,
  delFile: br,
  downloadFile: Fr,
  getDocPreviewUrl: Cr,
  getFileInfo: Sr,
  getFileList: gr,
  getFilePreviewUrlOrContent: Da,
  getPreview: nt,
  getSpaceUsage: Na,
  moveDirectory: Br,
  moveFile: Er,
  renameDirectory: Rr,
  renameFile: _r,
  renewTokenViaSdk: mr,
  resetClient: Qa,
  uploadFile: xr
}, Symbol.toStringTag, { value: "Module" })), uc = 3e3, ki = {
  success: "✓",
  error: "✕",
  warning: "!",
  info: "i"
}, Vi = {
  success: { bg: "#e6f7ee", border: "#b7eb8f", color: "#389e0d", iconBg: "#52c41a" },
  error: { bg: "#fff1f0", border: "#ffa39e", color: "#cf1322", iconBg: "#ff4d4f" },
  warning: { bg: "#fffbe6", border: "#ffe58f", color: "#ad6800", iconBg: "#faad14" },
  info: { bg: "#e6f4ff", border: "#91caff", color: "#0958d9", iconBg: "#1677ff" }
};
function fc() {
  let e = document.getElementById("__smh_toast_container__");
  return e || (e = document.createElement("div"), e.id = "__smh_toast_container__", Object.assign(e.style, {
    position: "fixed",
    top: "16px",
    left: "50%",
    transform: "translateX(-50%)",
    zIndex: "999999",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    gap: "8px",
    pointerEvents: "none"
  }), document.body.appendChild(e)), e;
}
const ie = {};
ie.notify = ({ type: e = "info", message: t, duration: a = uc }) => {
  if (typeof t != "string" || !t) return;
  const i = Vi[e] || Vi.info, r = ki[e] || ki.info, s = document.createElement("div");
  Object.assign(s.style, {
    display: "inline-flex",
    alignItems: "center",
    gap: "8px",
    padding: "8px 16px",
    borderRadius: "8px",
    background: i.bg,
    border: `1px solid ${i.border}`,
    color: i.color,
    fontSize: "14px",
    lineHeight: "1.5",
    boxShadow: "0 4px 12px rgba(0,0,0,0.1)",
    pointerEvents: "auto",
    opacity: "0",
    transform: "translateY(-8px)",
    transition: "opacity 0.25s ease, transform 0.25s ease",
    maxWidth: "400px",
    wordBreak: "break-word"
  });
  const o = document.createElement("span");
  Object.assign(o.style, {
    width: "18px",
    height: "18px",
    borderRadius: "50%",
    background: i.iconBg,
    color: "#fff",
    fontSize: "11px",
    fontWeight: "700",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: "0",
    lineHeight: "1"
  }), o.textContent = r;
  const n = document.createElement("span");
  n.textContent = t, s.appendChild(o), s.appendChild(n);
  const d = fc();
  d.appendChild(s), requestAnimationFrame(() => {
    s.style.opacity = "1", s.style.transform = "translateY(0)";
  }), setTimeout(() => {
    s.style.opacity = "0", s.style.transform = "translateY(-8px)", setTimeout(() => {
      s.remove(), d.children.length === 0 && d.remove();
    }, 250);
  }, a);
};
const Ac = [
  // {
  //   id: 'storage',
  //   title: '',
  //   items: [
  //     { id: 'personal', name: '云空间', icon: '/assets/CodeBubbyAssets/3970_352035/43.svg' },
  //   ],
  // },
];
function Qi(e, t = 20) {
  const a = (e || "").split(".").pop().toLowerCase(), i = { fontSize: t, flexShrink: 0 };
  return a === "__dir__" ? /* @__PURE__ */ g(Di, { style: { ...i, color: "#ffb020" } }) : ["jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "avif", "tpg", "heif"].includes(a) ? /* @__PURE__ */ g(ns, { style: { ...i, color: "#0abf5b" } }) : ["mp4", "avi", "mov", "wmv", "flv", "mkv", "webm"].includes(a) ? /* @__PURE__ */ g(cs, { style: { ...i, color: "#7b61ff" } }) : a === "pdf" ? /* @__PURE__ */ g(ls, { style: { ...i, color: "#e34d59" } }) : ["doc", "docx"].includes(a) ? /* @__PURE__ */ g(ds, { style: { ...i, color: "#3370ff" } }) : ["xls", "xlsx", "csv"].includes(a) ? /* @__PURE__ */ g(hs, { style: { ...i, color: "#2ba471" } }) : ["ppt", "pptx"].includes(a) ? /* @__PURE__ */ g(ps, { style: { ...i, color: "#ed7b2f" } }) : ["json", "js", "ts", "jsx", "tsx", "html", "css", "less", "scss", "py", "java", "go", "c", "cpp", "h", "rs", "rb", "php", "sh", "yaml", "yml", "xml", "sql"].includes(a) ? /* @__PURE__ */ g(ys, { style: { ...i, color: "#a0a3b1" } }) : ["txt", "md", "log", "ini", "conf", "cfg"].includes(a) ? /* @__PURE__ */ g(us, { style: { ...i, color: "#86909c" } }) : ["zip", "rar", "7z", "tar", "gz", "bz2", "xz", "tgz"].includes(a) ? /* @__PURE__ */ g(fs, { style: { ...i, color: "#c9a06e" } }) : ["mp3", "wav", "flac", "aac", "ogg", "wma", "m4a"].includes(a) ? /* @__PURE__ */ g(As, { style: { ...i, color: "#e95fbc" } }) : /* @__PURE__ */ g(Is, { style: { ...i, color: "#a8b0b8" } });
}
const Ic = ["json", "txt", "md", "log", "doc", "docx", "pdf", "xls", "xlsx", "ppt", "pptx"], mc = ["json", "txt", "md", "log"];
function _t(e) {
  const t = (e || "").split(".").pop().toLowerCase();
  return ["doc", "docx", "pdf", "xls", "xlsx", "ppt", "pptx", "txt"].includes(t) ? "doc" : ["jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "avif"].includes(t) ? "image" : ["mp4", "avi", "mov", "wmv", "flv", "mkv", "webm"].includes(t) ? "video" : "other";
}
function $e(e) {
  if (!e || e <= 0)
    return "0 B";
  const t = ["B", "KB", "MB", "GB", "TB"], a = Math.min(Math.floor(Math.log(e) / Math.log(1024)), t.length - 1), i = e / 1024 ** a;
  return `${i.toFixed(i >= 10 ? 0 : 1)} ${t[a]}`;
}
function gc(e) {
  if (!e)
    return "";
  const t = new Date(e);
  if (Number.isNaN(t.getTime()))
    return "";
  const a = (i) => String(i).padStart(2, "0");
  return `${t.getFullYear()}/${a(t.getMonth() + 1)}/${a(t.getDate())} ${a(t.getHours())}:${a(t.getMinutes())}`;
}
function bc() {
  const [e, t] = z(""), [a, i] = z(!1);
  return Ce(() => {
    function r() {
      const { expiresAt: o } = Oi();
      if (!o) {
        t("");
        return;
      }
      const n = Date.now(), d = Math.floor((o - n) / 1e3);
      if (d <= 0) {
        t("已过期"), i(!0);
        return;
      }
      i(!1);
      const c = Math.floor(d / 86400), l = Math.floor(d % 86400 / 3600), h = Math.floor(d % 3600 / 60), p = d % 60;
      c > 0 ? t(`${c}天${l}小时后过期`) : l > 0 ? t(`${l}小时${h}分钟后过期`) : h > 0 ? t(`${h}分${p}秒后过期`) : t(`${p}秒后过期`);
    }
    r();
    const s = setInterval(r, 60 * 1e3);
    return () => clearInterval(s);
  }, []), e ? /* @__PURE__ */ V(
    "span",
    {
      className: "cd-sidebar-user-card__expire",
      style: {
        fontSize: "11px",
        color: a ? "#e54545" : "#8c8c8c",
        marginTop: "2px"
      },
      title: (() => {
        const { expiresAt: r } = Oi();
        return r ? `过期时间：${new Date(r).toLocaleString()}` : "";
      })(),
      children: [
        "🔑 ",
        e
      ]
    }
  ) : null;
}
function vc({ username: e, showUserCard: t = !1, quota: a = null, onQuotaChange: i = null }) {
  const [r, s] = z("personal"), [o, n] = z(!1), [d, c] = z([]), [l, h] = z(!1), [p, y] = z(0), [u, A] = z(0), [I, b] = z(""), [S, w] = z(0), [R, P] = z(""), [B, T] = z(null), [j, q] = z(null), [H, Oe] = z([]), [Te, we] = z(!1), [Ee, Ue] = z(null), [Z, Zt] = z(""), [fe, yt] = z({}), [Xe, pe] = z(null), [ke, Fe] = z(null), [We, Ze] = z(!1), [Or, ut] = z(!1), [Le, Yt] = z("新建文件夹"), [Ur, Ta] = z(!1), [ee, Ve] = z([]), [La, Pa] = z(!1), [za, Ma] = z(!1), [kr, ft] = z(!1), [me, Ye] = z([]), [Ha, $a] = z([]), [Vr, ja] = z(!1), [ne, Pe] = z(1), [ze] = z(50), [Me, Ga] = z(0), [Be, Ja] = z(null), [qt, ea] = z(""), [Qr, Ka] = z(!1), At = Rt(null), qe = Rt(null), Xa = Rt(!0), et = Z.trim() ? d.filter((m) => (m.name || m.filename || "").toLowerCase().includes(Z.trim().toLowerCase())) : d, Dr = async (m) => {
    if (!m || m.type === "dir") {
      pe(null), Fe(null);
      return;
    }
    pe(m), Ze(!0);
    try {
      const E = await K.getFileInfo(m.path || "");
      Fe(E);
    } catch (E) {
      console.error("获取文件信息失败", E), Fe(null);
    } finally {
      Ze(!1);
    }
  };
  Ce(() => {
    (async () => {
      const E = d.filter((Q) => {
        const N = Q.name || Q.filename || "";
        return _t(N) === "image" && Q.type !== "dir";
      });
      for (const Q of E) {
        const N = Q.path || Q.name || Q.filename;
        if (!fe[N])
          try {
            const L = await K.getPreview(Q.path || "");
            yt((D) => ({ ...D, [N]: L }));
          } catch (L) {
            console.error("加载缩略图失败", L);
          }
      }
    })();
  }, [d]);
  const It = (m) => m ? Array.isArray(m.path) ? m.path.join("/") : m.path ? String(m.path).replace(/^\/+/, "") : [...H, m.name || m.filename].filter(Boolean).join("/") : "", Nr = async (m, E = {}) => {
    if (!m)
      return;
    const { skipEnterDir: Q = !1 } = E;
    if (!Q && m.type === "dir") {
      Kr(m.name || m.filename);
      return;
    }
    if (m.type === "dir")
      return;
    const N = m.name || m.filename || "";
    if (_t(N) === "image") {
      try {
        const G = await K.getPreview(m.path || ""), ae = Array.isArray(m.path) ? m.path.join("/") : m.path || m.name || m.filename || "";
        T({ url: G, name: N, path: ae });
      } catch (G) {
        console.error("获取图片预览失败", G);
      }
      return;
    }
    const D = (N.split(".").pop() || "").toLowerCase();
    if (Ic.includes(D)) {
      const G = Array.isArray(m.path) ? m.path.join("/") : m.path || m.name || m.filename || "";
      try {
        const ae = await K.getDocPreviewUrl(m.path || "");
        if (mc.includes(D)) {
          const Qe = await K.getPreview(m.path || "", !0);
          q({ name: N, path: G, url: ae, content: Qe });
        } else
          q({ name: N, path: G, url: ae });
      } catch {
        try {
          const Qe = await K.getPreview(m.path || "", !0);
          q({ name: N, path: G, content: Qe });
        } catch {
          q({ name: N, path: G, content: "无法加载内容" });
        }
      }
    }
  }, Tr = (m) => {
    if (!m)
      return;
    const E = m.name || m.filename || "该文件", Q = ri.confirm({
      header: "删除确认",
      body: `确认删除 ${E} 吗？`,
      theme: "warning",
      confirmBtn: "删除",
      onConfirm: async () => {
        Q.destroy();
        const N = It(m);
        if (N)
          try {
            m.type === "dir" ? await K.delDirectory(N) : await K.delFile(N), await _e(H.join("/"), ne), i == null || i();
          } catch (L) {
            console.error("删除失败", L);
          }
      },
      onClose: () => {
        Q.destroy();
      }
    });
  }, Lr = async (m) => {
    if (!(!m || m.type === "dir"))
      try {
        const E = m.name || m.filename || "文件";
        await K.downloadFile(m.path || "", E);
      } catch (E) {
        console.error("下载失败", E);
      }
  }, Pr = async () => {
    if (ee.length !== 0) {
      Pa(!0);
      try {
        for (const m of ee) {
          if (m.type === "dir")
            continue;
          const E = m.name || m.filename || "文件";
          await K.downloadFile(m.path || "", E), await new Promise((Q) => setTimeout(Q, 300));
        }
      } finally {
        Pa(!1);
      }
    }
  }, zr = () => {
    if (ee.length === 0)
      return;
    const m = ri.confirm({
      header: "批量删除确认",
      body: `确认删除选中的 ${ee.length} 个文件吗？`,
      theme: "warning",
      confirmBtn: "删除",
      onConfirm: async () => {
        m.destroy();
        try {
          for (const E of ee) {
            const Q = It(E);
            Q && (E.type === "dir" ? await K.delDirectory(Q) : await K.delFile(Q));
          }
          Ve([]), await _e(H.join("/"), ne), i == null || i();
        } catch (E) {
          console.error("批量删除失败", E);
        }
      },
      onClose: () => {
        m.destroy();
      }
    });
  }, tt = async (m = "") => {
    ja(!0);
    try {
      const Q = ((await K.getFileList(m, { page: 1, pageSize: 200 })).contents || []).filter((N) => N.type === "dir");
      $a(Q);
    } catch (E) {
      console.error("获取目录列表失败", E), $a([]);
    } finally {
      ja(!1);
    }
  }, Mr = () => {
    ee.length !== 0 && (Ye([]), ft(!0), tt(""));
  }, Hr = (m) => {
    const E = [...me, m];
    Ye(E), tt(E.join("/"));
  }, $r = () => {
    const m = me.slice(0, -1);
    Ye(m), tt(m.join("/"));
  }, Wa = (m) => {
    if (m < 0)
      Ye([]), tt("");
    else {
      const E = me.slice(0, m + 1);
      Ye(E), tt(E.join("/"));
    }
  }, jr = async () => {
    if (ee.length === 0) return;
    const m = me.join("/"), E = H.join("/");
    if (m === E) {
      ie.notify({ type: "warning", message: "目标目录与当前目录相同" });
      return;
    }
    Ma(!0);
    try {
      for (const Q of ee) {
        const N = It(Q);
        if (!N) continue;
        const L = Q.name || Q.filename, D = m ? `${m}/${L}` : L;
        Q.type === "dir" ? await K.moveDirectory(N, D) : await K.moveFile(N, D);
      }
      ie.notify({ type: "success", message: `已移动 ${ee.length} 个文件` }), Ve([]), ft(!1), await _e(H.join("/"), ne), i == null || i();
    } catch (Q) {
      console.error("批量移动失败", Q), ie.notify({ type: "error", message: "移动失败，请重试" });
    } finally {
      Ma(!1);
    }
  }, Gr = (m, E) => {
    E.stopPropagation();
    const Q = m.path || m.name || m.filename;
    Ve((N) => N.some((D) => (D.path || D.name || D.filename) === Q) ? N.filter((D) => (D.path || D.name || D.filename) !== Q) : [...N, m]);
  }, Jr = () => {
    const m = et.filter((E) => E.type !== "dir");
    ee.length === m.length ? Ve([]) : Ve(m);
  }, Za = (m) => {
    const E = m.path || m.name || m.filename;
    return ee.some((Q) => (Q.path || Q.name || Q.filename) === E);
  }, _e = async (m = "", E = 1) => {
    try {
      const Q = await K.getFileList(m, { page: E, pageSize: ze }), N = (Q.contents || []).map((L) => ({
        ...L,
        type: L.type || _t(L.name || L.filename || "")
      }));
      c(N), Ga(Q.totalNum || N.length);
    } catch (Q) {
      console.error("获取文件失败", Q), c([]), Ga(0);
    }
  };
  Ce(() => {
    Pe(1), Ve([]), _e(H.join("/"), 1);
  }, [H]), Ce(() => {
    if (Xa.current) {
      Xa.current = !1;
      return;
    }
    _e(H.join("/"), ne);
  }, [ne]);
  const Kr = (m) => {
    Oe((E) => [...E, m]);
  }, Xr = (m, E) => {
    E && E.stopPropagation();
    const Q = m.name || m.filename || "";
    Ja(m), ea(Q), setTimeout(() => {
      if (qe.current) {
        qe.current.focus();
        const N = Q.lastIndexOf(".");
        N > 0 && m.type !== "dir" ? qe.current.setSelectionRange(0, N) : qe.current.select();
      }
    }, 50);
  }, mt = () => {
    Ja(null), ea("");
  }, Ya = async () => {
    var N;
    if (!Be || !qt.trim()) {
      mt();
      return;
    }
    const m = Be.name || Be.filename || "", E = qt.trim();
    if (E === m) {
      mt();
      return;
    }
    if (/[\\/:*?"<>|]/.test(E)) {
      ie.notify({ type: "warning", message: '名称不支持特殊字符 "\\/:*?"<>|"' });
      return;
    }
    if (E.length > 255) {
      ie.notify({ type: "warning", message: "名称长度不能超过255个字" });
      return;
    }
    Ka(!0);
    try {
      const L = It(Be), D = L.split("/");
      D[D.length - 1] = E;
      const G = D.join("/");
      Be.type === "dir" ? await K.renameDirectory(L, G) : await K.renameFile(L, G), mt(), await _e(H.join("/"), ne);
    } catch (L) {
      console.error("重命名失败", L), ((N = L == null ? void 0 : L.response) == null ? void 0 : N.status) === 409 || (L == null ? void 0 : L.status) === 409 ? ie.notify({ type: "error", message: "目标名称已存在，请使用其他名称" }) : ie.notify({ type: "error", message: "重命名失败" });
    } finally {
      Ka(!1);
    }
  }, Wr = (m) => {
    m.key === "Enter" ? (m.preventDefault(), Ya()) : m.key === "Escape" && (m.preventDefault(), mt());
  }, Zr = async () => {
    if (!Le.trim())
      return;
    if (/[\\/:*?"<>|]/.test(Le)) {
      ie.notify({ type: "warning", message: '名称不支持特殊字符 "\\/:*?"<>|"' });
      return;
    }
    if (Le.length > 255) {
      ie.notify({ type: "warning", message: "名称长度不能超过255个字" });
      return;
    }
    Ta(!0);
    try {
      const E = H.length > 0 ? `${H.join("/")}/${Le.trim()}` : Le.trim();
      await K.createDirectory(E), await _e(H.join("/"), ne), ut(!1), Yt("新建文件夹");
    } catch (E) {
      console.error("创建文件夹失败", E), ie.notify({ type: "error", message: "创建文件夹失败" });
    } finally {
      Ta(!1);
    }
  }, Yr = async (m) => {
    var Q, N, L;
    const E = (Q = m.target.files) == null ? void 0 : Q[0];
    if (E) {
      h(!0), y(0), b(E.name), w(E.size || 0), P("");
      try {
        const D = H.length > 0 ? `${H.join("/")}/${E.name}` : E.name;
        await K.uploadFile(E, D, {
          onProgressCallback: (G, ae) => {
            y(G), A(ae);
          },
          onStateChangeCallback: (G) => P(G),
          onSuccessCallback: () => {
          },
          onErrorCallback: () => {
          }
        }), await _e(H.join("/"), ne), i == null || i();
      } catch (D) {
        const G = (D == null ? void 0 : D.code) || ((N = D == null ? void 0 : D.response) == null ? void 0 : N.code) || "", ae = ((L = D == null ? void 0 : D.response) == null ? void 0 : L.message) || (D == null ? void 0 : D.message) || "未知错误";
        G === "QuotaLimitReached" ? ie.notify({ type: "error", message: "空间配额不足，请清理文件或联系管理员扩容" }) : ie.notify({ type: "error", message: `文件上传失败：${ae}` });
      } finally {
        h(!1), y(0), A(0), b(""), w(0), P(""), At.current && (At.current.value = "");
      }
    }
  }, qr = async (m) => {
    var Q, N, L;
    m.preventDefault(), we(!1);
    const E = (Q = m.dataTransfer.files) == null ? void 0 : Q[0];
    if (E) {
      h(!0), y(0), A(0), b(E.name), w(E.size || 0), P("");
      try {
        const D = H.length > 0 ? `${H.join("/")}/${E.name}` : E.name;
        await K.uploadFile(E, D, {
          onProgressCallback: (G, ae) => {
            y(G), A(ae);
          },
          onStateChangeCallback: (G) => P(G),
          onSuccessCallback: () => {
          },
          onErrorCallback: () => {
          }
        }), await _e(H.join("/"), ne), i == null || i();
      } catch (D) {
        console.error("文件上传失败", D != null && D.toLogString ? D.toLogString() : D);
        const G = (D == null ? void 0 : D.code) || ((N = D == null ? void 0 : D.response) == null ? void 0 : N.code) || "", ae = ((L = D == null ? void 0 : D.response) == null ? void 0 : L.message) || (D == null ? void 0 : D.message) || "未知错误";
        G === "QuotaLimitReached" ? ie.notify({ type: "error", message: "空间配额不足，请清理文件或联系管理员扩容" }) : ie.notify({ type: "error", message: `文件上传失败：${ae}` });
      } finally {
        h(!1), y(0), A(0), b(""), w(0), P("");
      }
    }
  }, es = (m) => {
    m.preventDefault(), we(!0);
  }, ts = (m) => {
    m.currentTarget.contains(m.relatedTarget) || we(!1);
  }, as = (m) => /* @__PURE__ */ V("div", { className: "cd-sidebar-group", children: [
    m.title ? /* @__PURE__ */ g("div", { className: "cd-sidebar-group-title", children: m.title }) : null,
    m.items.map((E) => /* @__PURE__ */ V(
      "button",
      {
        type: "button",
        className: `cd-sidebar-item${r === E.id ? " active" : ""}`,
        onClick: () => s(E.id),
        children: [
          /* @__PURE__ */ g("span", { className: "cd-sidebar-icon", children: /* @__PURE__ */ g("img", { src: E.icon, alt: E.name }) }),
          E.name,
          E.suffix ? /* @__PURE__ */ g("span", { className: "cd-sidebar-item__suffix", children: /* @__PURE__ */ g("img", { src: E.suffix, alt: "suffix" }) }) : null
        ]
      },
      E.id
    ))
  ] }, m.id), gt = [
    { label: "全部文件", path: [] },
    ...H.map((m, E) => ({ label: m, path: H.slice(0, E + 1) }))
  ];
  Z.trim() ? `${et.length}` : `${d.length}`;
  const is = (m, E) => {
    const Q = m.name || m.filename || "未命名文件", N = m.type === "dir", L = N ? "dir" : _t(Q), D = m.path || `${Q}-${E}`, G = Ee === D, ae = L === "image", Qe = ae ? fe[m.path || m.name || m.filename] : null, qa = Be && (Be.path || Be.name || Be.filename) === (m.path || m.name || m.filename), rs = G && !qa && ee.length === 0 ? /* @__PURE__ */ V("div", { className: "cd-row-actions", children: [
      /* @__PURE__ */ g(
        "button",
        {
          className: "cd-row-action-icon",
          onClick: (te) => Xr(m, te),
          title: "重命名",
          children: /* @__PURE__ */ g(os, { size: "16" })
        }
      ),
      /* @__PURE__ */ g(
        "button",
        {
          className: "cd-row-action-icon",
          onClick: (te) => {
            te.stopPropagation(), Lr(m);
          },
          disabled: N,
          title: "下载",
          children: /* @__PURE__ */ g(ti, { size: "16" })
        }
      ),
      /* @__PURE__ */ g(
        "button",
        {
          className: "cd-row-action-icon cd-row-action-icon--danger",
          onClick: (te) => {
            te.stopPropagation(), Tr(m);
          },
          title: "删除",
          children: /* @__PURE__ */ g(ai, { size: "16" })
        }
      )
    ] }) : N ? "-" : $e(m.size);
    return /* @__PURE__ */ V(
      "div",
      {
        className: `cd-list-row ${(Xe == null ? void 0 : Xe.path) === m.path ? "selected" : ""} ${Za(m) ? "checked" : ""}`,
        onClick: () => Dr(m),
        onDoubleClick: () => Nr(m),
        tabIndex: 0,
        onMouseEnter: () => Ue(D),
        onMouseLeave: () => Ue(null),
        onFocus: () => Ue(D),
        onBlur: (te) => {
          te.currentTarget.contains(te.relatedTarget) || Ue(null);
        },
        children: [
          /* @__PURE__ */ g("div", { className: "cd-col-checkbox", children: !N && /* @__PURE__ */ g(
            "input",
            {
              type: "checkbox",
              checked: Za(m),
              onChange: (te) => Gr(m, te),
              onClick: (te) => te.stopPropagation(),
              className: "cd-checkbox"
            }
          ) }),
          /* @__PURE__ */ V("div", { className: "cd-col-name", children: [
            /* @__PURE__ */ g("span", { className: `cd-file-icon ${ae && Qe ? "cd-file-thumbnail" : ""}`, children: ae && Qe ? /* @__PURE__ */ g("img", { src: Qe, alt: Q, className: "cd-thumbnail-img" }) : Qi(N ? "__dir__" : Q) }),
            qa ? /* @__PURE__ */ g(
              "input",
              {
                ref: qe,
                className: "cd-rename-input",
                value: qt,
                onChange: (te) => ea(te.target.value),
                onKeyDown: Wr,
                onBlur: Ya,
                onClick: (te) => te.stopPropagation(),
                onDoubleClick: (te) => te.stopPropagation(),
                disabled: Qr,
                maxLength: 255
              }
            ) : /* @__PURE__ */ g("span", { className: `cd-file-name-text ${N ? "cd-no-select" : ""}`, children: Q })
          ] }),
          /* @__PURE__ */ g("div", { className: "cd-col-time", children: gc(m.updateTime || m.creationTime) }),
          /* @__PURE__ */ g("div", { className: "cd-col-size", children: rs })
        ]
      },
      D
    );
  };
  return /* @__PURE__ */ V("div", { className: "cloud-drive-layout", children: [
    o && /* @__PURE__ */ g("div", { className: "cd-sidebar-overlay", onClick: () => n(!1) }),
    /* @__PURE__ */ V("nav", { className: `cd-sidebar ${o ? "cd-sidebar--open" : "cd-sidebar--closed"}`, children: [
      t && /* @__PURE__ */ g("div", { className: "cd-sidebar-top", children: /* @__PURE__ */ g("div", { className: "cd-sidebar-user-card", children: /* @__PURE__ */ V("div", { className: "cd-sidebar-user-card__info", children: [
        /* @__PURE__ */ g("span", { className: "cd-sidebar-user-card__role", children: "租户空间" }),
        /* @__PURE__ */ g(bc, {}),
        a && a.total > 0 && /* @__PURE__ */ V("div", { className: "cd-sidebar-user-card__quota", children: [
          /* @__PURE__ */ g("div", { className: "cd-sidebar-user-card__quota-bar", children: /* @__PURE__ */ g(
            "div",
            {
              className: "cd-sidebar-user-card__quota-fill",
              style: { width: `${Math.min(a.used / a.total * 100, 100)}%` }
            }
          ) }),
          /* @__PURE__ */ V("span", { className: "cd-sidebar-user-card__quota-text", children: [
            $e(a.used),
            " / ",
            $e(a.total)
          ] })
        ] })
      ] }) }) }),
      Ac.map((m) => as(m))
    ] }),
    /* @__PURE__ */ V(
      "main",
      {
        className: "cd-main",
        onDragOver: es,
        onDrop: qr,
        onDragLeave: ts,
        onClick: (m) => {
          m.target.closest(".cd-list-row") || (pe(null), Fe(null));
        },
        children: [
          Te ? /* @__PURE__ */ V("div", { className: "cd-drag-overlay", children: [
            /* @__PURE__ */ g("span", { className: "cd-drag-icon", children: /* @__PURE__ */ V("svg", { width: "48", height: "48", viewBox: "0 0 24 24", fill: "none", stroke: "#0052d9", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
              /* @__PURE__ */ g("path", { d: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" }),
              /* @__PURE__ */ g("polyline", { points: "17 8 12 3 7 8" }),
              /* @__PURE__ */ g("line", { x1: "12", y1: "3", x2: "12", y2: "15" })
            ] }) }),
            "释放文件以上传"
          ] }) : null,
          /* @__PURE__ */ V("div", { className: "cd-top-bar", children: [
            /* @__PURE__ */ V("div", { className: "cd-top-bar__left", children: [
              /* @__PURE__ */ V(
                "button",
                {
                  type: "button",
                  className: "cd-top-bar__btn cd-top-bar__btn--new",
                  onClick: () => {
                    Yt("新建文件夹"), ut(!0);
                  },
                  children: [
                    /* @__PURE__ */ V("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
                      /* @__PURE__ */ g("line", { x1: "12", y1: "5", x2: "12", y2: "19" }),
                      /* @__PURE__ */ g("line", { x1: "5", y1: "12", x2: "19", y2: "12" })
                    ] }),
                    "新建"
                  ]
                }
              ),
              /* @__PURE__ */ V(
                "button",
                {
                  type: "button",
                  className: "cd-top-bar__btn cd-top-bar__btn--upload",
                  onClick: () => {
                    var m;
                    return (m = At.current) == null ? void 0 : m.click();
                  },
                  disabled: l,
                  children: [
                    /* @__PURE__ */ V("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
                      /* @__PURE__ */ g("path", { d: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" }),
                      /* @__PURE__ */ g("polyline", { points: "17 8 12 3 7 8" }),
                      /* @__PURE__ */ g("line", { x1: "12", y1: "3", x2: "12", y2: "15" })
                    ] }),
                    l ? "文件上传中..." : "上传"
                  ]
                }
              ),
              /* @__PURE__ */ g(
                "input",
                {
                  type: "file",
                  ref: At,
                  style: { display: "none" },
                  onChange: Yr
                }
              )
            ] }),
            /* @__PURE__ */ V("div", { className: "cd-top-bar__right", children: [
              l && /* @__PURE__ */ V("div", { className: "cd-upload-progress", children: [
                /* @__PURE__ */ V("div", { className: "cd-upload-progress__info", children: [
                  /* @__PURE__ */ g("span", { className: "cd-upload-progress__name", title: I, children: I }),
                  /* @__PURE__ */ g("span", { className: "cd-upload-progress__percent", children: R === "computing_hash" ? `秒传中 ${p}%` : `${p}% · ${$e(u)}/s` })
                ] }),
                /* @__PURE__ */ g("div", { className: "cd-upload-progress__bar", children: /* @__PURE__ */ g(
                  "div",
                  {
                    className: "cd-upload-progress__fill",
                    style: { width: `${p}%` }
                  }
                ) })
              ] }),
              a && /* @__PURE__ */ V("span", { className: "cd-top-bar__quota-text", children: [
                $e((a.used || 0) + (l ? Math.floor(S * p / 100) : 0)),
                a.total > 0 ? ` / ${$e(a.total)}` : ""
              ] })
            ] })
          ] }),
          H.length > 0 && /* @__PURE__ */ g("div", { className: "cd-breadcrumb", children: gt.map((m, E) => /* @__PURE__ */ V(ei.Fragment, { children: [
            /* @__PURE__ */ g(
              "button",
              {
                type: "button",
                className: E === gt.length - 1 ? "active" : "",
                title: m.label,
                onClick: () => {
                  E !== gt.length - 1 && Oe(m.path);
                },
                children: m.label
              }
            ),
            E < gt.length - 1 ? /* @__PURE__ */ g("span", { className: "cd-breadcrumb__sep", children: "/" }) : null
          ] }, m.label || E)) }),
          /* @__PURE__ */ V("div", { className: "cd-list-header", children: [
            /* @__PURE__ */ g("div", { className: "cd-col-checkbox", children: /* @__PURE__ */ g(
              "input",
              {
                type: "checkbox",
                checked: ee.length > 0 && ee.length === et.filter((m) => m.type !== "dir").length,
                onChange: Jr,
                className: "cd-checkbox"
              }
            ) }),
            /* @__PURE__ */ g("div", { className: "cd-col-name", children: "名称" }),
            /* @__PURE__ */ g("div", { className: "cd-col-time", children: "最近修改" }),
            /* @__PURE__ */ g("div", { className: "cd-col-size", children: "大小" })
          ] }),
          /* @__PURE__ */ g("div", { className: "cd-file-list-container", children: et.length === 0 ? /* @__PURE__ */ V("div", { className: "cd-file-empty", children: [
            /* @__PURE__ */ V("svg", { width: "64", height: "56", viewBox: "0 0 64 56", fill: "none", xmlns: "http://www.w3.org/2000/svg", children: [
              /* @__PURE__ */ g("path", { d: "M58 14H30L24 6H6C3.79 6 2 7.79 2 10V46C2 48.21 3.79 50 6 50H58C60.21 50 62 48.21 62 46V18C62 15.79 60.21 14 58 14Z", fill: "#E3E6EB" }),
              /* @__PURE__ */ g("path", { d: "M32 26V38M26 32H38", stroke: "#A6ADB4", strokeWidth: "2", strokeLinecap: "round" })
            ] }),
            Z.trim() ? "未找到匹配的文件" : "暂无文件"
          ] }) : et.map((m, E) => is(m, E)) }),
          Me > ze && /* @__PURE__ */ V("div", { className: "cd-pagination", children: [
            /* @__PURE__ */ V("span", { className: "cd-pagination__info", children: [
              "共 ",
              Me,
              " 项，第 ",
              ne,
              "/",
              Math.ceil(Me / ze),
              " 页"
            ] }),
            /* @__PURE__ */ V("div", { className: "cd-pagination__btns", children: [
              /* @__PURE__ */ g(
                "button",
                {
                  type: "button",
                  className: "cd-pagination__btn",
                  disabled: ne <= 1,
                  onClick: () => Pe((m) => Math.max(m - 1, 1)),
                  children: /* @__PURE__ */ g("svg", { width: "12", height: "12", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2.5", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ g("polyline", { points: "15 18 9 12 15 6" }) })
                }
              ),
              (() => {
                const m = Math.ceil(Me / ze), E = [], Q = 5;
                let N = Math.max(1, ne - Math.floor(Q / 2)), L = Math.min(m, N + Q - 1);
                L - N < Q - 1 && (N = Math.max(1, L - Q + 1)), N > 1 && (E.push(/* @__PURE__ */ g("button", { type: "button", className: "cd-pagination__btn cd-pagination__page", onClick: () => Pe(1), children: "1" }, 1)), N > 2 && E.push(/* @__PURE__ */ g("span", { className: "cd-pagination__ellipsis", children: "…" }, "s1")));
                for (let D = N; D <= L; D++)
                  E.push(
                    /* @__PURE__ */ g(
                      "button",
                      {
                        type: "button",
                        className: `cd-pagination__btn cd-pagination__page ${ne === D ? "cd-pagination__page--active" : ""}`,
                        onClick: () => Pe(D),
                        children: D
                      },
                      D
                    )
                  );
                return L < m && (L < m - 1 && E.push(/* @__PURE__ */ g("span", { className: "cd-pagination__ellipsis", children: "…" }, "s2")), E.push(/* @__PURE__ */ g("button", { type: "button", className: "cd-pagination__btn cd-pagination__page", onClick: () => Pe(m), children: m }, m))), E;
              })(),
              /* @__PURE__ */ g(
                "button",
                {
                  type: "button",
                  className: "cd-pagination__btn",
                  disabled: ne >= Math.ceil(Me / ze),
                  onClick: () => Pe((m) => Math.min(m + 1, Math.ceil(Me / ze))),
                  children: /* @__PURE__ */ g("svg", { width: "12", height: "12", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2.5", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ g("polyline", { points: "9 18 15 12 9 6" }) })
                }
              )
            ] })
          ] }),
          ee.length > 0 && /* @__PURE__ */ V("div", { className: "cd-batch-bar", children: [
            /* @__PURE__ */ V("div", { className: "cd-batch-bar__left", children: [
              /* @__PURE__ */ V("span", { className: "cd-batch-bar__count", children: [
                "已选 ",
                ee.length,
                " 项"
              ] }),
              /* @__PURE__ */ V(
                "button",
                {
                  type: "button",
                  className: "cd-batch-bar__btn",
                  onClick: Pr,
                  disabled: La,
                  children: [
                    /* @__PURE__ */ g(ti, { size: "12" }),
                    La ? "下载中..." : "下载"
                  ]
                }
              ),
              /* @__PURE__ */ V(
                "button",
                {
                  type: "button",
                  className: "cd-batch-bar__btn",
                  onClick: Mr,
                  children: [
                    /* @__PURE__ */ g(ms, { size: "12" }),
                    "移动"
                  ]
                }
              ),
              /* @__PURE__ */ V(
                "button",
                {
                  type: "button",
                  className: "cd-batch-bar__btn cd-batch-bar__btn--danger",
                  onClick: zr,
                  children: [
                    /* @__PURE__ */ g(ai, { size: "12" }),
                    "删除"
                  ]
                }
              )
            ] }),
            /* @__PURE__ */ g(
              "button",
              {
                type: "button",
                className: "cd-batch-bar__cancel",
                onClick: () => Ve([]),
                children: "取消选择"
              }
            )
          ] })
        ]
      }
    ),
    /* @__PURE__ */ g(
      Ec,
      {
        previewImg: B,
        setPreviewImg: T,
        previewDoc: j,
        setPreviewDoc: q
      }
    ),
    /* @__PURE__ */ g(
      ii,
      {
        visible: Or,
        header: "创建文件夹",
        onClose: () => ut(!1),
        onCancel: () => ut(!1),
        onConfirm: Zr,
        confirmBtn: { content: "确定", loading: Ur },
        cancelBtn: "取消",
        destroyOnClose: !0,
        children: /* @__PURE__ */ V("div", { className: "cd-create-folder-dialog", children: [
          /* @__PURE__ */ g(
            "input",
            {
              className: "smh-dialog__input",
              value: Le,
              onChange: (m) => Yt(m.target.value),
              maxLength: 255,
              placeholder: "请输入文件夹名称",
              autoFocus: !0
            }
          ),
          /* @__PURE__ */ g("div", { className: "cd-create-folder-tip", children: '名称不支持特殊字符 "\\/:*?"<>|"，长度不超过 255 个字' })
        ] })
      }
    ),
    /* @__PURE__ */ g(
      ii,
      {
        visible: kr,
        header: `移动 ${ee.length} 个文件到...`,
        onClose: () => ft(!1),
        onCancel: () => ft(!1),
        onConfirm: jr,
        confirmBtn: { content: za ? "移动中..." : "移动到此处", loading: za },
        cancelBtn: "取消",
        destroyOnClose: !0,
        children: /* @__PURE__ */ V("div", { className: "cd-move-dialog", children: [
          /* @__PURE__ */ V("div", { className: "cd-move-dialog__breadcrumb", children: [
            /* @__PURE__ */ g(
              "button",
              {
                type: "button",
                className: me.length === 0 ? "active" : "",
                onClick: () => Wa(-1),
                children: "根目录"
              }
            ),
            me.map((m, E) => /* @__PURE__ */ V(ei.Fragment, { children: [
              /* @__PURE__ */ g("span", { className: "cd-move-dialog__sep", children: "/" }),
              /* @__PURE__ */ g(
                "button",
                {
                  type: "button",
                  className: E === me.length - 1 ? "active" : "",
                  title: m,
                  onClick: () => Wa(E),
                  children: m
                }
              )
            ] }, E))
          ] }),
          /* @__PURE__ */ V("div", { className: "cd-move-dialog__list", children: [
            me.length > 0 && /* @__PURE__ */ V("div", { className: "cd-move-dialog__item cd-move-dialog__item--back", onClick: $r, children: [
              /* @__PURE__ */ g("svg", { width: "16", height: "16", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ g("polyline", { points: "15 18 9 12 15 6" }) }),
              /* @__PURE__ */ g("span", { children: "返回上级" })
            ] }),
            Vr ? /* @__PURE__ */ V("div", { className: "cd-move-dialog__empty", children: [
              /* @__PURE__ */ g("span", { className: "cd-move-dialog__spinner" }),
              "加载中..."
            ] }) : Ha.length === 0 ? /* @__PURE__ */ g("div", { className: "cd-move-dialog__empty", children: "暂无子文件夹" }) : Ha.map((m, E) => {
              const Q = m.name || m.filename, N = ee.some((L) => (L.name || L.filename) === Q && L.type === "dir");
              return /* @__PURE__ */ V(
                "div",
                {
                  className: `cd-move-dialog__item ${N ? "cd-move-dialog__item--disabled" : ""}`,
                  onClick: () => !N && Hr(Q),
                  children: [
                    /* @__PURE__ */ g(Di, { size: 18, style: { color: "#ffb020" } }),
                    /* @__PURE__ */ g("span", { className: "cd-move-dialog__item-name", title: Q, children: Q })
                  ]
                },
                E
              );
            })
          ] }),
          /* @__PURE__ */ V("div", { className: "cd-move-dialog__target", children: [
            "目标位置：",
            me.length > 0 ? `/${me.join("/")}` : "/（根目录）"
          ] })
        ] })
      }
    )
  ] });
}
function Sc({ item: e, onClose: t }) {
  const [a, i] = z(!0), [r, s] = z(!1);
  Ce(() => {
    function p(y) {
      y.key === "Escape" && t();
    }
    return document.addEventListener("keydown", p), () => document.removeEventListener("keydown", p);
  }, [t]);
  function o(p) {
    p.target === p.currentTarget && t();
  }
  function n() {
    i(!1);
  }
  function d() {
    i(!1), s(!0);
  }
  const c = e.path || "", l = !e.url && e.content, h = e.url ? `${e.url}${e.url.includes("?") ? "&" : "?"}ci-process=doc-preview&dstType=html&htmlwaterword=&htmlfillstyle=&htmlfront=&htmlrotate=&htmlhorizontal=&htmlvertical=` : "";
  return /* @__PURE__ */ V("div", { className: "doc-preview-mask", onClick: o, children: [
    /* @__PURE__ */ g("div", { className: "doc-preview-top", children: /* @__PURE__ */ g("div", { className: "doc-preview-top__info", children: /* @__PURE__ */ g("span", { className: "doc-preview-top__name", title: c, children: c || e.name }) }) }),
    /* @__PURE__ */ V("div", { className: "doc-preview-container", onClick: (p) => p.stopPropagation(), children: [
      !l && a && /* @__PURE__ */ V("div", { className: "doc-preview-loading", children: [
        /* @__PURE__ */ g("div", { className: "doc-preview-loading__spinner" }),
        /* @__PURE__ */ g("span", { className: "doc-preview-loading__text", children: "文档加载中..." })
      ] }),
      l ? (
        /* 纯文本展示 */
        /* @__PURE__ */ g("div", { className: "doc-preview-text", children: /* @__PURE__ */ g("pre", { className: "doc-preview-text__content", children: e.content || "无法加载文件内容" }) })
      ) : r ? /* @__PURE__ */ V("div", { className: "doc-preview-error", children: [
        /* @__PURE__ */ g("p", { className: "doc-preview-error__text", children: "文档预览加载失败" }),
        e.content && /* @__PURE__ */ g(
          "button",
          {
            className: "doc-preview-error__link",
            onClick: () => {
              s(!1), i(!1);
            },
            children: "切换为文本模式查看"
          }
        )
      ] }) : h && /* @__PURE__ */ g(
        "iframe",
        {
          className: "doc-preview-iframe",
          src: h,
          title: e.name || "文档预览",
          onLoad: n,
          onError: d,
          sandbox: "allow-scripts allow-same-origin allow-popups"
        }
      )
    ] }),
    /* @__PURE__ */ g("button", { className: "doc-preview-close", onClick: t, title: "关闭", children: "✕" })
  ] });
}
function wc({ item: e, onClose: t }) {
  Ce(() => {
    function r(s) {
      s.key === "Escape" && t();
    }
    return document.addEventListener("keydown", r), () => document.removeEventListener("keydown", r);
  }, [t]);
  function a(r) {
    r.target === r.currentTarget && t();
  }
  const i = e.path || "";
  return /* @__PURE__ */ V("div", { className: "img-preview-mask", onClick: a, children: [
    /* @__PURE__ */ g(
      "img",
      {
        className: "img-preview-photo",
        src: e.url,
        alt: e.name
      }
    ),
    i && /* @__PURE__ */ g("div", { className: "img-preview-top", children: /* @__PURE__ */ g("div", { className: "img-preview-top__info", children: /* @__PURE__ */ g("span", { className: "img-preview-top__path", title: i, children: i }) }) }),
    /* @__PURE__ */ g("button", { className: "img-preview-close", onClick: t, title: "关闭", children: "✕" })
  ] });
}
function Ec({ previewImg: e, setPreviewImg: t, previewDoc: a, setPreviewDoc: i }) {
  return /* @__PURE__ */ V(ss, { children: [
    e && /* @__PURE__ */ g(
      wc,
      {
        item: e,
        onClose: () => t(null)
      }
    ),
    a && /* @__PURE__ */ g(
      Sc,
      {
        item: a,
        onClose: () => i(null)
      }
    )
  ] });
}
function $c({
  basePath: e,
  libraryId: t,
  spaceId: a,
  getAccessToken: i,
  showUserCard: r = !0
}) {
  const [s, o] = z(!0), [n, d] = z(null), [c, l] = z(null), h = async () => {
    try {
      const p = await Na();
      p && l(p);
    } catch (p) {
      console.error("获取配额失败:", p.message);
    }
  };
  return Ce(() => {
    if (!e || !t || !a || typeof i != "function") {
      d("缺少必要参数：basePath、libraryId、spaceId、getAccessToken 均为必填项"), o(!1);
      return;
    }
    lc({
      basePath: e,
      libraryId: t,
      spaceId: a,
      getAccessToken: i,
      // UI 组件层提供 Toast 错误提示，服务层本身不依赖任何 UI
      onError: ({ message: p }) => ie.notify({ type: "error", message: p })
    }), Qa(), pc().then(() => h()).catch((p) => {
      console.error("初始化失败:", p.message);
    }).finally(() => {
      o(!1);
    });
  }, [e, t, a, i]), Ce(() => {
    if (s || n) return;
    const p = setInterval(() => {
      Ea() && Wt().catch(() => {
      });
    }, 30 * 1e3);
    return () => clearInterval(p);
  }, [s, n]), s ? /* @__PURE__ */ V("div", { style: { display: "flex", alignItems: "center", justifyContent: "center", height: "100%", gap: 10 }, children: [
    /* @__PURE__ */ g("span", { style: {
      width: 28,
      height: 28,
      border: "3px solid #e5e6eb",
      borderTopColor: "#0052d9",
      borderRadius: "50%",
      animation: "smhSpin 0.8s linear infinite",
      display: "inline-block"
    } }),
    /* @__PURE__ */ g("style", { children: "@keyframes smhSpin { to { transform: rotate(360deg); } }" }),
    /* @__PURE__ */ g("span", { style: { fontSize: 14, color: "rgba(0,0,0,0.45)" }, children: "正在初始化空间..." })
  ] }) : n ? /* @__PURE__ */ V("div", { style: { display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", height: "100%", gap: 16 }, children: [
    /* @__PURE__ */ V("svg", { width: "48", height: "48", viewBox: "0 0 48 48", fill: "none", children: [
      /* @__PURE__ */ g("circle", { cx: "24", cy: "24", r: "22", stroke: "#e34d59", strokeWidth: "3", fill: "#fff1f0" }),
      /* @__PURE__ */ g("path", { d: "M24 14v14", stroke: "#e34d59", strokeWidth: "3", strokeLinecap: "round" }),
      /* @__PURE__ */ g("circle", { cx: "24", cy: "34", r: "2", fill: "#e34d59" })
    ] }),
    /* @__PURE__ */ g("span", { style: { fontSize: 14, color: "rgba(0,0,0,0.45)" }, children: n })
  ] }) : /* @__PURE__ */ g("div", { style: { display: "flex", flexDirection: "column", height: "100%" }, children: /* @__PURE__ */ g("div", { style: { flex: 1, overflow: "hidden" }, children: /* @__PURE__ */ g(
    vc,
    {
      username: a,
      showUserCard: r,
      quota: c,
      onQuotaChange: h
    }
  ) }) });
}
export {
  vc as FilePage,
  $c as SpaceDrive,
  Hc as clearConfig,
  wr as createDirectory,
  vr as delDirectory,
  br as delFile,
  Fr as downloadFile,
  Wt as ensureValidToken,
  dc as getAccessToken,
  Va as getBasePath,
  Cr as getDocPreviewUrl,
  Sr as getFileInfo,
  gr as getFileList,
  Da as getFilePreviewUrlOrContent,
  ka as getLibraryId,
  nt as getPreview,
  Xt as getSpaceId,
  Na as getSpaceUsage,
  Oi as getTokenExpireInfo,
  pc as initToken,
  hc as isTokenExpired,
  Ea as isTokenExpiringSoon,
  Br as moveDirectory,
  Er as moveFile,
  Rr as renameDirectory,
  _r as renameFile,
  mr as renewTokenViaSdk,
  Qa as resetClient,
  lc as setSmhConfig,
  xr as uploadFile
};
