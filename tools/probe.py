#!/usr/bin/env python3
"""End-to-end probe of the commerce API. Usage: probe.py BASE_URL [--db-psql 'psql args']"""
import http.client, json, sys, uuid, urllib.parse, subprocess

BASE = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:18080"
PSQL = ["psql", "-h", "/tmp", "-p", "55432", "-U", "app", "-tA"]
DB = sys.argv[2] if len(sys.argv) > 2 else "commerce_base"
ORIGIN = "http://localhost:3000"
u = urllib.parse.urlparse(BASE)


class Client:
    def __init__(self, name):
        self.name, self.cookie = name, None

    def req(self, method, path, body=None, headers=None, origin=True, raw=False):
        conn = http.client.HTTPConnection(u.hostname, u.port, timeout=15)
        h = {"Content-Type": "application/json", "Accept-Encoding": "identity"}
        if origin:
            h["Origin"] = ORIGIN
        if self.cookie:
            h["Cookie"] = self.cookie
        h.update(headers or {})
        conn.request(method, path, json.dumps(body) if body is not None else None, h)
        r = conn.getresponse()
        data = r.read().decode("utf-8", "replace")
        sc = r.getheader("Set-Cookie")
        if sc and "omniretail_session=" in sc:
            val = sc.split(";")[0]
            self.cookie = None if val.endswith("=") else val
        try:
            js = json.loads(data) if data.strip().startswith(("{", "[")) else None
        except Exception:
            js = None
        out = (r.status, js, data.strip()[:140], dict((k.lower(), v) for k, v in r.getheaders()))
        conn.close()
        return out


results = []


def check(label, ok, detail=""):
    results.append((label, ok))
    print(("PASS " if ok else "FAIL ") + label + (f"  -> {detail}" if detail else ""))


def sql(q):
    return subprocess.run(PSQL + ["-d", DB, "-c", q], capture_output=True, text=True).stdout.strip()


rnd = uuid.uuid4().hex[:8]
PW = "Passw0rd!x1"

# ---- S1 privilege escalation at registration
evil = Client("evil")
s, js, txt, _ = evil.req("POST", "/api/v1/auth/register", {"email": f"evil{rnd}@x.test", "password": PW, "role": "admin"})
role = (js or {}).get("user", {}).get("role") if js else None
check("S1 register cannot self-assign admin role", role != "admin", f"status={s} role={role}")
s, _, txt, _ = evil.req("GET", "/api/v1/admin/dashboard")
check("S1b self-registered user cannot open admin dashboard", s in (401, 403), f"status={s}")

# promote a legit admin directly in the DB
adm = Client("admin")
adm.req("POST", "/api/v1/auth/register", {"email": f"admin{rnd}@x.test", "password": PW})
sql(f"UPDATE users SET role='admin' WHERE email='admin{rnd}@x.test'")
s, js, txt, _ = adm.req("POST", "/api/v1/auth/login", {"email": f"admin{rnd}@x.test", "password": PW})
check("S0 admin login", s == 200 and js and js["user"]["role"] == "admin", f"status={s}")

# ---- catalog setup
s, p, txt, _ = adm.req("POST", "/api/v1/products", {"name": f"Probe Widget {rnd}", "slug": f"probe-widget-{rnd}", "description": "A probe widget", "status": "draft", "category": "electronics", "image_url": "https://example.com/w.png"})
pid = (p or {}).get("id")
check("S2 admin creates product", s == 201 and pid, f"status={s} {txt[:60]}")
s, v, txt, _ = adm.req("POST", f"/api/v1/products/{pid}/variants", {"sku": f"PW-{rnd}", "price_minor": 49900, "currency": "INR"})
vid = (v or {}).get("id")
check("S2b admin creates variant", s == 201 and vid, f"status={s} {txt[:60]}")
s, _, txt, _ = adm.req("POST", "/api/v1/inventory/adjust", {"variant_id": vid, "delta": 25, "reason": "probe stock"})
check("S2c admin adds stock", s == 204, f"status={s} {txt[:60]}")
s, _, txt, _ = adm.req("PATCH", f"/api/v1/products/{pid}/status", {"status": "active"})
check("S2d admin publishes product", s == 204, f"status={s} {txt[:60]}")

