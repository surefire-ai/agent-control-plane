export function LoadingSkeleton() {
  return (
    <div className="space-y-5 animate-pulse" role="status" aria-label="Loading">
      {/* Header skeleton */}
      <div className="space-y-2">
        <div className="h-3 w-20 rounded bg-zinc-200/80" />
        <div className="h-7 w-56 rounded bg-zinc-200" />
        <div className="h-3 w-80 rounded bg-zinc-100" />
      </div>

      {/* Divider */}
      <div className="section-divider" />

      {/* Table-like skeleton */}
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="data-card flex items-center gap-4 rounded-lg p-4"
          >
            <div className="h-8 w-8 rounded-md bg-zinc-200/80" />
            <div className="flex-1 space-y-2">
              <div className="h-4 w-1/4 rounded bg-zinc-200" />
              <div className="h-3 w-1/3 rounded bg-zinc-100" />
            </div>
            <div className="h-5 w-16 rounded-full bg-zinc-100" />
          </div>
        ))}
      </div>
    </div>
  );
}
