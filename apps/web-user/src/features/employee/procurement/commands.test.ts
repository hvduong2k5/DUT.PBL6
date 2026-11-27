import { describe, expect, it, vi } from "vitest";
import { CommandError, commandStorageKey, createCommandRunner } from "./commands";

const actor = "11111111-1111-4111-8111-111111111111";
function store() {
  const data = new Map<string, string>();
  return { data, getItem: (key: string) => data.get(key) ?? null, setItem: (key: string, value: string) => { data.set(key, value); }, removeItem: (key: string) => { data.delete(key); } };
}
const success = () => Response.json({ id: actor }, { status: 201 });
describe("Procurement browser command retries", () => {
  it("sends same-origin cookies and a key, never a browser token or role header", async () => {
    const storage = store(), fetcher = vi.fn().mockResolvedValue(success());
    await createCommandRunner(storage, actor, fetcher)("POST", "suppliers", { name: "Supplier" });
    const [url, init] = fetcher.mock.calls[0];
    expect(url).toBe("/api/admin/procurement/suppliers"); expect(init.credentials).toBe("same-origin");
    expect(init.headers["Idempotency-Key"]).toMatch(/^procurement-/u);
    expect(init.headers.Authorization).toBeUndefined(); expect(init.headers["X-User-Role"]).toBeUndefined();
    expect(storage.data.size).toBe(0);
  });
  it("keeps an uncertain key across runner recreation without storing the business payload", async () => {
    const storage = store(), fetcher = vi.fn().mockRejectedValueOnce(new Error("lost response"));
    const payload = { code: "SUP-PRIVATE", name: "Private supplier" };
    await expect(createCommandRunner(storage, actor, fetcher)("POST", "suppliers", payload)).rejects.toMatchObject({ pending: true });
    const entry = storage.getItem(commandStorageKey(actor, "POST", "suppliers"))!;
    expect(entry).not.toContain("SUP-PRIVATE"); expect(entry).not.toContain("Private supplier");
    fetcher.mockResolvedValueOnce(success());
    await createCommandRunner(storage, actor, fetcher)("POST", "suppliers", { name: "Private supplier", code: "SUP-PRIVATE" });
    expect(fetcher.mock.calls[0][1].headers["Idempotency-Key"]).toBe(fetcher.mock.calls[1][1].headers["Idempotency-Key"]);
    expect(fetcher.mock.calls[0][1].body).toBe(fetcher.mock.calls[1][1].body);
    expect(storage.data.size).toBe(0);
  });
  it("rejects changed payload/version under an unresolved command, without silently minting a new key", async () => {
    const storage = store(), fetcher = vi.fn().mockRejectedValue(new Error("offline"));
    const run = createCommandRunner(storage, actor, fetcher);
    await expect(run("POST", "suppliers", { name: "First" })).rejects.toBeInstanceOf(CommandError);
    const before = [...storage.data.values()];
    await expect(run("POST", "suppliers", { name: "Changed" })).rejects.toMatchObject({ pending: true, code: "RETRY_PAYLOAD_MISMATCH" });
    expect(fetcher).toHaveBeenCalledTimes(1); expect([...storage.data.values()]).toEqual(before);
  });
  it("blocks double submission while a request is in flight", async () => {
    const storage = store(); let finish!: (response: Response) => void;
    const fetcher = vi.fn().mockImplementation(() => new Promise<Response>((resolve) => { finish = resolve; }));
    const run = createCommandRunner(storage, actor, fetcher);
    const first = run("POST", "suppliers", {});
    await expect(run("POST", "suppliers", {})).rejects.toThrow("đang được xử lý");
    await vi.waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1)); finish(success()); await first;
  });
  it("does not let a late response erase another pending command's retry evidence", async () => {
    const storage = store(); let finish!: (response: Response) => void;
    const fetcher = vi.fn().mockImplementation(() => new Promise<Response>((resolve) => { finish = resolve; }));
    const pending = createCommandRunner(storage, actor, fetcher)("POST", "suppliers", {});
    await vi.waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    const other = JSON.stringify({ key: "procurement-another-command", fingerprint: "a".repeat(64) });
    storage.setItem(commandStorageKey(actor, "POST", "suppliers"), other);
    finish(success()); await pending;
    expect(storage.getItem(commandStorageKey(actor, "POST", "suppliers"))).toBe(other);
  });
  it("does not automatically retry 409, keeps safe error metadata and allows a fresh explicit attempt", async () => {
    const storage = store(), fetcher = vi.fn().mockResolvedValueOnce(Response.json({ code: "CONFLICT", requestId: "conflict-1" }, { status: 409 }));
    const run = createCommandRunner(storage, actor, fetcher);
    await expect(run("POST", "suppliers", { name: "First" })).rejects.toMatchObject({ status: 409, pending: false, requestId: "conflict-1", code: "CONFLICT" });
    expect(fetcher).toHaveBeenCalledTimes(1); expect(storage.data.size).toBe(0);
    fetcher.mockResolvedValueOnce(success()); await run("POST", "suppliers", { name: "Changed" });
    expect(fetcher.mock.calls[0][1].headers["Idempotency-Key"]).not.toBe(fetcher.mock.calls[1][1].headers["Idempotency-Key"]);
  });
  it("keeps unresolved evidence after 5xx and after a subsequent expired session", async () => {
    const storage = store(), fetcher = vi.fn().mockResolvedValueOnce(Response.json({ code: "INTERNAL_ERROR" }, { status: 503 }));
    const run = createCommandRunner(storage, actor, fetcher);
    await expect(run("POST", "suppliers", {})).rejects.toMatchObject({ pending: true, status: 503 });
    fetcher.mockResolvedValueOnce(Response.json({ code: "UNAUTHENTICATED" }, { status: 401 }));
    await expect(run("POST", "suppliers", {})).rejects.toMatchObject({ pending: true, status: 401 });
    expect(storage.data.size).toBe(1);
    expect(fetcher.mock.calls[0][1].headers["Idempotency-Key"]).toBe(fetcher.mock.calls[1][1].headers["Idempotency-Key"]);
  });
  it("treats malformed success as uncertain, not an invitation to create another resource", async () => {
    const storage = store(), fetcher = vi.fn().mockResolvedValue(new Response("broken", { status: 201 }));
    await expect(createCommandRunner(storage, actor, fetcher)("POST", "suppliers", {})).rejects.toMatchObject({ pending: true });
    expect(storage.data.size).toBe(1);
  });
  it("requires a contracted synchronous result with a real UUID before clearing retry evidence", async () => {
    for (const response of [Response.json({}), Response.json({ id: actor }, { status: 202 })]) {
      const storage = store(), fetcher = vi.fn().mockResolvedValue(response);
      await expect(createCommandRunner(storage, actor, fetcher)("POST", "suppliers", {})).rejects.toMatchObject({ pending: true });
      expect(storage.data.size).toBe(1);
    }
  });
  it("fails closed if storage is unavailable or request exceeds 64 KiB", async () => {
    const storage = store(), fetcher = vi.fn();
    storage.setItem = () => { throw new Error("blocked storage"); };
    await expect(createCommandRunner(storage, actor, fetcher)("POST", "suppliers", {})).rejects.toThrow("khóa retry");
    await expect(createCommandRunner(store(), actor, fetcher)("POST", "suppliers", { notes: "x".repeat(65536) })).rejects.toThrow("64 KiB");
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("keeps decimal text significant and rejects unsupported routes", async () => {
    const storage = store(), fetcher = vi.fn().mockRejectedValue(new Error("offline"));
    const run = createCommandRunner(storage, actor, fetcher);
    await expect(run("POST", "suppliers", { quantity: "2.50" })).rejects.toMatchObject({ pending: true });
    await expect(run("POST", "suppliers", { quantity: "2.5" })).rejects.toMatchObject({ code: "RETRY_PAYLOAD_MISMATCH" });
    await expect(run("POST", "supply-plans", {})).rejects.toThrow("không được phép");
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});
