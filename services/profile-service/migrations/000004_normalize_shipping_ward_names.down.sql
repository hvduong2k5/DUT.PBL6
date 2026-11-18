-- The province remains in province_name. Restoring duplicate text would lose
-- the distinction between rows originally written with and without a suffix.
DROP FUNCTION IF EXISTS profile_normalize_ward_name(text, text);
