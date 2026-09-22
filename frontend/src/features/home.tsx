import { useQuery } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { Link } from "react-router";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Select, SelectItem, SelectPopup, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { toastManager } from "@/components/ui/toast";
import { curlCommands, openaiEnv } from "@/lib/curl";
import { highlightCommand, highlightSnippet } from "@/lib/highlight-command";
import { isSdkId, sdkExample, sdkOptions, type SdkId } from "@/lib/sdk-examples";
import { errorText } from "@/lib/desktop";
import { providerPages } from "@/lib/providers";
import { homeModelsQueryOptions, homeQueryOptions } from "@/requests/home";

async function copyText(value: string, title: string) {
  try {
    await navigator.clipboard.writeText(value);
    toastManager.add({ title, type: "success" });
  } catch {
    toastManager.add({ title: "复制失败，请重试", type: "error" });
  }
}

const ConnectPanel: FC<{ address: string; clientKey: string; running: boolean }> = ({
  address,
  clientKey,
  running,
}) => {
  const [showKey, setShowKey] = useState(false);
  const [sdk, setSdk] = useState<SdkId>("openai-python");
  if (!running) {
    return <p className="text-sm text-muted-foreground">服务尚未启动，请点击顶部「启动」后查看接入信息。</p>;
  }
  const visibleKey = showKey && clientKey ? clientKey : "••••••••";
  const env = openaiEnv(address, clientKey || "<API_KEY>");
  const visibleEnv = openaiEnv(address, visibleKey);
  const sdkOption = sdkOptions.find((item) => item.id === sdk) ?? sdkOptions[0];
  const example = sdkExample(sdkOption.id, address, clientKey || "<API_KEY>");
  const visibleExample = sdkExample(sdkOption.id, address, visibleKey);
  return <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
    <Card className="min-w-0 overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <h2 className="text-sm font-medium">地址</h2>
        <Button variant="outline" size="sm" disabled={!address} onClick={() => void copyText(address, "已复制地址")}>复制</Button>
      </div>
      <p className="truncate px-4 py-4 font-mono text-sm">{address || "还没有地址"}</p>
    </Card>
    <Card className="min-w-0 overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <h2 className="text-sm font-medium">客户端密钥</h2>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" aria-pressed={showKey} disabled={!clientKey} onClick={() => setShowKey(!showKey)}>{showKey ? "隐藏" : "显示"}</Button>
          <Button variant="outline" size="sm" disabled={!clientKey} onClick={() => void copyText(clientKey, "已复制 API key")}>复制</Button>
        </div>
      </div>
      <p className="truncate px-4 py-4 font-mono text-sm">{visibleKey}</p>
    </Card>
    <Card className="min-w-0 overflow-hidden md:col-span-2">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <h2 className="text-sm font-medium">环境变量</h2>
        <Button variant="outline" size="sm" disabled={!address || !clientKey} onClick={() => void copyText(env, "已复制环境变量")}>复制</Button>
      </div>
      <pre className="sh-code overflow-x-auto p-4 text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightCommand(visibleEnv) }} /></pre>
    </Card>
    <Card className="min-w-0 overflow-hidden md:col-span-2">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <h2 className="text-sm font-medium">SDK</h2>
          <Select value={sdk} onValueChange={(value) => { if (isSdkId(value)) setSdk(value); }}>
            <SelectTrigger size="sm" className="w-44" aria-label="SDK">
              <SelectValue>{sdkOption.label}</SelectValue>
            </SelectTrigger>
            <SelectPopup>
              {sdkOptions.map((item) => <SelectItem key={item.id} value={item.id}>{item.label}</SelectItem>)}
            </SelectPopup>
          </Select>
        </div>
        <Button variant="outline" size="sm" disabled={!address || !clientKey} onClick={() => void copyText(example, "已复制示例")}>复制</Button>
      </div>
      <pre className="sh-code overflow-x-auto p-4 text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightSnippet(visibleExample, sdkOption.lang) }} /></pre>
    </Card>
  </div>;
};

