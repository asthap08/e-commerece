"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import AuthPageShell from "@/components/AuthPageShell";
import { authenticate } from "@/lib/auth";
import { DEMO_MODE, saveDemoProfile } from "@/lib/demo";

export default function SignupPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [isError, setIsError] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    setIsError(false);
    if (password !== confirmPassword) {
      setIsError(true);
      setMessage("Passwords do not match.");
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim().toLowerCase())) {
      setIsError(true);
      setMessage("Please enter a valid email address.");
      return;
    }
    if (password.length < 8) {
      setIsError(true);
      setMessage("Password must be at least 8 characters.");
      return;
    }
    if (!/[A-Z]/.test(password) || !/[a-z]/.test(password) || !/[0-9]/.test(password) || !/[^A-Za-z0-9]/.test(password)) {
      setIsError(true);
      setMessage("Password must contain uppercase, lowercase, number, and special character.");
      return;
    }

    setLoading(true);
    const normalizedEmail = email.trim().toLowerCase();
    try {
      if (DEMO_MODE) {
        saveDemoProfile({ name: name.trim(), email: normalizedEmail });
        window.localStorage.setItem("Omni-demo-signed-in", "true");
        window.dispatchEvent(new Event("omni-auth-change"));
      } else {
        await authenticate("/api/v1/auth/register", {
          email: normalizedEmail,
          password,
          full_name: name.trim(),
        });
      }
      router.push("/account");
    } catch (reason) {
      setIsError(true);
      setMessage(reason instanceof Error ? reason.message : "Could not create your account. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPageShell
      title="Create your account"
      description="Join OmniRetail to keep your orders and shopping details together."
      alternateHref="/login"
      alternateText="Already have an account?"
      alternateLabel="Sign in"
    >
      <form onSubmit={submit}>
        <label className="auth-field" htmlFor="signup-name">
          Full name
          <input
            id="signup-name"
            autoComplete="name"
            required
            maxLength={100}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Your name"
          />
        </label>
        <label className="auth-field" htmlFor="signup-email">
          Email address
          <input
            id="signup-email"
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
        <label className="auth-field" htmlFor="signup-password">
          Password
          <input
            id="signup-password"
            type="password"
            autoComplete="new-password"
            minLength={8}
            maxLength={72}
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="At least 8 characters"
          />
        </label>
        <label className="auth-field" htmlFor="signup-confirm-password">
          Confirm password
          <input
            id="signup-confirm-password"
            type="password"
            autoComplete="new-password"
            minLength={8}
            maxLength={72}
            required
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            placeholder="Re-enter your password"
          />
        </label>
        {message && (
          <p className={`auth-message ${isError ? "error" : "success"}`} role={isError ? "alert" : "status"}>
            {message}
          </p>
        )}
        <button className="auth-submit" type="submit" disabled={loading}>
          {loading ? "Creating account…" : DEMO_MODE ? "Create demo account" : "Create account"}
        </button>
      </form>
    </AuthPageShell>
  );
}
