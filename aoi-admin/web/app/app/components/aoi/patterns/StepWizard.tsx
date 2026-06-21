import {
  useEffect,
  useRef,
} from "react";
import {
  AlertCircle,
  CheckCircle2,
  Circle,
  LoaderCircle,
  Lock,
  MinusCircle,
  type LucideIcon,
} from "lucide-react";

import { cn } from "~/lib/cn";

export type StepWizardStatus =
  | "blocked"
  | "current"
  | "failed"
  | "pending"
  | "running"
  | "skipped"
  | "succeeded";

export type StepWizardItem = {
  description?: string;
  disabled?: boolean;
  key: string;
  status: StepWizardStatus;
  statusLabel: string;
  title: string;
};

type StepWizardProps = {
  ariaLabel: string;
  className?: string;
  currentKey: string;
  items: StepWizardItem[];
  onSelect: (key: string) => void;
  progressLabel: string;
  progressValue: number;
};

export function StepWizard({
  ariaLabel,
  className,
  currentKey,
  items,
  onSelect,
  progressLabel,
  progressValue,
}: StepWizardProps) {
  const itemRefs = useRef(new Map<string, HTMLLIElement>());

  useEffect(() => {
    const item = itemRefs.current.get(currentKey);
    const list = item?.parentElement;

    if (!item || !list) {
      return;
    }

    const itemRect = item.getBoundingClientRect();
    const listRect = list.getBoundingClientRect();
    const itemLeft = itemRect.left - listRect.left + list.scrollLeft;
    const itemRight = itemLeft + itemRect.width;
    const visibleLeft = list.scrollLeft;
    const visibleRight = visibleLeft + list.clientWidth;

    if (itemLeft < visibleLeft) {
      list.scrollLeft = itemLeft;
      return;
    }

    if (itemRight > visibleRight) {
      list.scrollLeft = itemRight - list.clientWidth;
    }
  }, [currentKey, items.length]);

  return (
    <nav className={cn("aoi-step-wizard", className)} aria-label={ariaLabel}>
      <div className="aoi-step-wizard__summary">
        <span>{progressLabel}</span>
        <div
          className="aoi-step-wizard__progress"
          aria-label={progressLabel}
          aria-valuemax={100}
          aria-valuemin={0}
          aria-valuenow={progressValue}
          role="progressbar"
        >
          <span style={{ width: `${progressValue}%` }} />
        </div>
      </div>
      <ol className="aoi-step-wizard__list">
        {items.map((item, index) => {
          const current = item.key === currentKey;
          const Icon = iconForStatus(item.status);

          return (
            <li
              key={item.key}
              ref={(node) => {
                if (node) {
                  itemRefs.current.set(item.key, node);
                } else {
                  itemRefs.current.delete(item.key);
                }
              }}
            >
              <button
                className="aoi-step-wizard__button"
                data-status={item.status}
                type="button"
                aria-current={current ? "step" : undefined}
                aria-label={`${index + 1}. ${item.title}. ${item.statusLabel}`}
                disabled={item.disabled}
                onClick={() => onSelect(item.key)}
              >
                <span className="aoi-step-wizard__icon" aria-hidden="true">
                  <Icon size={18} />
                </span>
                <span className="aoi-step-wizard__body">
                  <span className="aoi-step-wizard__title">{item.title}</span>
                  {item.description ? (
                    <span className="aoi-step-wizard__description">{item.description}</span>
                  ) : null}
                </span>
                <span className="aoi-step-wizard__status">{item.statusLabel}</span>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

function iconForStatus(status: StepWizardStatus): LucideIcon {
  if (status === "failed") {
    return AlertCircle;
  }
  if (status === "blocked") {
    return Lock;
  }
  if (status === "running" || status === "current") {
    return LoaderCircle;
  }
  if (status === "skipped") {
    return MinusCircle;
  }
  if (status === "succeeded") {
    return CheckCircle2;
  }
  return Circle;
}
