/**
 * RadarWidget — Agent Design System (Light theme)
 * Adapted from Deep Space Tech radar with idle/hover progressive reveal
 * Colors: Brand Blue #007AFF / Purple #5856D6 for TDAI, Gray for Agent
 */

import { motion, AnimatePresence } from 'framer-motion';
import { useMemo } from 'react';

const BENCHMARK_DATA = [
  { label: '记住变化原因', labelShort: '记住变化原因', openClaw: 70.97, tdaiMemory: 88.89 },
  { label: '记住你说过的事', labelShort: '记住你说过的事', openClaw: 29.63, tdaiMemory: 79.07 },
  { label: '记住关键信息', labelShort: '记住关键信息', openClaw: 25.00, tdaiMemory: 76.47 },
  { label: '个性化推荐', labelShort: '个性化推荐', openClaw: 46.67, tdaiMemory: 76.36 },
  { label: '跨场景理解', labelShort: '跨场景理解', openClaw: 31.58, tdaiMemory: 78.95 },
  { label: '跟踪偏好变化', labelShort: '跟踪偏好变化', openClaw: 66.67, tdaiMemory: 83.45 },
  { label: '创意启发', labelShort: '创意启发', openClaw: 24.00, tdaiMemory: 45.16 },
];

const TOTAL = { openClaw: 47.85, tdaiMemory: 76.10 };

const CX = 195;
const CY = 160;
const R = 90;
const AXES = BENCHMARK_DATA.length;

function polar(angle: number, radius: number) {
  const rad = ((angle - 90) * Math.PI) / 180;
  return { x: CX + radius * Math.cos(rad), y: CY + radius * Math.sin(rad) };
}

function polyPoints(values: number[]): string {
  return values
    .map((v, i) => {
      const { x, y } = polar((360 / AXES) * i, (v / 100) * R);
      return `${x},${y}`;
    })
    .join(' ');
}

interface RadarWidgetProps {
  hovered: boolean;
  onHoverChange: (h: boolean) => void;
}

