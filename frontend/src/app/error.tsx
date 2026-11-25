"use client";

import { useEffect } from "react";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <main style={{ minHeight: "60vh", display: "grid", placeItems: "center", padding: "2rem", textAlign: "center" }}>
      <div>
        <h1 style={{ fontSize: "1.6rem" }}>Something went wrong</h1>
        <p style={{ color: "var(--text-secondary)" }}>An unexpected error occurred. Your cart and orders are safe.</p>
        <div style={{ display: "flex", gap: 12, justifyContent: "center" }}>
          <button className="btn-primary" onClick={reset} style={{ padding: "0.7rem 1.4rem" }}>Try again</button>
          {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
          <a href="/" className="btn-ghost" style={{ padding: "0.7rem 1.4rem", textDecoration: "none" }}>Go home</a>
        </div>
      </div>
    </main>
  );
}
