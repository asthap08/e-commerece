"use client";
import { useState } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { usePathname, useRouter } from "next/navigation";
import { Package, List, RotateCcw, Truck, Heart, User, LogOut, MapPin } from "lucide-react";
import Link from "next/link";
import { logout } from "@/lib/auth";
import { DEMO_MODE } from "@/lib/demo";
import { useCustomerAuth } from "@/lib/use-customer-auth";

const SIDEBAR_SECTIONS = [
  { id: "orders", label: "Your Orders", icon: <Package size={18} />, href: "/account/orders" },
  { id: "account", label: "Account & Lists", icon: <List size={18} />, href: "/account" },
  { id: "returns", label: "Returns & Refunds", icon: <RotateCcw size={18} />, href: "/returns" },
  { id: "track", label: "Track Order", icon: <Truck size={18} />, href: "/track-order" },
  { id: "addresses", label: "Saved Addresses", icon: <MapPin size={18} />, href: "/account/addresses" },
  { id: "wishlist", label: "Wishlist", icon: <Heart size={18} />, href: "/wishlist" },
  { id: "profile", label: "Profile", icon: <User size={18} />, href: "/account/profile" },
];

const CONTENT_MAP: Record<string, React.ReactNode> = {
  orders: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Your Orders</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Track orders, manage returns, and reorder</p>
      <Link href="/account/orders" style={{ color: "var(--accent-primary)", fontWeight: 700 }}>View your orders →</Link>
    </div>
  ),
  account: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Account & Lists</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Manage your account settings and preferences</p>
    </div>
  ),
  security: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Login & Security</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Change your password, email, phone, and security settings</p>
    </div>
  ),
  payment: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Payment Settings</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Manage your saved payment methods</p>
    </div>
  ),
  addresses: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Addresses</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Manage your shipping and billing addresses</p>
    </div>
  ),
  returns: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Returns & Refunds</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Return items and track your refunds</p>
    </div>
  ),
  track: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Track Order</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Track your packages and deliveries</p>
    </div>
  ),
  wishlist: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Wishlist</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Your saved items</p>
    </div>
  ),
  profile: (
    <div style={{ padding: "2rem" }}>
      <h2 style={{ margin: "0 0 0.5rem", fontSize: "1.25rem", fontWeight: 800 }}>Profile</h2>
      <p style={{ color: "#6b7280", fontSize: "0.9rem" }}>Edit your public name, profile image, and privacy settings</p>
    </div>
  ),
};

export default function AccountPage() {
  const pathname = usePathname();
  const router = useRouter();
  const authorized = useCustomerAuth();
  const [logoutError, setLogoutError] = useState("");
  const activeSection = SIDEBAR_SECTIONS.find((section) => pathname === section.href)?.id ?? "account";

  const handleSection = async (id: string) => {
    if (id === "logout") {
      try {
        if (!DEMO_MODE) await logout();
        window.localStorage.removeItem("Omni-demo-signed-in");
        window.dispatchEvent(new Event("omni-auth-change"));
        router.push("/login");
      } catch (reason) {
        setLogoutError(reason instanceof Error ? reason.message : "Could not sign out.");
      }
      return;
    }
    const section = SIDEBAR_SECTIONS.find(s => s.id === id);
    if (section && section.href) {
      router.push(section.href);
    }
  };

  if (!authorized) {
    return (
      <>
        <Navbar />
        <main role="status" style={{ minHeight: "50vh", display: "grid", placeItems: "center" }}>Redirecting to sign in…</main>
        <Footer />
      </>
    );
  }

  return (
    <>
      <Navbar />
      <main className="account-layout" style={{ paddingTop: "90px", minHeight: "100vh", display: "flex" }}>
        {/* Left Sidebar */}
        <aside className="account-sidebar">
          <div style={{ padding: "1.5rem", borderBottom: "1px solid var(--border-subtle)" }}>
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
              <div style={{ width: "36px", height: "36px", background: "linear-gradient(135deg, #14532d, #22c55e)", borderRadius: "8px", display: "flex", alignItems: "center", justifyContent: "center" }}>
                <User size={18} color="white" />
              </div>
              <div>
                <div style={{ fontWeight: 700, fontSize: "0.95rem" }}>Customer</div>
                <div style={{ fontSize: "0.8rem", color: "#6b7280" }}>Signed-in account</div>
              </div>
            </div>
          </div>

          <nav style={{ padding: "1rem 0" }}>
            {SIDEBAR_SECTIONS.map((section) => {
              const active = activeSection === section.id;
              return (
                <button
                  key={section.id}
                  onClick={() => handleSection(section.id)}
                  style={{
                    display: "flex", alignItems: "center", gap: "0.75rem",
                    width: "100%", padding: "0.875rem 1.5rem",
                    background: active ? "rgba(99,102,241,0.08)" : "transparent",
                    color: active ? "var(--accent-primary)" : "#4b5563",
                    border: "none",
                    borderLeft: active ? "3px solid var(--accent-primary)" : "none",
                    fontSize: "0.9rem", fontWeight: 600,
                    cursor: "pointer",
                    textAlign: "left",
                    transition: "all 0.2s",
                  }}
                >
                  {section.icon}
                  <span style={{ flex: 1 }}>{section.label}</span>
                  {active && <span style={{ fontSize: "0.7rem", fontWeight: 700, color: "var(--accent-primary)" }}>•</span>}
                </button>
              );
            })}
          </nav>

          <div style={{ padding: "1.5rem", borderTop: "1px solid var(--border-subtle)" }}>
            <button onClick={() => handleSection("logout")} style={{ display: "flex", alignItems: "center", gap: "0.75rem", width: "100%", padding: "0.875rem 1.5rem", background: "transparent", border: "none", color: "#dc2626", fontSize: "0.9rem", fontWeight: 600, cursor: "pointer", textAlign: "left" }}>
              <LogOut size={18} /> Sign Out
            </button>
          </div>
        </aside>

        {/* Main Content */}
        <div className="account-main">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem" }}>
            <h1 style={{ margin: 0, fontSize: "1.5rem", fontWeight: 800 }}>{SIDEBAR_SECTIONS.find(s => s.id === activeSection)?.label || "Account"}</h1>
          </div>
          {logoutError && <p role="alert" style={{ color: "var(--accent-rose)" }}>{logoutError}</p>}
          <div style={{ background: "var(--bg-card)", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", minHeight: "400px" }}>
            {CONTENT_MAP[activeSection] || CONTENT_MAP.orders}
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}