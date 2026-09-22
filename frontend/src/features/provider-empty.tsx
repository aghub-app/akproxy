import { type FC, type ReactNode } from "react";
import { Empty, EmptyContent, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { providerPages } from "@/lib/providers";

export const ProviderEmpty: FC<{ page: string; children: ReactNode }> = ({ page, children }) => {
  const meta = providerPages.find((item) => item.to === `/${page}`);
  const Icon = meta?.icon;
  return (
    <Empty>
      <EmptyHeader>
        <EmptyMedia>{Icon ? <Icon size={40} /> : null}</EmptyMedia>
        <EmptyTitle>{meta?.label ?? page}</EmptyTitle>
      </EmptyHeader>
      <EmptyContent>{children}</EmptyContent>
    </Empty>
  );
};
