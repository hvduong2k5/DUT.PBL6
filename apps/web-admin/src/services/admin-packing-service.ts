import type { PackingChecklistUpdateInput, PackingChecklistUpdateResult, PackingTaskCompleteInput, PackingTaskCompleteResult, PackingTaskDetail, PackingTaskList } from "@/lib/packing/types";
import { AdminPackingApiError } from "@/lib/packing/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/admin/packing${path}`, { ...init, cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminPackingApiError(response.status, payload);
  return payload as T;
}

export const adminPackingService = {
  tasks() { return request<PackingTaskList>("/tasks"); },
  task(taskId: string) { return request<PackingTaskDetail>(`/tasks/${encodeURIComponent(taskId)}`); },
  assignedTasks() { return request<PackingTaskList>("/workbench/tasks"); },
  assignedTask(taskId: string) { return request<PackingTaskDetail>(`/workbench/tasks/${encodeURIComponent(taskId)}`); },
  updateChecklist(taskId: string, itemId: string, input: PackingChecklistUpdateInput) { return request<PackingChecklistUpdateResult>(`/workbench/tasks/${encodeURIComponent(taskId)}/checklist/${encodeURIComponent(itemId)}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }); },
  completeTask(taskId: string, input: PackingTaskCompleteInput) { return request<PackingTaskCompleteResult>(`/workbench/tasks/${encodeURIComponent(taskId)}/complete`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }); }
};
