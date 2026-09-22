import { useQuery } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { Link } from "react-router";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Select, SelectItem, SelectPopup, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toastManager } from "@/components/ui/toast";
import { curlCommands } from "@/lib/curl";
import { errorText } from "@/lib/desktop";
import { providerPages } from "@/lib/providers";
import { homeModelsQueryOptions, homeQueryOptions } from "@/requests/home";

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

  const available = data.status.running && !models.isError ? (models.data ?? []) : [];
  const model = available.includes(selectedModel) ? selectedModel : (available[0] ?? "<MODEL_ID>");
  const commands = curlCommands(data.status.address, data.clientKey || "<API_KEY>", model);
  const displayed = curlCommands(data.status.address, showKey ? (data.clientKey || "<API_KEY>") : "••••••••", model);

  async function copy(command: string) {
    try {
      await navigator.clipboard.writeText(command);
      toastManager.add({ title: "已复制命令", type: "success" });
    } catch {
      toastManager.add({ title: "复制失败，请重试", type: "error" });
    }
  }

  return <section className="flex flex-col gap-5">
    <div><h1 className="text-xl font-semibold">开始调用</h1>
      <p className="mt-2 text-sm text-muted-foreground">复制命令，在终端调用本地代理。</p></div>
    {!data.status.running ? <p className="text-sm text-muted-foreground">服务尚未启动，请点击顶部「启动」后运行命令。</p> : null}
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
    {available.length === 0 ? <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground" role="status">
      <p>
        {models.isFetching && data.status.running ? "正在读取模型…" : models.isError && data.status.running ? errorText(models.error) : "暂无可选模型。"}
        {models.isFetching && data.status.running ? "" : " 请将命令中的 <MODEL_ID> 替换为模型 ID。"}
      </p>
      {data.status.running && !models.isFetching ? <Button variant="ghost" size="sm" onClick={() => void models.refetch()}>刷新模型</Button> : null}
    </div> : null}
    <p className="text-xs text-muted-foreground">复制内容包含第一把客户端密钥。模型是否支持对应 API 取决于上游。</p>
    {commands.map(({ title, command }, index) => <Card key={title} className="min-w-0 overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <h2 className="text-sm font-medium">{title}</h2>
        <Button variant="outline" size="sm" aria-label={`复制${title}命令`} onClick={() => void copy(command)}>复制</Button>
      </div>
      <pre className="overflow-x-auto p-4 text-xs leading-relaxed"><code>{displayed[index].command}</code></pre>
    </Card>)}
  </section>;
};
