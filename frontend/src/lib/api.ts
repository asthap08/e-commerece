export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

/** Fetch against the API with the session cookie. Responses are never HTTP-cached by the browser. */
export function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  return fetch(`${API_BASE_URL}${path}`, { cache: "no-store", ...init, headers, credentials: "include" });
}

/** Extracts the {"error": "..."} message the API returns (or plain text from proxies). */
export async function readError(response: Response, fallback: string): Promise<string> {
  const text = (await response.text().catch(() => "")).trim();
  if (!text) return fallback;
  try {
    const parsed = JSON.parse(text) as { error?: unknown };
    if (typeof parsed.error === "string" && parsed.error) return parsed.error;
  } catch {
    /* not JSON */
  }
  return text.length <= 200 ? text : fallback;
}

/** JSON request that throws ApiError with a human-readable message. GETs retry once on network failure only. */
export async function apiJSON<T>(path: string, init: RequestInit = {}, fallback = "Something went wrong. Please try again."): Promise<T> {
  let response: Response;
  try {
    response = await apiFetch(path, init);
  } catch {
    if ((init.method ?? "GET") !== "GET") throw new ApiError("Cannot reach the server. Check your connection and try again.", 0);
    try {
      response = await apiFetch(path, init);
    } catch {
      throw new ApiError("Cannot reach the server. Check your connection and try again.", 0);
    }
  }
  if (!response.ok) throw new ApiError(await readError(response, fallback), response.status);
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

export const fmtNum = (n: number) => n.toLocaleString("en-IN", { minimumFractionDigits: Number.isInteger(n) ? 0 : 2, maximumFractionDigits: 2 });
export const inr = (rupees: number) => `₹${fmtNum(rupees)}`;
export const inrMinor = (minor: number) => inr(minor / 100);
