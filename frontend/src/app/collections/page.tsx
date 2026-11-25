"use client";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { FEATURED_COLLECTIONS } from "@/lib/data";
import ProductCard from "@/components/ProductCard";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { useCatalogProducts, useCategories } from "@/lib/catalog";
import { DEMO_MODE } from "@/lib/demo";

export default function CollectionsPage() {
  const { products, loading, error } = useCatalogProducts();
  const categories = useCategories();
  // Live stores: collections are the real product categories (the demo list is fictional).
  const collections = DEMO_MODE
    ? FEATURED_COLLECTIONS
    : categories.map((c) => ({ id: c.name, title: c.name.charAt(0).toUpperCase() + c.name.slice(1), description: `Browse all ${c.name} products`, color: "", emoji: "🛍️", count: c.count }));
  const newArrivals = DEMO_MODE
    ? products.filter((product) => product.badge === "New" || product.badge === "Elite").slice(0, 4)
    : products.slice(0, 4);

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "72px", minHeight: "100vh" }}>
        {/* Hero */}
        <div style={{
          padding: "4rem 0",
          background: "linear-gradient(135deg, rgba(22,101,52,0.12), rgba(250,204,21,0.12))",
          borderBottom: "1px solid var(--border-subtle)",
          position: "relative", overflow: "hidden",
        }}>
          <div className="orb" style={{ width: "500px", height: "500px", background: "rgba(22,101,52,0.16)", top: "-200px", right: "0" }} />
          <div className="container-max" style={{ position: "relative", zIndex: 1 }}>
            <span className="badge badge-cyan" style={{ display: "inline-flex", marginBottom: "1rem" }}>✨ Curated for You</span>
            <h1 style={{ margin: "0 0 0.75rem", fontSize: "2.5rem" }}>
              Shop by <span className="gradient-text">Collection</span>
            </h1>
            <p style={{ margin: 0, color: "var(--text-muted)", fontSize: "1.05rem" }}>
              Handpicked themes for every lifestyle and occasion
            </p>
          </div>
        </div>

        <div className="container-max" style={{ padding: "3rem 1.5rem" }}>
          {/* Collection Cards */}
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: "1.5rem", marginBottom: "4rem" }}>
            {collections.map((col, i) => (
              <Link href={`/collections/${col.id}`} key={col.id} style={{ textDecoration: "none" }}>
                <div
                  className="glass-card animate-fade-up"
                  style={{
                    padding: "2.5rem 2rem",
                    animationDelay: `${i * 100}ms`,
                    cursor: "pointer",
                    minHeight: "220px",
                    display: "flex", flexDirection: "column", justifyContent: "space-between",
                  }}
                >
                  <div>
                    <div style={{ fontSize: "3.5rem", marginBottom: "1.25rem" }}>{col.emoji}</div>
                    <h2 style={{ margin: "0 0 0.625rem", fontSize: "1.4rem" }}>{col.title}</h2>
                    <p style={{ margin: "0 0 1.5rem", color: "var(--text-muted)", fontSize: "0.9rem", lineHeight: "1.6" }}>
                      {col.description}
                    </p>
                  </div>
                  <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                    <span className="badge badge-purple">{col.count} products</span>
                    <span style={{ color: "var(--accent-secondary)", display: "flex", alignItems: "center", gap: "0.25rem", fontSize: "0.875rem", fontWeight: 600 }}>
                      Explore <ChevronRight size={16} />
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>

          {/* New Arrivals */}
          <div>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
              <h2 style={{ margin: 0, fontSize: "1.75rem" }}>
                New <span className="gradient-text">Arrivals</span>
              </h2>
              <Link href="/products" style={{ textDecoration: "none" }}>
                <button className="btn-secondary" style={{ fontSize: "0.875rem", display: "flex", alignItems: "center", gap: "0.375rem" }}>
                  See All <ChevronRight size={16} />
                </button>
              </Link>
            </div>
            <div className="product-grid">
              {loading ? <p role="status">Loading products…</p> : error ? <p role="alert">{error}</p> : newArrivals.map((p, i) => (
                <ProductCard key={p.id} product={p} delay={i * 80} />
              ))}
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
