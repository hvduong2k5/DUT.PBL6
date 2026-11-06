CREATE OR REPLACE FUNCTION reject_supply_plan_revision_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'supply plan revisions are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER supply_plan_revisions_immutable
BEFORE UPDATE OR DELETE ON supply_plan_revisions
FOR EACH ROW EXECUTE FUNCTION reject_supply_plan_revision_mutation();
