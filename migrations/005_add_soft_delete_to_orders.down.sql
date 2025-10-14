-- Remove soft delete index
DROP INDEX IF EXISTS idx_orders_deleted_at;

-- Remove soft delete column
ALTER TABLE orders DROP COLUMN IF EXISTS deleted_at;

