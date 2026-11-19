"""Verify candidate DDL in an isolated, disposable PostgreSQL Docker container.

Explicit --docker required. No published ports, host mounts or existing DB access.
This verifies schema/selected constraints, not the future service implementation.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import time
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DDL = ROOT / "docs/02_architecture/services/fulfillment-service/fulfillment_persistence.candidate.sql"


def run(args, data=None):
    result = subprocess.run(args, input=data, text=True, encoding="utf-8", capture_output=True, check=False)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return result.stdout.strip()


def smoke_sql():
    return """
BEGIN;
INSERT INTO fulfillment.order_guards(order_id,highest_generation)
VALUES ('00000001-0001-4001-8001-000000000001',1);
INSERT INTO fulfillment.admissions(order_id,generation,state,resource_version)
VALUES ('00000001-0001-4001-8001-000000000001',1,'OPEN',1);
INSERT INTO fulfillment.tasks(task_id,order_id,ready_generation,workflow_state,version,
items_digest,warehouse_id,channel,cohort,input_snapshot_id,input_snapshot_version,input_snapshot,
policy_version,policy_snapshot,reservation_id,allocation_version,allocation_snapshot,evidence_mode,
admitted_at,updated_at)
VALUES ('00000002-0002-4002-8002-000000000002','00000001-0001-4001-8001-000000000001',1,
'READY',1,repeat('a',64),'WH-HUE-01','D2C_WEB','DOMAIN_TEST','snapshot-test',1,'{}',
'test-policy','{}','00000005-0005-4005-8005-000000000005',1,'{}','OPTIONAL',clock_timestamp(),clock_timestamp());
UPDATE fulfillment.admissions SET state='ACCEPTED', task_id='00000002-0002-4002-8002-000000000002',
decision_id='00000011-0011-4011-8011-000000000011',decision_at=clock_timestamp(),
items_digest=repeat('a',64),warehouse_id='WH-HUE-01',permit_id='test-permit',resource_version=2
WHERE order_id='00000001-0001-4001-8001-000000000001' AND generation=1;
UPDATE fulfillment.order_guards SET accepted_task_id='00000002-0002-4002-8002-000000000002'
WHERE order_id='00000001-0001-4001-8001-000000000001';
INSERT INTO fulfillment.packages(package_id,task_id,state,created_at)
VALUES ('00000003-0003-4003-8003-000000000003','00000002-0002-4002-8002-000000000002','DRAFT',clock_timestamp());
INSERT INTO fulfillment.task_revisions(task_id,resource_version,kind,order_id,ready_generation,items_digest,warehouse_id,occurred_at,snapshot)
VALUES ('00000002-0002-4002-8002-000000000002',1,'PACKING_ACCEPTED','00000001-0001-4001-8001-000000000001',1,repeat('a',64),'WH-HUE-01',clock_timestamp(),'{}');
SET CONSTRAINTS ALL IMMEDIATE;
DO $$
DECLARE rejected boolean;
BEGIN
    rejected := false;
    BEGIN
        UPDATE fulfillment.admissions SET state='OPEN',task_id=NULL
        WHERE order_id='00000001-0001-4001-8001-000000000001';
    EXCEPTION WHEN raise_exception THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'terminal admission rewrite was accepted'; END IF;

    rejected := false;
    BEGIN
        UPDATE fulfillment.order_guards SET highest_generation=0
        WHERE order_id='00000001-0001-4001-8001-000000000001';
    EXCEPTION WHEN raise_exception THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'generation regression was accepted'; END IF;

    rejected := false;
    BEGIN
        UPDATE fulfillment.order_guards SET accepted_task_id=NULL
        WHERE order_id='00000001-0001-4001-8001-000000000001';
    EXCEPTION WHEN raise_exception THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'accepted identity removal was accepted'; END IF;

    rejected := false;
    BEGIN
        INSERT INTO fulfillment.tasks SELECT '00000022-0022-4022-8022-000000000022'::uuid,
        order_id,ready_generation,workflow_state,resume_state,version,items_digest,warehouse_id,channel,
        cohort,assignee_id,input_snapshot_id,input_snapshot_version,input_snapshot,policy_version,policy_snapshot,
        reservation_id,allocation_version,allocation_snapshot,evidence_mode,packing_due_at,deadline_revision,
        admitted_at,started_at,completed_at,updated_at FROM fulfillment.tasks;
    EXCEPTION WHEN unique_violation THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'duplicate task/order was accepted'; END IF;

    rejected := false;
    BEGIN
        INSERT INTO fulfillment.task_lines(line_id,task_id,order_line_id,sku_code,product_name,packaging_text,required_quantity,fragile)
        VALUES ('00000004-0004-4004-8004-000000000004','00000002-0002-4002-8002-000000000002',
        '00000004-0004-4004-8004-000000000004','MX-GION-500G','TEST','500g',0,true);
    EXCEPTION WHEN check_violation THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'zero quantity was accepted'; END IF;

    rejected := false;
    BEGIN
        UPDATE fulfillment.packages SET state='SEALED',seal_code='SEAL-TEST-0001',weight_grams=1000,
        length_cm=10,width_cm=10,height_cm=10,fragile=true,sealed_at=clock_timestamp()
        WHERE package_id='00000003-0003-4003-8003-000000000003';
    EXCEPTION WHEN check_violation THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'seal without task version was accepted'; END IF;

    rejected := false;
    BEGIN UPDATE fulfillment.task_revisions SET snapshot='{"changed":true}';
    EXCEPTION WHEN raise_exception THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'immutable revision update was accepted'; END IF;

    rejected := false;
    BEGIN DELETE FROM fulfillment.task_revisions;
    EXCEPTION WHEN raise_exception THEN rejected := true; END;
    IF NOT rejected THEN RAISE EXCEPTION 'immutable revision delete was accepted'; END IF;
