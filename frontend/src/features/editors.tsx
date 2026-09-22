import type { FC } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ModelDraft, OpenAIDraft } from "@/lib/desktop";

export const StringList: FC<{
  values: string[];
  onChange: (values: string[]) => void;
  placeholder: string;
  secret?: boolean;
}> = ({ values, onChange, placeholder, secret = false }) => {
  const rows = values.length > 0 ? values : [""];
  return (
    <div className="flex w-full flex-col gap-2">
      {rows.map((value, index) => (
        <div className="flex gap-2" key={`${index}-${rows.length}`}>
          <Input
            type={secret ? "password" : "text"}
            value={value}
            placeholder={placeholder}
            autoComplete="off"
            onChange={(event) => {
              const next = [...rows];
              next[index] = event.target.value;
              onChange(next.filter((item, itemIndex) => item !== "" || itemIndex === index));
            }}
          />
          <Button
            variant="outline"
            onClick={() => onChange(rows.filter((_, itemIndex) => itemIndex !== index))}
          >
            移除
          </Button>
        </div>
      ))}
      <Button variant="outline" className="self-start" onClick={() => onChange([...rows, ""])}>
        添加
      </Button>
    </div>
  );
};

export const OpenAIDraftCard: FC<{
  draft: OpenAIDraft;
  onChange: (draft: OpenAIDraft) => void;
  onRemove: () => void;
}> = ({ draft, onChange, onRemove }) => {
  const models = draft.models.length > 0 ? draft.models : [{ name: "", alias: "" }];
  function updateModel(index: number, patch: Partial<ModelDraft>) {
    onChange({
      ...draft,
      models: models.map((model, itemIndex) =>
        itemIndex === index ? { ...model, ...patch } : model,
      ),
    });
  }
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-3">
      <Input
        value={draft.name}
        placeholder="名称"
        onChange={(event) => onChange({ ...draft, name: event.target.value })}
      />
      <Input
        value={draft.baseUrl}
        placeholder="Base URL"
        onChange={(event) => onChange({ ...draft, baseUrl: event.target.value })}
      />
      <StringList
        values={draft.apiKeys}
        placeholder="API key"
        secret
        onChange={(apiKeys) => onChange({ ...draft, apiKeys })}
      />
      <div className="flex flex-col gap-2">
        {models.map((model, index) => (
          <div className="flex gap-2" key={`${index}-${models.length}`}>
            <Input
              value={model.name}
              placeholder="上游模型名"
              onChange={(event) => updateModel(index, { name: event.target.value })}
            />
            <Input
              value={model.alias}
              placeholder="别名，可留空"
              onChange={(event) => updateModel(index, { alias: event.target.value })}
            />
          </div>
        ))}
        <Button
          variant="outline"
          className="self-start"
          onClick={() => onChange({ ...draft, models: [...models, { name: "", alias: "" }] })}
        >
          添加模型
        </Button>
      </div>
      <Button variant="outline" className="self-start" onClick={onRemove}>
        移除这个上游
      </Button>
    </div>
  );
};
