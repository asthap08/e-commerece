-- 0012: integrity, security and feature columns found during the code review.

-- Only known roles; the application also never lets a client choose its role.
UPDATE users SET role='customer' WHERE role NOT IN ('customer','admin');
DO $$ BEGIN
  ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('customer','admin'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Case-insensitive unique e-mail.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (lower(email));

-- Profile fields for the (previously unimplemented) account profile page.
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
UPDATE users SET failed_login_attempts = 0 WHERE failed_login_attempts IS NULL;

-- Nullable columns that Go code scans into plain strings: give them defaults.
UPDATE orders SET tracking_number='' WHERE tracking_number IS NULL;
UPDATE orders SET tracking_carrier='' WHERE tracking_carrier IS NULL;
UPDATE orders SET customer_name='' WHERE customer_name IS NULL;
UPDATE orders SET customer_email='' WHERE customer_email IS NULL;
UPDATE orders SET customer_phone='' WHERE customer_phone IS NULL;
UPDATE orders SET payment_method='cod' WHERE payment_method IS NULL;
UPDATE orders SET shipping_method='standard' WHERE shipping_method IS NULL;
UPDATE orders SET shipping_cost_minor=0 WHERE shipping_cost_minor IS NULL;
ALTER TABLE orders ALTER COLUMN tracking_number SET DEFAULT '', ALTER COLUMN tracking_carrier SET DEFAULT '',
  ALTER COLUMN customer_name SET DEFAULT '', ALTER COLUMN customer_email SET DEFAULT '', ALTER COLUMN customer_phone SET DEFAULT '';

-- Orders: coupon/discount snapshot + updated timestamp.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_minor BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS coupon_code TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);

-- Catalogue.
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS compare_at_minor BIGINT;
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
UPDATE products SET category='general' WHERE category IS NULL OR category='';
ALTER TABLE products ALTER COLUMN category SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_products_status_created ON products(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_images_product ON product_images(product_id, position);
CREATE INDEX IF NOT EXISTS idx_variants_product ON product_variants(product_id);

DO $$ BEGIN
  ALTER TABLE products ADD CONSTRAINT products_status_check CHECK (status IN ('draft','active','archived'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  ALTER TABLE product_variants ADD CONSTRAINT variants_price_check CHECK (price_minor >= 0 AND (compare_at_minor IS NULL OR compare_at_minor >= price_minor));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  ALTER TABLE inventory ADD CONSTRAINT inventory_nonneg CHECK (quantity >= 0 AND reserved >= 0 AND quantity >= reserved);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  ALTER TABLE cart_items ADD CONSTRAINT cart_items_qty_check CHECK (quantity > 0 AND quantity <= 99);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  ALTER TABLE order_items ADD CONSTRAINT order_items_qty_check CHECK (quantity > 0);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Reviews: one per user per product; review text defaults.
UPDATE reviews SET title='' WHERE title IS NULL;
UPDATE reviews SET body='' WHERE body IS NULL;
DELETE FROM reviews a USING reviews b WHERE a.product_id=b.product_id AND a.user_id=b.user_id AND a.created_at < b.created_at;
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_user_product ON reviews(user_id, product_id);

-- Addresses: at most one default per user.
UPDATE addresses a SET is_default=false WHERE is_default AND id <> (SELECT id FROM addresses b WHERE b.user_id=a.user_id AND b.is_default ORDER BY created_at DESC, id LIMIT 1);
CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_one_default ON addresses(user_id) WHERE is_default;
UPDATE addresses SET line2='' WHERE line2 IS NULL;

-- Coupons: expiry, minimum order, usage limits; case-insensitive codes.
ALTER TABLE coupons ADD COLUMN IF NOT EXISTS min_order_minor BIGINT NOT NULL DEFAULT 0;
ALTER TABLE coupons ADD COLUMN IF NOT EXISTS max_uses INT;
ALTER TABLE coupons ADD COLUMN IF NOT EXISTS used_count INT NOT NULL DEFAULT 0;
ALTER TABLE coupons ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
UPDATE coupons SET code=upper(code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_coupons_code_upper ON coupons (upper(code));
DO $$ BEGIN
  ALTER TABLE coupons ADD CONSTRAINT coupons_valid CHECK (discount_type IN ('percentage','fixed') AND value > 0 AND (discount_type <> 'percentage' OR value <= 100));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Returns: customers can request them; scanned columns must not be NULL.
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id);
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS admin_note TEXT NOT NULL DEFAULT '';
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE return_requests SET reason='' WHERE reason IS NULL;
UPDATE return_requests SET return_reason='' WHERE return_reason IS NULL;
UPDATE return_requests SET return_label_url='' WHERE return_label_url IS NULL;
UPDATE return_requests SET return_authorization='' WHERE return_authorization IS NULL;
UPDATE return_requests SET refund_amount_minor=0 WHERE refund_amount_minor IS NULL;
UPDATE return_requests SET refund_method='original' WHERE refund_method IS NULL;
ALTER TABLE return_requests ALTER COLUMN reason SET DEFAULT '', ALTER COLUMN return_reason SET DEFAULT '', ALTER COLUMN return_label_url SET DEFAULT '', ALTER COLUMN return_authorization SET DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_return_requests_user ON return_requests(user_id);

-- Shipments / audit / housekeeping indexes.
CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments(order_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_checkout_idem_created ON checkout_idempotency(created_at);
