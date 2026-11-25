"use client";
import { useEffect, useRef, useState } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { useStore } from "@/lib/store";
import { ArrowRight, CreditCard, MapPin, Package, Check, Wallet } from "lucide-react";
import { apiFetch, fmtNum } from "@/lib/api";
import Link from "next/link";
import StripePayment, { STRIPE_PUBLISHABLE_KEY } from "@/components/StripePayment";
import { createDemoOrder, DEMO_MODE } from "@/lib/demo";

type Step = "info" | "shipping" | "payment" | "confirmed";
const CHECKOUT_KEY_STORAGE = "omniretail.checkout.attempt";

const STEPS: { id: Step; label: string; icon: React.ReactNode }[] = [
  { id: "info", label: "Your Info", icon: <Package size={16} /> },
  { id: "shipping", label: "Shipping", icon: <MapPin size={16} /> },
  { id: "payment", label: "Payment", icon: <CreditCard size={16} /> },
  { id: "confirmed", label: "Confirmed", icon: <Check size={16} /> },
];

const STEP_ORDER: Step[] = ["info", "shipping", "payment", "confirmed"];

const PAYMENT_METHODS = [
  { id: "cod", label: "Cash on Delivery", icon: <Wallet size={18} />, desc: "Pay in cash when your order is delivered", requiresFields: false },
  ...(!DEMO_MODE && STRIPE_PUBLISHABLE_KEY
    ? [{ id: "card", label: "Credit / Debit card", icon: <CreditCard size={18} />, desc: "Pay securely online with Stripe", requiresFields: false }]
    : []),
  ...(DEMO_MODE
    ? [{ id: "demo_card", label: "Demo card (simulated)", icon: <CreditCard size={18} />, desc: "Completes a fake payment for the showcase only", requiresFields: false }]
    : []),
];

const SHIPPING_OPTIONS = [
  { id: "standard", label: "Standard Delivery", time: "5-7 business days", price: "FREE", cost: 0 },
  { id: "express", label: "Express Delivery", time: "2-3 business days", price: "₹149", cost: 149 },
  { id: "same_day", label: "Same Day Delivery", time: "Today by 8 PM", price: "₹299", cost: 299 },
];