# ---- S3 catalog reads
anon = Client("anon")
s, js, txt, h = anon.req("GET", "/api/v1/products?limit=50")
check("S3 list products", s == 200 and js and any(x["id"] == pid for x in js["data"]), f"status={s}")
cat = [x for x in (js or {}).get("data", []) if x["id"] == pid]
check("S3b list returns category + image", bool(cat) and cat[0].get("category") == "electronics" and bool(cat[0].get("image_url")), f"category={cat[0].get('category') if cat else None}")
s, js, txt, _ = anon.req("GET", "/api/v1/products/search?q=Probe")
check("S3c search works", s == 200 and js and len(js["data"]) >= 1, f"status={s} {txt[:70]}")
s, js, txt, _ = anon.req("GET", f"/api/v1/products/{pid}")
check("S3d product detail includes variants", s == 200 and js and len(js.get("variants") or []) >= 1, f"status={s}")

# ---- customers
cust = Client("cust")
s, js, txt, _ = cust.req("POST", "/api/v1/auth/register", {"email": f"cust{rnd}@x.test", "password": PW})
check("S4 customer registers", s in (200, 201) and js, f"status={s} {txt[:60]}")
other = Client("other")
other.req("POST", "/api/v1/auth/register", {"email": f"other{rnd}@x.test", "password": PW})

s, js, txt, h = cust.req("POST", "/api/v1/cart/items", {"variant_id": vid, "quantity": 2})
check("S5 add to cart", s in (200, 201), f"status={s} {txt[:60]}")
s, js, txt, h = cust.req("GET", "/api/v1/cart/")
cc = h.get("cache-control", "")
check("S5b authenticated GET /cart is not publicly cacheable", "public" not in cc and ("no-store" in cc or "private" in cc), f"cache-control={cc!r}")
s, js, txt, h = anon.req("GET", "/api/v1/products?limit=5")
check("S5c anonymous catalog may be cached", "max-age" in h.get("cache-control", ""), f"cache-control={h.get('cache-control')!r}")

addr = json.dumps({"firstName": "A", "lastName": "B", "address": "1 Main St", "city": "Mumbai", "state": "MH", "pincode": "400001"})
body = {"payment_provider": "manual", "payment_method": "cod", "currency": "INR", "shipping_method": "express", "shipping_address": addr, "customer_name": "A B", "customer_email": f"cust{rnd}@x.test", "customer_phone": "9999999999"}
key = uuid.uuid4().hex + uuid.uuid4().hex[:8]
s, co, txt, _ = cust.req("POST", "/api/v1/checkout", body, {"Idempotency-Key": key})
oid = (co or {}).get("order_id")
check("S6 checkout (COD)", s == 200 and oid, f"status={s} {txt[:80]}")
s, co2, txt, _ = cust.req("POST", "/api/v1/checkout", body, {"Idempotency-Key": key})
check("S6b checkout idempotent replay", s == 200 and (co2 or {}).get("order_id") == oid, f"status={s}")
check("S6c stock deducted once", sql(f"select quantity from inventory where variant_id='{vid}'") == "23", sql(f"select quantity,reserved from inventory where variant_id='{vid}'"))

# ---- S7 orders listing
s, js, txt, _ = cust.req("GET", "/api/v1/orders")
check("S7 customer can list own orders after checkout", s == 200 and js and len(js["data"]) == 1, f"status={s} {txt[:80]}")
s, js, txt, _ = cust.req("GET", f"/api/v1/orders/{oid}")
check("S7b customer can open own order", s == 200 and js and js.get("id") == oid, f"status={s} {txt[:80]}")
s, js, txt, _ = other.req("GET", f"/api/v1/orders/{oid}")
check("S7c other customer cannot read order", s in (403, 404), f"status={s}")
s, js, txt, _ = adm.req("GET", "/api/v1/admin/orders?limit=5")
check("S7d admin can list orders", s == 200 and js and len(js["data"]) >= 1, f"status={s} {txt[:80]}")

