import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Account",
  description: "OmniRetail account page.",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
