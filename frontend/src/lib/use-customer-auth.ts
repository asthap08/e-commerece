"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { DEMO_MODE } from "@/lib/demo";

export function useCustomerAuth(): boolean {
  const router = useRouter();
  const [authorized, setAuthorized] = useState(false);

  useEffect(() => {
    let active = true;
    const check = async () => {
      if (DEMO_MODE) {
        if (window.localStorage.getItem("Omni-demo-signed-in") === "true") {
          if (active) setAuthorized(true);
        } else if (active) {
          router.replace("/login");
        }
        return;
      }
      try {
        const response = await apiFetch("/api/v1/auth/me");
        if (active && response.ok) setAuthorized(true);
        else if (active) router.replace("/login");
      } catch {
        if (active) router.replace("/login");
      }
    };
    void check();
    return () => { active = false; };
  }, [router]);

  return authorized;
}
