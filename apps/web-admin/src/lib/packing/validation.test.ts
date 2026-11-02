import { describe, expect, it } from "vitest";
import { parseAdminPackingPath, validatePackingChecklistUpdate, validatePackingTaskComplete } from "./validation";

describe("admin packing route validation", () => {
  it("separates management and assigned workbench routes", () => {
    expect(parseAdminPackingPath(["tasks"])).toEqual({ kind: "task-list" });
    expect(parseAdminPackingPath(["tasks", "pack-001"])).toEqual({ kind: "task-detail", taskId: "pack-001" });
    expect(parseAdminPackingPath(["workbench", "tasks"])).toEqual({ kind: "workbench-list" });
    expect(parseAdminPackingPath(["workbench", "tasks", "pack-001"])).toEqual({ kind: "workbench-detail", taskId: "pack-001" });
    expect(parseAdminPackingPath(["workbench", "tasks", "pack-001", "checklist", "check-01"])).toEqual({ kind: "checklist-update", taskId: "pack-001", itemId: "check-01" });
    expect(parseAdminPackingPath(["workbench", "tasks", "pack-001", "complete"])).toEqual({ kind: "task-complete", taskId: "pack-001" });
  });

  it("validates checklist result and failure note", () => {
    const valid = validatePackingChecklistUpdate({ state: "PASSED", expectedRevision: 2, idempotencyKey: "packing-check-0001" });
    expect(valid.ok).toBe(true);
    const invalid = validatePackingChecklistUpdate({ state: "FAILED", note: "Sai", expectedRevision: 2, idempotencyKey: "packing-check-0002" });
    expect(invalid.ok).toBe(false);
  });

  it("requires revision and idempotency for completion", () => {
    expect(validatePackingTaskComplete({ expectedRevision: 3, idempotencyKey: "packing-complete-0001" }).ok).toBe(true);
    expect(validatePackingTaskComplete({ expectedRevision: 0, idempotencyKey: "short" }).ok).toBe(false);
  });
});
