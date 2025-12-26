-- Add request_id column to orders table
ALTER TABLE orders ADD COLUMN request_id VARCHAR(255) NOT NULL;

-- Add unique index for request_id queries
CREATE UNIQUE INDEX idx_orders_request_id ON orders(request_id);

