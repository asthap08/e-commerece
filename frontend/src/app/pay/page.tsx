"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import StripePayment from "@/components/StripePayment";
import { apiJSON, fmtNum } from "@/lib/api";
import { useCustomerAuth } from "@/lib/use-customer-auth";

function PayView() {
  const authorized = useCustomerAuth();
  const router = useRouter();
  const orderId = useSearchParams().get("order") ?? "";
  const [state, setState] = useState<{ secret: string; total: number } | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!authorized || !orderId) return;
    apiJSON<{ client_secret: string; amount_minor?: number }>("/api/v1/payments/intent", { method: "POST", body: JSON.stringify({ order_id: orderId }) }, "This order cannot be paid right now.")
      .then((r) => apiJSON<{ total_minor: number }>(`/api/v1/orders/${orderId}`).then((o) => setState({ secret: r.client_secret, total: o.total_minor / 100 })))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "This order cannot be paid right now."));
  }, [authorized, orderId]);

  if (!authorized) return null;
  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 520, margin: "0 auto", padding: "2.5rem 1.25rem 4rem" }}>
        <h1 style={{ fontSize: "1.5rem", marginTop: 0 }}>Pay for your order</h1>
        {!orderId && <p role="alert">No order selected. <Link href="/account/orders" style={{ fontWeight: 600 }}>Go to My Orders</Link></p>}
        {error && <p role="alert" style={{ color: "var(--accent-rose)" }}>{error} <Link href="/account/orders" style={{ fontWeight: 600 }}>Back to My Orders</Link></p>}
        {state && (
          <div className="glass-card" style={{ padding: "1.5rem" }}>
            <p style={{ marginTop: 0 }}>Order #{orderId.slice(0, 8)} · Total ₹{fmtNum(state.total)}</p>
            <StripePayment clientSecret={state.secret} amount={`₹${fmtNum(state.total)}`} onPaid={() => router.push("/account/orders")} />
          </div>
        )}
      </main>
      <Footer />
    </>
  );
}

export default function PayPage() {
  return <Suspense fallback={null}><PayView /></Suspense>;
}