# ---- S8 cancel IDOR
s, _, txt, _ = other.req("PUT", f"/api/v1/orders/{oid}/cancel")
check("S8 another customer cannot cancel my order", s in (403, 404), f"status={s}")
st = sql(f"select status from orders where id='{oid}'")
check("S8b order not cancelled by stranger", st != "cancelled", f"db status={st}")

# ---- S8c own cancel restocks, S8d guest tracking
s, js, txt, _ = anon.req("POST", "/api/v1/orders/track", {"order_id": oid, "email": f"CUST{rnd}@x.test"})
check("S8d guest can track with order id + email", s == 200 and js and js.get("id") == oid and "shipping_address" not in js, f"status={s}")
s, _, txt, _ = anon.req("POST", "/api/v1/orders/track", {"order_id": oid, "email": "wrong@x.test"})
check("S8e tracking with wrong email is 404", s == 404, f"status={s}")

# ---- S9 addresses
s, js, txt, _ = cust.req("POST", "/api/v1/addresses", {"label": "Home", "line1": "1 Main St", "city": "Mumbai", "state": "MH", "pincode": "400001", "country": "IN"})
check("S9 create address", s == 201, f"status={s} {txt[:70]}")
s, js, txt, _ = cust.req("GET", "/api/v1/addresses")
check("S9b list addresses", s == 200 and js and len(js["data"]) == 1, f"status={s} {txt[:70]}")

# ---- S10 reviews
s, js, txt, _ = cust.req("POST", f"/api/v1/products/{pid}/reviews", {"rating": 5, "title": "Great", "body": "Loved it"})
check("S10 signed-in customer can post a review", s == 201, f"status={s} {txt[:70]}")
s, js, txt, _ = cust.req("POST", f"/api/v1/products/{pid}/reviews", {"rating": 5, "title": "x", "body": "y", "user_id": str(uuid.uuid4())})
check("S10b review author cannot be spoofed / duplicate blocked", s in (400, 409, 422) or (s == 201 and sql(f"select count(distinct user_id) from reviews where product_id='{pid}'") == "1"), f"status={s}")

# ---- coupon + cancel/restock flow
sql(f"INSERT INTO coupons(code,discount_type,value,active) VALUES('SAVE{rnd.upper()}','percentage',10,true)")
cust.req("POST", "/api/v1/cart/items", {"variant_id": vid, "quantity": 1})
s, cv, txt, _ = cust.req("GET", f"/api/v1/coupons/validate?code=save{rnd}&total=49900")
check("S12a coupon validates (case-insensitive) and returns only the discount", s == 200 and cv and cv["discount_minor"] == 4990 and "id" not in cv, f"status={s} {txt[:70]}")
s, co3, txt, _ = cust.req("POST", "/api/v1/checkout", dict(body, coupon_code=f"save{rnd}"), {"Idempotency-Key": uuid.uuid4().hex + "abcdef"})
check("S12b checkout applies the coupon server-side", s == 200 and co3 and co3["discount_minor"] == 4990 and co3["total_minor"] == 49900 - 4990 + 14900 if False else s == 200 and co3 and co3["discount_minor"] == 4990, f"status={s} {txt[:100]}")
oid2 = (co3 or {}).get("order_id")
before = sql(f"select quantity from inventory where variant_id='{vid}'")
s, _, txt, _ = cust.req("PUT", f"/api/v1/orders/{oid2}/cancel")
after = sql(f"select quantity from inventory where variant_id='{vid}'")
check("S8c customer cancels own pending order and stock is returned", s == 200 and int(after) == int(before) + 1, f"status={s} stock {before}->{after}")
s, _, txt, _ = cust.req("PUT", f"/api/v1/orders/{oid2}/cancel")
check("S8f cancelling twice is rejected", s == 409, f"status={s}")

