-- Priority is now optional at creation; agent/admin assign it later.
ALTER TABLE tickets ALTER COLUMN priority DROP NOT NULL;
ALTER TABLE tickets ALTER COLUMN priority DROP DEFAULT;
UPDATE tickets SET priority = NULL WHERE priority = '';
