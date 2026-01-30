-- Add order_id column to orders table
ALTER TABLE orders
ADD COLUMN order_id UUID
NOT NULL DEFAULT gen_random_uuid();

-- Add index to order_id column
CREATE INDEX IF NOT EXISTS idx_orders_order_id ON orders(order_id);