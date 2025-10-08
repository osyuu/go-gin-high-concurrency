-- Drop constraints
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_user_id;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_ticket_id;

-- Drop orders table
DROP TABLE IF EXISTS orders;