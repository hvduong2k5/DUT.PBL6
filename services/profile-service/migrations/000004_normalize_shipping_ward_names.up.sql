CREATE OR REPLACE FUNCTION profile_normalize_ward_name(ward text, province text)
RETURNS text
LANGUAGE plpgsql IMMUTABLE STRICT
AS $$
DECLARE
    normalized text := btrim(ward);
    province_suffix text := ', ' || btrim(province);
BEGIN
    IF btrim(province) = '' THEN
        RETURN ward;
    END IF;
    WHILE right(normalized, length(province_suffix)) = province_suffix LOOP
        normalized := rtrim(left(normalized, -length(province_suffix)));
    END LOOP;
    RETURN CASE WHEN normalized = '' THEN ward ELSE normalized END;
END;
$$;

UPDATE shipping_addresses
SET ward_name = profile_normalize_ward_name(ward_name, province_name)
WHERE ward_name IS DISTINCT FROM profile_normalize_ward_name(ward_name, province_name);
