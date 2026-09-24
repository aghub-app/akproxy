import { PlayIcon } from "@phosphor-icons/react";
import { useQuery } from "@tanstack/react-query";
import { Clipboard } from "@wailsio/runtime";
import { motion } from "motion/react";
import { type FC, useRef, useState } from "react";
import { Link } from "react-router";
import { maskClientKey } from "@/components/client-key-display";
import { Button } from "@/components/ui/button";
import { Card, CardFrame, CardFrameAction, CardFrameHeader, CardFrameTitle, CardPanel } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Select, SelectItem, SelectPopup, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { toastManager } from "@/components/ui/toast";
import { curlCommands, openaiEnv } from "@/lib/curl";
import { highlightCommand, highlightSnippet } from "@/lib/highlight-command";
import { isSdkId, sdkExample, sdkOptions, type SdkId } from "@/lib/sdk-examples";
import { errorText } from "@/lib/desktop";
import { providerPages } from "@/lib/providers";
import { currentLocale, translate } from "@/lib/i18n";
import { usePresentation } from "@/presentation";
import { homeModelsQueryOptions, homeQueryOptions } from "@/requests/home";
import { useUIMotion } from "@/lib/ui-motion";

async function copyText(value: string, title: string) {
  try {
    await Clipboard.SetText(value);
    toastManager.add({ title, type: "success" });
  } catch {
    toastManager.add({ title: translate("复制失败，请重试", currentLocale()), type: "error" });
  }
}

const DoodleArrow: FC = () => (
  <svg viewBox="0 0 220 170" className="pointer-events-none absolute top-1 left-full ms-1 hidden h-40 w-48 -translate-x-[100px] -translate-y-[150px] text-foreground/80 sm:block" aria-hidden="true">
    <path d="M20 146C48 149 66 128 82 109C96 92 107 88 119 93C135 100 132 119 117 121C101 123 96 105 105 91C116 73 141 76 150 92C160 110 115 135 131 122C160 99 158 56 198 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M178 28C186 26 193 25 198 24C192 31 188 39 186 47" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" transform="rotate(-15 198 24)" />
    <path d="M167 17l2 5 5 2-5 2-2 5-2-5-5-2 5-2z" fill="currentColor" />
    <circle cx="20" cy="146" r="2.5" fill="currentColor" />
  </svg>
);

type HomeEntrance = {
  enter: false | { opacity: number; transform: string };
  transition: ReturnType<typeof useUIMotion>["transition"];
};

const HomeStopped: FC<HomeEntrance> = ({ enter, transition }) => {
  const { t } = usePresentation();
  return (
    <motion.div className="flex flex-1 flex-col" initial={enter} animate={{ opacity: 1, transform: "none" }} transition={transition}>
      <Empty className="-translate-y-8">
        <EmptyHeader className="relative">
          <EmptyMedia><PlayIcon size={40} weight="duotone" /></EmptyMedia>
          <EmptyTitle>{t("启动服务")}</EmptyTitle>
          <EmptyDescription>
            {t("服务这会儿还停着。去窗口右上角点一下，它就在这台机器上跑起来。")}
          </EmptyDescription>
          <DoodleArrow />
        </EmptyHeader>
      </Empty>
    </motion.div>
  );
};

