Test scripts used for the review (they target a local API + Postgres; adjust ports/DB names at the top of each file):
- probe.py — 59-check API regression (`python3 probe.py http://127.0.0.1:18080 <dbname>`)
- probe2.py — password reset, SMTP e-mails, worker (needs a local SMTP sink on :2525)
- probe3.py — oversell race + idempotency + latency
- browser_checkout_flow.py — Selenium/Firefox storefront flow
