"use client";

import { useMemo, useState } from "react";
import { loadStripe } from "@stripe/stripe-js";
import { Elements, PaymentElement, useElements, useStripe } from "@stripe/react-stripe-js";

export const STRIPE_PUBLISHABLE_KEY = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? "";

function PayForm({ amount, onPaid }: { amount: string; onPaid: () => void }) {
  const stripe = useStripe();
  const elements = useElements();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!stripe || !elements) return;
    setBusy(true);
    setError("");
    // Cards confirm in place; redirect-based methods (3-D Secure, banks) return to the tracking page.
    const result = await stripe.confirmPayment({
      elements,
      redirect: "if_required",
      confirmParams: { return_url: `${window.location.origin}/account/orders` },
    });
    if (result.error) {
      setError(result.error.message ?? "Your payment could not be completed.");
      setBusy(false);
      return;
    }
    onPaid();
  }

  return (
    <form onSubmit={submit} style={{ display: "grid", gap: "1.25rem" }}>
      <PaymentElement />
      {error && <p role="alert" style={{ margin: 0, color: "var(--accent-rose)" }}>{error}</p>}
      <button className="btn-primary" disabled={!stripe || busy} style={{ padding: "0.9rem" }}>{busy ? "Processing…" : `Pay ${amount}`}</button>
    </form>
  );
}

/** Card payment for an already-created order. The order is only marked paid by Stripe's webhook, never by the browser. */
export default function StripePayment({ clientSecret, amount, onPaid }: { clientSecret: string; amount: string; onPaid: () => void }) {
  const stripePromise = useMemo(() => loadStripe(STRIPE_PUBLISHABLE_KEY), []);
  return (
    <Elements stripe={stripePromise} options={{ clientSecret, appearance: { theme: "stripe" } }}>
      <PayForm amount={amount} onPaid={onPaid} />
    </Elements>
  );
}
