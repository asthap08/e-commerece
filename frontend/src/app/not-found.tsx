import Link from "next/link";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";

export default function NotFound() {
  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "90px", minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center" }}>
        <div style={{ textAlign: "center", padding: "2rem" }}>
          <div style={{ fontSize: "6rem", fontWeight: 800, color: "#064e3b", lineHeight: 1 }}>404</div>
          <h1 style={{ fontSize: "1.5rem", fontWeight: 700, margin: "1rem 0 0.5rem" }}>Page not found</h1>
          <p style={{ color: "#6b7280", marginBottom: "2rem" }}>
            The page you&apos;re looking for doesn&apos;t exist or has been moved.
          </p>
          <Link href="/" style={{ textDecoration: "none" }}>
            <button style={{ padding: "0.75rem 1.5rem", background: "#064e3b", color: "white", border: "none", borderRadius: "999px", fontWeight: 600, cursor: "pointer" }}>
              Go Home
            </button>
          </Link>
        </div>
      </main>
      <Footer />
    </>
  );
}
