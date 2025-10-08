-- Create tickets table
CREATE TABLE IF NOT EXISTS tickets (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    total_stock INT NOT NULL,
    remaining_stock INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Add constraints
    CONSTRAINT tickets_stock_check CHECK (remaining_stock >= 0),
    CONSTRAINT tickets_total_stock_check CHECK (total_stock >= remaining_stock),
    CONSTRAINT tickets_price_check CHECK (price >= 0)
);


-- Add index
CREATE INDEX IF NOT EXISTS idx_tickets_event_id ON tickets(event_id);
CREATE INDEX IF NOT EXISTS idx_tickets_remaining_stock ON tickets(remaining_stock);