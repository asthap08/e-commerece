"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { ArrowLeft } from "lucide-react";
import { DEMO_MODE, getDemoProfile, saveDemoProfile } from "@/lib/demo";
import { apiJSON } from "@/lib/api";
import { useCustomerAuth } from "@/lib/use-customer-auth";

type Profile = { name: string; email: string; phone: string };
type Note = { ok: boolean; text: string } | null;

export default function ProfilePage() {
  const authorized = useCustomerAuth();
  const [form, setForm] = useState<Profile>({ name: "", email: "", phone: "" });
  const [note, setNote] = useState<Note>(null);
  const [busy, setBusy] = useState(false);
  const [pw, setPw] = useState({ current: "", next: "", confirm: "" });
  const [pwNote, setPwNote] = useState<Note>(null);

  useEffect(() => {
    if (!authorized) return;
    if (DEMO_MODE) {
      const saved = getDemoProfile();
      // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
      if (saved) setForm({ name: saved.name, email: saved.email, phone: saved.phone ?? "" });
      return;
    }
    apiJSON<{ full_name: string; email: string; phone: string }>("/api/v1/auth/me")
      .then((u) => setForm({ name: u.full_name, email: u.email, phone: u.phone }))
      .catch((e: unknown) => setNote({ ok: false, text: e instanceof Error ? e.message : "Could not load your profile." }));
  }, [authorized]);

  if (!authorized) {
    return (<><Navbar /><main role="status" style={{ minHeight: "50vh", display: "grid", placeItems: "center" }}>Redirecting to sign in…</main><Footer /></>);
  }

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true); setNote(null);
    try {
      if (DEMO_MODE) saveDemoProfile({ name: form.name.trim(), email: form.email.trim().toLowerCase(), phone: form.phone.trim() });
      else await apiJSON("/api/v1/auth/me", { method: "PUT", body: JSON.stringify({ full_name: form.name, phone: form.phone }) }, "Could not save your profile.");
      setNote({ ok: true, text: "Profile saved." });
      window.dispatchEvent(new Event("omni-auth-change"));
    } catch (err) {
      setNote({ ok: false, text: err instanceof Error ? err.message : "Could not save your profile." });
    } finally { setBusy(false); }
  };

  const changePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwNote(null);
    if (pw.next !== pw.confirm) { setPwNote({ ok: false, text: "The new passwords do not match." }); return; }
    try {
      await apiJSON("/api/v1/auth/password", { method: "PUT", body: JSON.stringify({ current_password: pw.current, new_password: pw.next }) }, "Could not change your password.");
      setPw({ current: "", next: "", confirm: "" });
      setPwNote({ ok: true, text: "Password updated. Your other devices have been signed out." });
    } catch (err) {
      setPwNote({ ok: false, text: err instanceof Error ? err.message : "Could not change your password." });
    }
  };

  const field = (label: string, key: keyof Profile, props: React.InputHTMLAttributes<HTMLInputElement> = {}) => (
    <label>{label}<input value={form[key]} onChange={(e) => setForm({ ...form, [key]: e.target.value })} className="input-field" style={{ display: "block", width: "100%" }} {...props} /></label>
  );

  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 560, margin: "0 auto", padding: "2rem 1.25rem 4rem" }}>
        <Link href="/account" style={{ display: "inline-flex", alignItems: "center", gap: 6, color: "var(--text-secondary)", textDecoration: "none", marginBottom: "1rem" }}><ArrowLeft size={16} /> Account</Link>
        <h1 style={{ fontSize: "1.6rem", marginTop: 0 }}>Your profile</h1>
        <form onSubmit={save} style={{ display: "grid", gap: "0.9rem", marginBottom: "2.5rem" }}>
          {field("Full name", "name", { maxLength: 100, autoComplete: "name" })}
          {field("Email", "email", { type: "email", readOnly: !DEMO_MODE, "aria-readonly": !DEMO_MODE, style: { display: "block", width: "100%", opacity: DEMO_MODE ? 1 : 0.7 } })}
          {field("Phone", "phone", { type: "tel", autoComplete: "tel", maxLength: 20 })}
          <button className="btn-primary" disabled={busy} style={{ padding: "0.8rem", justifySelf: "start", minWidth: 140 }}>{busy ? "Saving…" : "Save changes"}</button>
          {note && <p role={note.ok ? "status" : "alert"} style={{ margin: 0, color: note.ok ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{note.text}</p>}
        </form>

        {!DEMO_MODE && (
          <form onSubmit={changePassword} style={{ display: "grid", gap: "0.9rem" }}>
            <h2 style={{ fontSize: "1.2rem", margin: 0 }}>Change password</h2>
            <label>Current password<input type="password" autoComplete="current-password" value={pw.current} onChange={(e) => setPw({ ...pw, current: e.target.value })} className="input-field" style={{ display: "block", width: "100%" }} required /></label>
            <label>New password<input type="password" autoComplete="new-password" value={pw.next} onChange={(e) => setPw({ ...pw, next: e.target.value })} className="input-field" style={{ display: "block", width: "100%" }} required minLength={8} maxLength={72} /></label>
            <label>Confirm new password<input type="password" autoComplete="new-password" value={pw.confirm} onChange={(e) => setPw({ ...pw, confirm: e.target.value })} className="input-field" style={{ display: "block", width: "100%" }} required /></label>
            <small style={{ color: "var(--text-muted)" }}>8–72 characters with upper and lower case, a number and a symbol.</small>
            <button className="btn-ghost" style={{ padding: "0.8rem", justifySelf: "start", minWidth: 160 }}>Update password</button>
            {pwNote && <p role={pwNote.ok ? "status" : "alert"} style={{ margin: 0, color: pwNote.ok ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{pwNote.text}</p>}
          </form>
        )}
      </main>
      <Footer />
    </>
  );
}
