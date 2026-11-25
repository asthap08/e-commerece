import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Returns",
  description: "OmniRetail returns page.",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