export default function CheckoutPage() {
  const { cart, cartTotal, cartDiscount, clearCart, loadCart, cartError } = useStore();
  const idempotencyKey = useRef<string | null>(null);
  const [step, setStep] = useState<Step>("info");
  const [loading, setLoading] = useState(false);
  const [cartLoading, setCartLoading] = useState(!DEMO_MODE);
  const [error, setError] = useState("");
  const [confirmedOrderId, setConfirmedOrderId] = useState("");
  const [confirmedTotal, setConfirmedTotal] = useState<number | null>(null);
  const [cardOrder, setCardOrder] = useState<{ id: string; secret: string; total: number } | null>(null);
  const { coupon, promoCode } = useStore();
  const [form, setForm] = useState({
    firstName: "", lastName: "", email: "", phone: "",
    address: "", city: "", state: "", pincode: "",
    cardNumber: "", expiry: "", cvv: "", cardName: "",
    paymentMethod: "cod",
    shippingMethod: "standard",
  });

  // Prefill contact + default saved address for signed-in shoppers (live mode).
  useEffect(() => {
    if (DEMO_MODE) return;
    let active = true;
    (async () => {
      try {
        const me = await apiFetch("/api/v1/auth/me");
        if (!me.ok || !active) return;
        const user = (await me.json()) as { email?: string; full_name?: string; phone?: string };
        const [first = "", ...rest] = (user.full_name ?? "").split(" ");
        setForm((f) => ({ ...f, email: f.email || user.email || "", firstName: f.firstName || first, lastName: f.lastName || rest.join(" "), phone: f.phone || user.phone || "" }));
        const addr = await apiFetch("/api/v1/addresses");
        if (!addr.ok || !active) return;
        const list = ((await addr.json()) as { data: { line1: string; line2?: string; city: string; state: string; pincode: string; is_default: boolean }[] }).data;
        const d = list.find((x) => x.is_default) ?? list[0];
        if (d) setForm((f) => ({ ...f, address: f.address || [d.line1, d.line2].filter(Boolean).join(", "), city: f.city || d.city, state: f.state || d.state, pincode: f.pincode || d.pincode }));
      } catch { /* prefill is a convenience only */ }
    })();
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (DEMO_MODE) return;
    void loadCart().catch(() => undefined).finally(() => setCartLoading(false));
  }, [loadCart]);

  const total = cartTotal();
  const discount = cartDiscount();
  const shippingOpt = SHIPPING_OPTIONS.find(o => o.id === form.shippingMethod);
  const shipping = shippingOpt ? shippingOpt.cost : 0;
  const taxableTotal = Math.max(0, total - discount);
  const tax = DEMO_MODE ? Math.round(taxableTotal * 0.05) : 0;
  const grand = taxableTotal + shipping + tax;

  const currentIdx = STEP_ORDER.indexOf(step);

  const update = (k: string, v: string) => {
    idempotencyKey.current = null;
    setForm((f) => ({ ...f, [k]: v }));
  };

  const next = () => {
    const idx = STEP_ORDER.indexOf(step);
    if (idx < STEP_ORDER.length - 1) {
      if (step === "payment") {
        placeOrder();
      } else {
        const requiredFields =
          step === "info"
            ? [
                ["First name", form.firstName],
                ["Last name", form.lastName],
                ["Email", form.email],
                ["Phone", form.phone],
              ]
            : [
                ["Street address", form.address],
                ["City", form.city],
                ["State", form.state],
                ["PIN code", form.pincode],
              ];
        const missing = requiredFields.find(([, value]) => !value.trim());
        if (missing) {
          setError(`${missing[0]} is required.`);
          return;
        }
        if (step === "info" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
          setError("Enter a valid email address.");
          return;
        }
        setError("");
        setStep(STEP_ORDER[idx + 1]);
      }
    }
  };

  const validateForm = () => {
    if (!form.firstName.trim() || !form.lastName.trim()) return "First and last name are required";
    if (!form.email.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) return "Valid email is required";
    if (!form.phone.trim() || form.phone.replace(/\D/g, "").length < 10) return "Valid phone number is required";
    if (!form.address.trim()) return "Street address is required";
    if (!form.city.trim()) return "City is required";
    if (!form.state.trim()) return "State is required";
    if (!form.pincode.trim() || form.pincode.replace(/\D/g, "").length < 6) return "Valid PIN code is required";
    if (form.paymentMethod === "card") {
      if (!form.cardNumber.replace(/\s/g, "") || form.cardNumber.replace(/\s/g, "").length < 16) return "Valid card number is required";
      if (!form.cardName.trim()) return "Cardholder name is required";
      if (!form.expiry.trim() || !/^\d{2}\/\d{2}$/.test(form.expiry)) return "Valid expiry date (MM/YY) is required";
      if (!form.cvv.trim() || form.cvv.length < 3) return "Valid CVV is required";
    }
    return null;
  };

  const placeOrder = async () => {
    const validationError = validateForm();
    if (validationError) {
      setError(validationError);
      return;
    }
    setLoading(true);
    setError("");

    try {
      if (DEMO_MODE) {
        const order = createDemoOrder({
          items: cart,
          customer: {
            name: `${form.firstName} ${form.lastName}`.trim(),
            email: form.email,
          },
          address: `${form.address}, ${form.city}, ${form.state} ${form.pincode}`,
          paymentMethod: form.paymentMethod,
          shippingMethod: form.shippingMethod,
          discountMinor: discount,
          shippingMinor: shipping,
          taxMinor: tax,
        });
        setConfirmedOrderId(order.id);
        setStep("confirmed");
        clearCart();
        setLoading(false);
        return;
      }

      const shippingAddress = JSON.stringify({
        firstName: form.firstName,
        lastName: form.lastName,
        address: form.address,
        city: form.city,
        state: form.state,
        pincode: form.pincode,
      });
      const checkoutBody = {
        payment_provider: form.paymentMethod === "card" ? "stripe" : "manual",
        payment_method: form.paymentMethod,
        currency: "INR",
        shipping_method: form.shippingMethod,
        shipping_address: shippingAddress,
        customer_name: form.firstName + " " + form.lastName,
        customer_email: form.email,
        customer_phone: form.phone,
        ...(coupon ? { coupon_code: coupon.code } : {}),
      };
      const serializedCheckoutBody = JSON.stringify(checkoutBody);
      const fingerprint = serializedCheckoutBody;
      if (!idempotencyKey.current) {
        const savedFingerprint = window.sessionStorage.getItem(`${CHECKOUT_KEY_STORAGE}.fingerprint`);
        const savedKey = window.sessionStorage.getItem(`${CHECKOUT_KEY_STORAGE}.key`);
        idempotencyKey.current = savedFingerprint === fingerprint && savedKey
          ? savedKey
          : crypto.randomUUID();
        window.sessionStorage.setItem(`${CHECKOUT_KEY_STORAGE}.fingerprint`, fingerprint);
        window.sessionStorage.setItem(`${CHECKOUT_KEY_STORAGE}.key`, idempotencyKey.current);
      }

      const res = await apiFetch("/api/v1/checkout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey.current,
        },
        body: serializedCheckoutBody,
      });

      const body = await res.text();
      let data: { error?: string; order_id?: string; total_minor?: number } = {};
      try {
        data = JSON.parse(body) as { error?: string; order_id?: string; total_minor?: number };
      } catch {
        if (res.ok) throw new Error("The checkout service returned an invalid response.");
      }

      if (!res.ok) {
        setError(data.error || body.trim() || "Failed to place order");
        setLoading(false);
        return;
      }
      if (!data.order_id) throw new Error("The checkout service did not return an order ID.");

      if (form.paymentMethod === "card") {
        // The order exists (stock reserved); collect the card before showing "confirmed".
        const intent = await apiFetch("/api/v1/payments/intent", { method: "POST", body: JSON.stringify({ order_id: data.order_id }) });
        const intentBody = (await intent.json().catch(() => ({}))) as { client_secret?: string; error?: string };
        if (!intent.ok || !intentBody.client_secret) {
          setError(intentBody.error || "We could not start the card payment. Your order is saved — you can retry from My Orders.");
          setLoading(false);
          return;
        }
        setCardOrder({ id: data.order_id, secret: intentBody.client_secret, total: typeof data.total_minor === "number" ? data.total_minor / 100 : grand });
        setLoading(false);
        return;
      }
      setLoading(false);
      setConfirmedOrderId(data.order_id);
      // The cart is cleared next, so keep the server-confirmed total (it used to show ₹0).
      setConfirmedTotal(typeof data.total_minor === "number" ? data.total_minor / 100 : grand);
      setStep("confirmed");
      clearCart();
      window.sessionStorage.removeItem(`${CHECKOUT_KEY_STORAGE}.fingerprint`);
      window.sessionStorage.removeItem(`${CHECKOUT_KEY_STORAGE}.key`);
    } catch (checkoutError) {
      setError(checkoutError instanceof Error ? checkoutError.message : "Checkout could not be completed. Please retry.");
      setLoading(false);
    }
  };

  const isShippingSelected = (id: string) => form.shippingMethod === id;
  const isPaymentSelected = (id: string) => form.paymentMethod === id;
  const activeMethod = PAYMENT_METHODS.find(m => m.id === form.paymentMethod);

  if (cartLoading) {
    return <main style={{ padding: "8rem 1rem", textAlign: "center" }} role="status">Loading your cart…</main>;
  }

  if (cart.length === 0 && step !== "confirmed") {
    if (cardOrder) {
    return (
      <>
        <Navbar />
        <main style={{ maxWidth: 520, margin: "0 auto", padding: "2.5rem 1.25rem 4rem" }}>
          <h1 style={{ fontSize: "1.5rem", marginTop: 0 }}>Pay for your order</h1>
          <p style={{ color: "var(--text-secondary)" }}>Order #{cardOrder.id.slice(0, 8)} · Total ₹{fmtNum(cardOrder.total)}</p>
          <div className="glass-card" style={{ padding: "1.5rem" }}>
            <StripePayment
              clientSecret={cardOrder.secret}
              amount={`₹${fmtNum(cardOrder.total)}`}
              onPaid={() => {
                setConfirmedOrderId(cardOrder.id);
                setConfirmedTotal(cardOrder.total);
                setCardOrder(null);
                setStep("confirmed");
                clearCart();
                window.sessionStorage.removeItem(`${CHECKOUT_KEY_STORAGE}.fingerprint`);
                window.sessionStorage.removeItem(`${CHECKOUT_KEY_STORAGE}.key`);
              }}
            />
          </div>
          <p style={{ fontSize: "0.8rem", color: "var(--text-muted)", marginTop: "1rem" }}>Your order is held for 2 hours. If you leave, you can pay later from <a href="/account/orders" style={{ textDecoration: "underline" }}>My Orders</a>.</p>
        </main>
        <Footer />
      </>
    );
  }

  return (
      <>
        <Navbar />
        <main style={{ paddingTop: "72px", minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center" }}>
          <div style={{ textAlign: "center" }}>
            <div style={{ fontSize: "4rem", marginBottom: "1rem" }}>🛒</div>
            <h2>{cartError || "Your cart is empty"}</h2>
            {cartError && <p role="alert" style={{ color: "var(--accent-rose)" }}>{cartError}</p>}
            <Link href="/products" style={{ textDecoration: "none" }}>
              <button className="btn-primary" style={{ marginTop: "1rem" }}>Shop Now</button>
            </Link>
            {cartError && <Link href="/login" style={{ display: "block", marginTop: "1rem" }}>Sign in to continue</Link>}
          </div>
        </main>
        <Footer />
      </>
    );
  }

  const requiresCardFields = activeMethod ? activeMethod.requiresFields : false;

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "72px", minHeight: "100vh" }}>
        {DEMO_MODE && (
          <div className="container-max" style={{ padding: "1rem 1.5rem 0" }}>
            <p role="note" style={{ margin: 0, color: "#92400e", fontSize: "0.85rem" }}>
              Showcase checkout: demo orders are stored only in this browser. The illustrative 5% demo tax is not a GST calculation.
            </p>
          </div>
        )}
        <div style={{ padding: "2rem 0", background: "var(--bg-secondary)", borderBottom: "1px solid var(--border-subtle)" }}>
          <div className="container-max">
            <h1 style={{ margin: 0, fontSize: "2rem" }}>Checkout</h1>
          </div>
        </div>

        <div className="container-max" style={{ padding: "2.5rem 1.5rem" }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "0", marginBottom: "3rem", overflowX: "auto" }}>
            {STEPS.map((s, i) => {
              const sIdx = STEP_ORDER.indexOf(s.id);
              const done = sIdx < currentIdx;
              const active = s.id === step;
              return (
                <div key={s.id} style={{ display: "flex", alignItems: "center" }}>
                  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.375rem" }}>
                    <div className={`step-indicator ${active ? "step-active" : done ? "step-done" : "step-idle"}`}>
                      {done ? <Check size={16} /> : s.icon}
                    </div>
                    <span style={{ fontSize: "0.75rem", fontWeight: 600, color: active ? "var(--accent-secondary)" : done ? "var(--accent-emerald)" : "var(--text-muted)", whiteSpace: "nowrap" }}>
                      {s.label}
                    </span>
                  </div>
                  {i < STEPS.length - 1 && (
                    <div style={{ width: "60px", height: "2px", background: done ? "var(--accent-emerald)" : "var(--border-subtle)", margin: "0 0.5rem 1.25rem", flexShrink: 0, transition: "background 0.5s ease" }} />
                  )}
                </div>
              );
            })}
          </div>

          {error && step !== "payment" && (
            <div role="alert" style={{ maxWidth: 720, margin: "0 auto 1rem", padding: "0.875rem", background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)", borderRadius: "var(--radius-md)", color: "#ef4444", fontSize: "0.85rem" }}>
              {error}
            </div>
          )}

          {step === "confirmed" ? (
            <div style={{ maxWidth: "480px", margin: "0 auto", textAlign: "center", padding: "3rem 1.5rem" }}>
              <div style={{
                width: "80px", height: "80px",
                background: "linear-gradient(135deg, var(--accent-emerald), #059669)",
                borderRadius: "50%",
                display: "flex", alignItems: "center", justifyContent: "center",
                margin: "0 auto 2rem",
                boxShadow: "0 0 40px rgba(16,185,129,0.4)",
              }}>
                <Check size={36} color="white" />
              </div>
              <h2 style={{ fontSize: "2rem", marginBottom: "0.75rem" }}>Order Placed</h2>
              <p style={{ color: "var(--text-secondary)", marginBottom: "2rem", lineHeight: "1.6" }}>
                {DEMO_MODE
                  ? "This is a simulated showcase order. No real payment was taken and no shipment will be sent."
                  : "Your order has been placed. Payment is due when your order is delivered."}
              </p>
              <div className="glass-card" style={{ padding: "1.25rem", marginBottom: "2rem", textAlign: "left" }}>
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.5rem" }}>
                  <span style={{ color: "var(--text-muted)" }}>Order ID</span>
                  <span style={{ fontFamily: "monospace", fontWeight: 700 }}>#{confirmedOrderId}</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.5rem" }}>
                  <span style={{ color: "var(--text-muted)" }}>Payment Method</span>
                  <span style={{ fontWeight: 600 }}>{activeMethod?.label || "Card"}</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span style={{ color: "var(--text-muted)" }}>Order Total</span>
                  <span className="gradient-text" style={{ fontWeight: 800 }}>₹{(confirmedTotal ?? grand).toLocaleString("en-IN")}</span>
                </div>
              </div>
              <div style={{ marginBottom: "2rem" }}>
                <p style={{ fontSize: "0.85rem", color: "var(--text-muted)" }}>
                  {DEMO_MODE
                    ? "Simulation only — this showcase does not charge a payment method."
                    : form.paymentMethod === "cod" ? "Pay cash to the delivery partner upon arrival." :
                   form.paymentMethod === "upi" ? "Complete the UPI payment via your UPI app." :
                   "Your card payment has been processed securely."}
                </p>
              </div>
              <Link href={`/track-order?id=${encodeURIComponent(confirmedOrderId)}`} style={{ display: "block", marginBottom: "1rem", color: "var(--accent-secondary)" }}>
                Track this order
              </Link>
              <Link href="/" style={{ textDecoration: "none" }}>
                <button className="btn-primary" style={{ width: "100%", padding: "0.875rem" }}>
                  Continue Shopping
                </button>
              </Link>
            </div>
          ) : (
            <div className="checkout-grid" style={{ display: "grid", gridTemplateColumns: "1fr 380px", gap: "2rem", alignItems: "start" }}>
              <div>
              <div className="glass-card" style={{ padding: "2rem" }}>
                {step === "info" && (
                  <div>
                    <h2 style={{ margin: "0 0 1.5rem", fontSize: "1.25rem" }}>Personal Information</h2>
                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
                      {([
                        { k: "firstName", label: "First Name", placeholder: "John" },
                        { k: "lastName", label: "Last Name", placeholder: "Doe" },
                      ] as const).map(({ k, label, placeholder }) => (
                        <div key={k}>
                          <label className="form-label">{label}</label>
                          <input className="input-field" placeholder={placeholder} value={form[k]} onChange={(e) => update(k, e.target.value)} />
                        </div>
                      ))}
                    </div>
                    <div style={{ marginTop: "1rem" }}>
                      <label className="form-label">Email Address</label>
                      <input className="input-field" type="email" placeholder="john@example.com" value={form.email} onChange={(e) => update("email", e.target.value)} />
                    </div>
                    <div style={{ marginTop: "1rem" }}>
                      <label className="form-label">Phone Number</label>
                      <input className="input-field" type="tel" placeholder="+91 99999 99999" value={form.phone} onChange={(e) => update("phone", e.target.value)} />
                    </div>
                  </div>
                )}

                {step === "shipping" && (
                  <div>
                    <h2 style={{ margin: "0 0 1.5rem", fontSize: "1.25rem" }}>Shipping Address</h2>
                    <div style={{ marginBottom: "1rem" }}>
                      <label className="form-label">Street Address</label>
                      <input className="input-field" placeholder="123 Main Street, Apt 4B" value={form.address} onChange={(e) => update("address", e.target.value)} />
                    </div>
                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
                      {([
                        { k: "city", label: "City", placeholder: "Mumbai" },
                        { k: "state", label: "State", placeholder: "Maharashtra" },
                      ] as const).map(({ k, label, placeholder }) => (
                        <div key={k}>
                          <label className="form-label">{label}</label>
                          <input className="input-field" placeholder={placeholder} value={form[k]} onChange={(e) => update(k, e.target.value)} />
                        </div>
                      ))}
                    </div>
                    <div style={{ marginTop: "1rem" }}>
                      <label className="form-label">PIN Code</label>
                      <input className="input-field" placeholder="400001" value={form.pincode} onChange={(e) => update("pincode", e.target.value)} />
                    </div>

                    <div style={{ marginTop: "1.5rem" }}>
                      <label className="form-label">Delivery Option</label>
                      <div style={{ display: "flex", flexDirection: "column", gap: "0.625rem", marginTop: "0.5rem" }}>
                        {SHIPPING_OPTIONS.map((opt) => (
                          <label key={opt.id} style={{
                            display: "flex", alignItems: "center", justifyContent: "space-between",
                            padding: "0.875rem 1rem",
                            border: `1px solid ${isShippingSelected(opt.id) ? "var(--accent-secondary)" : "var(--border-subtle)"}`,
                            borderRadius: "var(--radius-md)",
                            cursor: "pointer",
                            background: isShippingSelected(opt.id) ? "rgba(99,102,241,0.05)" : "var(--bg-glass)",
                            transition: "all 0.2s",
                          }} onClick={() => update("shippingMethod", opt.id)}>
                            <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
                              <input type="radio" name="shipping" value={opt.id} checked={isShippingSelected(opt.id)} readOnly style={{ accentColor: "var(--accent-primary)" }} />
                              <div>
                                <p style={{ margin: 0, fontWeight: 600, fontSize: "0.9rem" }}>{opt.label}</p>
                                <p style={{ margin: 0, fontSize: "0.75rem", color: "var(--text-muted)" }}>{opt.time}</p>
                              </div>
                            </div>
                            <span style={{ fontWeight: 700, fontSize: "0.9rem", color: opt.price === "FREE" ? "var(--accent-emerald)" : "var(--text-primary)" }}>
                              {opt.price}
                            </span>
                          </label>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {step === "payment" && (
                  <div>
                    <h2 style={{ margin: "0 0 1.5rem", fontSize: "1.25rem" }}>Payment</h2>
                    <div style={{ marginBottom: "1.5rem" }}>
                      <label className="form-label">Payment Method</label>
                      <div style={{ display: "flex", flexDirection: "column", gap: "0.625rem" }}>
                        {PAYMENT_METHODS.map((method) => (
                          <label key={method.id} style={{
                            display: "flex", alignItems: "center", justifyContent: "space-between",
                            padding: "0.875rem 1rem",
                            border: `1px solid ${isPaymentSelected(method.id) ? "var(--accent-secondary)" : "var(--border-subtle)"}`,
                            borderRadius: "var(--radius-md)",
                            cursor: "pointer",
                            background: isPaymentSelected(method.id) ? "rgba(99,102,241,0.05)" : "var(--bg-glass)",
                            transition: "all 0.2s",
                          }} onClick={() => update("paymentMethod", method.id)}>
                            <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
                              <input type="radio" name="payment" value={method.id} checked={isPaymentSelected(method.id)} readOnly style={{ accentColor: "var(--accent-primary)" }} />
                              <div>
                                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                                  {method.icon}
                                  <span style={{ fontWeight: 600, fontSize: "0.9rem" }}>{method.label}</span>
                                </div>
                                <p style={{ margin: 0, fontSize: "0.75rem", color: "var(--text-muted)" }}>{method.desc}</p>
                              </div>
                            </div>
                            <span style={{ fontWeight: 600, fontSize: "0.85rem", color: "var(--accent-secondary)" }}>
                              {form.paymentMethod === method.id ? "Selected" : "Available"}
                            </span>
                          </label>
                        ))}
                      </div>
                    </div>

                    {!requiresCardFields && (
                      <div style={{ background: "rgba(99,102,241,0.08)", border: "1px solid rgba(99,102,241,0.2)", borderRadius: "var(--radius-md)", padding: "1rem" }}>
                        <p style={{ margin: 0, fontSize: "0.8rem", color: "var(--accent-secondary)", fontWeight: 600 }}>
                          Pay in cash when your order is delivered to your doorstep.
                        </p>
                      </div>
                    )}

                    {error && (
                      <div style={{ marginTop: "1rem", padding: "0.875rem", background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)", borderRadius: "var(--radius-md)", color: "#ef4444", fontSize: "0.85rem" }}>
                        {error}
                      </div>
                    )}
                  </div>
                )}
              </div>

              <div style={{ marginTop: "2rem", display: "flex", justifyContent: currentIdx > 0 ? "space-between" : "flex-end" }}>
                {currentIdx > 0 && (
                  <button type="button" className="btn-ghost" disabled={loading} onClick={() => { setError(""); setStep(STEP_ORDER[currentIdx - 1]); }} style={{ padding: "0.875rem 1.5rem" }}>
                    ← Back
                  </button>
                )}
                <button
                  className="btn-primary"
                  onClick={next}
                  disabled={loading}
                  style={{ padding: "0.875rem 2rem", display: "flex", alignItems: "center", gap: "0.5rem", opacity: loading ? 0.7 : 1 }}
                >
                  {loading ? (
                    <span style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                      <span style={{ width: "16px", height: "16px", border: "2px solid rgba(255,255,255,0.3)", borderTopColor: "white", borderRadius: "50%", animation: "spin 0.8s linear infinite", display: "inline-block" }} />
                      Processing…
                    </span>
                  ) : step === "payment" ? (
                    <>
                      <>Place {DEMO_MODE ? "Demo Order" : "Order"} ₹{fmtNum(grand)} <ArrowRight size={18} /></>
                    </>
                  ) : (
                    <>Continue <ArrowRight size={18} /></>
                  )}
                </button>
              </div>
              </div>

              <div className="glass-card" style={{ padding: "1.5rem", position: "sticky", top: "90px" }}>
                <h3 style={{ margin: "0 0 1.25rem", fontSize: "1.1rem" }}>Order Summary</h3>
                <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem", marginBottom: "1.25rem" }}>
                  {cart.map((item) => (
                    <div key={item.id} style={{ display: "flex", gap: "0.75rem", alignItems: "center" }}>
                      <div style={{ width: "48px", height: "48px", background: "var(--bg-card)", borderRadius: "8px", display: "flex", alignItems: "center", justifyContent: "center", fontSize: "1.5rem", border: "1px solid var(--border-subtle)", flexShrink: 0 }}>
                        {item.image.startsWith("http") || item.image.startsWith("/")
                          // eslint-disable-next-line @next/next/no-img-element
                          ? <img src={item.image} alt="" style={{ width: "100%", height: "100%", objectFit: "cover", borderRadius: 6 }} />
                          : item.image}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <p style={{ margin: 0, fontSize: "0.8rem", fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{item.name}</p>
                        <p style={{ margin: 0, fontSize: "0.7rem", color: "var(--text-muted)" }}>Qty: {item.quantity}</p>
                      </div>
                      <span style={{ fontSize: "0.875rem", fontWeight: 700, flexShrink: 0 }}>₹{(item.price * item.quantity).toLocaleString()}</span>
                    </div>
                  ))}
                </div>

                <div className="section-divider" style={{ marginBottom: "1rem" }} />
                <div style={{ display: "flex", flexDirection: "column", gap: "0.625rem" }}>
                  {[
                    { label: "Subtotal", value: `₹${fmtNum(total)}` },
                    ...(discount > 0 ? [{ label: `${coupon?.code ?? promoCode ?? "Coupon"} discount`, value: `−₹${fmtNum(discount)}` }] : []),
                    { label: "Shipping", value: shipping === 0 ? "FREE" : `₹${shipping}`, color: shipping === 0 ? "var(--accent-emerald)" : undefined },
                    ...(DEMO_MODE ? [{ label: "Demo tax (illustrative 5%)", value: `₹${fmtNum(tax)}` }] : []),
                  ].map(({ label, value, color }) => (
                    <div key={label} style={{ display: "flex", justifyContent: "space-between", fontSize: "0.875rem" }}>
                      <span style={{ color: "var(--text-muted)" }}>{label}</span>
                      <span style={{ color: color || "var(--text-secondary)" }}>{value}</span>
                    </div>
                  ))}
                </div>
                <div className="section-divider" style={{ margin: "1rem 0" }} />
                <div style={{ display: "flex", justifyContent: "space-between", fontWeight: 800, fontSize: "1.1rem" }}>
                  <span>Grand Total</span>
                  <span className="gradient-text">₹{fmtNum(grand)}</span>
                </div>
              </div>
            </div>
          )}
        </div>
        <Footer />
        <style>{`
          @keyframes spin { to { transform: rotate(360deg); } }
          @media (max-width: 900px) {
            .checkout-grid { grid-template-columns: 1fr !important; }
          }
        `}</style>
      </main>
    </>
  );
}