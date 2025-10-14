-- Add soft delete column to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL;

-- Add index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

