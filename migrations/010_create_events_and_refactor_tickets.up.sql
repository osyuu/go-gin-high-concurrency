-- Step 1: Create events table
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Backfill — 以 event_id 為主，一個 event_id 一筆 event；name 先取一個代表值
INSERT INTO events (id, name)
SELECT event_id, MIN(event_name) FROM tickets GROUP BY event_id;

-- Step 2.1: Set serial to MAX(id)+1, avoid duplicate
SELECT setval(pg_get_serial_sequence('events', 'id'), (SELECT COALESCE(MAX(id), 1) FROM events));

-- Step 2.2: Add index to events table
CREATE INDEX IF NOT EXISTS idx_events_event_id ON events(event_id);

-- Step 3: Add foreign key constraint to tickets table
ALTER TABLE tickets
ADD CONSTRAINT fk_tickets_events
FOREIGN KEY (event_id) REFERENCES events(id);

-- Step 4: Add name column to tickets table
ALTER TABLE tickets ADD COLUMN name VARCHAR(255);

-- Step 5: Backfill — set event_name to name
UPDATE tickets
SET name = event_name;

-- Step 5.1: Add NOT NULL constraint to name column
ALTER TABLE tickets ALTER COLUMN name SET NOT NULL;

-- Step 6: Drop event_name column
ALTER TABLE tickets DROP COLUMN event_name;
