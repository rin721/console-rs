import { Skeleton, SkeletonText } from "~/components/aoi/primitives/Skeleton";
import { cn } from "~/lib/cn";

type StatGridSkeletonProps = {
  count?: number;
};

export function StatGridSkeleton({ count = 5 }: StatGridSkeletonProps) {
  return (
    <div className="aoi-admin-stat-grid" aria-hidden="true">
      {Array.from({ length: count }).map((_, index) => (
        <article className="aoi-admin-stat-card aoi-admin-stat-card--loading" key={index}>
          <Skeleton className="aoi-skeleton--icon" />
          <div>
            <Skeleton className="aoi-skeleton--label" />
            <Skeleton className="aoi-skeleton--value" />
          </div>
        </article>
      ))}
    </div>
  );
}

type TableSkeletonProps = {
  caption?: string;
  columns?: number;
  rows?: number;
};

export function TableSkeleton({ caption, columns = 4, rows = 6 }: TableSkeletonProps) {
  return (
    <div className="aoi-data-table-wrap" aria-label={caption}>
      <table className="aoi-data-table aoi-data-table--loading" aria-busy="true">
        {caption ? <caption>{caption}</caption> : null}
        <thead>
          <tr>
            {Array.from({ length: columns }).map((_, index) => (
              <th key={index} scope="col">
                <Skeleton className="aoi-skeleton--table-heading" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }).map((_, rowIndex) => (
            <tr key={rowIndex}>
              {Array.from({ length: columns }).map((_, columnIndex) => (
                <td key={columnIndex}>
                  <Skeleton
                    className={cn(
                      "aoi-skeleton--table-cell",
                      columnIndex === 0 && "aoi-skeleton--table-cell-wide",
                    )}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

type PanelSkeletonProps = {
  rows?: number;
};

export function PanelSkeleton({ rows = 4 }: PanelSkeletonProps) {
  return (
    <div className="aoi-panel-skeleton" aria-hidden="true">
      <Skeleton className="aoi-skeleton--panel-title" />
      <SkeletonText lines={rows} />
    </div>
  );
}

export function FormSkeleton({ fields = 4 }: { fields?: number }) {
  return (
    <div className="aoi-form-skeleton" aria-hidden="true">
      {Array.from({ length: fields }).map((_, index) => (
        <div className="aoi-form-skeleton__field" key={index}>
          <Skeleton className="aoi-skeleton--label" />
          <Skeleton className="aoi-skeleton--input" />
        </div>
      ))}
    </div>
  );
}
