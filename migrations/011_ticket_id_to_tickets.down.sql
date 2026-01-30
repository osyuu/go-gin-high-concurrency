-- Drop index on ticket_id
DROP INDEX IF EXISTS idx_tickets_ticket_id;

-- Drop ticket_id column from tickets table
ALTER TABLE tickets DROP COLUMN IF EXISTS ticket_id;