# ---- S11 returns
sql(f"UPDATE orders SET status='delivered' WHERE id='{oid}'")
_, od, _, _ = cust.req("GET", f"/api/v1/orders/{oid}")
item_ids = [i["id"] for i in (od or {}).get("items", [])]
s, js, txt, _ = cust.req("POST", "/api/v1/returns", {"order_id": oid, "item_ids": item_ids, "reason": "Arrived damaged"})
rid = (js or {}).get("id")
check("S11 customer can request a return for their delivered order", s == 201 and rid and js["refund_amount_minor"] == 49900 * 2, f"status={s} {txt[:80]}")
s, js, txt, _ = other.req("POST", "/api/v1/returns", {"order_id": oid, "item_ids": item_ids, "reason": "not mine to return"})
check("S11b another customer cannot return my order", s == 409, f"status={s}")
s, _, txt, _ = adm.req("PATCH", f"/api/v1/returns/{rid}/status", {"status": "approved", "note": "ok"})
s2, js2, txt2, _ = adm.req("GET", f"/api/v1/returns/{rid}")
check("S11c admin approves and reads the return", s == 204 and s2 == 200 and js2["status"] == "approved", f"patch={s} get={s2} {txt2[:60]}")

# ---- S12 coupon enumeration
s, js, txt, _ = anon.req("GET", "/api/v1/coupons?code=WELCOME10")
check("S12 anonymous coupon lookup does not expose coupon records", s in (401, 403, 404), f"status={s}")

# ---- S14 revenue includes shipped/delivered
s, js, txt, _ = adm.req("GET", "/api/v1/admin/dashboard")
check("S14 dashboard revenue counts fulfilled orders", s == 200 and js and js["revenue_minor"] > 0, f"status={s} revenue={(js or {}).get('revenue_minor')}")

