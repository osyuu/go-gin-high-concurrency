-- Add ticket_id column to tickets table
ALTER TABLE tickets
ADD COLUMN ticket_id UUID
NOT NULL DEFAULT gen_random_uuid();

-- Add index to ticket_id column
CREATE INDEX IF NOT EXISTS idx_tickets_ticket_id ON tickets(ticket_id);