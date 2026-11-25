"use client";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { apiFetch, readError } from "@/lib/api";
import { DEMO_MODE } from "@/lib/demo";

export type CartItem = {
  id: number | string;
  variantId?: string;
  name: string;
  price: number;
  originalPrice?: number;
  image: string;
  quantity: number;
  category: string;
  available?: number;
  productId?: string;
};

type WishlistItem = { id: number | string; variantId?: string; name: string; price: number; image: string };

type RecentlyViewedItem = { id: number | string; name: string; price: number; image: string; viewedAt: number };

type ApiCartItem = {
  variant_id: string;
  quantity: number;
  name: string;
  price_minor: number;
  image_url?: string;
  product_id?: string;
  available?: number;
};

type ApiCart = { items: ApiCartItem[] };

type StoreState = {
  cart: CartItem[];
  wishlist: WishlistItem[];
  recentlyViewed: RecentlyViewedItem[];
  cartOpen: boolean;
  cartError: string;
  searchQuery: string;
  promoCode: string;
  coupon: { code: string; type: "percentage" | "fixed"; value: number } | null;
  applyCoupon: (code: string) => Promise<void>;
  removeCoupon: () => void;
  addToCart: (item: Omit<CartItem, "quantity">, quantity?: number) => Promise<void>;
  removeFromCart: (id: number | string) => Promise<void>;
  updateQty: (id: number | string, qty: number) => Promise<void>;
  loadCart: () => Promise<void>;
  clearCart: () => void;
  clearCartError: () => void;
  toggleWishlist: (item: WishlistItem) => void;
  isWishlisted: (id: number | string) => boolean;
  setCartOpen: (open: boolean) => void;
  addToRecentlyViewed: (item: Omit<RecentlyViewedItem, "viewedAt">) => void;
  setSearchQuery: (q: string) => void;
  setPromoCode: (code: string) => void;
  cartDiscount: () => number;
  cartTotal: () => number;
  cartCount: () => number;
};

async function requestCart(path: string, init?: RequestInit): Promise<ApiCart> {
  const response = await apiFetch(path, init);
  if (!response.ok) {
    if (response.status === 401) throw new Error("Sign in before managing your cart.");
    throw new Error(await readError(response, "The cart could not be updated."));
  }
  return (await response.json()) as ApiCart;
}

function toCartItems(cart: ApiCart): CartItem[] {
  return cart.items.map((item) => ({
    id: item.variant_id,
    variantId: item.variant_id,
    name: item.name,
    price: item.price_minor / 100,
    originalPrice: item.price_minor / 100,
    image: item.image_url || "📦",
    quantity: item.quantity,
    category: "General",
    available: item.available,
    productId: item.product_id,
  }));
}

function cartErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "The cart could not be updated.";
}

