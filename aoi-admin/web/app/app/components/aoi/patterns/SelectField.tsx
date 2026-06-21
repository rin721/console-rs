import type { SelectHTMLAttributes } from "react";
import { useId } from "react";

export type SelectOption = {
  label: string;
  value: string;
};

type SelectFieldProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, "children"> & {
  error?: string;
  help?: string;
  label: string;
  options: SelectOption[];
};

export function SelectField({ error, help, id, label, options, ...props }: SelectFieldProps) {
  const generatedId = useId();
  const selectId = id ?? generatedId;
  const helpId = help ? `${selectId}-help` : undefined;
  const errorId = error ? `${selectId}-error` : undefined;
  const describedBy = [helpId, errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="aoi-form-field">
      <label htmlFor={selectId}>{label}</label>
      <select id={selectId} aria-describedby={describedBy} aria-invalid={Boolean(error)} {...props}>
        {options.map((option) => (
          <option key={option.value || "all"} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
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
