import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Checkout",
  description: "OmniRetail checkout page.",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
