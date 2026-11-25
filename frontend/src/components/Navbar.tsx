"use client";
import { useEffect, useState, useSyncExternalStore } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useStore } from "@/lib/store";
import { Search, ShoppingCart, Menu, X, ChevronDown } from "lucide-react";
import { DEMO_MODE } from "@/lib/demo";
import { apiFetch } from "@/lib/api";
import { logout } from "@/lib/auth";

export default function Navbar() {
  const router = useRouter();
  const { cartCount, cartOpen, setCartOpen, searchQuery, setSearchQuery } = useStore();
  const count = cartCount();
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [name, setName] = useState("");
  const [mobileOpen, setMobileOpen] = useState(false);
  const mounted = useSyncExternalStore(
    (onStoreChange) => useStore.persist.onFinishHydration(onStoreChange),
    () => useStore.persist.hasHydrated(),
    () => false
  );
  useEffect(() => {
    let active = true;
    const refreshAuth = async () => {
      if (DEMO_MODE) {
        if (active) { setIsLoggedIn(window.localStorage.getItem("Omni-demo-signed-in") === "true"); setName("Customer"); }
        return;
      }
      try {
        const response = await apiFetch("/api/v1/auth/me");
        if (active) setIsLoggedIn(response.ok);
        if (active && response.ok) {
          const me = (await response.json()) as { full_name?: string; email?: string };
          setName((me.full_name || me.email?.split("@")[0] || "Customer").split(" ")[0]);
        }
      } catch {
        if (active) setIsLoggedIn(false);
      }
    };
    void refreshAuth();
    window.addEventListener("storage", refreshAuth);
    window.addEventListener("omni-auth-change", refreshAuth);
    return () => {
      active = false;
      window.removeEventListener("storage", refreshAuth);
      window.removeEventListener("omni-auth-change", refreshAuth);
    };
  }, []);

  return (
    <>
      <header className="w-full sticky top-0 z-50 flex flex-col shadow-sm">
        {/* Tier 1: Main Search & Account Bar */}
        <div className="bg-primary-container text-on-primary px-3 sm:px-4 py-2.5 flex flex-wrap sm:flex-nowrap items-center justify-between gap-x-3 gap-y-2 border-b border-primary">
          <div className="flex items-center gap-6 shrink-0 mr-auto sm:mr-0">
            <Link href="/" className="flex items-center gap-1.5 focus:outline-none">
              <span className="text-secondary-container text-2xl">🌿</span>
              <span className="font-headline-lg text-headline-lg font-bold tracking-tight text-on-primary">OmniRetail</span>
            </Link>
          </div>

          {/* Omnibox Search Bar */}
          <div className="order-last basis-full sm:order-none sm:basis-auto flex-1 max-w-4xl sm:mx-2 min-w-0">
            <form
              className="flex items-center w-full h-10 rounded-lg overflow-hidden bg-surface-container-lowest focus-within:ring-2 focus-within:ring-secondary-container transition-all shadow-inner"
              onSubmit={(event) => {
                event.preventDefault();
                router.push(searchQuery.trim() ? `/products?q=${encodeURIComponent(searchQuery.trim())}` : "/products");
              }}
            >
              <div className="relative bg-surface-container-low border-r border-outline-variant h-full hidden sm:flex items-center px-3">
                <span className="font-label-sm text-label-sm font-semibold text-on-surface">Shop</span>
              </div>
              <input
                aria-label="Search products"
                className="w-full h-full px-3 text-on-surface font-body-md text-body-md placeholder:text-on-surface-variant focus:outline-none border-none ring-0 focus:ring-0 bg-transparent"
                placeholder="Search products, brands and deals"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <button className="h-full px-5 bg-secondary-container hover:bg-secondary text-primary-container flex items-center justify-center transition-colors" type="submit">
                <Search className="w-5 h-5 font-bold" />
              </button>
            </form>
          </div>

          {/* Mobile Menu Button */}
          <button
            className="lg:hidden flex items-center justify-center w-10 h-10 rounded hover:bg-primary/20 transition-colors"
            onClick={() => setMobileOpen(!mobileOpen)}
            aria-expanded={mobileOpen}
            aria-label="Toggle menu"
          >
            {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>

          {/* Trailing Action Badges */}
          <div className="flex items-center gap-1 lg:gap-3 shrink-0">
            {isLoggedIn ? (
              <Link href="/account" className="hidden sm:flex flex-col text-left px-2 py-1 rounded hover:border hover:border-outline-variant transition-all">
                <span className="font-body-xs text-body-xs text-on-primary-container">Hello, {name || "Customer"}</span>
                <div className="flex items-center">
                  <span className="font-label-md text-label-md font-bold">Account &amp; Lists</span>
                  <ChevronDown className="w-3 h-3 ml-0.5" />
                </div>
              </Link>
            ) : (
              <Link href="/login" className="hidden sm:flex flex-col text-left px-2 py-1 rounded hover:border hover:border-outline-variant transition-all">
                <span className="font-body-xs text-body-xs text-on-primary-container">Hello, Sign in</span>
                <div className="flex items-center">
                  <span className="font-label-md text-label-md font-bold">Account &amp; Lists</span>
                  <ChevronDown className="w-3 h-3 ml-0.5" />
                </div>
              </Link>
            )}
            {isLoggedIn && (
              <button
                onClick={async () => {
                  try {
                    if (DEMO_MODE) { window.localStorage.removeItem("Omni-demo-signed-in"); window.dispatchEvent(new Event("omni-auth-change")); }
                    else await logout();
                  } finally { router.push("/"); }
                }}
                className="hidden sm:flex flex-col text-left px-2 py-1 rounded hover:border hover:border-outline-variant transition-all"
              >
                <span className="font-body-xs text-body-xs text-on-primary-container">Not you?</span>
                <span className="font-label-md text-label-md font-bold">Sign out</span>
              </button>
            )}
            <Link href="/account/orders" className="hidden sm:flex flex-col text-left px-2 py-1 rounded hover:border hover:border-outline-variant transition-all">
              <span className="font-body-xs text-body-xs text-on-primary-container">Returns</span>
              <span className="font-label-md text-label-md font-bold">&amp; Orders</span>
            </Link>
            <button
              onClick={() => setCartOpen(!cartOpen)}
              aria-label={`Open cart, ${mounted ? count : 0} items`}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded hover:border hover:border-outline-variant transition-all relative"
            >
              <div className="relative flex items-center">
                <ShoppingCart className="text-secondary-container w-8 h-8" />
                {mounted && count > 0 && (
                  <span className="absolute -top-1.5 left-4 bg-secondary-container text-primary-container font-headline-sm text-headline-sm font-bold w-5 h-5 rounded-full flex items-center justify-center text-xs">
                    {count}
                  </span>
                )}
              </div>
              <span className="hidden md:inline font-headline-sm text-headline-sm font-bold pt-2">Cart ({mounted ? count : 0})</span>
            </button>
          </div>
        </div>

        {mobileOpen && (
          <nav aria-label="Mobile" className="lg:hidden bg-primary text-on-primary px-4 py-3 flex flex-col gap-3 text-body-sm">
            <Link href="/products" onClick={() => setMobileOpen(false)}>All products</Link>
            <Link href="/deals" onClick={() => setMobileOpen(false)}>Deals</Link>
            <Link href="/track-order" onClick={() => setMobileOpen(false)}>Track an order</Link>
            <Link href="/wishlist" onClick={() => setMobileOpen(false)}>Wishlist</Link>
            <Link href={isLoggedIn ? "/account" : "/login"} onClick={() => setMobileOpen(false)}>{isLoggedIn ? "My account" : "Sign in"}</Link>
          </nav>
        )}

        {/* Tier 2: Sub-navigation & Departments */}
        <div className="bg-primary text-on-primary px-4 py-1.5 flex items-center justify-between text-body-sm font-body-sm overflow-x-auto no-scrollbar">
          <div className="flex items-center gap-5 whitespace-nowrap">
            <Link href="/products" className="flex items-center gap-1.5 font-bold hover:text-secondary-fixed transition-colors">
              <Menu className="w-5 h-5" />
              <span>Shop all</span>
            </Link>
            <Link href="/deals" className="text-secondary-container font-bold border-b-2 border-secondary-container pb-0.5">Today&apos;s Deals</Link>
            <Link href="/products?sort=rating" className="text-on-primary font-medium hover:text-secondary-container transition-colors">Best Sellers</Link>
            <Link href="/products?category=electronics" className="text-on-primary font-medium hover:text-secondary-container transition-colors">Electronics</Link>
            <Link href="/products?category=home" className="text-on-primary font-medium hover:text-secondary-container transition-colors">Home &amp; Living</Link>
            <Link href="/returns" className="text-on-primary font-medium hover:text-secondary-container transition-colors">Returns</Link>
            <Link href="/collections" className="hidden lg:inline text-on-primary font-medium hover:text-secondary-container transition-colors">Collections</Link>
          </div>
          <div className="hidden md:flex items-center gap-2 pl-4 shrink-0">
            <span className="font-body-xs text-body-xs text-surface-variant font-medium">
              {DEMO_MODE ? "Demo store · Sample products and checkout" : "Thoughtful picks for everyday living"}
            </span>
          </div>
        </div>
      </header>

    </>
  );
}
