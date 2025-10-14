-- Remove soft delete index
DROP INDEX IF EXISTS idx_tickets_deleted_at;

-- Remove soft delete column
ALTER TABLE tickets DROP COLUMN IF EXISTS deleted_at;

