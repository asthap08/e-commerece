"use client";
import { Suspense, useEffect, useMemo, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import Navbar from "@/components/Navbar";
import ProductCard from "@/components/ProductCard";
import Footer from "@/components/Footer";
import { CATEGORIES } from "@/lib/data";
import { useStore } from "@/lib/store";
import { useCatalogProducts, useCategories } from "@/lib/catalog";
import { DEMO_MODE } from "@/lib/demo";
import { Search, X } from "lucide-react";

const SORTS = [
  { value: "newest", label: "Newest" },
  { value: "price_asc", label: "Price: Low to High" },
  { value: "price_desc", label: "Price: High to Low" },
  { value: "rating", label: "Top Rated" },
];

function useDebounced<T>(value: T, ms = 350): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

function ProductsView() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const { setSearchQuery } = useStore();

  // The URL is the source of truth, so filters survive refresh/back and can be shared.
  const q = params.get("q") ?? "";
  const category = params.get("category") ?? "all";
  const sort = params.get("sort") ?? "newest";
  const urlMin = Number(params.get("min") ?? 0) || 0;
  const urlMax = Number(params.get("max") ?? 0) || 0;

  const [qInput, setQInput] = useState(q);
  const [minInput, setMinInput] = useState(urlMin ? String(urlMin) : "");
  const [maxInput, setMaxInput] = useState(urlMax ? String(urlMax) : "");
  const debouncedQ = useDebounced(qInput);

  const update = (patch: Record<string, string>) => {
    const next = new URLSearchParams(params.toString());
    Object.entries(patch).forEach(([k, v]) => (v && v !== "all" && !(k === "sort" && v === "newest") ? next.set(k, v) : next.delete(k)));
    router.replace(`${pathname}${next.toString() ? `?${next}` : ""}`, { scroll: false });
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { if (debouncedQ !== q) update({ q: debouncedQ.trim() }); }, [debouncedQ]);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
  useEffect(() => { setQInput(q); setSearchQuery(q); }, [q, setSearchQuery]);

  const live = useCatalogProducts({ q, category, sort, minPrice: urlMin, maxPrice: urlMax });
  const apiCategories = useCategories();

  const categories = DEMO_MODE
    ? CATEGORIES
    : [{ id: "all", label: "All products", icon: "" }, ...apiCategories.map((c) => ({ id: c.name, label: `${c.name[0]?.toUpperCase()}${c.name.slice(1)} (${c.count})`, icon: "" }))];

  // Demo data is in memory, so filter it here; live data is already filtered by the API.
  const shown = useMemo(() => {
    if (!DEMO_MODE) return live.products;
    const needle = q.toLowerCase();
    return live.products
      .filter((p) => (category === "all" || p.category === category) && (!needle || p.name.toLowerCase().includes(needle)) && (!urlMin || p.price >= urlMin) && (!urlMax || p.price <= urlMax))
      .sort((a, b) => (sort === "price_asc" ? a.price - b.price : sort === "price_desc" ? b.price - a.price : sort === "rating" ? b.rating - a.rating : 0));
  }, [live.products, q, category, sort, urlMin, urlMax]);

  const hasFilters = Boolean(q || category !== "all" || urlMin || urlMax);

  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 1280, margin: "0 auto", padding: "1.5rem 1.25rem 4rem" }}>
        <h1 style={{ fontSize: "1.6rem", margin: "0 0 1rem" }}>{q ? `Results for “${q}”` : "All products"}</h1>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "0.75rem", alignItems: "end", marginBottom: "1rem" }}>
          <label style={{ position: "relative", display: "block" }}>
            <span className="sr-only">Search products</span>
            <Search size={16} style={{ position: "absolute", left: 10, top: 12, color: "var(--text-muted)" }} />
            <input value={qInput} onChange={(e) => setQInput(e.target.value)} placeholder="Search products" className="input-field" style={{ width: "100%", paddingLeft: 32 }} />
          </label>
          <label>Sort
            <select value={sort} onChange={(e) => update({ sort: e.target.value })} className="input-field" style={{ width: "100%", display: "block" }}>
              {SORTS.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
            </select>
          </label>
          <div style={{ display: "flex", gap: 8, alignItems: "end" }}>
            <label style={{ flex: 1 }}>Min ₹<input inputMode="numeric" value={minInput} onChange={(e) => setMinInput(e.target.value.replace(/\D/g, ""))} className="input-field" style={{ width: "100%" }} /></label>
            <label style={{ flex: 1 }}>Max ₹<input inputMode="numeric" value={maxInput} onChange={(e) => setMaxInput(e.target.value.replace(/\D/g, ""))} className="input-field" style={{ width: "100%" }} /></label>
            <button className="btn-secondary" onClick={() => update({ min: minInput, max: maxInput })} style={{ padding: "0.6rem 1rem" }}>Apply</button>
          </div>
        </div>

        <div role="tablist" aria-label="Categories" style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: "1rem" }}>
          {categories.map((c) => (
            <button key={c.id} role="tab" aria-selected={category === c.id} onClick={() => update({ category: c.id })}
              style={{ padding: "0.4rem 0.9rem", borderRadius: 999, cursor: "pointer", border: `1px solid ${category === c.id ? "var(--accent-primary)" : "var(--border-subtle)"}`, background: category === c.id ? "var(--accent-primary)" : "var(--bg-card)", color: category === c.id ? "#fff" : "var(--text-primary)" }}>
              {c.icon} {c.label}
            </button>
          ))}
          {hasFilters && (
            <button onClick={() => { setMinInput(""); setMaxInput(""); setQInput(""); router.replace(pathname, { scroll: false }); }} style={{ background: "none", border: "none", cursor: "pointer", color: "var(--accent-rose)", display: "inline-flex", alignItems: "center", gap: 4 }}><X size={14} /> Clear filters</button>
          )}
        </div>

        <p role="status" aria-live="polite" style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>
          {live.loading ? "Loading…" : `${DEMO_MODE ? shown.length : live.total} product${(DEMO_MODE ? shown.length : live.total) === 1 ? "" : "s"}`}
        </p>

        {live.error && (
          <div role="alert" style={{ padding: "1rem", border: "1px solid #fecaca", background: "#fef2f2", color: "var(--accent-rose)", borderRadius: 10, margin: "0.75rem 0" }}>
            {live.error} <button onClick={() => router.refresh()} style={{ textDecoration: "underline", background: "none", border: "none", cursor: "pointer", color: "inherit" }}>Retry</button>
          </div>
        )}

        {!live.loading && !live.error && shown.length === 0 ? (
          <div style={{ textAlign: "center", padding: "3rem 0", color: "var(--text-secondary)" }}>
            <p style={{ fontSize: "1.1rem", fontWeight: 600 }}>No products match your filters.</p>
            <p>Try a different search or clear the filters.</p>
          </div>
        ) : (
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(230px, 1fr))", gap: "1.25rem" }}>
            {shown.map((p, i) => <ProductCard key={p.id} product={p} delay={Math.min(i, 8) * 40} />)}
          </div>
        )}

        {live.hasMore && (
          <div style={{ textAlign: "center", marginTop: "2rem" }}>
            <button className="btn-ghost" onClick={() => void live.loadMore()} disabled={live.loadingMore} style={{ padding: "0.7rem 2rem" }}>{live.loadingMore ? "Loading…" : "Load more"}</button>
          </div>
        )}
      </main>
      <Footer />
    </>
  );
}

export default function ProductsPage() {
  return (
    <Suspense fallback={null}>
      <ProductsView />
    </Suspense>
  );
}