export const HomePage: FC = () => {
  const home = useQuery(homeQueryOptions());
  const [selectedModel, setSelectedModel] = useState("");
  const [showKey, setShowKey] = useState(false);
  const data = home.data;
  const models = useQuery(homeModelsQueryOptions(
    !home.isError && data?.hasCredentials && data.status.running ? data.status.address : null,
  ));

  if (home.isError) {
    return <div role="alert" className="flex flex-col items-start gap-3">
      <p>{errorText(home.error)}</p>
      <Button variant="outline" onClick={() => void home.refetch()}>重试</Button>
    </div>;
  }
  if (!data) return <p className="text-sm text-muted-foreground">正在读取首页…</p>;
  if (!data.hasCredentials) {
    return <section className="flex flex-col gap-6">
      <div><h1 className="text-xl font-semibold">添加你的第一个上游</h1>
        <p className="mt-2 text-sm text-muted-foreground">选择一个提供商，登录账号或配置 API key。</p></div>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
        {providerPages.map(({ to, label, icon: Icon }) => (
          <Card key={to} render={<Link to={to} />} className="items-center gap-4 p-6 transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring">
            <Icon size={36} /><span className="text-sm font-medium">{label}</span>
          </Card>
        ))}
      </div>
    </section>;
  }

  const running = data.status.running;
  const available = running && !models.isError ? (models.data ?? []) : [];
  const model = available.includes(selectedModel) ? selectedModel : (available[0] ?? "<MODEL_ID>");
  const commands = curlCommands(data.status.address, data.clientKey || "<API_KEY>", model);
  const displayed = curlCommands(data.status.address, showKey ? (data.clientKey || "<API_KEY>") : "••••••••", model);

  return <section className="flex flex-col gap-5">
    <h1 className="text-xl font-semibold">开始调用</h1>
    <Tabs className="gap-5" defaultValue="connect">
      <TabsList>
        <TabsTab value="connect">接入</TabsTab>
        <TabsTab value="test">测试</TabsTab>
      </TabsList>
      <TabsPanel value="connect">
        <ConnectPanel address={data.status.address} clientKey={data.clientKey} running={running} />
      </TabsPanel>
      <TabsPanel value="test" className="flex flex-col gap-5">
        {!running ? <p className="text-sm text-muted-foreground">服务尚未启动，请点击顶部「启动」后复制命令。</p> : null}
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <label id="home-model-label" className="text-sm font-medium">模型</label>
            <Select value={model} disabled={available.length === 0} onValueChange={(value) => { if (typeof value === "string") setSelectedModel(value); }}>
              <SelectTrigger aria-labelledby="home-model-label"><SelectValue>{model}</SelectValue></SelectTrigger>
              <SelectPopup>{available.map((id) => <SelectItem key={id} value={id}>{id}</SelectItem>)}</SelectPopup>
            </Select>
          </div>
          <Button variant="outline" aria-pressed={showKey} onClick={() => setShowKey(!showKey)}>{showKey ? "隐藏密钥" : "显示密钥"}</Button>
        </div>
        {running && available.length === 0 ? <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground" role="status">
          <p>
            {models.isFetching ? "正在读取模型…" : models.isError ? errorText(models.error) : "暂无可选模型。"}
            {models.isFetching ? "" : " 请将命令中的 <MODEL_ID> 替换为模型 ID。"}
          </p>
          {!models.isFetching ? <Button variant="ghost" size="sm" onClick={() => void models.refetch()}>刷新模型</Button> : null}
        </div> : null}
        {running ? <p className="text-xs text-muted-foreground">复制内容包含第一把客户端密钥。模型是否支持对应 API 取决于上游。</p> : null}
        {commands.map(({ title, command }, index) => <Card key={title} className="min-w-0 overflow-hidden">
          <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
            <h2 className="text-sm font-medium">{title}</h2>
            <Button
              variant="outline"
              size="sm"
              aria-label={`复制${title}命令`}
              disabled={!running}
              onClick={() => { if (running) void copyText(command, "已复制命令"); }}
            >复制</Button>
          </div>
          <pre className="sh-code overflow-x-auto p-4 text-xs leading-relaxed"><code dangerouslySetInnerHTML={{ __html: highlightCommand(displayed[index].command) }} /></pre>
        </Card>)}
      </TabsPanel>
    </Tabs>
  </section>;
};
