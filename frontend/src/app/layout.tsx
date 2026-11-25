import type { Metadata } from "next";
import { Inter, Plus_Jakarta_Sans } from "next/font/google";
import CartDrawer from "@/components/CartDrawer";
import DemoModeBanner from "@/components/DemoModeBanner";
import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const plusJakarta = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-plus-jakarta",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "OmniRetail — Everyday finds, thoughtfully picked",
    template: "%s | OmniRetail",
  },
  description:
    "Discover everyday essentials and thoughtful picks at OmniRetail.",
  keywords: ["ecommerce", "shopping", "marketplace", "online store", "omniretail"],
  authors: [{ name: "OmniRetail Team" }],
  openGraph: {
    type: "website",
    locale: "en_IN",
    siteName: "OmniRetail",
    title: "OmniRetail — Everyday finds, thoughtfully picked",
    description: "Discover curated products at OmniRetail.",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={`${inter.variable} ${plusJakarta.variable}`} data-scroll-behavior="smooth">
      <body id="top">
        <DemoModeBanner />
        {children}
        <CartDrawer />
      </body>
    </html>
  );
}
