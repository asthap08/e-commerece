"use client";
import { useCallback, useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import Link from "next/link";
import { Truck, Package, RefreshCw, RotateCcw, XCircle } from "lucide-react";
import { apiFetch, apiJSON, fmtNum } from "@/lib/api";
import { cancelDemoOrder, DEMO_MODE, listDemoOrders } from "@/lib/demo";
import { useStore } from "@/lib/store";
import { useCustomerAuth } from "@/lib/use-customer-auth";

type OrderItem = {
  id: string;
  variant_id?: string;
  name: string;
  quantity: number;
  price_minor: number;
  image?: string;
};

type Order = {
  payment_method?: string;
  id: string;
  status: string;
  total_minor: number;
  created_at: string;
  tracking_number: string;
  items: OrderItem[];
};

const STATUS_CONFIG: Record<string, { label: string; color: string; bg: string; steps: string[] }> = {
  pending:   { label: "Pending", color: "#f59e0b", bg: "#fef3c7", steps: ["Ordered", "Preparing", "Shipped", "Out for Delivery", "Delivered"] },
  paid:      { label: "Payment Confirmed", color: "#3b82f6", bg: "#dbeafe", steps: ["Ordered", "Preparing", "Shipped", "Out for Delivery", "Delivered"] },
  processing:{ label: "Processing", color: "#8b5cf6", bg: "#e9d5ff", steps: ["Ordered", "Preparing", "Shipped", "Out for Delivery", "Delivered"] },
  shipped:   { label: "Shipped", color: "#06b6d4", bg: "#cffafe", steps: ["Ordered", "Preparing", "Shipped", "Out for Delivery", "Delivered"] },
  delivered: { label: "Delivered", color: "#10b981", bg: "#d1fae5", steps: ["Ordered", "Preparing", "Shipped", "Out for Delivery", "Delivered"] },
  cancelled: { label: "Cancelled", color: "#ef4444", bg: "#fee2e2", steps: ["Ordered", "Cancelled"] },
  payment_failed: { label: "Payment Failed", color: "#dc2626", bg: "#fee2e2", steps: ["Ordered", "Payment Failed"] },
};

type OrderStatus = keyof typeof STATUS_CONFIG;

const FILTER_OPTIONS: Array<string | "all"> = ["all", "pending", "paid", "processing", "shipped", "delivered", "cancelled", "payment_failed"];

export default function OrdersPage() {
  const authorized = useCustomerAuth();
  const { addToCart, loadCart, setCartOpen } = useStore();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string | "all">("all");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const loadOrders = useCallback(async () => {
    setLoading(true);
    setError("");
    setNotice("");
    if (DEMO_MODE) {
      setOrders(listDemoOrders());
      setLoading(false);
      return;
    }
    try {
      const response = await apiFetch("/api/v1/orders");
      if (!response.ok) {
        const detail = (await response.text()).trim();
        throw new Error(response.status === 401 ? "Sign in to view your orders." : detail || "Could not load orders.");
      }
      const payload: unknown = await response.json();
      if (typeof payload !== "object" || payload === null || !("data" in payload) || !Array.isArray(payload.data)) {
        throw new Error("The orders service returned an invalid response.");
      }
      setOrders(payload.data as Order[]);
    } catch (reason) {
      setOrders([]);
      setError(reason instanceof Error ? reason.message : "Could not load orders.");
    } finally {
      setLoading(false);
    }
  }, []);

  async function reorder(order: Order) {
    setError("");
    setNotice("");
    try {
      for (const item of order.items ?? []) {
        await addToCart({
          id: item.variant_id ?? item.id,
          variantId: item.variant_id,
          name: item.name,
          price: item.price_minor / 100,
          image: item.image ?? "📦",
          category: "",
        }, item.quantity);
      }
      if (!DEMO_MODE) await loadCart();
      setCartOpen(true);
      setNotice("Order items were added to your cart.");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not reorder these items.");
    }
  }

  async function cancelOrder(order: Order) {
    setError("");
    setNotice("");
    if (!DEMO_MODE && !window.confirm("Cancel this order? Items go back into stock and this cannot be undone.")) return;
    try {
      if (DEMO_MODE) {
        cancelDemoOrder(order.id);
        setOrders(listDemoOrders());
        setNotice("Demo order cancelled in this browser.");
      } else {
        await apiJSON(`/api/v1/orders/${encodeURIComponent(order.id)}/cancel`, { method: "PUT" }, "Could not cancel this order.");
        await loadOrders();
        setNotice("Your order was cancelled.");
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not cancel this order.");
    }
  }

  useEffect(() => {
    if (!authorized) return;
    const frame = window.requestAnimationFrame(loadOrders);
    return () => window.cancelAnimationFrame(frame);
  }, [authorized, loadOrders]);

  const filtered = filter === "all"
    ? orders
    : orders.filter((o) => o.status === filter);

  const statusCfg = (s: string) => STATUS_CONFIG[s as OrderStatus] ?? { label: s, color: "#6b7280", bg: "#e5e7eb", steps: [] };

  const progressPct = (status: string): number => ({
    pending: 15,
    paid: 25,
    processing: 40,
    shipped: 70,
    delivered: 100,
  })[status] ?? 0;

  if (!authorized || loading) {
    return (
      <>
        <Navbar />
        <main role="status" style={{ paddingTop: "80px", minHeight: "50vh", display: "flex", alignItems: "center", justifyContent: "center" }}>
          {authorized ? <div style={{ fontSize: "3rem" }}>⏳</div> : "Redirecting to sign in…"}
        </main>
        <Footer />
      </>
    );
  }

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "90px", minHeight: "100vh" }} className="container-max">
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "2rem", flexWrap: "wrap", gap: "1rem" }}>
          <div>
            <h1 style={{ margin: 0, fontSize: "1.75rem", fontWeight: 800 }}>Your Orders</h1>
            <p style={{ margin: "0.25rem 0 0", color: "#6b7280", fontSize: "0.9rem" }}>Track orders, manage returns, and reorder</p>
          </div>
          <button type="button" onClick={() => void loadOrders()} disabled={loading} style={{ display: "flex", alignItems: "center", gap: "0.5rem", padding: "0.625rem 1rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "white", cursor: loading ? "wait" : "pointer", fontSize: "0.875rem", fontWeight: 600 }}>
            <RefreshCw size={16} /> Refresh
          </button>
        </div>

        {error && <p role="alert" style={{ color: "#991b1b", background: "#fef2f2", padding: "0.75rem", borderRadius: "var(--radius-md)" }}>{error}</p>}
        {notice && <p role="status" style={{ color: "var(--accent-emerald)", background: "#f0fdf4", padding: "0.75rem", borderRadius: "var(--radius-md)" }}>{notice}</p>}

        <div style={{ display: "flex", gap: "0.5rem", marginBottom: "1.5rem", overflowX: "auto", paddingBottom: "0.5rem" }}>
          {FILTER_OPTIONS.map((f) => {
            const cfg = statusCfg(f as OrderStatus);
            const active = filter === f;
            const count = orders.filter(o => f === "all" || o.status === f).length;
            return (
              <button
                key={f}
                onClick={() => setFilter(f)}
                style={{
                  display: "flex", alignItems: "center", gap: "0.5rem",
                  padding: "0.5rem 0.875rem", borderRadius: "999px",
                  border: `1px solid ${active ? cfg.color : "var(--border-subtle)"}`,
                  background: active ? cfg.color : "transparent",
                  color: active ? "white" : "#6b7280",
                  fontSize: "0.8rem", fontWeight: 600, cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
              >
                <span style={{ width: "6px", height: "6px", borderRadius: "50%", background: active ? "white" : cfg.color, display: "inline-block" }}></span>
                {cfg.label} <span style={{ fontSize: "0.7rem", opacity: 0.8 }}>({count})</span>
              </button>
            );
          })}
        </div>

        {error ? (
          <div style={{ textAlign: "center", padding: "3rem 1rem" }}>
            <p>We couldn&apos;t load your orders.</p>
            <button type="button" className="btn-primary" onClick={() => void loadOrders()}>Try again</button>
            {!DEMO_MODE && <p><Link href="/login">Sign in</Link> if you have not already.</p>}
          </div>
        ) : filtered.length === 0 ? (
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "4rem 1rem", textAlign: "center", color: "#6b7280" }}>
            <div style={{ fontSize: "4rem", marginBottom: "1rem" }}>📦</div>
            <h2 style={{ margin: 0, fontSize: "1.25rem", fontWeight: 600 }}>No orders yet</h2>
            <p style={{ margin: "0.25rem 0 1rem", fontSize: "0.9rem" }}>Start shopping to see your orders here.</p>
            <Link href="/products" style={{ display: "inline-flex", padding: "0.75rem 1.5rem", background: "var(--accent-primary)", color: "white", borderRadius: "var(--radius-md)", fontWeight: 600, textDecoration: "none" }}>
              Shop Now
            </Link>
          </div>
        ) : (
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(340px, 1fr))", gap: "1.5rem" }}>
            {filtered.map((o) => {
              const cfg = statusCfg(o.status);
              const pct = progressPct(o.status);
              const total = (o.total_minor || 0) / 100;
              const items = o.items || [];
              const isTrackable = ["pending", "paid", "processing", "shipped"].includes(o.status);
              const isReturnable = o.status === "delivered";
              const isReorderable = ["delivered", "cancelled"].includes(o.status);
              return (
                <div key={o.id} style={{ border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", overflow: "hidden", background: "var(--bg-card)", display: "flex", flexDirection: "column" }}>
                  <div style={{ padding: "1rem 1.25rem", borderBottom: "1px solid var(--border-subtle)", display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                    <div>
                      <div style={{ fontSize: "0.85rem", fontWeight: 700, fontFamily: "monospace", color: "#6b7280" }}>Order #{o.id.slice(0, 8).toUpperCase()}</div>
                      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "0.25rem" }}>
                        <span style={{ fontSize: "0.75rem", padding: "0.15rem 0.5rem", borderRadius: "4px", background: cfg.bg, color: cfg.color, fontWeight: 600, textTransform: "uppercase" }}>
                          {cfg.label}
                        </span>
                        {o.tracking_number && (
                          <span style={{ fontSize: "0.75rem", color: "#6b7280", fontFamily: "monospace" }}>#{o.tracking_number.slice(-6)}</span>
                        )}
                      </div>
                    </div>
                    <div style={{ textAlign: "right" }}>
                      <div style={{ fontWeight: 800, fontSize: "1.1rem" }}>₹{fmtNum(total)}</div>
                      <div style={{ fontSize: "0.75rem", color: "#6b7280" }}>{new Date(o.created_at).toLocaleDateString("en-IN")}</div>
                    </div>
                  </div>
                  <div style={{ padding: "0.75rem 1.25rem", borderBottom: "1px solid var(--border-subtle)" }}>
                    <div style={{ display: "flex", alignItems: "center", gap: "0.375rem" }}>
                      <div style={{ width: "100%", height: "4px", background: "#e5e7eb", borderRadius: "2px", overflow: "hidden" }}>
                        <div style={{ width: `${pct}%`, height: "100%", background: cfg.color }}></div>
                      </div>
                      <span style={{ fontSize: "0.7rem", color: cfg.color, fontWeight: 600, minWidth: "100px", textAlign: "right" }}>{cfg.label}</span>
                    </div>
                  </div>
                  <div style={{ padding: "1rem 1.25rem", flex: 1, overflow: "auto" }}>
                    <div style={{ display: "flex", flexDirection: "column", gap: "0.625rem" }}>
                      {items.slice(0, 4).map((item) => (
                        <div key={item.id} style={{ display: "flex", gap: "0.75rem", alignItems: "center" }}>
                          <div aria-hidden="true" style={{ width: "48px", height: "48px", background: "var(--bg-secondary)", backgroundImage: item.image?.startsWith("https://") || item.image?.startsWith("/") ? `url(${JSON.stringify(item.image)})` : undefined, backgroundSize: "cover", backgroundPosition: "center", borderRadius: "6px", display: "flex", alignItems: "center", justifyContent: "center", fontSize: "1.25rem", flexShrink: 0, border: "1px solid var(--border-subtle)" }}>
                            {!item.image?.startsWith("https://") && !item.image?.startsWith("/") ? item.image || "📦" : null}
                          </div>
                          <div style={{ flex: 1, minWidth: 0 }}>
                            <p style={{ margin: 0, fontSize: "0.85rem", fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{item.name}</p>
                            <p style={{ margin: 0, fontSize: "0.7rem", color: "#6b7280" }}>Qty: {item.quantity}</p>
                          </div>
                          <span style={{ fontSize: "0.85rem", fontWeight: 700, flexShrink: 0 }}>₹{(item.price_minor / 100).toLocaleString("en-IN")}</span>
                        </div>
                      ))}
                      {items.length > 4 && (
                        <div style={{ display: "flex", justifyContent: "center", padding: "0.5rem", color: "#6b7280", fontSize: "0.75rem", fontStyle: "italic" }}>
                          +{items.length - 4} more item{items.length > 5 ? "s" : ""}
                        </div>
                      )}
                    </div>
                  </div>
                  <div style={{ padding: "1rem 1.25rem", borderTop: "1px solid var(--border-subtle)", background: "var(--bg-secondary)" }}>
                    <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(110px, 1fr))", gap: "0.625rem" }}>
                      {isTrackable && (
                        <Link href={`/track-order?id=${encodeURIComponent(o.id)}`} style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.375rem", padding: "0.625rem 0.5rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "white", cursor: "pointer", fontSize: "0.75rem", fontWeight: 600, color: "var(--text-primary)", textDecoration: "none" }}>
                          <Truck size={16} /> Track Package
                        </Link>
                      )}
                      {isReturnable && (
                        <Link href={`/returns?order=${encodeURIComponent(o.id)}`} style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.375rem", padding: "0.625rem 0.5rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "white", cursor: "pointer", fontSize: "0.75rem", fontWeight: 600, color: "var(--text-primary)", textDecoration: "none" }}>
                          <RotateCcw size={16} style={{ color: "#dc2626" }} /> Return Items
                        </Link>
                      )}
                      {isReorderable && (
                        <button type="button" onClick={() => void reorder(o)} style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.375rem", padding: "0.625rem 0.5rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "white", cursor: "pointer", fontSize: "0.75rem", fontWeight: 600, color: "var(--text-primary)" }}>
                          <Package size={16} /> Re-order
                        </button>
                      )}
                      {!DEMO_MODE && o.payment_method === "card" && ["pending", "payment_failed"].includes(o.status) && (
                    <Link href={`/pay?order=${o.id}`} className="btn-primary" style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: "0.75rem", textDecoration: "none", fontSize: "0.85rem", fontWeight: 600 }}>
                      Pay now
                    </Link>
                  )}
                  {(DEMO_MODE ? ["pending", "processing"] : ["pending", "payment_failed"]).includes(o.status) && (
                        <button type="button" onClick={() => void cancelOrder(o)} style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.375rem", padding: "0.625rem 0.5rem", border: "1px solid #dc2626", borderRadius: "var(--radius-md)", background: "white", cursor: "pointer", fontSize: "0.75rem", fontWeight: 600, color: "#dc2626" }}>
                          <XCircle size={16} style={{ color: "#dc2626" }} /> Cancel Order
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </main>
      <Footer />
    </>
  );
}