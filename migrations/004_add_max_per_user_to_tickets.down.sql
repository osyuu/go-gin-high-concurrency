-- Drop constraints
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_max_per_user_check;

-- Drop max_per_user column
ALTER TABLE tickets DROP COLUMN IF EXISTS max_per_user;
