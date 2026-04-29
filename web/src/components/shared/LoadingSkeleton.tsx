export function LoadingSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      <div className="h-8 w-48 rounded bg-zinc-200" />
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-16 rounded-lg border border-zinc-200 bg-white/80 p-4">
            <div className="h-4 w-1/3 rounded bg-zinc-200" />
            <div className="mt-2 h-3 w-1/4 rounded bg-zinc-100" />
          </div>
        ))}
      </div>
    </div>
  );
}
