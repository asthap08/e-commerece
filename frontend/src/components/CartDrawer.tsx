"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { X, Plus, Minus, ShoppingBag, Trash2, ArrowRight, Tag } from "lucide-react";
import { useStore } from "@/lib/store";
import { DEMO_MODE } from "@/lib/demo";
import { inr } from "@/lib/api";

const MAX_QTY = 99;

export default function CartDrawer() {
  const { cart, cartOpen, cartError, clearCartError, loadCart, setCartOpen, removeFromCart, updateQty, cartTotal, cartDiscount, coupon, promoCode, applyCoupon, removeCoupon } = useStore();
  const [promoInput, setPromoInput] = useState("");
  const [promoMessage, setPromoMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const closeRef = useRef<HTMLButtonElement>(null);
  const total = cartTotal();
  const discount = cartDiscount();
  const appliedCode = coupon?.code ?? (promoCode || "");

  useEffect(() => {
    if (!DEMO_MODE) void loadCart().catch(() => undefined);
  }, [loadCart]);

  // Escape closes, focus moves into the dialog when it opens, page scroll is locked.
  useEffect(() => {
    if (!cartOpen) return;
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setCartOpen(false);
    document.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [cartOpen, setCartOpen]);

  if (!cartOpen) return null;

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    try {
      await fn();
    } catch {
      /* the store exposes the message through cartError */
    } finally {
      setBusy(false);
    }
  };

  const submitCoupon = async () => {
    setPromoMessage("");
    try {
      await applyCoupon(promoInput);
      setPromoMessage("Coupon applied.");
      setPromoInput("");
    } catch (e) {
      setPromoMessage(e instanceof Error ? e.message : "That coupon could not be applied.");
    }
  };

  return (
    <>
      <div className="drawer-overlay" onClick={() => setCartOpen(false)} aria-hidden="true" />
      <div className="drawer animate-scale-in" role="dialog" aria-modal="true" aria-label="Shopping cart" style={{ transform: "translateX(0)", display: "flex", flexDirection: "column" }}>
        <div style={{ padding: "1.25rem 1.5rem", borderBottom: "1px solid var(--border-subtle)", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <h2 style={{ margin: 0, fontSize: "1.1rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <ShoppingBag size={18} /> Your cart <span style={{ color: "var(--text-muted)", fontWeight: 500 }}>({cart.reduce((n, i) => n + i.quantity, 0)})</span>
          </h2>
          <button ref={closeRef} onClick={() => setCartOpen(false)} aria-label="Close cart" style={{ background: "none", border: "none", cursor: "pointer", padding: "0.25rem", color: "var(--text-secondary)" }}>
            <X size={20} />
          </button>
        </div>

        {cartError && (
          <div role="alert" style={{ margin: "1rem 1.5rem 0", padding: "0.75rem", background: "#fef2f2", border: "1px solid #fecaca", color: "var(--accent-rose)", borderRadius: "var(--radius-md)", fontSize: "0.85rem", display: "flex", justifyContent: "space-between", gap: "0.5rem" }}>
            <span>{cartError}</span>
            <button onClick={clearCartError} aria-label="Dismiss message" style={{ background: "none", border: "none", cursor: "pointer", color: "inherit" }}>×</button>
          </div>
        )}

        <div style={{ flex: 1, overflowY: "auto", padding: "1rem 1.5rem" }}>
          {cart.length === 0 ? (
            <div style={{ textAlign: "center", padding: "3rem 0", color: "var(--text-secondary)" }}>
              <ShoppingBag size={40} style={{ opacity: 0.4 }} />
              <p style={{ fontWeight: 600, margin: "0.75rem 0 0.25rem" }}>Your cart is empty</p>
              <p style={{ margin: "0 0 1rem", fontSize: "0.85rem" }}>Browse the catalogue and add something you like.</p>
              <Link href="/products" onClick={() => setCartOpen(false)} className="btn-primary" style={{ display: "inline-block", padding: "0.6rem 1.25rem", textDecoration: "none" }}>Shop products</Link>
            </div>
          ) : (
            <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: "0.75rem" }}>
              {cart.map((item) => {
                const limit = Math.min(MAX_QTY, item.available ?? MAX_QTY);
                return (
                  <li key={item.id} style={{ display: "flex", gap: "0.75rem", padding: "0.75rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "var(--bg-card)" }}>
                    <div style={{ width: 56, height: 56, borderRadius: 8, background: "var(--bg-secondary)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: "1.75rem", overflow: "hidden", flexShrink: 0 }}>
                      {item.image.startsWith("http") || item.image.startsWith("/")
                        // eslint-disable-next-line @next/next/no-img-element
                        ? <img src={item.image} alt="" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                        : item.image}
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <p style={{ margin: 0, fontSize: "0.88rem", fontWeight: 600 }} className="truncate-2">{item.name}</p>
                      <p style={{ margin: "0.15rem 0 0.5rem", fontSize: "0.85rem", color: "var(--accent-primary)", fontWeight: 600 }}>{inr(item.price)}</p>
                      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                        <button disabled={busy} aria-label={`Decrease quantity of ${item.name}`} onClick={() => run(() => updateQty(item.id, item.quantity - 1))} style={qtyBtn}><Minus size={13} /></button>
                        <span aria-live="polite" style={{ minWidth: 20, textAlign: "center", fontWeight: 600 }}>{item.quantity}</span>
                        <button disabled={busy || item.quantity >= limit} aria-label={`Increase quantity of ${item.name}`} onClick={() => run(() => updateQty(item.id, item.quantity + 1))} style={qtyBtn}><Plus size={13} /></button>
                        <button disabled={busy} aria-label={`Remove ${item.name} from cart`} onClick={() => run(() => removeFromCart(item.id))} style={{ ...qtyBtn, marginLeft: "auto", color: "var(--accent-rose)" }}><Trash2 size={13} /></button>
                      </div>
                      {item.available !== undefined && item.available <= 5 && <p style={{ margin: "0.4rem 0 0", fontSize: "0.72rem", color: "var(--accent-amber)" }}>Only {item.available} left in stock</p>}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {cart.length > 0 && (
          <div style={{ padding: "1rem 1.5rem 1.5rem", borderTop: "1px solid var(--border-subtle)", background: "var(--bg-card)" }}>
            {appliedCode ? (
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "0.75rem", fontSize: "0.85rem", color: "var(--accent-emerald)" }}>
                <span style={{ display: "flex", alignItems: "center", gap: "0.35rem" }}><Tag size={14} /> {appliedCode} applied</span>
                <button onClick={() => { removeCoupon(); setPromoMessage(""); }} style={{ background: "none", border: "none", textDecoration: "underline", cursor: "pointer", color: "var(--text-secondary)", fontSize: "0.8rem" }}>Remove</button>
              </div>
            ) : (
              <div style={{ marginBottom: "0.75rem" }}>
                <div style={{ display: "flex", gap: "0.5rem" }}>
                  <input value={promoInput} onChange={(e) => setPromoInput(e.target.value)} onKeyDown={(e) => e.key === "Enter" && void submitCoupon()} aria-label="Coupon code" placeholder="Coupon code" className="input-field" style={{ flex: 1, padding: "0.5rem 0.75rem" }} />
                  <button onClick={() => void submitCoupon()} disabled={!promoInput.trim()} className="btn-secondary" style={{ padding: "0.5rem 0.9rem" }}>Apply</button>
                </div>
                {promoMessage && <p role="status" style={{ margin: "0.4rem 0 0", fontSize: "0.78rem", color: promoMessage === "Coupon applied." ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{promoMessage}</p>}
              </div>
            )}
            <div style={{ display: "grid", gap: "0.35rem", fontSize: "0.9rem", marginBottom: "0.9rem" }}>
              <Row label="Subtotal" value={inr(total)} />
              {discount > 0 && <Row label="Discount" value={`− ${inr(discount)}`} accent />}
              <Row label="Shipping" value="Calculated at checkout" muted />
              <div style={{ display: "flex", justifyContent: "space-between", fontWeight: 700, fontSize: "1.05rem", paddingTop: "0.5rem", borderTop: "1px solid var(--border-subtle)" }}>
                <span>Total</span><span>{inr(total - discount)}</span>
              </div>
            </div>
            <Link href="/checkout" onClick={() => setCartOpen(false)} className="btn-primary" style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "0.5rem", padding: "0.8rem", textDecoration: "none", width: "100%" }}>
              Proceed to checkout <ArrowRight size={16} />
            </Link>
          </div>
        )}
      </div>
    </>
  );
}

const qtyBtn: React.CSSProperties = { width: 28, height: 28, display: "inline-flex", alignItems: "center", justifyContent: "center", border: "1px solid var(--border-subtle)", background: "var(--bg-card)", borderRadius: 6, cursor: "pointer" };

function Row({ label, value, accent, muted }: { label: string; value: string; accent?: boolean; muted?: boolean }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", color: accent ? "var(--accent-emerald)" : muted ? "var(--text-muted)" : "var(--text-secondary)" }}>
      <span>{label}</span><span>{value}</span>
    </div>
  );
}
