import { useEffect, useId, useRef, useState } from "react";
import type { FormEvent } from "react";
import { authenticate, validateAuth } from "../auth";
import type { AuthFields, AuthMode, FieldErrors } from "../auth";
import Icon from "./Icon";

function PasswordField({
  label,
  name,
  value,
  error,
  hint,
  disabled,
  onChange,
}: {
  label: string;
  name: "password" | "confirmPassword";
  value: string;
  error?: string;
  hint?: string;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const id = useId();
  const [visible, setVisible] = useState(false);
  return (
    <div className="form-field">
      <label htmlFor={id}>{label}</label>
      <div className={`input-wrap ${error ? "has-error" : ""}`}>
        <Icon name="lock" />
        <input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          value={value}
          required
          placeholder={
            name === "confirmPassword"
              ? "Enter your password again"
              : "Enter your password"
          }
          autoComplete={
            hint || name === "confirmPassword"
              ? "new-password"
              : "current-password"
          }
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={Boolean(error)}
          aria-describedby={error || hint ? `${id}-note` : undefined}
        />
        <button
          type="button"
          className="password-toggle"
          onClick={() => setVisible(!visible)}
          aria-label={`${visible ? "Hide" : "Show"} ${label.toLowerCase()}`}
          aria-pressed={visible}
          disabled={disabled}
        >
          <Icon name={visible ? "eye-off" : "eye"} />
        </button>
      </div>
      {(error || hint) && (
        <p id={`${id}-note`} className={error ? "field-error" : "field-hint"}>
          {error || hint}
        </p>
      )}
    </div>
  );
}

export default function AuthForm({
  mode,
  onSwitch,
  onAuthenticated,
}: {
  mode: AuthMode;
  onSwitch: () => void;
  onAuthenticated: () => void;
}) {
  const signup = mode === "signup";
  const emailId = useId();
  const formRef = useRef<HTMLFormElement>(null);
  const requestRef = useRef<AbortController | null>(null);
  const [fields, setFields] = useState<AuthFields>({
    email: "",
    password: "",
    confirmPassword: "",
  });
  const [errors, setErrors] = useState<FieldErrors>({});
  const [requestError, setRequestError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => () => requestRef.current?.abort(), []);

  function updateField(name: keyof AuthFields, value: string) {
    setFields((current) => ({ ...current, [name]: value }));
    setErrors((current) => ({
      ...current,
      [name]: undefined,
      ...(name === "password" ? { confirmPassword: undefined } : {}),
    }));
    setRequestError("");
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (requestRef.current) return;
    const nextErrors = validateAuth(mode, fields);
    setErrors(nextErrors);
    setRequestError("");
    if (Object.keys(nextErrors).length) {
      const firstInvalid = Object.keys(nextErrors)[0];
      formRef.current
        ?.querySelector<HTMLInputElement>(`[name="${firstInvalid}"]`)
        ?.focus();
      return;
    }
    const controller = new AbortController();
    requestRef.current = controller;
    setPending(true);
    try {
      await authenticate(mode, fields, controller.signal);
      if (!controller.signal.aborted) {
        setFields({ email: "", password: "", confirmPassword: "" });
        onAuthenticated();
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        setRequestError(
          error instanceof Error
            ? error.message
            : "Something went wrong. Please try again.",
        );
      }
    } finally {
      if (!controller.signal.aborted) setPending(false);
      requestRef.current = null;
    }
  }

  return (
    <div className="auth-content">
      <div className="form-heading">
        <p className="eyebrow">
          {signup ? "MAKE YOURSELF AT HOME" : "YOUR PROPERTY WORKSPACE"}
        </p>
        <h1>{signup ? "Create your account." : "Welcome back."}</h1>
        <p className="form-description">
          {signup
            ? "A fresh perspective on property starts here."
            : "Sign in to see your next move more clearly."}
        </p>
      </div>
      <form
        ref={formRef}
        onSubmit={handleSubmit}
        noValidate
        aria-label={signup ? "Create account" : "Sign in"}
        aria-busy={pending}
      >
        <div className="form-field">
          <label htmlFor={emailId}>Email address</label>
          <div className={`input-wrap ${errors.email ? "has-error" : ""}`}>
            <Icon name="mail" />
            <input
              id={emailId}
              name="email"
              type="email"
              placeholder="you@example.com"
              autoComplete="email"
              autoCapitalize="none"
              spellCheck={false}
              required
              value={fields.email}
              onChange={(event) => updateField("email", event.target.value)}
              disabled={pending}
              aria-invalid={Boolean(errors.email)}
              aria-describedby={errors.email ? `${emailId}-error` : undefined}
            />
          </div>
          {errors.email && (
            <p id={`${emailId}-error`} className="field-error">
              {errors.email}
            </p>
          )}
        </div>
        <PasswordField
          label="Password"
          name="password"
          value={fields.password}
          error={errors.password}
          disabled={pending}
          hint={
            signup
              ? "9–72 characters. English letters and numbers only."
              : undefined
          }
          onChange={(value) => updateField("password", value)}
        />
        {signup && (
          <PasswordField
            label="Confirm password"
            name="confirmPassword"
            value={fields.confirmPassword}
            error={errors.confirmPassword}
            disabled={pending}
            onChange={(value) => updateField("confirmPassword", value)}
          />
        )}
        {requestError && (
          <p className="request-error" role="alert">
            {requestError}
          </p>
        )}
        <button type="submit" className="primary-button" disabled={pending}>
          <span>
            {pending
              ? signup
                ? "Creating your account…"
                : "Signing in…"
              : signup
                ? "Create account"
                : "Sign in"}
          </span>
          {pending ? (
            <span className="spinner" aria-hidden="true" />
          ) : (
            <Icon name="arrow" />
          )}
        </button>
      </form>
      <div className="form-divider">
        <span>YOUR NEXT MOVE STARTS HERE</span>
      </div>
      <p className="switch-copy">
        {signup ? "Already have an account?" : "New to Property Co Pilot?"}
        <button
          type="button"
          className="text-button"
          onClick={onSwitch}
          disabled={pending}
        >
          {signup ? "Sign in" : "Create account"}
          <span aria-hidden="true">↗</span>
        </button>
      </p>
    </div>
  );
}
