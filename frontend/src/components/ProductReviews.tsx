"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ApiError, apiFetch, apiJSON } from "@/lib/api";
import { DEMO_MODE } from "@/lib/demo";

type Review = { id: string; author: string; verified_purchase: boolean; rating: number; title: string; body: string; created_at: string };
type Meta = { average_rating: number; review_count: number };

const Stars = ({ n }: { n: number }) => <span aria-label={`${n} out of 5 stars`} style={{ color: "var(--accent-amber)", letterSpacing: 1 }}>{"★".repeat(n)}{"☆".repeat(5 - n)}</span>;

/** Public reviews list + review form. Guests can read reviews; posting needs sign-in (no redirect). */
export default function ProductReviews({ productId }: { productId: string }) {
  const [reviews, setReviews] = useState<Review[]>([]);
  const [meta, setMeta] = useState<Meta>({ average_rating: 0, review_count: 0 });
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [rating, setRating] = useState(5);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await apiJSON<{ data: Review[]; meta: Meta }>(`/api/v1/products/${productId}/reviews`);
      setReviews(r.data);
      setMeta(r.meta);
    } catch { /* reviews are optional content */ }
  }, [productId]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
    if (DEMO_MODE) { setSignedIn(false); return; }
    void load();
    apiFetch("/api/v1/auth/me").then((r) => setSignedIn(r.ok)).catch(() => setSignedIn(false));
  }, [load]);

  if (DEMO_MODE) return null;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg(null);
    setBusy(true);
    try {
      await apiJSON(`/api/v1/products/${productId}/reviews`, { method: "POST", body: JSON.stringify({ rating, title, body }) }, "Your review could not be saved.");
      setMsg({ ok: true, text: "Thanks! Your review has been posted." });
      setTitle(""); setBody(""); setRating(5);
      void load();
    } catch (err) {
      setMsg({ ok: false, text: err instanceof ApiError ? err.message : "Your review could not be saved." });
    } finally {
      setBusy(false);
    }
  };

  return (
    <section aria-labelledby="reviews-h" style={{ marginTop: "3rem" }}>
      <h2 id="reviews-h" style={{ fontSize: "1.3rem", marginBottom: "0.25rem" }}>Customer reviews</h2>
      <p style={{ color: "var(--text-secondary)", margin: "0 0 1.25rem" }}>
        {meta.review_count > 0 ? <><Stars n={Math.round(meta.average_rating)} /> {meta.average_rating.toFixed(1)} · {meta.review_count} review{meta.review_count === 1 ? "" : "s"}</> : "No reviews yet — be the first."}
      </p>
      <div style={{ display: "grid", gap: "1rem", marginBottom: "2rem" }}>
        {reviews.map((r) => (
          <article key={r.id} style={{ padding: "1rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "var(--bg-card)" }}>
            <div style={{ display: "flex", gap: "0.5rem", alignItems: "center", flexWrap: "wrap" }}>
              <Stars n={r.rating} />
              {r.title && <strong>{r.title}</strong>}
            </div>
            {r.body && <p style={{ margin: "0.5rem 0", whiteSpace: "pre-wrap" }}>{r.body}</p>}
            <small style={{ color: "var(--text-muted)" }}>
              {r.author} · {new Date(r.created_at).toLocaleDateString("en-IN", { dateStyle: "medium" })}
              {r.verified_purchase && <span style={{ color: "var(--accent-emerald)", marginLeft: 8 }}>✓ Verified purchase</span>}
            </small>
          </article>
        ))}
      </div>
      {signedIn === false ? (
        <p><Link href="/login" style={{ color: "var(--accent-primary)", fontWeight: 600 }}>Sign in</Link> to write a review.</p>
      ) : signedIn ? (
        <form onSubmit={submit} style={{ display: "grid", gap: "0.75rem", maxWidth: 560 }}>
          <h3 style={{ margin: 0 }}>Write a review</h3>
          <label>Rating
            <select value={rating} onChange={(e) => setRating(Number(e.target.value))} className="input-field" style={{ display: "block", marginTop: 4 }}>
              {[5, 4, 3, 2, 1].map((n) => <option key={n} value={n}>{n} star{n > 1 ? "s" : ""}</option>)}
            </select>
          </label>
          <label>Title<input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={120} className="input-field" style={{ display: "block", width: "100%", marginTop: 4 }} /></label>
          <label>Review<textarea value={body} onChange={(e) => setBody(e.target.value)} maxLength={2000} rows={4} className="input-field" style={{ display: "block", width: "100%", marginTop: 4 }} /></label>
          {msg && <p role={msg.ok ? "status" : "alert"} style={{ margin: 0, color: msg.ok ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{msg.text}</p>}
          <button className="btn-primary" disabled={busy} style={{ padding: "0.7rem 1.25rem", justifySelf: "start" }}>{busy ? "Posting…" : "Post review"}</button>
        </form>
      ) : null}
    </section>
  );
}
