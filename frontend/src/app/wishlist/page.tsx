"use client";
import { fmtNum } from "@/lib/api";
import { useStore } from "@/lib/store";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { Heart, ShoppingCart, Trash2 } from "lucide-react";
import Link from "next/link";

export default function WishlistPage() {
  const { wishlist, toggleWishlist, addToCart, setCartOpen } = useStore();

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "72px", minHeight: "100vh" }}>
        <div style={{ padding: "3rem 0 2rem", background: "var(--bg-secondary)", borderBottom: "1px solid var(--border-subtle)" }}>
          <div className="container-max">
            <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
              <Heart size={28} color="#f43f5e" fill="#f43f5e" />
              <h1 style={{ margin: 0, fontSize: "2.25rem" }}>My Wishlist</h1>
            </div>
            <p style={{ margin: "0.5rem 0 0", color: "var(--text-muted)" }}>{wishlist.length} saved items</p>
          </div>
        </div>

        <div className="container-max" style={{ padding: "3rem 1.5rem" }}>
          {wishlist.length === 0 ? (
            <div style={{ textAlign: "center", padding: "5rem 1rem" }}>
              <div style={{ fontSize: "5rem", marginBottom: "1.5rem" }}>💔</div>
              <h2 style={{ marginBottom: "0.75rem" }}>Your wishlist is empty</h2>
              <p style={{ color: "var(--text-muted)", marginBottom: "2rem" }}>
                Start adding items you love!
              </p>
              <Link href="/products" style={{ textDecoration: "none" }}>
                <button className="btn-primary" style={{ padding: "0.875rem 2rem" }}>
                  Explore Products
                </button>
              </Link>
            </div>
          ) : (
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: "1.25rem" }}>
              {wishlist.map((item, i) => (
                <div
                  key={item.id}
                  className="glass-card animate-fade-up"
                  style={{ padding: "1.25rem", animationDelay: `${i * 80}ms` }}
                >
                  <div style={{
                    height: "140px", background: "var(--bg-card)",
                    borderRadius: "var(--radius-md)",
                    display: "flex", alignItems: "center", justifyContent: "center",
                    fontSize: "3.5rem", marginBottom: "1rem",
                    border: "1px solid var(--border-subtle)",
                    backgroundImage: item.image.startsWith("http") || item.image.startsWith("/") ? `url("${item.image}")` : undefined,
                    backgroundPosition: "center",
                    backgroundSize: "cover",
                  }}>
                    {item.image.startsWith("http") || item.image.startsWith("/") ? null : item.image}
                  </div>
                  <h3 style={{ margin: "0 0 0.375rem", fontSize: "0.95rem", fontWeight: 600 }}>{item.name}</h3>
                  <p className="price-tag" style={{ marginBottom: "1rem" }}>₹{fmtNum(item.price)}</p>
                  <div style={{ display: "flex", gap: "0.5rem" }}>
                    <button className="btn-primary" style={{ flex: 1, padding: "0.625rem", fontSize: "0.85rem", display: "flex", alignItems: "center", justifyContent: "center", gap: "0.375rem" }}
                      onClick={() => void addToCart({ id: item.id, variantId: item.variantId, name: item.name, price: item.price, image: item.image, category: "" }).then(() => setCartOpen(true)).catch(() => setCartOpen(true))}>
                      <ShoppingCart size={15} /> Add to Cart
                    </button>
                    <button className="btn-ghost" style={{ padding: "0.625rem", color: "#f43f5e", border: "1px solid rgba(244,63,94,0.3)", borderRadius: "var(--radius-md)" }}
                      onClick={() => toggleWishlist(item)}>
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </main>
      <Footer />
    </>
  );
}
