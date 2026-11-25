"use client";
import { useState, useEffect } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import ProductCard from "@/components/ProductCard";
import { Flame, Clock, Tag } from "lucide-react";
import { useCatalogProducts } from "@/lib/catalog";
import { DEMO_MODE } from "@/lib/demo";

const DEALS_END_TIME = new Date("2026-12-31T23:59:59").getTime();

function useCountdown(targetTime: number) {
  const [time, setTime] = useState({ h: "00", m: "00", s: "00" });
  useEffect(() => {
    const tick = () => {
      const diff = targetTime - Date.now();
      if (diff <= 0) return;
      const h = Math.floor(diff / 3600000);
      const m = Math.floor((diff % 3600000) / 60000);
      const s = Math.floor((diff % 60000) / 1000);
      setTime({ h: String(h).padStart(2, "0"), m: String(m).padStart(2, "0"), s: String(s).padStart(2, "0") });
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [targetTime]);
  return time;
}

export default function DealsPage() {
  const time = useCountdown(DEALS_END_TIME);
  const { products, loading, error } = useCatalogProducts();

  const hotDeals = DEMO_MODE ? products.filter((p) => p.discount >= 28) : [...products].filter((p) => p.discount > 0).sort((a, b) => b.discount - a.discount);
  const flashDeals = DEMO_MODE ? products.filter((p) => p.discount >= 35) : hotDeals.filter((p) => p.discount >= 30);

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "72px", minHeight: "100vh" }}>
        {/* Hero */}
        <section style={{
          padding: "4rem 0",
          background: "linear-gradient(135deg, rgba(244,63,94,0.2), rgba(249,115,22,0.15), rgba(0,0,0,0))",
          borderBottom: "1px solid rgba(244,63,94,0.2)",
          position: "relative", overflow: "hidden",
        }}>
          <div className="orb" style={{ width: "400px", height: "400px", background: "rgba(244,63,94,0.2)", top: "-150px", left: "-100px" }} />
          <div className="container-max" style={{ position: "relative", zIndex: 1 }}>
            <div style={{ display: "flex", alignItems: "center", gap: "1rem", marginBottom: "1.5rem", flexWrap: "wrap" }}>
              <div style={{
                padding: "0.75rem",
                background: "rgba(244,63,94,0.2)",
                borderRadius: "14px", border: "1px solid rgba(244,63,94,0.4)",
              }}>
                <Flame size={28} color="#f43f5e" />
              </div>
              <div>
                <h1 style={{ margin: 0, fontSize: "2.5rem" }}>
                  Hot <span style={{ background: "linear-gradient(135deg,#f43f5e,#f97316)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent" }}>Deals</span>
                </h1>
                <p style={{ margin: 0, color: "var(--text-muted)" }}>Limited time offers — don&apos;t miss out!</p>
              </div>

              {/* Live Countdown */}
              <div style={{ marginLeft: "auto", display: DEMO_MODE ? "flex" : "none", alignItems: "center", gap: "0.75rem" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "0.375rem", color: "var(--text-muted)", fontSize: "0.85rem" }}>
                  <Clock size={16} /> Ends in
                </div>
                <div style={{ display: "flex", gap: "0.375rem" }}>
                  {[
                    { v: time.h, l: "HRS" },
                    { v: time.m, l: "MIN" },
                    { v: time.s, l: "SEC" },
                  ].map(({ v, l }) => (
                    <div key={l} style={{ display: "flex", flexDirection: "column", alignItems: "center" }}>
                      <div style={{
                        background: "var(--bg-card)", border: "1px solid rgba(244,63,94,0.4)",
                        borderRadius: "8px", padding: "0.375rem 0.75rem",
                        fontFamily: "monospace", fontSize: "1.4rem", fontWeight: 800, color: "#f43f5e",
                        minWidth: "52px", textAlign: "center",
                        boxShadow: "0 0 20px rgba(244,63,94,0.2)",
                      }}>
                        {v}
                      </div>
                      <span style={{ fontSize: "0.6rem", color: "var(--text-muted)", marginTop: "0.2rem" }}>{l}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Flash Deals */}
        <section style={{ padding: "3rem 0", background: "var(--bg-secondary)" }}>
          <div className="container-max">
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginBottom: "2rem" }}>
              <Tag size={20} color="#f59e0b" />
              <h2 style={{ margin: 0, fontSize: "1.5rem" }}>
                <span style={{ background: "linear-gradient(135deg,#f59e0b,#fbbf24)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent" }}>Flash Deals</span>
                {" "}— 35%+ Off
              </h2>
            </div>
            <div className="product-grid">
              {loading ? <p role="status">Loading deals…</p> : error ? <p role="alert">{error}</p> : flashDeals.length ? flashDeals.map((p, i) => <ProductCard key={p.id} product={p} delay={i * 60} />) : <p>No active flash deals are available.</p>}
            </div>
          </div>
        </section>

        {/* All Hot Deals */}
        <section style={{ padding: "3rem 0" }}>
          <div className="container-max">
            <h2 style={{ margin: "0 0 2rem", fontSize: "1.5rem" }}>
              All <span className="gradient-text">Hot Deals</span>
            </h2>
            <div className="product-grid">
              {loading ? <p role="status">Loading deals…</p> : error ? <p role="alert">{error}</p> : hotDeals.length ? hotDeals.map((p, i) => <ProductCard key={p.id} product={p} delay={i * 50} />) : <p>No active deals are available.</p>}
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}
