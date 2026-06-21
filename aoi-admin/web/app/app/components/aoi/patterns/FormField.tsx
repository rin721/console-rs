import type { InputHTMLAttributes } from "react";
import { useId } from "react";

type FormFieldProps = InputHTMLAttributes<HTMLInputElement> & {
  error?: string;
  help?: string;
  label: string;
};

export function FormField({ error, help, id, label, ...props }: FormFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const helpId = help ? `${inputId}-help` : undefined;
  const errorId = error ? `${inputId}-error` : undefined;
  const describedBy = [helpId, errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="aoi-form-field">
      <label htmlFor={inputId}>{label}</label>
      <input id={inputId} aria-describedby={describedBy} aria-invalid={Boolean(error)} {...props} />
      {help ? (
        <span id={helpId} className="aoi-form-field__help">
          {help}
        </span>
      ) : null}
      {error ? (
        <span id={errorId} className="aoi-form-field__error">
          {error}
        </span>
      ) : null}
    </div>
  );
}
