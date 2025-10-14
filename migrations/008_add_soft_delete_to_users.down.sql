-- Remove soft delete index
DROP INDEX IF EXISTS idx_users_deleted_at;

-- Remove soft delete column
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