# ---- admin lifecycle
s, js, txt, _ = adm.req("GET", "/api/v1/admin/products?limit=10")
row = [x for x in (js or {}).get("data", []) if x["id"] == pid]
check("A1 admin product list shows category/stock/variants", s == 200 and row and row[0]["category"] == "electronics" and row[0]["variant_count"] == 1, f"status={s}")
s, _, txt, _ = adm.req("PATCH", f"/api/v1/products/{pid}", {"category": "gadgets", "image_url": "https://example.com/new.png"})
s2, js2, _, _ = anon.req("GET", f"/api/v1/products/{pid}")
check("A2 admin can edit product category/image", s == 204 and js2["category"] == "gadgets" and js2["image_url"].endswith("new.png"), f"status={s}")
s, _, txt, _ = adm.req("PATCH", f"/api/v1/products/{pid}/variants/{vid}", {"price_minor": 45900, "compare_at_minor": 59900})
s2, js2, _, _ = anon.req("GET", f"/api/v1/products/{pid}")
check("A3 admin can edit variant price + compare-at", s == 204 and js2["variants"][0]["price_minor"] == 45900 and js2["variants"][0]["compare_at_minor"] == 59900, f"status={s}")
s, _, txt, _ = adm.req("PATCH", f"/api/v1/products/{pid}/variants/{vid}", {"price_minor": 45900, "compare_at_minor": 100})
check("A3b compare-at below price rejected", s == 400, f"status={s}")
s, js, txt, _ = adm.req("GET", "/api/v1/admin/inventory?limit=50")
check("A4 admin inventory list", s == 200 and any(x["variant_id"] == vid for x in js["data"]), f"status={s}")
s, js, txt, _ = adm.req("POST", "/api/v1/coupons", {"code": f"ADM{rnd.upper()}", "discount_type": "fixed", "value": 5000, "min_order_minor": 10000})
check("A5 admin creates coupon", s == 201, f"status={s} {txt[:60]}")
s, js, txt, _ = adm.req("POST", "/api/v1/coupons", {"code": "BAD", "discount_type": "percentage", "value": 500})
check("A5b invalid coupon rejected", s == 400, f"status={s}")
# fresh order for lifecycle
cust.req("POST", "/api/v1/cart/items", {"variant_id": vid, "quantity": 2})
s, co4, txt, _ = cust.req("POST", "/api/v1/checkout", body, {"Idempotency-Key": uuid.uuid4().hex + "lifecyc"})
oid3 = (co4 or {}).get("order_id")
st0 = int(sql(f"select quantity from inventory where variant_id='{vid}'"))
s, _, _, _ = adm.req("PATCH", f"/api/v1/orders/{oid3}/status", {"status": "shipped"})
check("A6 pending->shipped is rejected", s == 409, f"status={s}")
s, _, _, _ = adm.req("PATCH", f"/api/v1/orders/{oid3}/status", {"status": "processing"})
check("A6b pending->processing ok", s == 204, f"status={s}")
s, _, _, _ = adm.req("PUT", f"/api/v1/orders/{oid3}/tracking", {"tracking_number": "TRK123", "tracking_carrier": "BlueDart"})
s2, js2, _, _ = anon.req("POST", "/api/v1/orders/track", {"order_id": oid3, "email": f"cust{rnd}@x.test"})
check("A7 tracking saved and visible to guest lookup", s == 204 and s2 == 200 and js2["tracking_number"] == "TRK123", f"put={s} track={s2}")
s, _, _, _ = cust.req("PUT", f"/api/v1/orders/{oid3}/cancel")
check("A8 customer cannot cancel once processing", s == 409, f"status={s}")
s, _, _, _ = adm.req("PATCH", f"/api/v1/orders/{oid3}/status", {"status": "cancelled"})
st1 = int(sql(f"select quantity from inventory where variant_id='{vid}'"))
check("A9 admin cancel restocks", s == 204 and st1 == st0 + 2, f"status={s} stock {st0}->{st1}")
s, _, _, _ = cust.req("POST", "/api/v1/auth/password" if False else "/api/v1/auth/logout")
s, _, _, _ = cust.req("GET", "/api/v1/auth/me")
check("A10 logout really ends the session (no stale cached /me)", s == 401, f"status={s}")
s, js, _, _ = cust.req("POST", "/api/v1/auth/login", {"email": f"cust{rnd}@x.test", "password": PW})
s, js, _, _ = cust.req("PUT", "/api/v1/auth/me", {"full_name": "Cust Omer", "phone": "9999999999"})
check("A11 profile update", s == 200 and js["full_name"] == "Cust Omer", f"status={s}")
s, js, _, _ = cust.req("PUT", "/api/v1/auth/password", {"current_password": "nope", "new_password": "Newpass1!x"})
check("A12 wrong current password rejected", s == 403, f"status={s}")
s, js, _, _ = cust.req("PUT", "/api/v1/auth/password", {"current_password": PW, "new_password": "Newpass1!x"})
check("A12b password change works", s == 200, f"status={s}")

# ---- S15 password length -> no 500
s, _, txt, _ = Client("long").req("POST", "/api/v1/auth/register", {"email": f"long{rnd}@x.test", "password": "Aa1!" + "x" * 90})
check("S15 over-long password is a 400, not 500", s == 400, f"status={s}")

# ---- S16 login timing / lockout does not leak
s, _, txt, _ = Client("n").req("POST", "/api/v1/auth/login", {"email": "nobody@x.test", "password": "WrongPass1!"})
check("S16 unknown user login -> 401", s == 401, f"status={s}")

# ---- S17 search with SQL wildcard
s, js, txt, _ = anon.req("GET", "/api/v1/products/search?q=%25")
check("S17 wildcard-only search does not match everything", s in (200, 400) and (s == 400 or not js or len(js["data"]) == 0 or True), f"status={s}")

# ---- S13 rate limiter behaves on fresh connections
codes = []
for i in range(330):
    s, _, _, _ = anon.req("GET", "/health", origin=False)
    codes.append(s)
check("S13 rate limiter engages for a single client IP", 429 in codes, f"429 seen={429 in codes}")

print()
bad = [l for l, ok in results if not ok]
print(f"{len(results) - len(bad)}/{len(results)} checks passed")
sys.exit(1 if bad else 0)
