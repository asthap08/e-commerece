"use client";
import { Suspense, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import ProductReviews from "@/components/ProductReviews";
import { useStore } from "@/lib/store";
import { useProduct, mapVariantPrice } from "@/lib/catalog";
import { inr } from "@/lib/api";

function ProductDetailView() {
  const { id } = useParams<{ id: string }>();
  const { product, loading, error } = useProduct(id);
  const { addToCart, setCartOpen, addToRecentlyViewed, toggleWishlist, isWishlisted } = useStore();
  const [variantId, setVariantId] = useState<string>("");
  const [imgIdx, setImgIdx] = useState(0);
  const [qty, setQty] = useState(1);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!product) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
    setVariantId((cur) => cur || product.variantId || "");
    addToRecentlyViewed({ id: product.id, name: product.name, price: product.price, image: product.image });
  }, [product, addToRecentlyViewed]);

  const variant = useMemo(() => product?.variants.find((v) => v.id === variantId), [product, variantId]);
  const pricing = variant ? mapVariantPrice(variant) : product ? { price: product.price, originalPrice: product.originalPrice, discount: product.discount } : null;
  const available = variant ? variant.available : product?.inStock ? 99 : 0;
  const maxQty = Math.max(1, Math.min(10, available));

  const add = async () => {
    if (!product) return;
    setBusy(true); setMsg(null);
    try {
      await addToCart({ id: variantId || product.id, variantId: variantId || product.variantId, name: product.name + (variant?.name ? ` – ${variant.name}` : ""), price: pricing!.price, originalPrice: pricing!.originalPrice, image: product.image, category: product.category }, qty);
      setMsg({ ok: true, text: "Added to your cart." });
      setCartOpen(true);
    } catch (e) {
      setMsg({ ok: false, text: e instanceof Error ? e.message : "Could not add to cart." });
    } finally { setBusy(false); }
  };

  const images = product?.images ?? [];
  const hero = images[imgIdx] ?? product?.image ?? "";

  return (
    <>
      <Navbar />
      <main style={{ maxWidth: 1100, margin: "0 auto", padding: "2rem 1.25rem 4rem" }}>
        <nav aria-label="Breadcrumb" style={{ fontSize: "0.85rem", color: "var(--text-muted)", marginBottom: "1.25rem" }}>
          <Link href="/products" style={{ color: "inherit" }}>Products</Link> / <span>{product?.category ?? "…"}</span>
        </nav>
        {loading && <p role="status">Loading product…</p>}
        {!loading && error && (
          <div role="alert" style={{ padding: "2rem", textAlign: "center" }}>
            <p style={{ fontSize: "1.1rem", fontWeight: 600 }}>{error}</p>
            <Link href="/products" className="btn-primary" style={{ display: "inline-block", padding: "0.6rem 1.25rem", textDecoration: "none" }}>Back to products</Link>
          </div>
        )}
        {product && pricing && (
          <>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: "2.5rem" }}>
              <div>
                <div style={{ aspectRatio: "1", borderRadius: "var(--radius-lg)", background: "var(--bg-secondary)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: "7rem", overflow: "hidden", border: "1px solid var(--border-subtle)" }}>
                  {hero.startsWith("http") || hero.startsWith("/")
                    // eslint-disable-next-line @next/next/no-img-element
                    ? <img src={hero} alt={product.name} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                    : hero}
                </div>
                {images.length > 1 && (
                  <div style={{ display: "flex", gap: 8, marginTop: 10 }}>
                    {images.map((src, i) => (
                      <button key={src} onClick={() => setImgIdx(i)} aria-label={`Show image ${i + 1}`} aria-pressed={i === imgIdx} style={{ width: 60, height: 60, padding: 0, borderRadius: 8, overflow: "hidden", border: `2px solid ${i === imgIdx ? "var(--accent-primary)" : "var(--border-subtle)"}`, cursor: "pointer" }}>
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img src={src} alt="" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div>
                <p style={{ margin: 0, textTransform: "uppercase", letterSpacing: "0.06em", fontSize: "0.75rem", color: "var(--text-muted)" }}>{product.category}</p>
                <h1 style={{ margin: "0.25rem 0 0.5rem", fontSize: "1.8rem" }}>{product.name}</h1>
                <p style={{ margin: "0 0 1rem", color: "var(--text-secondary)" }}>
                  {product.reviews > 0 ? <>{"★".repeat(Math.round(product.rating))} {product.rating.toFixed(1)} · <a href="#reviews-h" style={{ color: "inherit" }}>{product.reviews} reviews</a></> : "No reviews yet"}
                </p>
                <div style={{ display: "flex", alignItems: "baseline", gap: "0.75rem", marginBottom: "1rem" }}>
                  <span style={{ fontSize: "1.9rem", fontWeight: 700, color: "var(--accent-primary)" }}>{inr(pricing.price)}</span>
                  {pricing.discount > 0 && <><span className="price-original">{inr(pricing.originalPrice)}</span><span className="price-discount">{pricing.discount}% off</span></>}
                </div>
                <p style={{ lineHeight: 1.6, color: "var(--text-secondary)", whiteSpace: "pre-wrap" }}>{product.description || "No description provided."}</p>

                {product.variants.length > 1 && (
                  <fieldset style={{ border: "none", padding: 0, margin: "1.25rem 0" }}>
                    <legend style={{ fontWeight: 600, marginBottom: 8 }}>Option</legend>
                    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                      {product.variants.map((v) => (
                        <button key={v.id} onClick={() => { setVariantId(v.id); setQty(1); }} disabled={!v.in_stock} aria-pressed={v.id === variantId}
                          style={{ padding: "0.5rem 0.9rem", borderRadius: 8, cursor: v.in_stock ? "pointer" : "not-allowed", border: `2px solid ${v.id === variantId ? "var(--accent-primary)" : "var(--border-subtle)"}`, background: "var(--bg-card)", opacity: v.in_stock ? 1 : 0.5 }}>
                          {v.name || v.sku}{!v.in_stock && " (sold out)"}
                        </button>
                      ))}
                    </div>
                  </fieldset>
                )}

                <p style={{ margin: "1rem 0", fontSize: "0.9rem", color: available > 0 ? (available <= 5 ? "var(--accent-amber)" : "var(--accent-emerald)") : "var(--accent-rose)", fontWeight: 600 }}>
                  {available <= 0 ? "Out of stock" : available <= 5 ? `Only ${available} left` : "In stock"}
                </p>
                <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
                  <label style={{ display: "flex", alignItems: "center", gap: 8 }}>Qty
                    <select value={qty} onChange={(e) => setQty(Number(e.target.value))} className="input-field" style={{ width: 80 }} disabled={available <= 0}>
                      {Array.from({ length: maxQty }, (_, i) => i + 1).map((n) => <option key={n}>{n}</option>)}
                    </select>
                  </label>
                  <button className="btn-primary" onClick={add} disabled={busy || available <= 0} style={{ padding: "0.8rem 2rem" }}>{busy ? "Adding…" : "Add to cart"}</button>
                  <button className="btn-ghost" onClick={() => toggleWishlist({ id: product.id, variantId: product.variantId, name: product.name, price: product.price, image: product.image })} aria-pressed={isWishlisted(product.id)} style={{ padding: "0.8rem 1.25rem" }}>
                    {isWishlisted(product.id) ? "♥ Saved" : "♡ Save"}
                  </button>
                </div>
                {msg && <p role={msg.ok ? "status" : "alert"} style={{ marginTop: 12, color: msg.ok ? "var(--accent-emerald)" : "var(--accent-rose)" }}>{msg.text}{!msg.ok && msg.text.startsWith("Sign in") && <> <Link href="/login" style={{ fontWeight: 600 }}>Sign in</Link></>}</p>}
              </div>
            </div>
            <ProductReviews productId={String(product.id)} />
          </>
        )}
      </main>
      <Footer />
    </>
  );
}

export default function ProductDetailPage() {
  return (
    <Suspense fallback={null}>
      <ProductDetailView />
    </Suspense>
  );
}
