"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { createDemoReturn, DEMO_MODE, listDemoOrders, listDemoReturns } from "@/lib/demo";
import { apiJSON, inrMinor } from "@/lib/api";
import { useCustomerAuth } from "@/lib/use-customer-auth";

type Item = { id: string; name: string; quantity: number; price_minor: number };
type EligibleOrder = { id: string; created_at: string; status: string; items: Item[] };
type ReturnRow = { id: string; order_id: string; status: string; reason: string; refund_amount_minor?: number; created_at: string; admin_note?: string; label?: string };

const STATUS_TEXT: Record<string, string> = {
  requested: "Waiting for review", approved: "Approved — send the items back", received: "Items received", refunded: "Refunded", rejected: "Declined",
};

function ReturnsView() {
  const authorized = useCustomerAuth();
  const params = useSearchParams();
  const [orders, setOrders] = useState<EligibleOrder[]>([]);
  const [returns, setReturns] = useState<ReturnRow[]>([]);
  const [orderId, setOrderId] = useState(params.get("order") ?? "");
  const [picked, setPicked] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (DEMO_MODE) {
      setOrders(listDemoOrders().filter((o) => o.status === "delivered"));
      setReturns(listDemoReturns().map((r) => ({ ...r })));
      return;
    }
    try {
      const [o, r] = await Promise.all([
        apiJSON<{ data: EligibleOrder[] }>("/api/v1/orders?limit=100"),
        apiJSON<{ data: ReturnRow[] }>("/api/v1/returns/mine"),
      ]);
      const open = new Set(r.data.filter((x) => x.status !== "rejected").map((x) => x.order_id));
      setOrders(o.data.filter((x) => x.status === "delivered" && !open.has(x.id)));
      setReturns(r.data);
    } catch (e) {
      setMsg({ ok: false, text: e instanceof Error ? e.message : "Could not load your orders." });
    }
  }, []);

  // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
  useEffect(() => { if (authorized) void load(); }, [authorized, load]);

  const order = orders.find((o) => o.id === orderId);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
  useEffect(() => { setPicked(order ? order.items.map((i) => i.id) : []); }, [order]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!order || picked.length === 0 || reason.trim().length < 5) {
      setMsg({ ok: false, text: "Choose an order, at least one item, and tell us why (a few words)." });
      return;
    }
    setBusy(true); setMsg(null);
    try {
      if (DEMO_MODE) {
        const first = order.items.find((i) => i.id === picked[0])!;
        createDemoReturn({ order_id: order.id, item_name: first.name, reason: reason.trim() });
      } else {
        await apiJSON("/api/v1/returns", { method: "POST", body: JSON.stringify({ order_id: order.id, item_ids: picked, reason: reason.trim() }) }, "Your return could not be submitted.");
      }
      setMsg({ ok: true, text: "Return request submitted. We’ll review it and update the status below." });
      setOrderId(""); setReason("");
      await load();
    } catch (err) {
      setMsg({ ok: false, text: err instanceof Error ? err.message : "Your return could not be submitted." });
    } finally { setBusy(false); }
  };

  if (!authorized) return null;

  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 820, margin: "0 auto", padding: "2rem 1.25rem 4rem" }}>
        <h1 style={{ fontSize: "1.7rem", marginBottom: "0.25rem" }}>Returns</h1>
        <p style={{ color: "var(--text-secondary)", marginTop: 0 }}>Only delivered orders can be returned. Refunds go back to the original payment method.</p>

        <form onSubmit={submit} style={{ display: "grid", gap: "0.9rem", padding: "1.25rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", background: "var(--bg-card)", margin: "1.25rem 0" }}>
          <h2 style={{ margin: 0, fontSize: "1.1rem" }}>Start a return</h2>
          {orders.length === 0 ? (
            <p style={{ margin: 0, color: "var(--text-secondary)" }}>You have no delivered orders that are eligible for a return. <Link href="/account/orders" style={{ color: "var(--accent-primary)", fontWeight: 600 }}>View your orders</Link></p>
          ) : (
            <>
              <label>Order
                <select value={orderId} onChange={(e) => setOrderId(e.target.value)} className="input-field" style={{ display: "block", width: "100%" }}>
                  <option value="">Select a delivered order…</option>
                  {orders.map((o) => <option key={o.id} value={o.id}>#{o.id.slice(0, 8)} · {new Date(o.created_at).toLocaleDateString("en-IN")}</option>)}
                </select>
              </label>
              {order && (
                <fieldset style={{ border: "none", padding: 0, margin: 0 }}>
                  <legend style={{ fontWeight: 600, marginBottom: 6 }}>Items to return</legend>
                  {order.items.map((it) => (
                    <label key={it.id} style={{ display: "flex", gap: 8, alignItems: "center", padding: "4px 0" }}>
                      <input type="checkbox" checked={picked.includes(it.id)} onChange={(e) => setPicked((p) => (e.target.checked ? [...p, it.id] : p.filter((x) => x !== it.id)))} />
                      <span style={{ flex: 1 }}>{it.name} × {it.quantity}</span><span>{inrMinor(it.price_minor * it.quantity)}</span>
                    </label>
                  ))}
                </fieldset>
              )}
              <label>Reason<textarea value={reason} onChange={(e) => setReason(e.target.value)} rows={3} maxLength={1000} className="input-field" style={{ display: "block", width: "100%" }} placeholder="e.g. Arrived damaged, wrong size…" /></label>
              <button className="btn-primary" disabled={busy || !order} style={{ padding: "0.8rem", justifySelf: "start", minWidth: 180 }}>{busy ? "Submitting…" : "Submit return request"}</button>
            </>
          )}
          {msg && <p role={msg.ok ? "status" : "alert"} style={{ margin: 0, color: msg.ok ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{msg.text}</p>}
        </form>

        <h2 style={{ fontSize: "1.1rem" }}>Your return requests</h2>
        {returns.length === 0 ? <p style={{ color: "var(--text-muted)" }}>No return requests yet.</p> : (
          <ul style={{ listStyle: "none", padding: 0, display: "grid", gap: 10 }}>
            {returns.map((r) => (
              <li key={r.id} style={{ padding: "0.9rem 1rem", border: "1px solid var(--border-subtle)", borderRadius: 10, background: "var(--bg-card)" }}>
                <div style={{ display: "flex", justifyContent: "space-between", flexWrap: "wrap", gap: 6 }}>
                  <strong>Order #{r.order_id.slice(0, 8)}</strong>
                  <span style={{ color: r.status === "rejected" ? "var(--accent-rose)" : "var(--accent-primary)", fontWeight: 600 }}>{STATUS_TEXT[r.status] ?? r.status}</span>
                </div>
                <p style={{ margin: "0.35rem 0 0", color: "var(--text-secondary)" }}>{r.reason}</p>
                {r.admin_note && <p style={{ margin: "0.35rem 0 0", fontSize: "0.85rem" }}>Note from us: {r.admin_note}</p>}
                {!DEMO_MODE && typeof r.refund_amount_minor === "number" && <p style={{ margin: "0.35rem 0 0", fontSize: "0.85rem", color: "var(--text-muted)" }}>Refund amount: {inrMinor(r.refund_amount_minor)}</p>}
              </li>
            ))}
          </ul>
        )}
      </main>
      <Footer />
    </>
  );
}

export default function ReturnsPage() {
  return <Suspense fallback={null}><ReturnsView /></Suspense>;
}