const ConnectPanel: FC<{ address: string; clientKeys: string[]; clientKey: string; onSelectKey: (key: string) => void }> = ({
  address,
  clientKeys,
  clientKey,
  onSelectKey,
}) => {
  const { t } = usePresentation();
  const [revealedKey, setRevealedKey] = useState<string | null>(null);
  const [sdk, setSdk] = useState<SdkId>("openai-python");
  const showKey = clientKey !== "" && revealedKey === clientKey;
  const visibleKey = showKey && clientKey ? clientKey : "••••••••";
  const env = openaiEnv(address, clientKey || "<API_KEY>");
  const visibleEnv = openaiEnv(address, visibleKey);
  const sdkOption = sdkOptions.find((item) => item.id === sdk) ?? sdkOptions[0];
  const example = sdkExample(sdkOption.id, address, clientKey || "<API_KEY>");
  const visibleExample = sdkExample(sdkOption.id, address, visibleKey);
  return <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
    <CardFrame className="min-w-0">
      <CardFrameHeader className="px-3 py-2">
        <CardFrameTitle render={<h2 />}>{t("地址")}</CardFrameTitle>
        <CardFrameAction>
          <Button variant="outline" size="sm" disabled={!address} onClick={() => void copyText(address, t("已复制地址"))}>{t("复制")}</Button>
        </CardFrameAction>
      </CardFrameHeader>
      <Card className="min-w-0 flex-1 overflow-hidden">
        <CardPanel className="flex min-w-0 items-center p-3 font-mono text-sm"><p className="truncate">{address || t("还没有地址")}</p></CardPanel>
      </Card>
    </CardFrame>
    <CardFrame className="min-w-0">
      <CardFrameHeader className="px-3 py-2">
        <CardFrameTitle render={<h2 />}>{t("密钥")}</CardFrameTitle>
        <CardFrameAction className="gap-2">
          {clientKeys.length > 1 ? <Select value={String(clientKeys.indexOf(clientKey))} onValueChange={(index) => {
            if (typeof index !== "string") return;
            const key = clientKeys[Number(index)];
            if (key) { setRevealedKey(null); onSelectKey(key); }
          }}>
            <SelectTrigger size="sm" className="w-52" aria-label={t("选择客户端密钥")}><SelectValue>{maskClientKey(clientKey)}</SelectValue></SelectTrigger>
            <SelectPopup>{clientKeys.map((key, index) => <SelectItem key={key} value={String(index)}>{maskClientKey(key)}</SelectItem>)}</SelectPopup>
          </Select> : null}
          <Button type="button" variant="outline" size="sm" aria-pressed={showKey} disabled={!clientKey} onClick={() => setRevealedKey(showKey ? null : clientKey)}>{showKey ? t("隐藏密钥") : t("显示密钥")}</Button>
          <Button variant="outline" size="sm" disabled={!clientKey} onClick={() => void copyText(clientKey, t("已复制 API key"))}>{t("复制")}</Button>
        </CardFrameAction>
      </CardFrameHeader>
      <Card className="min-w-0 flex-1 overflow-hidden">
        <CardPanel className="flex min-w-0 items-center p-3 font-mono text-sm">
          <span className="min-w-0 truncate">{visibleKey}</span>
        </CardPanel>
      </Card>
    </CardFrame>
    <CardFrame className="min-w-0 md:col-span-2">
      <CardFrameHeader className="px-3 py-2">
        <CardFrameTitle render={<h2 />}>{t("环境变量")}</CardFrameTitle>
        <CardFrameAction>
          <Button variant="outline" size="sm" disabled={!address || !clientKey} onClick={() => void copyText(env, t("已复制环境变量"))}>{t("复制")}</Button>
        </CardFrameAction>
      </CardFrameHeader>
      <Card className="min-w-0 overflow-hidden">
        <CardPanel className="min-w-0 p-3"><pre className="sh-code overflow-x-auto text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightCommand(visibleEnv) }} /></pre></CardPanel>
      </Card>
    </CardFrame>
    <CardFrame className="min-w-0 md:col-span-2">
      <CardFrameHeader className="px-3 py-2">
        <div className="flex min-w-0 items-center gap-3">
          <CardFrameTitle render={<h2 />}>SDK</CardFrameTitle>
          <Select value={sdk} onValueChange={(value) => { if (isSdkId(value)) setSdk(value); }}>
            <SelectTrigger size="sm" className="w-44" aria-label="SDK">
              <SelectValue>{sdkOption.label}</SelectValue>
            </SelectTrigger>
            <SelectPopup>
              {sdkOptions.map((item) => <SelectItem key={item.id} value={item.id}>{item.label}</SelectItem>)}
            </SelectPopup>
          </Select>
        </div>
        <CardFrameAction>
          <Button variant="outline" size="sm" disabled={!address || !clientKey} onClick={() => void copyText(example, t("已复制示例"))}>{t("复制")}</Button>
        </CardFrameAction>
      </CardFrameHeader>
      <Card className="min-w-0 overflow-hidden">
        <CardPanel className="min-w-0 p-3"><pre className="sh-code overflow-x-auto text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightSnippet(visibleExample, sdkOption.lang) }} /></pre></CardPanel>
      </Card>
    </CardFrame>
  </div>;
};

