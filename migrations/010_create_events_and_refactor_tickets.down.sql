-- Step 1: Add event_name column back to tickets
ALTER TABLE tickets ADD COLUMN event_name VARCHAR(255);

-- Step 2: Copy name back to event_name
UPDATE tickets SET event_name = name;

-- Step 2.1: Add NOT NULL constraint to event_name column
ALTER TABLE tickets ALTER COLUMN event_name SET NOT NULL;

-- Step 3: Drop name column
ALTER TABLE tickets DROP COLUMN name;

-- Step 4: Drop foreign key constraint
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS fk_tickets_events;

-- Step 5: Drop events table (index idx_events_event_id is dropped with the table)
DROP TABLE IF EXISTS events;