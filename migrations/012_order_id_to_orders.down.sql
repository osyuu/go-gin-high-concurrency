-- Drop index on order_id
DROP INDEX IF EXISTS idx_orders_order_id;

-- Drop order_id column from orders table
ALTER TABLE orders DROP COLUMN IF EXISTS order_id;
