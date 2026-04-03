UPDATE demos SET map_name = 'unknown' WHERE map_name IS NULL;
ALTER TABLE demos ALTER COLUMN map_name SET NOT NULL;
ALTER TABLE demos ALTER COLUMN map_name DROP DEFAULT;
