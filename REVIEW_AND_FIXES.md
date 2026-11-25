# Review & fixes — OmniRetail commerce

Scope: Go API + PostgreSQL schema, Next.js storefront, Vite admin. Method: read all source, reproduced suspected bugs against the **unmodified** API on a real PostgreSQL 18 (16/35 probe checks passed), fixed, then re-ran (now **59/59**). Probe script: `tools/probe.py` (below).

## Critical / security (all fixed)
| # | Issue | Fix |
|---|---|---|
| 1 | **Anyone could register as admin** — `/auth/register` accepted a client `role`; DB had no role constraint | Role never read from client; `CHECK (role IN ('customer','admin'))`. **Audit your existing admins** (see README) |
| 2 | **Any signed-in user could cancel any order** (even shipped), no restock | Cancel is ownership-checked, state-checked (pending/payment_failed only), transactional, **restocks** |
| 3 | `Cache-Control: public, max-age=300` on *every* GET incl. cart/orders/auth/admin → stale carts, user still "logged in" after logout | Only anonymous catalogue reads are cacheable (60s); everything else `no-store` |
| 4 | Rate limiter never fired (keyed on `ip:port`) and permanently blocked busy clients; `Retry-After` was `"1m0s"`; `X-Forwarded-For` trusted blindly | Fixed-window per IP, numeric `Retry-After`, stricter login/register limit, `TRUST_PROXY` opt-in |
| 5 | Reviews took `user_id` from the request body (spoofable; the UI sent none → always failed) | Author from session; one review per user/product |
| 6 | Coupon endpoint public, returned full records (code enumeration) | Admin list/create; shoppers get only the computed discount |
| 7 | CSRF check read the wrong cookie name; raw DB errors returned to clients; no request-body size cap; 401/403 plain text | Fixed; JSON errors everywhere; 1 MiB cap; login timing equalised |

