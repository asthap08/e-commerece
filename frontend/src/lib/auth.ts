import { apiFetch, readError } from "@/lib/api";
import { useStore } from "@/lib/store";

type AuthResponse = {
  user?: { id?: unknown; role?: unknown };
};

export async function authenticate(
  path: "/api/v1/auth/login" | "/api/v1/auth/register",
  payload: Record<string, string>,
): Promise<void> {
  const response = await apiFetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await response.text();
  let data: AuthResponse = {};
  try {
    data = JSON.parse(body) as AuthResponse;
  } catch {
    if (response.ok) throw new Error("The server returned an invalid response.");
  }

  if (!response.ok) {
    throw new Error(await readError(new Response(body), `Request failed (${response.status}).`));
  }
  if (typeof data.user?.id !== "string" || typeof data.user.role !== "string") {
    throw new Error("The server did not create a session. Please try again.");
  }
  window.dispatchEvent(new Event("omni-auth-change"));
}

export async function logout(): Promise<void> {
  const response = await apiFetch("/api/v1/auth/logout", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Could not end the session."));
  }
  useStore.getState().clearCart();
  window.dispatchEvent(new Event("omni-auth-change"));
}
