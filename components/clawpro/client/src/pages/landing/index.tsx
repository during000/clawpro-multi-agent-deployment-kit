/**
 * LandingPage - 永辉 AI Agent 管控平台 首页
 *
 * 基于 Figma 高保真设计稿 1:1 还原
 * 包含：Navbar 滚动效果 + 1920 等比缩放
 */
import { useEffect, useRef } from "react";

import "./landing.css";
import Features from "./Features";
import Hero from "./Hero";
import Navbar from "./Navbar";

export default function LandingPage() {
  const rootRef = useRef<HTMLDivElement | null>(null);

  // ---------- Navbar 滚动效果 ----------
  useEffect(() => {
    const navbar = rootRef.current?.querySelector(".yh-navbar");
    if (!navbar) return;
    const onScroll = () => {
      if (window.scrollY > 10) navbar.classList.add("is-scrolled");
      else navbar.classList.remove("is-scrolled");
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // ---------- 入场动画：所有 .yh-reveal 元素挂载后立即播放 ----------
  // 不再使用 IntersectionObserver，避免首次加载时下方模块要等用户滚动才出现，
  // 误让用户以为页面已无更多内容。
  // 各元素的 delay 通过 inline style 的 --yh-d 控制，自然形成瀑布式呈现。
  // Footer 不参与（按需求保持静态）。
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const targets = root.querySelectorAll<HTMLElement>(
      ".yh-hero .yh-reveal",
    );
    if (!targets.length) return;
    // 等下一帧再加 class，确保浏览器能跑到 transition
    const raf = requestAnimationFrame(() => {
      targets.forEach((el) => el.classList.add("yh-revealed"));
    });
    return () => cancelAnimationFrame(raf);
  }, []);

  // ---------- Features 区域滚动渐显 ----------
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const features = root.querySelector<HTMLElement>(".yh-features");
    if (!features) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          features.classList.add("yh-features-visible");
          // 触发 Features 内部元素的 reveal 动画
          features
            .querySelectorAll<HTMLElement>(".yh-reveal")
            .forEach((el) => el.classList.add("yh-revealed"));
          observer.disconnect();
        }
      },
      { threshold: 0.05 },
    );
    observer.observe(features);
    return () => observer.disconnect();
  }, []);

  // ---------- 1920 等比缩放 ----------
  // Navbar 和页面其他内容统一按 1920 设计稿等比缩放，避免窗口 < 1920px 时
  // navbar 看起来"巨大"而下方内容已被 zoom 压缩造成的视觉割裂。
  // 同时设置最小缩放下限（MIN_SCALE），避免窗口越缩越小、内容跟着无限变小，
  // 视口宽度低于阈值后页面按 1920 基准展示，多余部分由水平滚动条承载。
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const wrapper = root.querySelector<HTMLElement>(".yh-page-wrapper");
    const navbar = root.querySelector<HTMLElement>(".yh-navbar");
    const hero = root.querySelector<HTMLElement>(".yh-hero");
    if (!wrapper) return;
    const DESIGN_WIDTH = 1920;
    // 最小缩放下限：避免窗口越小内容越小、出现"无限缩放"。
    // 视口宽度低于 MIN_VIEWPORT 时，缩放比锁定为 MIN_SCALE，
    // 页面按 1920 设计稿宽度展示，超出部分通过水平滚动查看。
    const MIN_SCALE = 0.667; // 约对应 1280px 视口
    const MIN_VIEWPORT = DESIGN_WIDTH * MIN_SCALE; // ≈ 1280
    const applyScale = () => {
      const w = window.innerWidth || document.documentElement.clientWidth;
      const vh = window.innerHeight || document.documentElement.clientHeight;
      let scale: number;
      if (w >= DESIGN_WIDTH) {
        scale = 1;
      } else if (w <= MIN_VIEWPORT) {
        scale = MIN_SCALE;
      } else {
        scale = w / DESIGN_WIDTH;
      }
      // Hero 占满整屏：wrapper 被 zoom 缩放后，渲染高度 = cssHeight * scale，
      // 因此 cssHeight = 视口高 / scale，渲染后恰好等于一个视口高度
      if (hero) {
        hero.style.height = `${vh / scale}px`;
      }
      // Navbar 不参与 zoom，固定真实尺寸（64px 高 / 28px padding / 28px logo），
      // 与用户端 TopNav (TenantLayout) 完全对齐，避免 landing ↔ 用户端 跳转时
      // 左上角标题位置发生像素级跳动。scroll-padding-top 直接 64px 即可。
      document.documentElement.style.scrollPaddingTop = `64px`;
      if (w >= DESIGN_WIDTH) {
        // 大屏：恢复 1:1
        (wrapper.style as unknown as { zoom: string }).zoom = "";
        wrapper.style.transform = "";
        wrapper.style.width = "";
        if (navbar) {
          (navbar.style as unknown as { zoom: string }).zoom = "";
        }
        return;
      }
      const wStyle = wrapper.style as unknown as { zoom: string };
      wStyle.zoom = String(scale);
      // 不支持 zoom 的浏览器（如 Firefox 老版本）回退到 transform
      if (typeof wStyle.zoom === "undefined" || wStyle.zoom === "") {
        wrapper.style.transform = `scale(${scale})`;
        wrapper.style.transformOrigin = "top left";
        wrapper.style.width = `${DESIGN_WIDTH}px`;
      }
      // Navbar 始终保持真实尺寸（不跟随 zoom），见上方注释
      if (navbar) {
        const nStyle = navbar.style as unknown as { zoom: string };
        nStyle.zoom = "";
      }
    };
    window.addEventListener("resize", applyScale);
    window.addEventListener("load", applyScale);
    applyScale();
    return () => {
      window.removeEventListener("resize", applyScale);
      window.removeEventListener("load", applyScale);
    };
  }, []);

  return (
    <div className="yh-root" ref={rootRef}>
      {/* 视频背景：放在 zoom 容器外，固定铺满视口 */}
      <video
        className="yh-hero-bg"
        src="/landing-assets/banner/login-bg.mp4"
        autoPlay
        loop
        muted
        playsInline
        preload="auto"
        aria-hidden="true"
      />

      {/* Navbar 与下方内容统一按 1920 等比缩放（在 useEffect 里通过 zoom 实现） */}
      <Navbar />

      <div className="yh-page-wrapper">
        <Hero />
        <Features />
      </div>
    </div>
  );
}