END;
$$;
ROLLBACK;
SELECT 'constraints-ok';
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--docker", action="store_true", help="Create a disposable isolated test container")
    parser.add_argument("--image", default="postgres:16-alpine")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    if not args.docker:
        parser.error("--docker is required; no existing database is used")
    image_id = run(["docker", "image", "inspect", args.image, "--format", "{{.Id}}"])
    name = "codex-fulfillment-ddl-" + uuid.uuid4().hex[:12]
    container = run(["docker", "run", "--rm", "-d", "--name", name, "--network", "none",
                     "--memory", "256m", "--tmpfs", "/var/lib/postgresql/data",
                     "-e", "POSTGRES_PASSWORD=design-only-ephemeral", args.image])
    if not re.fullmatch(r"[0-9a-f]{64}", container):
        raise RuntimeError("Unexpected container identity; no broad cleanup attempted")
    result = {"result": "PENDING", "runtime_acceptance": "PENDING", "image": args.image,
              "image_id": image_id, "container": name, "network": "none", "host_mounts": False,
              "published_ports": False, "scope": "Candidate schema and eight selected constraint smoke checks; not service concurrency/E2E."}
    try:
        for _ in range(100):
            ready = subprocess.run(["docker", "exec", container, "pg_isready", "-U", "postgres"],
                                   capture_output=True, check=False)
            if ready.returncode == 0:
                break
            time.sleep(0.2)
        else:
            raise RuntimeError("Isolated PostgreSQL did not become ready")
        psql = ["docker", "exec", "-i", container, "psql", "-X", "-qAt", "-U", "postgres", "-v", "ON_ERROR_STOP=1"]
        run(psql, DDL.read_text(encoding="utf-8"))
        result["postgres_version"] = run(psql, "SELECT version();")
        result["tables_created"] = int(run(psql, "SELECT count(*) FROM information_schema.tables WHERE table_schema='fulfillment' AND table_type='BASE TABLE';"))
        if result["tables_created"] != 16:
            raise RuntimeError("Unexpected candidate table count")
        if run(psql, smoke_sql()) != "constraints-ok":
            raise RuntimeError("Constraint smoke test did not finish")
        result["constraint_checks"] = 8
        result["result"] = "PASS"
    except Exception as error:
        result.update(result="FAIL", error=str(error))
    finally:
        run(["docker", "stop", "-t", "5", container])
        result["disposable_container_stopped"] = True
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, ensure_ascii=True, indent=2))
    return 0 if result["result"] == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
