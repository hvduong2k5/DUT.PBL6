"use client";

import { InputHTMLAttributes, ReactNode, useState } from "react";
import { getPasswordChecks } from "@/lib/auth/validation";

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: string;
  icon?: string;
}

export function FormField({ label, error, hint, icon, id, type, ...inputProps }: FieldProps) {
  const [visible, setVisible] = useState(false);
  const isPassword = type === "password";
  const inputType = isPassword && visible ? "text" : type;
  const describedBy = [error ? `${id}-error` : "", hint ? `${id}-hint` : ""].filter(Boolean).join(" ") || undefined;

  return (
    <label className="form-field" htmlFor={id}>
      <span className="field-label">{label}{inputProps.required ? <b aria-hidden="true"> *</b> : null}</span>
      <span className={`input-wrap ${error ? "has-error" : ""}`}>
        {icon ? <span className="input-icon" aria-hidden="true">{icon}</span> : null}
        <input id={id} type={inputType} aria-invalid={Boolean(error)} aria-describedby={describedBy} {...inputProps} />
        {isPassword ? (
          <button className="visibility-button" type="button" onClick={() => setVisible((value) => !value)} aria-label={visible ? "Ẩn mật khẩu" : "Hiện mật khẩu"}>
            {visible ? "◉" : "◎"}
          </button>
        ) : null}
      </span>
      {error ? <span className="field-error" id={`${id}-error`} role="alert">{error}</span> : null}
      {hint ? <span className="field-hint" id={`${id}-hint`}>{hint}</span> : null}
    </label>
  );
}

export function PasswordChecklist({ password }: { password: string }) {
  return (
    <div className="password-checklist" aria-label="Điều kiện mật khẩu">
      {getPasswordChecks(password).map((check) => (
        <span className={check.valid ? "valid" : ""} key={check.code}>
          <b aria-hidden="true">{check.valid ? "✓" : "○"}</b>{check.label}
        </span>
      ))}
    </div>
  );
}

export function StatusNotice({ tone = "error", children }: { tone?: "error" | "success" | "info"; children: ReactNode }) {
  return <div className={`status-notice ${tone}`} role={tone === "error" ? "alert" : "status"}>{children}</div>;
}

export function SubmitButton({ pending, children }: { pending: boolean; children: ReactNode }) {
  return <button className="primary-button" type="submit" disabled={pending}>{pending ? <><span className="spinner" />Đang xử lý...</> : children}</button>;
}
