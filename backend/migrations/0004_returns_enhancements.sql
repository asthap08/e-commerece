-- Enhance return_requests table with refund and item details
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS refund_amount_minor BIGINT DEFAULT 0;
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS refund_method TEXT DEFAULT 'original';
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS return_reason TEXT;
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS return_items JSONB DEFAULT '[]'::jsonb;
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS return_label_url TEXT;
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS return_authorization TEXT;
ALTER TABLE return_requests ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;

-- Index for faster returns lookups
CREATE INDEX IF NOT EXISTS idx_return_requests_order_id ON return_requests(order_id);
CREATE INDEX IF NOT EXISTS idx_return_requests_status ON return_requests(status);