import { type ReactNode } from "react";
import { motion, useIsPresent } from "motion/react";
import { useUIMotion } from "@/lib/ui-motion";

export function AnimatedField({ children }: { children: (present: boolean) => ReactNode }) {
  const present = useIsPresent();
  const { reduced, transition } = useUIMotion(0.16);
  const hidden = { opacity: 0, transform: reduced ? "none" : "translateY(-4px)" };

  return (
    <motion.div
      initial={hidden}
      animate={{ opacity: 1, transform: "none" }}
      exit={hidden}
      transition={transition}
      inert={!present}
      aria-hidden={!present}
    >
      {children(present)}
    </motion.div>
  );
}
