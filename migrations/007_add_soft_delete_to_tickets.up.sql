-- Add soft delete column to tickets table
ALTER TABLE tickets ADD COLUMN deleted_at TIMESTAMP NULL;

-- Add index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_tickets_deleted_at ON tickets(deleted_at);

