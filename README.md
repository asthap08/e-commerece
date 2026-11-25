# OmniRetail commerce

Go API (`backend/`), Next.js storefront (`frontend/`), Vite admin (`admin/`), PostgreSQL.
See **REVIEW_AND_FIXES.md** for everything that was found and changed.

## Run
```
docker compose up --build        # storefront :3000, admin :3001, api :8080
```
Or manually: `cd backend && go run ./cmd/api` (migrations run automatically), `cd frontend && npm i && npm run dev`, `cd admin && npm i && npm run dev`.

## Creating an administrator
Self-registration always creates a *customer* (the API no longer accepts a `role`). Register normally, then promote the account once, from the database:
```
psql "$DATABASE_URL" -c "UPDATE users SET role='admin' WHERE lower(email)=lower('you@example.com');"
```
**If you ran an earlier build, audit existing admins** — anyone could previously register as admin:
```
psql "$DATABASE_URL" -c "SELECT id,email,created_at FROM users WHERE role='admin' ORDER BY created_at;"
```

## Production notes
- Run `worker` alongside `api` (cancels unpaid card orders after 2h and returns their stock; purges expired sessions).
- Set `TRUST_PROXY=true` only behind a reverse proxy you control; otherwise leave it false.
- Never commit `.env` files (they are git-ignored now).

## Go-live checklist
1. **Backend env** (`backend/.env.example`): `APP_ENV=production`, `DATABASE_URL` (managed Postgres, `sslmode=require`), `CORS_ORIGINS` = your **https** storefront + admin origins, `CHECKOUT_ENABLED=true`, `PUBLIC_URL`, SMTP settings, and (for cards) `STRIPE_SECRET_KEY` + `STRIPE_WEBHOOK_SECRET`. The API refuses to boot with unsafe combinations.
2. **Frontend build args**: `NEXT_PUBLIC_API_BASE_URL` (https API URL), `NEXT_PUBLIC_DEMO_MODE=false`, `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`. Admin: `VITE_API_BASE_URL` at build, `API_ORIGIN` at runtime.
3. **Same-site hosting**: serve the storefront, admin and API under one registrable domain (e.g. `shop.example.com`, `admin.example.com`, `api.example.com`) so session cookies work.
4. **Reverse proxy / TLS** in front of the API (set `TRUST_PROXY=true` only then). Terminate HTTPS; the API sets `Secure` cookies in production.
5. **Stripe**: create a webhook endpoint `https://api.example.com/api/v1/payments/webhook` for `payment_intent.succeeded`, `payment_intent.payment_failed`, `payment_intent.canceled`.
6. **Run the worker** (`docker run … worker`) — it frees stock from unpaid card orders after 2h.
7. **Create your admin** and audit existing admins (see above). Back up Postgres.
8. **Business decisions not encoded in software**: prices are treated as tax-inclusive (no GST breakdown/invoice PDF); shipping fees are fixed (₹0/149/299); refunds are issued manually in Stripe, then marked refunded in the admin.
