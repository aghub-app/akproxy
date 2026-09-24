import { useReducedMotion } from "motion/react";

const easeOut = [0.23, 1, 0.32, 1] as const;

export function useUIMotion(duration: number) {
  const reduced = useReducedMotion();
  return {
    reduced,
    transition: { type: "tween" as const, duration: reduced ? 0.1 : duration, ease: easeOut },
  };
}
