-- initial schema placeholder
CREATE TABLE IF NOT EXISTS health_checks (
    id SERIAL PRIMARY KEY,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS products (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    slug text UNIQUE NOT NULL,
    description text,
    status text NOT NULL DEFAULT 'draft',
    created_at timestamPTZ NOT NULL DEFAULT now(),
    updated_at timestamPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS product_variants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku text UNIQUE NOT NULL,
    price_minor bigint NOT NULL,
    currency text NOT NULL DEFAULT 'INR',
    created_at timestamPTZ NOT NULL DEFAULT now()
);
