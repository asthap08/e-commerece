"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import AuthPageShell from "@/components/AuthPageShell";
import { authenticate } from "@/lib/auth";
import { DEMO_MODE, getDemoProfile, saveDemoProfile } from "@/lib/demo";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [isError, setIsError] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    setIsError(false);
    setLoading(true);
    const normalizedEmail = email.trim().toLowerCase();
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
      setIsError(true);
      setMessage("Please enter a valid email address.");
      setLoading(false);
      return;
    }
    if (password.length < 8) {
      setIsError(true);
      setMessage("Password must be at least 8 characters.");
      setLoading(false);
      return;
    }
    try {
      if (DEMO_MODE) {
        const savedProfile = getDemoProfile();
        saveDemoProfile({
          name: savedProfile?.email === normalizedEmail
            ? savedProfile.name
            : normalizedEmail.split("@")[0],
          email: normalizedEmail,
        });
        window.localStorage.setItem("Omni-demo-signed-in", "true");
        window.dispatchEvent(new Event("omni-auth-change"));
      } else {
        await authenticate("/api/v1/auth/login", {
          email: normalizedEmail,
          password,
        });
      }
      router.push("/account");
    } catch (reason) {
      setIsError(true);
      setMessage(reason instanceof Error ? reason.message : "Could not sign in. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPageShell
      title="Welcome back"
      description="Sign in to see your orders, manage your account, and pick up where you left off."
      alternateHref="/signup"
      alternateText="New to OmniRetail?"
      alternateLabel="Create an account"
    >
      <form onSubmit={submit}>
        <label className="auth-field" htmlFor="login-email">
          Email address
          <input
            id="login-email"
            type="email"
            autoComplete="email"
            autoCapitalize="none"
            spellCheck={false}
            required
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="you@example.com"
          />
        </label>
        <label className="auth-field" htmlFor="login-password">
          Password
          <input
            id="login-password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="Enter your password"
          />
        </label>
        {message && (
          <p className={`auth-message ${isError ? "error" : "success"}`} role={isError ? "alert" : "status"}>
            {message}
          </p>
        )}
        <button className="auth-submit" type="submit" disabled={loading}>
          {loading ? "Signing in…" : DEMO_MODE ? "Continue with demo account" : "Sign in"}
        </button>
      <p style={{ textAlign: "center", marginTop: "1rem", fontSize: "0.9rem" }}>
          <Link href="/forgot-password" style={{ color: "var(--accent-primary)" }}>Forgot your password?</Link>
        </p>
      </form>
    </AuthPageShell>
  );
}
