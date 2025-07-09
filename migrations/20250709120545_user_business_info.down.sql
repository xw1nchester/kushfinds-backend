ALTER TABLE users
  DROP COLUMN IF EXISTS is_admin;

DROP TABLE IF EXISTS business_profiles;

DROP TABLE IF EXISTS business_categories;

DROP TABLE IF EXISTS business_industries;

DROP TABLE IF EXISTS regions;

DROP TABLE IF EXISTS states;

DROP TABLE IF EXISTS countries;