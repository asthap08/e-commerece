import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Products",
  description: "Browse our wide selection of products across all categories.",
};

export default function ProductsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
