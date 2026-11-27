-- V14: Ngăn chặn chu trình đa cấp trong cây danh mục (Circular hierarchy prevention trigger)

CREATE OR REPLACE FUNCTION catalog_check_category_circular()
RETURNS TRIGGER AS $$
DECLARE
    curr_parent UUID;
    depth INT := 0;
BEGIN
    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.parent_id = NEW.id THEN
        RAISE EXCEPTION 'Category cannot be its own parent'
            USING ERRCODE = '23514'; -- check_violation
    END IF;

    IF TG_OP = 'UPDATE' THEN
        curr_parent := NEW.parent_id;
        WHILE curr_parent IS NOT NULL LOOP
            IF curr_parent = NEW.id THEN
                RAISE EXCEPTION 'Circular hierarchy detected: category % is an ancestor of %', NEW.id, NEW.parent_id
                    USING ERRCODE = '23514'; -- check_violation
            END IF;
            depth := depth + 1;
            IF depth > 100 THEN
                RAISE EXCEPTION 'Category hierarchy depth exceeded maximum limit (100)'
                    USING ERRCODE = '23514';
            END IF;
            SELECT parent_id INTO curr_parent FROM categories WHERE id = curr_parent;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_categories_check_circular ON categories;

CREATE TRIGGER trg_categories_check_circular
    BEFORE INSERT OR UPDATE OF parent_id ON categories
    FOR EACH ROW
    EXECUTE FUNCTION catalog_check_category_circular();

COMMENT ON FUNCTION catalog_check_category_circular() IS 'Kiểm tra và ngăn chặn chu trình đa cấp trong phân cấp danh mục categories';