export const HomePage: FC = () => {
  const { t, locale } = usePresentation();
  const home = useQuery(homeQueryOptions());
  const [selectedModel, setSelectedModel] = useState("");
  const [selectedKey, setSelectedKey] = useState("");
  const [revealedTestKey, setRevealedTestKey] = useState<string | null>(null);
  const data = home.data;
  const state = data ? !data.hasCredentials ? "empty" : data.status.running ? "running" : "stopped" : null;
  const lastState = useRef<string | null>(null);
  const enteredState = useRef<string | null>(null);
  if (state !== lastState.current) {
    enteredState.current = lastState.current === null ? null : state;
    lastState.current = state;
  }
  const animateState = enteredState.current === state;
  const { reduced, transition } = useUIMotion(0.18);
  const enter = animateState ? { opacity: 0, transform: reduced ? "none" : "translateY(6px)" } : false;
  const models = useQuery(homeModelsQueryOptions(
    !home.isError && data?.hasCredentials && data.status.running ? data.status.address : null,
  ));

  if (home.isError) {
    return <div role="alert" className="flex flex-col items-start gap-3">
      <p>{errorText(home.error, locale)}</p>
      <Button variant="outline" onClick={() => void home.refetch()}>{t("重试")}</Button>
    </div>;
  }
  if (!data) return <p className="text-sm text-muted-foreground">{t("正在读取首页…")}</p>;
  if (!data.hasCredentials) {
    return <motion.section className="flex flex-col gap-6" initial={enter} animate={{ opacity: 1, transform: "none" }} transition={transition}>
      <div><h1 className="text-xl font-semibold">{t("添加你的第一个上游")}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t("选择一个提供商，登录账号或配置 API key。")}</p></div>
      <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
        {providerPages.map(({ to, label, icon: Icon }) => (
          <Card key={to} render={<Link to={to} />} className="items-center gap-3 p-4 transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring">
            <Icon size={32} /><span className="text-sm font-medium">{label}</span>
          </Card>
        ))}
      </div>
    </motion.section>;
  }

  if (!data.status.running) return <HomeStopped enter={enter} transition={transition} />;

  const clientKeys = data.clientKeys ?? [];
  const clientKey = clientKeys.includes(selectedKey) ? selectedKey : (clientKeys[0] ?? "");
  const showKey = clientKey !== "" && revealedTestKey === clientKey;
  const available = !models.isError ? (models.data ?? []) : [];
  const model = available.includes(selectedModel) ? selectedModel : (available[0] ?? "<MODEL_ID>");
  const commands = curlCommands(data.status.address, clientKey || "<API_KEY>", model);
  const displayed = curlCommands(data.status.address, showKey ? (clientKey || "<API_KEY>") : "••••••••", model);

  return <motion.section className="flex flex-col gap-5" initial={enter} animate={{ opacity: 1, transform: "none" }} transition={transition}>
    <h1 className="text-xl font-semibold">{t("开始调用")}</h1>
    <Tabs className="gap-5" defaultValue="connect">
      <TabsList>
        <TabsTab value="connect">{t("接入")}</TabsTab>
        <TabsTab value="test">{t("测试")}</TabsTab>
      </TabsList>
      <TabsPanel value="connect">
        <ConnectPanel address={data.status.address} clientKeys={clientKeys} clientKey={clientKey} onSelectKey={(key) => { setSelectedKey(key); setRevealedTestKey(null); }} />
      </TabsPanel>
      <TabsPanel value="test" className="flex flex-col gap-4">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <label id="home-model-label" className="text-sm font-medium">{t("模型")}</label>
            <Select value={model} disabled={available.length === 0} onValueChange={(value) => { if (typeof value === "string") setSelectedModel(value); }}>
              <SelectTrigger aria-labelledby="home-model-label"><SelectValue>{model}</SelectValue></SelectTrigger>
              <SelectPopup>{available.map((id) => <SelectItem key={id} value={id}>{id}</SelectItem>)}</SelectPopup>
            </Select>
          </div>
          <Button variant="outline" aria-pressed={showKey} onClick={() => setRevealedTestKey(showKey ? null : clientKey)}>{showKey ? t("隐藏密钥") : t("显示密钥")}</Button>
        </div>
        {available.length === 0 ? <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground" role="status">
          <p>
            {models.isFetching ? t("正在读取模型…") : models.isError ? errorText(models.error, locale) : t("暂无可选模型。")}
            {models.isFetching ? "" : t(" 请将命令中的 <MODEL_ID> 替换为模型 ID。")}
          </p>
          {!models.isFetching ? <Button variant="ghost" size="sm" onClick={() => void models.refetch()}>{t("刷新模型")}</Button> : null}
        </div> : null}
        <p className="text-xs text-muted-foreground">{t("复制内容包含所选客户端密钥。模型是否支持对应 API 取决于上游。")}</p>
        {commands.map(({ title, command }, index) => <CardFrame key={title} className="min-w-0">
          <CardFrameHeader className="px-3 py-2">
            <CardFrameTitle render={<h2 />}>{t(title)}</CardFrameTitle>
            <CardFrameAction>
              <Button variant="outline" size="sm" aria-label={t("复制{title}命令", { title: t(title) })} onClick={() => void copyText(command, t("已复制命令"))}>{t("复制")}</Button>
            </CardFrameAction>
          </CardFrameHeader>
          <Card className="min-w-0 overflow-hidden">
            <CardPanel className="min-w-0 p-3"><pre className="sh-code overflow-x-auto text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightCommand(displayed[index].command) }} /></pre></CardPanel>
          </Card>
        </CardFrame>)}
      </TabsPanel>
    </Tabs>
  </motion.section>;
};
