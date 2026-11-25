"use client";
import { useParams } from "next/navigation";
import { Suspense } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import ProductCard from "@/components/ProductCard";
import { FEATURED_COLLECTIONS } from "@/lib/data";
import { useCatalogProducts } from "@/lib/catalog";
import { DEMO_MODE } from "@/lib/demo";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

function CollectionDetailInner() {
  const params = useParams();
  const slug = (params?.slug as string) || "";
  const col = FEATURED_COLLECTIONS.find((c) => c.id === slug);
  const router = useRouter();
  useEffect(() => {
    // Live stores have no curated collections: a collection is a product category.
    if (!DEMO_MODE && slug) router.replace(`/products?category=${encodeURIComponent(slug)}`);
  }, [slug, router]);
  const { products, loading, error } = useCatalogProducts();
  const items = products.slice(0, 12);
  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "72px", minHeight: "100vh" }}>
        <div style={{ padding: "3rem 0", borderBottom: "1px solid var(--border-subtle)" }}>
          <div className="container-max">
            <h1 style={{ fontSize: "2rem", fontWeight: 800 }}>{col?.title || slug}</h1>
            <p style={{ color: "var(--text-muted)" }}>{col?.description}</p>
          </div>
        </div>
        <div className="container-max" style={{ padding: "2rem 1.5rem" }}>
          {loading ? <p role="status">Loading products…</p> : error ? <p role="alert">{error}</p> : (
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))", gap: "1.5rem" }}>
              {items.map((p) => <ProductCard key={p.id} product={p} />)}
            </div>
          )}
        </div>
      </main>
      <Footer />
    </>
  );
}

export default function CollectionDetailPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <CollectionDetailInner />
    </Suspense>
  );
}
