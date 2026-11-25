import { DEMO_MODE } from "@/lib/demo";
import Link from "next/link";

const footerGroups = [
  {
    title: "Explore",
    links: [
      { label: "All products", href: "/products" },
      { label: "Today's deals", href: "/deals" },
      { label: "Collections", href: "/collections" },
      { label: "Wishlist", href: "/wishlist" },
    ],
  },
  {
    title: "Your account",
    links: [
      { label: "Sign in", href: "/login" },
      { label: "Create account", href: "/signup" },
      { label: "Your orders", href: "/account/orders" },
      { label: "Track an order", href: "/track-order" },
    ],
  },
  {
    title: "Help",
    links: [
      { label: "Returns", href: "/returns" },
      { label: "Account profile", href: "/account/profile" },
      { label: "Browse products", href: "/products" },
    ],
  },
];

export default function Footer() {
  return (
    <footer className="w-full bg-primary-container text-on-primary-container border-t border-primary mt-auto">
      <a
        className="block w-full py-3.5 bg-primary text-center font-label-md text-label-md font-semibold text-on-primary hover:bg-primary-container hover:text-secondary-fixed transition-colors"
        href="#top"
      >
        Back to top
      </a>
      <div className="max-w-7xl mx-auto px-6 py-10 grid grid-cols-2 md:grid-cols-4 gap-8">
        {footerGroups.map((group) => (
          <div key={group.title}>
            <h2 className="font-headline-sm text-headline-sm font-bold text-on-primary mb-3">
              {group.title}
            </h2>
            <ul className="space-y-2 font-body-sm text-body-sm">
              {group.links.map((item) => (
                <li key={item.href}>
                  <Link className="hover:text-on-primary hover:underline" href={item.href}>
                    {item.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
        <div>
          <h2 className="font-headline-sm text-headline-sm font-bold text-on-primary mb-3">
            OmniRetail
          </h2>
          <p className="max-w-xs font-body-sm text-body-sm leading-6">
            A demo shopping experience for discovering everyday essentials and thoughtful finds.
          </p>
        </div>
      </div>
      <div className="border-t border-primary/80 py-6">
        <div className="max-w-7xl mx-auto px-6 flex flex-col sm:flex-row items-center justify-center gap-5 text-on-primary">
          <Link href="/" className="flex items-center gap-2 font-headline-md text-headline-md font-bold">
            <span className="text-secondary-container text-2xl" aria-hidden="true">🌿</span>
            OmniRetail
          </Link>
          <span className="font-label-sm text-label-sm">India · INR (₹)</span>
        </div>
      </div>
      <div className="bg-primary py-5 text-center text-body-xs text-on-primary-container">
        <p>OmniRetail{DEMO_MODE ? " · Demo storefront — sample orders are simulated." : ""}</p>
      </div>
    </footer>
  );
}