## Functional bugs (all fixed)
- **Product search always 500** (SELECT 9 cols, Scan 10). `min_price/max_price` were ignored.
- **Orders list/detail failed** after any checkout (nullable `tracking_*` scanned into `string`) — same bug class in **returns** and addresses.
- **Addresses API panicked on every call** (read a context key nothing sets).
- Bcrypt >72-byte passwords → 500; password regex rejected `- _ .` and spaces.
- Dashboard revenue counted only `status='paid'` → vanished once packed; COD never counted.
- Checkout stock deducted up front but **never returned** on cancel/failed payment → worker now cancels unpaid card orders after 2h and restocks.
- Late Stripe success on a cancelled order made Stripe retry forever → recorded as "REFUND REQUIRED" in audit log.
- Migrations re-ran every boot, unlocked → versioned `schema_migrations` + advisory lock (safe on existing DBs; tested).
- Redis was required but never used → optional. Dockerfile: non-root, healthcheck, worker built, `.dockerignore`.
- **Storefront:** checkout confirmation showed **₹0** (cart cleared before render); hard-coded `localhost:8080` (checkout, cart); coupons never reached the server (customer saw a discount they weren't given); `apiRequest` retried POSTs/4xx up to 3× (unused, removed); cart persisted in localStorage across users; anonymous visitors saw "Sign in…" error in the cart; raw image URLs printed as text; `0% off`/`0 ★` shown for live products; `ProductReviews` redirected guests to login and swallowed errors; no logout button, dead mobile menu, category links all `/products`; footer said "Demo" in live mode; standalone Docker build would crash (`server.js` nested) — fixed `next.config`/Dockerfile.
- **Schema:** added CHECKs (status, price, stock, qty, coupon), unique lower(email), one-default-address index, one-review-per-user, FK-supporting indexes, NULL-safe defaults.

## Features added
Storefront: **product detail page** (variants, gallery, stock, reviews) · server-side filter/sort/**pagination**, URL-driven & debounced search · real categories, ratings, compare-at discounts · working **coupons** end-to-end · **guest order tracking** (order id + email) · **customer returns** (eligibility + refund priced server-side) · **profile + change password** · saved addresses linked & prefilled in checkout · customer order cancel · checkout Back button · cart drawer a11y (dialog, Esc, focus, stock limits) · global cart drawer.
Admin (rewritten, live-only): order search/filter/pagination + **detail drawer, status workflow, tracking numbers** · product create/**edit/publish/archive**, category & image, **variants + initial stock + compare-at** · inventory list with low-stock filter & adjust · **coupon management** · **returns queue** (approve/receive/refund/reject with notes) · dashboard (today, open orders, returns, low stock).
API: profile, password change, categories, product/variant update, guest tracking, customer returns, coupons admin, paginated admin lists, worker jobs.

## Verified
`go vet`/`go build`/unit tests (Go 1.27.3); end-to-end probe 59/59 on fresh DB and upgrade of a pre-fix DB; storefront `tsc` + ESLint clean + `next build` OK (live mode); admin `tsc` + `vite build` OK. Added tests: rate limiter, cache headers, discounts, slugify/LIKE-escape, registration validation.

# Round 2 — production-readiness audit (this build)

Method: re-reviewed my own changes, then drove the **real site in headless Firefox** (signup → cart → coupon → checkout → orders → tracking → cancel → logout; password reset; deals/collections; phone-width layout), plus concurrency tests and a production-mode boot.

## Bugs found in round 1 and fixed now
| Issue | Fix |
|---|---|
| **API refused to start in production with checkout on** (hard-coded guard) — the site could never take a live order | Replaced with real preconditions: https-only CORS origins, Stripe key+webhook secret both-or-neither. Tested (boots with https, rejects http) |
| Checkout Continue button sat in the sidebar, order summary pushed below (visible in browser) | Form + buttons wrapped in one column |
| Sidebar said "DEMO10 discount" for any coupon; `₹1,799.1` money formatting everywhere | Real code shown; central `fmtNum` (₹1,799.10) |
| Phone layout: header overflowed, cart pushed off-screen, page scrolled sideways | Responsive top bar; verified no horizontal overflow on 5 pages |
| Signup ignored the name field; "Add to Cart" on sold-out items; unstyled Apply buttons | Fixed |
| Coupon use-count never released when an order was cancelled | Restored on customer/admin cancel and by the worker |
| Fragile `err.Error()=="no rows"` check in reviews; 2 stale integration tests (old migration behavior, impossible password) | Fixed; **all Go tests incl. DB integration pass** |
| Dockerfiles: admin had no security headers / API origin config; frontend CSP missing | nginx template with CSP/headers; Next CSP (Stripe-aware), HSTS, X-Frame-Options, Referrer/Permissions-Policy; `error.tsx`; `robots.txt` |

## Gaps closed (were "not done" in round 1)
- **Card payments**: Stripe Payment Element in checkout + **/pay page and "Pay now"** on unpaid card orders (order is only marked paid by Stripe's webhook, never the browser).
- **Password reset** (single-use, 1-hour, hashed token, revokes all sessions, no account enumeration) + forgot/reset pages.
- **E-mail** over SMTP: password reset + order confirmation (tested against a real SMTP server).
- **Deals** (real discounted products) and **Collections** (real categories) use live data; fake countdown hidden outside demo mode.
- CI workflow (`.github/workflows/ci.yml`), docker-compose, go-live checklist in README.

## Evidence
| Check | Result |
|---|---|
| API end-to-end probe (`tools/probe.py`) on a fresh DB | **59/59** |
| Reset/e-mail/worker probe (`tools/probe2.py`) | **16/16** |
| Oversell race: 8 buyers, 3 units | exactly 3 orders, 5 rejected, stock 0 |
| 5 simultaneous identical checkout requests (same key) | exactly 1 order |
| Catalogue under load (8 threads × 25) | p95 ≈ 14 ms |
| Go vet / unit / DB integration tests | all pass |
| Storefront `tsc` + ESLint + production `next build` | clean |
| Admin `tsc` + `vite build` | clean |
| Real browser flows (Firefox, production build) | 31 of 32 checks pass; the 1 "fail" is my test comparing a lower-case ID to the page's upper-case display |
| Upgrade of a pre-fix database | works (versioned migrations replay safely) |

# Read before you ship — what I could NOT verify or did not build
1. **Stripe was not tested against Stripe** (no keys/network in my environment). The server-side payment/webhook logic is covered by the project's own tests and the checkout→intent request shape matches, but run one real test-mode payment (card `4242 4242 4242 4242`) and one webhook delivery before going live.
2. **SMTP**: verified against a local test server only — try your real provider (SPF/DKIM are your DNS's job).
3. **No guest cart** (shoppers must sign in to add to cart). **No tax/GST invoices** — prices are treated as tax-inclusive; shipping fees are fixed (₹0/149/299). **Refunds are manual** (refund in Stripe, then mark refunded in admin).
4. **Existing admin accounts**: before this fix anyone could register as admin — audit (README) and rotate credentials/sessions if the old build was ever public.
5. Browser testing was Firefox headless only (no Safari/Chrome/real phones). Rate limiting is per API instance (use your proxy/CDN for global limits).
6. Docker images were not built here (no Docker); the Go/Next/Vite builds the Dockerfiles run were. `go.mod`/`go.sum` are unchanged (verified with Go 1.27.3).
7. Shipping/digital/notification admin modules were not changed. Fonts load from Google at build time (needs internet during `docker build`).
