interface LoadingSkeletonProps {
  /** Layout variant — defaults to "list" */
  variant?: "list" | "detail" | "table";
}

/** Generic list-page skeleton (header + card rows) */
function ListSkeleton() {
  return (
    <div className="space-y-5">
      {/* PageHeader skeleton */}
      <div className="space-y-2">
        <div className="skeleton-shimmer h-3 w-20 rounded bg-zinc-200/60" />
        <div className="skeleton-shimmer h-7 w-56 rounded bg-zinc-200/80" />
        <div className="skeleton-shimmer h-3 w-80 rounded bg-zinc-100" />
      </div>

      <div className="section-divider" />

      {/* Summary strip skeleton */}
      <div className="flex items-center gap-4">
        <div className="skeleton-shimmer h-5 w-24 rounded-full bg-zinc-200/60" />
        <div className="skeleton-shimmer h-5 w-20 rounded-full bg-zinc-200/40" />
        <div className="skeleton-shimmer h-5 w-16 rounded-full bg-zinc-200/40" />
      </div>

      {/* Table-like rows */}
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="data-card flex items-center gap-4 rounded-lg p-4"
          >
            <div className="skeleton-shimmer h-8 w-8 rounded-md bg-zinc-200/60" />
            <div className="flex-1 space-y-2">
              <div className="skeleton-shimmer h-4 w-1/4 rounded bg-zinc-200/80" />
              <div className="skeleton-shimmer h-3 w-1/3 rounded bg-zinc-100" />
            </div>
            <div className="skeleton-shimmer h-5 w-16 rounded-full bg-zinc-100" />
            <div className="skeleton-shimmer h-4 w-20 rounded bg-zinc-200/40" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** Detail-page skeleton (header + partitioned sections) */
function DetailSkeleton() {
  return (
    <div className="space-y-6">
      {/* PageHeader */}
      <div className="space-y-2">
        <div className="skeleton-shimmer h-3 w-16 rounded bg-zinc-200/60" />
        <div className="skeleton-shimmer h-8 w-64 rounded bg-zinc-200/80" />
        <div className="flex items-center gap-3 mt-2">
          <div className="skeleton-shimmer h-5 w-20 rounded-full bg-zinc-200/60" />
          <div className="skeleton-shimmer h-5 w-32 rounded bg-zinc-100" />
        </div>
      </div>

      <div className="section-divider" />

      {/* Detail sections */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="data-card rounded-lg p-5 space-y-4">
            <div className="skeleton-shimmer h-3 w-24 rounded bg-zinc-200/60" />
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, j) => (
                <div key={j} className="flex items-center justify-between">
                  <div className="skeleton-shimmer h-3 w-20 rounded bg-zinc-100" />
                  <div className="skeleton-shimmer h-3 w-32 rounded bg-zinc-200/60" />
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* Action bar skeleton */}
      <div className="flex gap-3 justify-end">
        <div className="skeleton-shimmer h-9 w-20 rounded-md bg-zinc-200/60" />
        <div className="skeleton-shimmer h-9 w-24 rounded-md bg-zinc-200/80" />
      </div>
    </div>
  );
}

/** Full table skeleton (header + data rows) */
function TableSkeleton() {
  return (
    <div className="space-y-5">
      <div className="space-y-2">
        <div className="skeleton-shimmer h-7 w-48 rounded bg-zinc-200/80" />
      </div>

      <div className="data-card rounded-lg overflow-hidden">
        {/* Table header */}
        <div className="flex items-center gap-4 border-b border-zinc-200/60 bg-zinc-50/80 px-5 py-3">
          <div className="skeleton-shimmer h-3 w-8 rounded bg-zinc-200/40" />
          <div className="skeleton-shimmer h-3 w-24 rounded bg-zinc-200/40" />
          <div className="flex-1" />
          <div className="skeleton-shimmer h-3 w-16 rounded bg-zinc-200/40" />
          <div className="skeleton-shimmer h-3 w-20 rounded bg-zinc-200/40" />
        </div>
        {/* Table rows */}
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-4 border-b border-zinc-100 px-5 py-4 last:border-b-0"
          >
            <div className="skeleton-shimmer h-7 w-7 rounded-md bg-zinc-200/60" />
            <div className="skeleton-shimmer h-4 w-32 rounded bg-zinc-200/80" />
            <div className="flex-1" />
            <div className="skeleton-shimmer h-5 w-16 rounded-full bg-zinc-100" />
            <div className="skeleton-shimmer h-3 w-24 rounded bg-zinc-200/40" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function LoadingSkeleton({ variant = "list" }: LoadingSkeletonProps) {
  return (
    <div className="animate-pulse" role="status" aria-label="Loading">
      {variant === "list" && <ListSkeleton />}
      {variant === "detail" && <DetailSkeleton />}
      {variant === "table" && <TableSkeleton />}
    </div>
  );
}
