"use client";
import { useState } from "react";
import { useStore } from "@/lib/store";
import type { Product } from "@/lib/data";
import { DEMO_MODE } from "@/lib/demo";
import Link from "next/link";
import { Heart, ShoppingCart, Zap } from "lucide-react";
import { inr, fmtNum } from "@/lib/api";

type Props = { product: Product; delay?: number };

export default function ProductCard({ product, delay = 0 }: Props) {
  const { addToCart, toggleWishlist, isWishlisted } = useStore();
  const [added, setAdded] = useState(false);
  const [error, setError] = useState("");
  const wishlisted = isWishlisted(product.id);
  const canAdd = product.inStock && (Boolean(product.variantId) || DEMO_MODE);

  const handleAdd = async () => {
    setError("");
    try {
      await addToCart({
        id: product.id,
        variantId: product.variantId,
        name: product.name,
        price: product.price,
        originalPrice: product.originalPrice,
        image: product.image,
        category: product.category,
      });
      setAdded(true);
      setTimeout(() => setAdded(false), 1800);
    } catch (addError) {
      setError(addError instanceof Error ? addError.message : "The item could not be added.");
    }
  };

  const handleWishlist = () => {
    toggleWishlist({ id: product.id, variantId: product.variantId, name: product.name, price: product.price, image: product.image });
  };

  return (
    <div
      className="product-card animate-fade-up"
      style={{ animationDelay: `${delay}ms` }}
    >
      {/* Image Area */}
      <div style={{ position: "relative", padding: "1.25rem 1.25rem 0" }}>
        {/* Badge */}
        {product.badge && (
          <div style={{ position: "absolute", top: "0.75rem", left: "0.75rem", zIndex: 2 }}>
            <span className={`badge ${
              product.badge === "New" ? "badge-cyan" :
              product.badge === "Hot" ? "badge-rose" :
              product.badge === "Elite" ? "badge-purple" :
              "badge-amber"
            }`}>
              {product.badge === "Hot" && <Zap size={9} />}
              {product.badge}
            </span>
          </div>
        )}

        {/* Wishlist */}
        <button
          onClick={handleWishlist}
          aria-label={wishlisted ? `Remove ${product.name} from wishlist` : `Add ${product.name} to wishlist`}
          aria-pressed={wishlisted}
          style={{
            position: "absolute", top: "0.75rem", right: "0.75rem", zIndex: 2,
            background: wishlisted ? "rgba(244,63,94,0.2)" : "rgba(255,255,255,0.05)",
            border: `1px solid ${wishlisted ? "rgba(244,63,94,0.4)" : "rgba(255,255,255,0.1)"}`,
            borderRadius: "50%", width: "34px", height: "34px",
            display: "flex", alignItems: "center", justifyContent: "center",
            cursor: "pointer", transition: "all 0.3s ease",
          }}
        >
          <Heart
            size={15}
            fill={wishlisted ? "#f43f5e" : "none"}
            color={wishlisted ? "#f43f5e" : "#94a3b8"}
          />
        </button>

        {/* Emoji Image */}
        <Link href={`/products/${product.id}`} aria-label={`View ${product.name}`} style={{ display: "block", textDecoration: "none", color: "inherit" }}>
        <div style={{
          height: "140px",
          background: "linear-gradient(135deg, rgba(22,101,52,0.08), rgba(250,204,21,0.08))",
          borderRadius: "12px",
          display: "flex", alignItems: "center", justifyContent: "center",
          fontSize: "4rem",
          border: "1px solid rgba(255,255,255,0.05)",
          position: "relative", overflow: "hidden",
        }}>
          {product.image.startsWith("http") || product.image.startsWith("/")
            // eslint-disable-next-line @next/next/no-img-element
            ? <img src={product.image} alt={product.name} loading="lazy" style={{ position: "absolute", inset: 0, width: "100%", height: "100%", objectFit: "cover" }} />
            : product.image}
          {!product.inStock && (
            <div style={{
              position: "absolute", inset: 0,
              background: "rgba(0,0,0,0.6)",
              display: "flex", alignItems: "center", justifyContent: "center",
              borderRadius: "12px",
            }}>
              <span style={{ fontSize: "0.75rem", color: "#f9fafb", fontWeight: 600 }}>Out of Stock</span>
            </div>
          )}
        </div>
        </Link>
      </div>

      {/* Info */}
      <div style={{ padding: "1rem 1.25rem 1.25rem" }}>
        <p style={{ margin: "0 0 0.25rem", fontSize: "0.7rem", color: "var(--text-outline)", textTransform: "uppercase", letterSpacing: "0.06em" }}>
          {product.category}
        </p>
        <h3 className="truncate-2" style={{ margin: "0 0 0.5rem", fontSize: "0.9rem", fontWeight: 600, lineHeight: "1.4", height: "2.52rem" }}>
          <Link href={`/products/${product.id}`} style={{ color: "inherit", textDecoration: "none" }}>{product.name}</Link>
        </h3>

        {/* Rating */}
        <div style={{ display: "flex", alignItems: "center", gap: "0.375rem", marginBottom: "0.75rem", minHeight: "1.1rem" }}>
          {product.reviews > 0 ? (
            <>
              <div className="stars" style={{ fontSize: "0.75rem" }} aria-hidden="true">
                {"★".repeat(Math.round(product.rating))}{"☆".repeat(5 - Math.round(product.rating))}
              </div>
              <span style={{ fontSize: "0.75rem", fontWeight: 600, color: "var(--text-secondary)" }}>{product.rating.toFixed(1)}</span>
              <span style={{ fontSize: "0.7rem", color: "var(--text-muted)" }}>({fmtNum(product.reviews)})</span>
            </>
          ) : (
            <span style={{ fontSize: "0.7rem", color: "var(--text-muted)" }}>No reviews yet</span>
          )}
        </div>

        {/* Price */}
        <div style={{ display: "flex", alignItems: "baseline", gap: "0.5rem", marginBottom: "1rem", flexWrap: "wrap" }}>
          <span className="price-tag">{inr(product.price)}</span>
          {product.discount > 0 && (
            <>
              <span className="price-original">{inr(product.originalPrice)}</span>
              <span className="price-discount">{product.discount}% off</span>
            </>
          )}
        </div>

        {/* Add to Cart */}
        <button
          className="btn-primary"
          onClick={handleAdd}
          disabled={!canAdd}
          style={{
            width: "100%",
            padding: "0.625rem",
            fontSize: "0.85rem",
            display: "flex", alignItems: "center", justifyContent: "center", gap: "0.4rem",
            background: added
              ? "linear-gradient(135deg, #10b981, #059669)"
              : !canAdd
              ? "var(--border-subtle)"
              : undefined,
            cursor: canAdd ? "pointer" : "not-allowed",
          }}
        >
          {added ? (
            <><span>✓</span> Added!</>
          ) : (
            <><ShoppingCart size={15} /> {product.inStock ? "Add to Cart" : "Sold out"}</>
          )}
        </button>
        {error && <p role="alert" style={{ margin: "0.5rem 0 0", color: "var(--accent-rose)", fontSize: "0.75rem" }}>{error}</p>}
      </div>
    </div>
  );
}
