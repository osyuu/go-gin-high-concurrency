-- Drop RESTRICT foreign key constraints
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_user_id;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_ticket_id;

-- Restore original CASCADE foreign key constraints
ALTER TABLE orders ADD CONSTRAINT fk_orders_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) 
    ON DELETE CASCADE;

ALTER TABLE orders ADD CONSTRAINT fk_orders_ticket_id 
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) 
    ON DELETE CASCADE;

