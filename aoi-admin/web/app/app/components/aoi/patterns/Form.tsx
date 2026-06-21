import type { ComponentProps, ReactNode } from "react";
import {
  FormProvider,
  useFormContext,
  type FieldErrors,
  type FieldValues,
  type Path,
  type SubmitHandler,
  type UseFormReturn,
} from "react-hook-form";

import { FormField } from "./FormField";

type AoiFormProps<TValues extends FieldValues> = {
  children: ReactNode;
  className?: string;
  form: UseFormReturn<TValues>;
  onSubmit: SubmitHandler<TValues>;
};

type AoiTextFieldProps<TValues extends FieldValues> = Omit<
  ComponentProps<typeof FormField>,
  "error" | "name"
> & {
  name: Path<TValues>;
};

export function AoiForm<TValues extends FieldValues>({
  children,
  className,
  form,
  onSubmit,
}: AoiFormProps<TValues>) {
  return (
    <FormProvider {...form}>
      <form
        className={className}
        noValidate
        onSubmit={(event) => void form.handleSubmit(onSubmit)(event)}
      >
        {children}
      </form>
    </FormProvider>
  );
}

export function AoiTextField<TValues extends FieldValues>({
  name,
  ...props
}: AoiTextFieldProps<TValues>) {
  const {
    formState: { errors },
    register,
  } = useFormContext<TValues>();

  return <FormField error={fieldErrorMessage(errors, name)} {...props} {...register(name)} />;
}

export function fieldErrorMessage<TValues extends FieldValues>(
  errors: FieldErrors<TValues>,
  name: Path<TValues>,
) {
  const value = name.split(".").reduce<unknown>((current, segment) => {
    if (current && typeof current === "object" && segment in current) {
      return (current as Record<string, unknown>)[segment];
    }
    return undefined;
  }, errors);

  if (value && typeof value === "object" && "message" in value) {
    const message = (value as { message?: unknown }).message;
    return typeof message === "string" ? message : undefined;
  }
  return undefined;
}
