import { useEffect } from "react";

export function useJsonLd(id: string, value: Record<string, unknown> | null) {
  useEffect(() => {
    const selector = `script[type="application/ld+json"][data-aoi-jsonld="${id}"]`;
    const existing = document.querySelector<HTMLScriptElement>(selector);

    if (!value) {
      existing?.remove();
      return;
    }

    const script = existing ?? document.createElement("script");
    script.type = "application/ld+json";
    script.dataset.aoiJsonld = id;
    script.text = JSON.stringify(value).replace(/</g, "\\u003c");

    if (!existing) {
      document.head.append(script);
    }

    return () => {
      script.remove();
    };
  }, [id, value]);
}
