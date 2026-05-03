import { Plus, X } from "lucide-react";
import { Button } from "@/components/shared/Button";
import { Card } from "@/components/shared/Card";

interface BindingPanelProps {
  title: string;
  description: string;
  addLabel: string;
  items: unknown[];
  onAdd: () => void;
  onRemove: (index: number) => void;
  renderItem: (item: unknown, index: number) => React.ReactNode;
  emptyMessage?: string;
}

export function BindingPanel({
  title,
  description,
  addLabel,
  items,
  onAdd,
  onRemove,
  renderItem,
  emptyMessage,
}: BindingPanelProps) {
  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h4 className="text-sm font-semibold text-zinc-800">{title}</h4>
          <p className="text-xs text-zinc-500">{description}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={onAdd} type="button">
          <Plus className="mr-1 h-3.5 w-3.5" />
          {addLabel}
        </Button>
      </div>

      {items.length === 0 && (
        <Card className="p-4">
          <p className="text-center text-sm text-zinc-400">
            {emptyMessage ?? addLabel}
          </p>
        </Card>
      )}

      <div className="space-y-2">
        {items.map((item, index) => (
          <div
            key={index}
            className="flex items-center gap-2 rounded-md border border-zinc-200 bg-zinc-50/50 p-3"
          >
            <div className="flex-1">{renderItem(item, index)}</div>
            <button
              type="button"
              onClick={() => onRemove(index)}
              className="shrink-0 rounded p-1 text-zinc-400 hover:bg-rose-50 hover:text-rose-600"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

// Convenience components for common binding types

interface StringArrayBindingPanelProps {
  title: string;
  description: string;
  addLabel: string;
  items: string[];
  onChange: (items: string[]) => void;
  placeholder?: string;
}

export function StringArrayBindingPanel({
  title,
  description,
  addLabel,
  items,
  onChange,
  placeholder,
}: StringArrayBindingPanelProps) {
  return (
    <BindingPanel
      title={title}
      description={description}
      addLabel={addLabel}
      items={items}
      onAdd={() => onChange([...items, ""])}
      onRemove={(index) => onChange(items.filter((_, i) => i !== index))}
      renderItem={(item, index) => (
        <input
          className="block w-full rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-950 placeholder-zinc-400 outline-none transition focus:border-teal-600 focus:ring-2 focus:ring-teal-500/20"
          value={item as string}
          placeholder={placeholder}
          onChange={(e) => {
            const updated = [...items];
            updated[index] = e.target.value;
            onChange(updated);
          }}
        />
      )}
    />
  );
}