export function RadarWidget({ hovered, onHoverChange }: RadarWidgetProps) {
  const ocPoints = useMemo(() => polyPoints(BENCHMARK_DATA.map((d) => d.openClaw)), []);
  const tdPoints = useMemo(() => polyPoints(BENCHMARK_DATA.map((d) => d.tdaiMemory)), []);

  const rings = useMemo(
    () =>
      [1, 2, 3, 4, 5].map((n) => {
        const r = (n / 5) * R;
        const pts = Array.from({ length: AXES }, (_, j) => {
          const { x, y } = polar((360 / AXES) * j, r);
          return `${x},${y}`;
        }).join(' ');
        return <polygon key={n} points={pts} fill="none" stroke="rgba(0,0,0,0.06)" strokeWidth={0.8} />;
      }),
    []
  );

  const axisLines = useMemo(
    () =>
      BENCHMARK_DATA.map((_, i) => {
        const { x, y } = polar((360 / AXES) * i, R);
        return <line key={i} x1={CX} y1={CY} x2={x} y2={y} stroke="rgba(0,0,0,0.06)" strokeWidth={0.8} />;
      }),
    []
  );

  const axisLabels = useMemo(
    () =>
      BENCHMARK_DATA.map((d, i) => {
        const angle = (360 / AXES) * i;
        const { x, y } = polar(angle, R + 35);
        const norm = ((angle % 360) + 360) % 360;
        let anchor: 'start' | 'middle' | 'end' = 'middle';
        if (norm > 20 && norm < 160) anchor = 'start';
        else if (norm > 200 && norm < 340) anchor = 'end';
        return (
          <text
            key={i}
            x={x}
            y={y}
            textAnchor={anchor}
            dominantBaseline="middle"
            fill={hovered ? '#374151' : '#9ca3af'}
            fontSize={11}
            fontFamily="Inter, sans-serif"
            fontWeight={hovered ? 500 : 400}
            style={{ transition: 'fill 0.3s, font-weight 0.3s' }}
          >
            {d.labelShort}
          </text>
        );
      }),
    [hovered]
  );

  return (
    <motion.div
      onMouseEnter={() => onHoverChange(true)}
      onMouseLeave={() => onHoverChange(false)}
      className="relative cursor-pointer select-none w-full"
      style={{ maxWidth: 420, aspectRatio: '390 / 320' }}
    >
      <svg viewBox="0 0 390 320" className="w-full h-full">
        <defs>
          <radialGradient id="rwGlow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor={hovered ? 'rgba(0,122,255,0.06)' : 'rgba(0,122,255,0.02)'} />
            <stop offset="100%" stopColor="rgba(0,0,0,0)" />
          </radialGradient>
          <linearGradient id="brandGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#007AFF" />
            <stop offset="100%" stopColor="#5856D6" />
          </linearGradient>
          <filter id="blueGlow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {/* Background glow */}
        <circle cx={CX} cy={CY} r={R + 10} fill="url(#rwGlow)" />

        {/* Breathing pulse ring (idle tease) */}
        {!hovered && (
          <motion.circle
            cx={CX}
            cy={CY}
            r={R + 6}
            fill="none"
            stroke="rgba(0,122,255,0.1)"
            strokeWidth={1}
            animate={{
              r: [R + 4, R + 12, R + 4],
              opacity: [0.2, 0.5, 0.2],
            }}
            transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
          />
        )}

        {rings}
        {axisLines}

        {/* Percentage labels on rings — removed per design request */}

        {/* Agent area — always visible */}
        <motion.polygon
          points={ocPoints}
          fill="rgba(156,163,175,0.12)"
          stroke="#9ca3af"
          strokeWidth={1.5}
          strokeLinejoin="round"
          animate={{ opacity: hovered ? 0.5 : 1 }}
          transition={{ duration: 0.4 }}
        />

        {/* Ghost TDAI outline — idle tease (flickers) */}
        {!hovered && (
          <motion.polygon
            points={tdPoints}
            fill="none"
            stroke="rgba(0,122,255,0.15)"
            strokeWidth={1}
            strokeLinejoin="round"
            strokeDasharray="4 6"
            animate={{ opacity: [0, 0.3, 0], strokeDashoffset: [0, -20] }}
            transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
          />
        )}

        {/* TDAI area — full on hover */}
        <motion.polygon
          points={tdPoints}
          fill="rgba(0,122,255,0.08)"
          stroke="url(#brandGrad)"
          strokeWidth={2}
          strokeLinejoin="round"
          filter="url(#blueGlow)"
          initial={{ opacity: 0, scale: 0 }}
          animate={hovered ? { opacity: 1, scale: 1 } : { opacity: 0, scale: 0 }}
          transition={{ duration: 0.5, type: 'spring', stiffness: 200 }}
          style={{ transformOrigin: `${CX}px ${CY}px` }}
        />

        {/* Data dots */}
        {BENCHMARK_DATA.map((d, i) => {
          const angle = (360 / AXES) * i;
          const oc = polar(angle, (d.openClaw / 100) * R);
          const td = polar(angle, (d.tdaiMemory / 100) * R);
          return (
            <g key={i}>
              <circle cx={oc.x} cy={oc.y} r={2.5} fill="#9ca3af" opacity={0.9} />
              {hovered && (
                <motion.circle
                  cx={td.x}
                  cy={td.y}
                  r={3}
                  fill="#007AFF"
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ delay: i * 0.04, type: 'spring' }}
                  style={{ filter: 'drop-shadow(0 0 4px rgba(0,122,255,0.5))' }}
                />
              )}
            </g>
          );
        })}

        {axisLabels}
      </svg>

      {/* Center label */}
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        <AnimatePresence mode="wait">
          {hovered ? (
            <motion.div
              key="after"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              transition={{ duration: 0.25 }}
              className="text-center"
            >
              <div className="text-[10px] text-blue-500 font-medium mb-0.5">Memory Pro 版</div>
              <div
                className="text-lg font-bold font-mono"
                style={{ color: '#007AFF', textShadow: '0 0 12px rgba(0,122,255,0.25)' }}
              >
                {TOTAL.tdaiMemory}%
              </div>
            </motion.div>
          ) : (
            <motion.div
              key="before"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              transition={{ duration: 0.25 }}
              className="text-center"
            >
              <div className="text-[10px] text-gray-400 mb-0.5">Agent 原生</div>
              <div className="text-lg font-bold font-mono text-gray-400">
                {TOTAL.openClaw}%
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Idle hover hint */}
      {!hovered && (
        <motion.div
          className="absolute -bottom-2 left-1/2 -translate-x-1/2 pointer-events-none"
          animate={{ opacity: [0.3, 0.7, 0.3], y: [0, -3, 0] }}
          transition={{ duration: 2, repeat: Infinity, ease: 'easeInOut' }}
        >
          <svg width="20" height="10" viewBox="0 0 20 10" fill="none">
            <path d="M4 8L10 3L16 8" stroke="#007AFF" strokeWidth="1.5" strokeLinecap="round" opacity="0.5" />
          </svg>
        </motion.div>
      )}
    </motion.div>
  );
}
