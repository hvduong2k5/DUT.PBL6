BEGIN;

DROP TABLE IF EXISTS refund_status_history;
DROP TABLE IF EXISTS refund_obligations;
DROP TABLE IF EXISTS returned_goods_tracking;
DROP TABLE IF EXISTS return_status_history;
DROP TABLE IF EXISTS return_decision_lines;
DROP TABLE IF EXISTS return_decisions;
DROP TABLE IF EXISTS return_evidence;
DROP TABLE IF EXISTS return_case_lines;
DROP TABLE IF EXISTS return_cases;

COMMIT;
