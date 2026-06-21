import { AlertCircle, Info } from "lucide-react";

type StateBlockIntent = "danger" | "info";

type StateBlockProps = {
  action?: React.ReactNode;
  description: string;
  intent?: StateBlockIntent;
  title: string;
};

export function StateBlock({ action, description, intent = "info", title }: StateBlockProps) {
  const Icon = intent === "danger" ? AlertCircle : Info;

  return (
    <section className={`aoi-state-block aoi-state-block--${intent}`} aria-live="polite">
      <Icon aria-hidden="true" size={22} />
      <div>
        <h2>{title}</h2>
        <p>{description}</p>
      </div>
      {action}
    </section>
  );
}
