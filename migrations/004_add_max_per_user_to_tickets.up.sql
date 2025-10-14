-- Update tickets table
ALTER TABLE tickets ADD COLUMN max_per_user INTEGER NOT NULL DEFAULT 4;

-- Add constraints
ALTER TABLE tickets ADD CONSTRAINT tickets_max_per_user_check
 CHECK (max_per_user > 0 AND max_per_user <= 20);

-- Set max_per_user to 4
UPDATE tickets SET max_per_user = 4;