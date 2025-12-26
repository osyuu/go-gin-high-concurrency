-- Remove request_id unique index
DROP INDEX IF EXISTS idx_orders_request_id;

-- Remove request_id column from orders table
ALTER TABLE orders DROP COLUMN IF EXISTS request_id;
