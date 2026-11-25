ALTER TABLE products ADD COLUMN IF NOT EXISTS category TEXT DEFAULT 'general';
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
