import type { CartItem } from "@/lib/store";

export const DEMO_MODE =
  process.env.NEXT_PUBLIC_DEMO_MODE === "true" ||
  (process.env.NODE_ENV !== "production" &&
    process.env.NEXT_PUBLIC_DEMO_MODE !== "false");

const ORDERS_KEY = "Omni-demo-orders";
const RETURNS_KEY = "Omni-demo-returns";
const PROFILE_KEY = "Omni-demo-profile";

export type DemoOrder = {
  id: string;
  created_at: string;
  status: string;
  payment_method: string;
  shipping_method: string;
  total_minor: number;
  subtotal_minor: number;
  shipping_minor: number;
  tax_minor: number;
  tracking_number: string;
  tracking_carrier: string;
  customer_name: string;
  customer_email: string;
  address: string;
  items: Array<{
    id: string;
    name: string;
    quantity: number;
    price_minor: number;
    image: string;
  }>;
};

export type DemoReturn = {
  id: string;
  order_id: string;
  item_name: string;
  reason: string;
  status: "requested";
  created_at: string;
};

export type DemoProfile = {
  name: string;
  email: string;
  phone?: string;
};

function readList<T>(key: string): T[] {
  if (typeof window === "undefined") return [];
  try {
    const value: unknown = JSON.parse(window.localStorage.getItem(key) ?? "[]");
    return Array.isArray(value) ? (value as T[]) : [];
  } catch {
    return [];
  }
}

export function listDemoOrders(): DemoOrder[] {
  return readList<DemoOrder>(ORDERS_KEY);
}

export function cancelDemoOrder(id: string): void {
  const orders = listDemoOrders();
  const order = orders.find((candidate) => candidate.id === id);
  if (!order) throw new Error("Demo order not found.");
  if (!["pending", "processing"].includes(order.status)) {
    throw new Error("This demo order can no longer be cancelled.");
  }
  order.status = "cancelled";
  window.localStorage.setItem(ORDERS_KEY, JSON.stringify(orders));
}

export function getDemoOrder(id: string): DemoOrder | undefined {
  return listDemoOrders().find((order) => order.id.toLowerCase() === id.toLowerCase());
}

export function createDemoOrder(input: {
  items: CartItem[];
  customer: DemoProfile;
  address: string;
  paymentMethod: string;
  shippingMethod: string;
  discountMinor: number;
  shippingMinor: number;
  taxMinor: number;
}): DemoOrder {
  const rawSubtotal = input.items.reduce(
    (sum, item) => sum + item.price * item.quantity,
    0,
  );
  const subtotal = rawSubtotal - input.discountMinor;
  const order: DemoOrder = {
    id: `DEMO-${Date.now().toString(36).toUpperCase()}`,
    created_at: new Date().toISOString(),
    status: "processing",
    payment_method: input.paymentMethod,
    shipping_method: input.shippingMethod,
    total_minor: (subtotal + input.shippingMinor + input.taxMinor) * 100,
    subtotal_minor: subtotal * 100,
    shipping_minor: input.shippingMinor * 100,
    tax_minor: input.taxMinor * 100,
    tracking_number: `DEMO${Math.random().toString(36).slice(2, 10).toUpperCase()}`,
    tracking_carrier: "Demo Express",
    customer_name: input.customer.name,
    customer_email: input.customer.email,
    address: input.address,
    items: input.items.map((item, index) => ({
      id: `${orderItemId(index)}`,
      name: item.name,
      quantity: item.quantity,
      price_minor: item.price * 100,
      image: item.image,
    })),
  };
  window.localStorage.setItem(
    ORDERS_KEY,
    JSON.stringify([order, ...listDemoOrders()]),
  );
  return order;
}

function orderItemId(index: number): string {
  return `demo-item-${index + 1}`;
}

export function listDemoReturns(): DemoReturn[] {
  return readList<DemoReturn>(RETURNS_KEY);
}

export function createDemoReturn(input: Omit<DemoReturn, "id" | "status" | "created_at">): DemoReturn {
  const request: DemoReturn = {
    ...input,
    id: `RET-${Date.now().toString(36).toUpperCase()}`,
    status: "requested",
    created_at: new Date().toISOString(),
  };
  window.localStorage.setItem(
    RETURNS_KEY,
    JSON.stringify([request, ...listDemoReturns()]),
  );
  return request;
}

export function saveDemoProfile(profile: DemoProfile): void {
  window.localStorage.setItem(PROFILE_KEY, JSON.stringify(profile));
}

export function getDemoProfile(): DemoProfile | null {
  if (typeof window === "undefined") return null;
  try {
    const value: unknown = JSON.parse(window.localStorage.getItem(PROFILE_KEY) ?? "null");
    if (isDemoProfile(value)) {
      return {
        name: value.name,
        email: value.email,
        ...(value.phone ? { phone: value.phone } : {}),
      };
    }
    return null;
  } catch {
    return null;
  }
}

function isDemoProfile(value: unknown): value is DemoProfile {
  if (typeof value !== "object" || value === null) return false;
  const profile = value as Record<string, unknown>;
  return (
    typeof profile.name === "string" &&
    typeof profile.email === "string" &&
    (profile.phone === undefined || typeof profile.phone === "string")
  );
}
