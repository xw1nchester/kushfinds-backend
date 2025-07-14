DROP TABLE IF EXISTS business_profiles;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_admin,
    DROP COLUMN IF EXISTS country_id,
    DROP COLUMN IF EXISTS state_id,
    DROP COLUMN IF EXISTS region_id,
    DROP COLUMN IF EXISTS age,
    DROP COLUMN IF EXISTS phone_number;

ALTER TABLE users 
    DROP CONSTRAINT IF EXISTS fk_country_id,
    DROP CONSTRAINT IF EXISTS fk_state_id,
    DROP CONSTRAINT IF EXISTS fk_region_id;

DROP TABLE IF EXISTS business_industries;

DROP TABLE IF EXISTS regions;

DROP TABLE IF EXISTS states;

DROP TABLE IF EXISTS countries;