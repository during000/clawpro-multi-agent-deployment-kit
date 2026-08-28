/**
 * useBrandLogo - 让组件实时跟随全局 Logo 变化
 */
import { useEffect, useState } from "react";
import { getBrandLogo, subscribeBrandLogo } from "./brandLogo";

export function useBrandLogo(): string {
  const [logo, setLogo] = useState<string>(() => getBrandLogo());
  useEffect(() => subscribeBrandLogo(setLogo), []);
  return logo;
}
