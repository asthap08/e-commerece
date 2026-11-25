"use client";

import { useState } from "react";
import AuthPageShell from "@/components/AuthPageShell";
import { apiJSON } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await apiJSON("/api/v1/auth/forgot", { method: "POST", body: JSON.stringify({ email: email.trim() }) }, "Could not send the reset link. Please try again.");
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send the reset link.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthPageShell title="Forgot your password?" description="Enter your email and we’ll send you a link to choose a new one." alternateHref="/login" alternateText="Remembered it?" alternateLabel="Sign in">
      {done ? (
        <p role="status" style={{ color: "var(--accent-emerald)" }}>If that email has an account, a reset link is on its way. It expires in 1 hour.</p>
      ) : (
        <form onSubmit={submit} style={{ display: "grid", gap: "1rem" }}>
          <label>Email<input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} className="input-field" style={{ display: "block", width: "100%" }} /></label>
          {error && <p role="alert" style={{ margin: 0, color: "var(--accent-rose)" }}>{error}</p>}
          <button className="btn-primary" disabled={busy} style={{ padding: "0.8rem" }}>{busy ? "Sending…" : "Send reset link"}</button>
        </form>
      )}
    </AuthPageShell>
  );
}
