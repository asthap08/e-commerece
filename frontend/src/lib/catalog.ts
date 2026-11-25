"use client";
import { useCallback, useEffect, useState } from "react";
import { apiJSON } from "@/lib/api";
import { DEMO_MODE } from "@/lib/demo";
import { PRODUCTS, type Product } from "@/lib/data";

export type ApiVariant = {
  id: string;
  sku: string;
  name: string;
  price_minor: number;
  compare_at_minor?: number | null;
  currency: string;
  in_stock: boolean;
  available: number;
};

export type ApiProduct = {
  id: string;
  name: string;
  slug: string;
  description: string;
  category: string;
  image_url?: string;
  images?: string[];
  rating: number;
  review_count: number;
  variants: ApiVariant[];
};

export type ProductDetail = Product & { slug: string; images: string[]; variants: ApiVariant[] };

export function mapVariantPrice(variant: ApiVariant) {
  const price = variant.price_minor / 100;
  const original = variant.compare_at_minor ? variant.compare_at_minor / 100 : price;
  return { price, originalPrice: original, discount: original > price ? Math.round(((original - price) / original) * 100) : 0 };
}

export function mapProduct(product: ApiProduct): Product | null {
  const variants = product.variants ?? [];
  const variant = variants.find((v) => v.in_stock) ?? variants[0];
  if (!variant || variant.currency !== "INR") return null;
  return {
    id: product.id,
    variantId: variant.id,
    name: product.name,
    description: product.description,
    ...mapVariantPrice(variant),
    image: product.image_url || "📦",
    category: product.category || "general",
    rating: product.rating ?? 0,
    reviews: product.review_count ?? 0,
    inStock: variant.in_stock,
    tags: [],
  };
}

export type CatalogQuery = { q?: string; category?: string; minPrice?: number; maxPrice?: number; sort?: string };
const PAGE = 24;

/** Server-side filtered, paginated catalogue. `loadMore` appends the next page. */
export function useCatalogProducts(query: CatalogQuery = {}) {
  const { q = "", category = "all", minPrice = 0, maxPrice = 0, sort = "newest" } = query;
  const [products, setProducts] = useState<Product[]>(DEMO_MODE ? PRODUCTS : []);
  const [total, setTotal] = useState(DEMO_MODE ? PRODUCTS.length : 0);
  const [loading, setLoading] = useState(!DEMO_MODE);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState("");

  const build = useCallback(
    (offset: number) => {
      const params = new URLSearchParams({ limit: String(PAGE), offset: String(offset), sort });
      if (q) params.set("q", q);
      if (category !== "all") params.set("category", category);
      if (minPrice > 0) params.set("min_price", String(minPrice));
      if (maxPrice > 0) params.set("max_price", String(maxPrice));
      return `/api/v1/products?${params.toString()}`;
    },
    [q, category, minPrice, maxPrice, sort],
  );

  useEffect(() => {
    if (DEMO_MODE) return;
    let active = true;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
    setLoading(true);
    setError("");
    apiJSON<{ data: ApiProduct[]; meta: { total: number } }>(build(0), {}, "The product catalog could not be loaded.")
      .then((payload) => {
        if (!active) return;
        setProducts(payload.data.map(mapProduct).filter((p): p is Product => p !== null));
        setTotal(payload.meta.total);
      })
      .catch((e: unknown) => active && setError(e instanceof Error ? e.message : "The product catalog could not be loaded."))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [build]);

  const loadMore = useCallback(async () => {
    if (DEMO_MODE) return;
    setLoadingMore(true);
    try {
      const payload = await apiJSON<{ data: ApiProduct[]; meta: { total: number } }>(build(products.length));
      setProducts((prev) => [...prev, ...payload.data.map(mapProduct).filter((p): p is Product => p !== null)]);
      setTotal(payload.meta.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load more products.");
    } finally {
      setLoadingMore(false);
    }
  }, [build, products.length]);

  return { products, total, loading, loadingMore, error, hasMore: !DEMO_MODE && products.length < total, loadMore };
}

export function useCategories() {
  const [categories, setCategories] = useState<{ name: string; count: number }[]>([]);
  useEffect(() => {
    if (DEMO_MODE) return;
    apiJSON<{ data: { name: string; count: number }[] }>("/api/v1/products/categories")
      .then((r) => setCategories(r.data))
      .catch(() => setCategories([]));
  }, []);
  return categories;
}

export function useProduct(idOrSlug: string) {
  const [product, setProduct] = useState<ProductDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    if (DEMO_MODE) {
      const demo = PRODUCTS.find((p) => String(p.id) === idOrSlug);
      // eslint-disable-next-line react-hooks/set-state-in-effect -- loading/reset state tied to the fetch lifecycle
      setProduct(demo ? { ...demo, slug: String(demo.id), images: [demo.image], variants: [] } : null);
      setLoading(false);
      if (!demo) setError("This product could not be found.");
      return;
    }
    apiJSON<ApiProduct>(`/api/v1/products/${encodeURIComponent(idOrSlug)}`, {}, "This product could not be found.")
      .then((p) => {
        if (!active) return;
        const base = mapProduct(p);
        if (!base) throw new Error("This product is not available.");
        setProduct({ ...base, slug: p.slug, images: p.images?.length ? p.images : p.image_url ? [p.image_url] : [], variants: p.variants });
      })
      .catch((e: unknown) => active && setError(e instanceof Error ? e.message : "This product could not be found."))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [idOrSlug]);
  return { product, loading, error };
}
