import Link from "next/link";
import { ArrowLeft, Leaf, ShieldCheck, Truck } from "lucide-react";
import { DEMO_MODE } from "@/lib/demo";

type Props = {
  title: string;
  description: string;
  alternateHref: string;
  alternateText: string;
  alternateLabel: string;
  children: React.ReactNode;
};

export default function AuthPageShell({
  title,
  description,
  alternateHref,
  alternateText,
  alternateLabel,
  children,
}: Props) {
  return (
    <main className="auth-page">
      <section className="auth-promo" aria-label="OmniRetail">
        <Link href="/" className="auth-brand">
          <span className="auth-brand-icon"><Leaf size={22} /></span>
          <span>OmniRetail</span>
        </Link>
        <div className="auth-promo-content">
          <span className="auth-eyebrow">A BETTER WAY TO SHOP</span>
          <h1>Good finds.<br />Bright days.</h1>
          <p>Discover everyday essentials and thoughtful picks, all in one place.</p>
          <div className="auth-benefits">
            <span><ShieldCheck size={18} /> A simple, secure account</span>
            <span><Truck size={18} /> Order updates in one place</span>
          </div>
        </div>
        <p className="auth-promo-foot">Thoughtful shopping, made easy.</p>
      </section>

      <section className="auth-form-side">
        <Link href="/" className="auth-back"><ArrowLeft size={16} /> Back to store</Link>
        <div className="auth-card">
          <div className="auth-mobile-brand">
            <span className="auth-brand-icon"><Leaf size={19} /></span>
            <span>OmniRetail</span>
          </div>
          {DEMO_MODE && (
            <p className="auth-demo-note" role="note">
              Demo account only. Use sample details; nothing is sent to a live account service.
            </p>
          )}
          <h2>{title}</h2>
          <p className="auth-description">{description}</p>
          {children}
          <p className="auth-alternate">
            {alternateText} <Link href={alternateHref}>{alternateLabel}</Link>
          </p>
          <p className="auth-legal">
            By continuing, you agree to the store&apos;s terms and privacy notice.
          </p>
        </div>
      </section>
    </main>
  );
}