export const useStore = create<StoreState>()(
  persist(
    (set, get) => ({
      cart: [],
      wishlist: [],
      recentlyViewed: [],
      cartOpen: false,
      cartError: "",
      searchQuery: "",
      promoCode: "",
      coupon: null,

      applyCoupon: async (code) => {
        const clean = code.trim().toUpperCase();
        if (!clean) throw new Error("Enter a coupon code.");
        if (DEMO_MODE) {
          if (clean !== "DEMO10") throw new Error("That coupon code is not valid.");
          set({ promoCode: clean });
          return;
        }
        const subtotalMinor = Math.round(get().cartTotal() * 100);
        const response = await apiFetch(`/api/v1/coupons/validate?code=${encodeURIComponent(clean)}&total=${subtotalMinor}`);
        if (response.status === 401) throw new Error("Sign in to use a coupon.");
        if (!response.ok) throw new Error(await readError(response, "That coupon code is not valid."));
        const data = (await response.json()) as { code: string; discount_type: "percentage" | "fixed"; value: number };
        set({ coupon: { code: data.code, type: data.discount_type, value: data.value } });
      },
      removeCoupon: () => set({ coupon: null, promoCode: "" }),

      addToCart: async (item, quantity = 1) => {
        set({ cartError: "" });
        if (!Number.isInteger(quantity) || quantity <= 0) {
          throw new Error("Quantity must be a positive whole number.");
        }
        try {
          if (item.variantId) {
            const cart = await requestCart("/api/v1/cart/items", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ variant_id: item.variantId, quantity }),
            });
            set({ cart: toCartItems(cart) });
            return;
          }
          if (!DEMO_MODE) throw new Error("This product is not connected to the live catalog.");
          set((state) => {
            const existing = state.cart.find((cartItem) => cartItem.id === item.id);
            if (existing) {
              return {
                cart: state.cart.map((cartItem) =>
                  cartItem.id === item.id ? { ...cartItem, quantity: cartItem.quantity + quantity } : cartItem
                ),
              };
            }
            return { cart: [...state.cart, { ...item, quantity }] };
          });
        } catch (error) {
          set({ cartError: cartErrorMessage(error) });
          throw error;
        }
      },

      removeFromCart: async (id) => {
        const item = get().cart.find((cartItem) => cartItem.id === id);
        if (item?.variantId) {
          set({ cartError: "" });
          try {
            const cart = await requestCart(`/api/v1/cart/items/${encodeURIComponent(item.variantId)}`, {
              method: "DELETE",
            });
            set({ cart: toCartItems(cart) });
          } catch (error) {
            set({ cartError: cartErrorMessage(error) });
            throw error;
          }
          return;
        }
        set((state) => ({ cart: state.cart.filter((cartItem) => cartItem.id !== id) }));
      },

      updateQty: async (id, qty) => {
        const item = get().cart.find((cartItem) => cartItem.id === id);
        if (item?.variantId) {
          if (qty <= 0) {
            await get().removeFromCart(id);
            return;
          }
          set({ cartError: "" });
          try {
            const cart = await requestCart(`/api/v1/cart/items/${encodeURIComponent(item.variantId)}`, {
              method: "PUT",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ quantity: qty }),
            });
            set({ cart: toCartItems(cart) });
          } catch (error) {
            set({ cartError: cartErrorMessage(error) });
            throw error;
          }
          return;
        }
        set((state) => ({
          cart:
            qty <= 0
              ? state.cart.filter((cartItem) => cartItem.id !== id)
              : state.cart.map((cartItem) =>
                  cartItem.id === id ? { ...cartItem, quantity: qty } : cartItem
                ),
        }));
      },

      loadCart: async () => {
        set({ cartError: "" });
        try {
          const cart = await requestCart("/api/v1/cart/");
          set({ cart: toCartItems(cart) });
        } catch (error) {
          set({ cart: [] });
          // Not signed in is a normal state for a visitor, not an error to display.
          if (error instanceof Error && error.message.startsWith("Sign in")) return;
          set({ cartError: cartErrorMessage(error) });
          throw error;
        }
      },

      clearCart: () => set({ cart: [], promoCode: "", coupon: null }),
      clearCartError: () => set({ cartError: "" }),

      toggleWishlist: (item) =>
        set((state) => {
          const exists = state.wishlist.some((wishlisted) => wishlisted.id === item.id);
          return {
            wishlist: exists
              ? state.wishlist.filter((wishlisted) => wishlisted.id !== item.id)
              : [...state.wishlist, item],
          };
        }),

      isWishlisted: (id) => get().wishlist.some((item) => item.id === id),

      setCartOpen: (open) => set({ cartOpen: open }),
      addToRecentlyViewed: (item) => set((state) => {
        const filtered = state.recentlyViewed.filter((rv) => rv.id !== item.id);
        return { recentlyViewed: [{ ...item, viewedAt: Date.now() }, ...filtered].slice(0, 10) };
      }),
      setSearchQuery: (q) => set({ searchQuery: q }),
      setPromoCode: (code) => set({ promoCode: code }),

      // Mirrors the server's integer maths so the preview equals what checkout charges.
      cartDiscount: () => {
        const subtotal = get().cart.reduce((sum, item) => sum + item.price * item.quantity, 0);
        const coupon = get().coupon;
        if (coupon) {
          const minor = Math.round(subtotal * 100);
          const off = coupon.type === "percentage" ? Math.floor((minor * coupon.value) / 100) : Math.min(coupon.value, minor);
          return off / 100;
        }
        return DEMO_MODE && get().promoCode === "DEMO10" ? Math.round(subtotal * 0.1) : 0;
      },

      cartTotal: () => get().cart.reduce((sum, item) => sum + item.price * item.quantity, 0),

      cartCount: () => get().cart.reduce((sum, item) => sum + item.quantity, 0),
    }),
    {
      name: "Omni-store",
      // A signed-in cart lives on the server; persisting it locally showed one
      // user's cart to the next person on a shared device.
      partialize: (state) => ({
        wishlist: state.wishlist,
        recentlyViewed: state.recentlyViewed,
        ...(DEMO_MODE ? { cart: state.cart, promoCode: state.promoCode } : {}),
      }),
    }
  )
);
