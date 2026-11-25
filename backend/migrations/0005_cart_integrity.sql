WITH grouped_items AS (
    SELECT cart_id, variant_id, (array_agg(id ORDER BY id))[1] AS keeper_id, sum(quantity)::integer AS quantity
    FROM cart_items
    GROUP BY cart_id, variant_id
    HAVING count(*) > 1
)
UPDATE cart_items AS item
SET quantity = grouped_items.quantity
FROM grouped_items
WHERE item.id = grouped_items.keeper_id;

DELETE FROM cart_items AS item
USING cart_items AS keeper
WHERE item.cart_id = keeper.cart_id
  AND item.variant_id = keeper.variant_id
  AND keeper.id < item.id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_items_cart_variant_unique
    ON cart_items(cart_id, variant_id);

WITH ranked_carts AS (
    SELECT
        id,
        first_value(id) OVER (
            PARTITION BY CASE
                WHEN user_id IS NOT NULL THEN 'user:' || user_id::text
                ELSE 'session:' || session_id
            END
            ORDER BY id
        ) AS keeper_id
    FROM carts
    WHERE user_id IS NOT NULL OR session_id IS NOT NULL
),
duplicate_items AS (
    SELECT ranked_carts.keeper_id, item.variant_id, sum(item.quantity)::integer AS quantity
    FROM ranked_carts
    JOIN cart_items AS item ON item.cart_id = ranked_carts.id
    WHERE ranked_carts.id <> ranked_carts.keeper_id
    GROUP BY ranked_carts.keeper_id, item.variant_id
)
INSERT INTO cart_items(cart_id, variant_id, quantity)
SELECT keeper_id, variant_id, quantity
FROM duplicate_items
ON CONFLICT (cart_id, variant_id)
DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity;

WITH ranked_carts AS (
    SELECT
        id,
        first_value(id) OVER (
            PARTITION BY CASE
                WHEN user_id IS NOT NULL THEN 'user:' || user_id::text
                ELSE 'session:' || session_id
            END
            ORDER BY id
        ) AS keeper_id
    FROM carts
    WHERE user_id IS NOT NULL OR session_id IS NOT NULL
)
DELETE FROM carts AS cart
USING ranked_carts
WHERE cart.id = ranked_carts.id
  AND ranked_carts.id <> ranked_carts.keeper_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_carts_user_id_unique
    ON carts(user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_carts_session_id_unique
    ON carts(session_id) WHERE session_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_items_cart_variant_unique
    ON cart_items(cart_id, variant_id);
