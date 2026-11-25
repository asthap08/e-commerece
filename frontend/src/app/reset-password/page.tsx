"use client";

import { Suspense, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import AuthPageShell from "@/components/AuthPageShell";
import { apiJSON } from "@/lib/api";

function ResetForm() {
  const token = useSearchParams().get("token") ?? "";
  const [pw, setPw] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (pw !== confirm) return setError("The passwords do not match.");
    setBusy(true);
    try {
      await apiJSON("/api/v1/auth/reset", { method: "POST", body: JSON.stringify({ token, new_password: pw }) }, "Could not reset your password.");
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not reset your password.");
    } finally {
      setBusy(false);
    }
  }

  if (!token) return <p role="alert">This reset link is incomplete. <Link href="/forgot-password" style={{ color: "var(--accent-primary)", fontWeight: 600 }}>Request a new one</Link>.</p>;
  if (done) return <p role="status" style={{ color: "var(--accent-emerald)" }}>Password updated. <Link href="/login" style={{ color: "var(--accent-primary)", fontWeight: 600 }}>Sign in</Link> with your new password.</p>;
  return (
    <form onSubmit={submit} style={{ display: "grid", gap: "1rem" }}>
      <label>New password<input type="password" required minLength={8} maxLength={72} autoComplete="new-password" value={pw} onChange={(e) => setPw(e.target.value)} className="input-field" style={{ display: "block", width: "100%" }} /></label>
      <label>Confirm password<input type="password" required autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} className="input-field" style={{ display: "block", width: "100%" }} /></label>
      <small style={{ color: "var(--text-muted)" }}>8–72 characters with upper and lower case, a number and a symbol.</small>
      {error && <p role="alert" style={{ margin: 0, color: "var(--accent-rose)" }}>{error}{error.includes("invalid or has expired") && <> <Link href="/forgot-password" style={{ fontWeight: 600 }}>Request a new link</Link></>}</p>}
      <button className="btn-primary" disabled={busy} style={{ padding: "0.8rem" }}>{busy ? "Saving…" : "Set new password"}</button>
    </form>
  );
}

export default function ResetPasswordPage() {
  return (
    <AuthPageShell title="Choose a new password" description="You’ll be signed out of all other devices." alternateHref="/login" alternateText="Back to" alternateLabel="Sign in">
      <Suspense fallback={null}><ResetForm /></Suspense>
    </AuthPageShell>
  );
}
