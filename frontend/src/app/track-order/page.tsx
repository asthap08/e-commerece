"use client";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { ApiError, apiJSON, inrMinor } from "@/lib/api";
import { DEMO_MODE, getDemoOrder } from "@/lib/demo";

type Tracked = {
  id: string;
  status: string;
  total_minor: number;
  created_at: string;
  payment_method: string;
  shipping_method: string;
  tracking_number: string;
  tracking_carrier: string;
  items: { id: string; name: string; quantity: number; price_minor: number }[];
};

const STEPS = [
  { key: "pending", label: "Order placed" },
  { key: "processing", label: "Being prepared" },
  { key: "shipped", label: "Shipped" },
  { key: "delivered", label: "Delivered" },
];
// paid == payment received, still waiting to be prepared
const stepIndex = (status: string) => (status === "paid" ? 0 : Math.max(0, STEPS.findIndex((s) => s.key === status)));

function TrackView() {
  const params = useSearchParams();
  const [orderId, setOrderId] = useState(params.get("id") ?? "");
  const [email, setEmail] = useState("");
  const [order, setOrder] = useState<Tracked | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
  useEffect(() => { const id = params.get("id"); if (id) setOrderId(id); }, [params]);

  const lookup = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(""); setOrder(null); setBusy(true);
    try {
      if (DEMO_MODE) {
        const d = getDemoOrder(orderId.trim().replace(/^#/, ""));
        if (!d) throw new ApiError("No order matches that id.", 404);
        setOrder({ ...d, items: d.items });
      } else {
        setOrder(await apiJSON<Tracked>("/api/v1/orders/track", { method: "POST", body: JSON.stringify({ order_id: orderId.trim().replace(/^#/, ""), email: email.trim() }) }, "We could not look up that order."));
      }
    } catch (err) {
      setError(err instanceof ApiError && err.status === 404 ? "We couldn’t find an order with that ID and email. Check both and try again." : err instanceof Error ? err.message : "Lookup failed.");
    } finally { setBusy(false); }
  };

  const cancelled = order?.status === "cancelled" || order?.status === "payment_failed";
  const idx = order ? stepIndex(order.status) : 0;

  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 720, margin: "0 auto", padding: "2rem 1.25rem 4rem" }}>
        <h1 style={{ fontSize: "1.7rem", marginBottom: "0.25rem" }}>Track your order</h1>
        <p style={{ color: "var(--text-secondary)", marginTop: 0 }}>Enter the order ID from your confirmation{DEMO_MODE ? "" : " and the email you used at checkout"}. No account needed.</p>
        <form onSubmit={lookup} style={{ display: "grid", gap: "0.75rem", margin: "1.25rem 0" }}>
          <label>Order ID<input required value={orderId} onChange={(e) => setOrderId(e.target.value)} className="input-field" style={{ width: "100%", display: "block", fontFamily: "monospace" }} /></label>
          {!DEMO_MODE && <label>Email<input required type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="input-field" style={{ width: "100%", display: "block" }} /></label>}
          <button className="btn-primary" disabled={busy} style={{ padding: "0.8rem", justifySelf: "start", minWidth: 160 }}>{busy ? "Looking up…" : "Track order"}</button>
        </form>
        {error && <p role="alert" style={{ color: "var(--accent-rose)", padding: "0.75rem", background: "#fef2f2", border: "1px solid #fecaca", borderRadius: 10 }}>{error}</p>}

        {order && (
          <section aria-live="polite" style={{ border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", padding: "1.25rem", background: "var(--bg-card)" }}>
            <div style={{ display: "flex", justifyContent: "space-between", flexWrap: "wrap", gap: 8 }}>
              <strong style={{ fontFamily: "monospace" }}>#{order.id}</strong>
              <span>{new Date(order.created_at).toLocaleDateString("en-IN", { dateStyle: "long" })}</span>
            </div>
            {cancelled ? (
              <p style={{ color: "var(--accent-rose)", fontWeight: 600 }}>{order.status === "cancelled" ? "This order was cancelled." : "Payment for this order failed."}</p>
            ) : (
              <ol style={{ listStyle: "none", padding: 0, margin: "1.25rem 0", display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 8 }}>
                {STEPS.map((s, i) => (
                  <li key={s.key} aria-current={i === idx ? "step" : undefined} style={{ textAlign: "center", fontSize: "0.8rem", color: i <= idx ? "var(--accent-primary)" : "var(--text-muted)", fontWeight: i === idx ? 700 : 500 }}>
                    <div style={{ height: 6, borderRadius: 3, background: i <= idx ? "var(--accent-primary)" : "var(--border-subtle)", marginBottom: 6 }} />
                    {s.label}
                  </li>
                ))}
              </ol>
            )}
            {order.tracking_number && (
              <p style={{ margin: "0 0 1rem" }}>Tracking: <strong style={{ fontFamily: "monospace" }}>{order.tracking_number}</strong>{order.tracking_carrier && <> via {order.tracking_carrier}</>}</p>
            )}
            <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "grid", gap: 6 }}>
              {order.items.map((it) => <li key={it.id} style={{ display: "flex", justifyContent: "space-between" }}><span>{it.name} × {it.quantity}</span><span>{inrMinor(it.price_minor * it.quantity)}</span></li>)}
            </ul>
            <p style={{ textAlign: "right", fontWeight: 700, margin: "1rem 0 0" }}>Total {inrMinor(order.total_minor)}</p>
          </section>
        )}
      </main>
      <Footer />
    </>
  );
}

export default function TrackOrderPage() {
  return <Suspense fallback={null}><TrackView /></Suspense>;
}
