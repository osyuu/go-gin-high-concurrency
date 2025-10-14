-- Add soft delete column to orders table
ALTER TABLE orders ADD COLUMN deleted_at TIMESTAMP NULL;

-- Add index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at ON orders(deleted_at);

