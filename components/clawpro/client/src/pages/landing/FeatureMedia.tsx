/**
 * FeatureMedia - 卡片插图：normal 静态图，hover 时播放视频
 * 鼠标移出后视频归零暂停，重新显示静态图
 */
import { useRef, useState } from "react";

interface FeatureMediaProps {
  staticSrc: string;
  videoSrc: string;
  alt?: string;
  className?: string;
}

export default function FeatureMedia({
  staticSrc,
  videoSrc,
  alt = "",
  className = "",
}: FeatureMediaProps) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [hovering, setHovering] = useState(false);

  const handleEnter = () => {
    setHovering(true);
    const v = videoRef.current;
    if (!v) return;
    try {
      v.currentTime = 0;
    } catch {
      /* noop */
    }
    const pr = v.play();
    if (pr && typeof pr.catch === "function") pr.catch(() => {});
  };

  const handleLeave = () => {
    setHovering(false);
    const v = videoRef.current;
    if (!v) return;
    v.pause();
    try {
      v.currentTime = 0;
    } catch {
      /* noop */
    }
  };

  return (
    <div
      className={`yh-fm ${className}`}
      onMouseEnter={handleEnter}
      onMouseLeave={handleLeave}
    >
      <img
        src={staticSrc}
        alt={alt}
        className="yh-fm-static"
        style={{ opacity: hovering ? 0 : 1 }}
      />
      <video
        ref={videoRef}
        className="yh-fm-video"
        muted
        loop
        playsInline
        preload="metadata"
        poster={staticSrc}
        style={{ opacity: hovering ? 1 : 0 }}
      >
        <source src={videoSrc} type="video/mp4" />
      </video>
    </div>
  );
}
