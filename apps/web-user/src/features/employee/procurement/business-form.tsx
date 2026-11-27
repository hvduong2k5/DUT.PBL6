"use client";

import { useId, useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { buildPayload, type Draft, type Field, type FormSpec, type RecordData } from "./forms";
import { CommandError, createCommandRunner } from "./commands";

export function BusinessForm({ spec, onSuccess, onCancel }: { spec: FormSpec; onSuccess: (result: RecordData) => Promise<void>; onCancel: () => void }) {
  const { session } = useEmployeeSession();
  const formID = useId();
  const [draft, setDraft] = useState<Draft>(() => ({ values: { ...spec.initial }, lines: spec.initialLines.map((line) => ({ ...line })) }));
  const [busy, setBusy] = useState(false), [pending, setPending] = useState(false), [conflict, setConflict] = useState(false);
  const [failure, setFailure] = useState<CommandError>();
  const [completed, setCompleted] = useState(false);
  const runner = useRef<ReturnType<typeof createCommandRunner> | null>(null);
  const inFlight = useRef(false);
  const inputLocked = busy || completed || conflict || pending && failure?.code !== "RETRY_PAYLOAD_MISMATCH";
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (inFlight.current || completed || conflict && !pending) return;
    inFlight.current = true; setBusy(true); setFailure(undefined);
    try {
      const payload = buildPayload(spec, draft);
      if (!runner.current) runner.current = createCommandRunner(window.sessionStorage, session.employee.employeeId);
      const result = await runner.current(spec.method, spec.path, payload);
      setCompleted(true); setPending(false);
      // Replays may be frozen snapshots. Parent must GET current state before
      // rendering any next-step controls, never treat this result as current.
      try { await onSuccess(result); }
      catch { setFailure(new CommandError("Yêu cầu đã được ghi nhận, nhưng chưa tải lại được dữ liệu. Không gửi lại; hãy dùng nút tải lại.")); }
    } catch (cause) {
      const error = cause instanceof CommandError ? cause : new CommandError(cause instanceof Error ? cause.message : "Không thể gửi yêu cầu.");
      setFailure(error); setPending(error.pending); setConflict(error.status === 409);
    } finally { inFlight.current = false; setBusy(false); }
  }
  function input(field: Field, value: string, update: (value: string) => void, suffix: string) {
    const common = { id: `proc-${formID}-${spec.action}-${suffix}-${field.name}`, value: value ?? "", onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => update(event.target.value), required: field.required && !field.allowEmpty, disabled: inputLocked, readOnly: field.locked, maxLength: field.maxLength ?? 4000, "aria-describedby": field.hint ? `proc-hint-${formID}-${suffix}-${field.name}` : undefined };
    return <label key={field.name} className="proc-field" htmlFor={common.id}><span>{field.label}{field.required && !field.allowEmpty ? " *" : ""}</span>
      {field.options || field.type === "boolean" ? <select {...common} disabled={common.disabled || field.locked}><option value="">Chọn…</option>{(field.options ?? ["true", "false"]).map((option) => <option key={option} value={option}>{option === "true" ? "Bật" : option === "false" ? "Tắt" : option}</option>)}</select>
        : field.type === "textarea" ? <textarea {...common} rows={3} />
        : <input {...common} type={field.type === "date" ? "date" : field.type === "email" ? "email" : "text"} inputMode={field.type === "quantity" ? "decimal" : field.type === "integer" ? "numeric" : undefined} />}
      {field.hint && <small id={`proc-hint-${formID}-${suffix}-${field.name}`}>{field.hint}</small>}
    </label>;
  }
  const setValue = (name: string, value: string) => setDraft((old) => ({ ...old, values: { ...old.values, [name]: value } }));
  return <section className="proc-card" aria-label={spec.title}>
    <h3>{spec.title}</h3>
    <p>Kiểm tra kỹ trước khi xác nhận. Quyền, phạm vi và trạng thái được backend kiểm tra lại.</p>
    {spec.action === "po-send" && <p>Thao tác này ghi nhận nhân viên đã gửi PO; hệ thống không tự gửi email cho nhà cung cấp.</p>}
    {spec.action === "po-close" && <p>Đóng phần chưa nhận còn lại. Không xóa lượng đã nhận, không hoàn tiền hay điều chỉnh tồn kho.</p>}
    {spec.action === "receipt-inspect" && <p>Chấp nhận ở Procurement không đồng nghĩa hàng được bán: Inventory vẫn quản lý QUARANTINE và QC riêng.</p>}
    {spec.action.startsWith("contract-") && <p>Chỉ quản lý dữ liệu hợp đồng; chưa hỗ trợ tải tệp hoặc ký điện tử.</p>}
    <form onSubmit={(event) => void submit(event)}>
      <div className="proc-fields">{spec.fields.map((field) => input(field, field.locked ? spec.initial[field.name] : draft.values[field.name], (value) => setValue(field.name, value), "header"))}</div>
      {spec.lineFields && <fieldset disabled={inputLocked}><legend>Các dòng chi tiết</legend>
        {draft.lines.map((line, index) => <fieldset key={index} className="proc-line"><legend>Dòng {index + 1}</legend>
          {spec.action === "receipt-inspect" && <p>Thực nhận: {String((spec.source.lines as RecordData[])[index].quantity_received)}; tổng chấp nhận + từ chối phải khớp.</p>}
          <div className="proc-fields">{spec.lineFields!.map((field) => input(field, field.locked ? spec.initialLines[index]?.[field.name] : line[field.name], (value) => setDraft((old) => ({ ...old, lines: old.lines.map((item, row) => row === index ? { ...item, [field.name]: value } : item) })), `line-${index}`))}</div>
          {!spec.fixedLines && <button type="button" disabled={draft.lines.length <= 1} onClick={() => setDraft((old) => ({ ...old, lines: old.lines.filter((_, row) => row !== index) }))}>Bỏ dòng {index + 1}</button>}
        </fieldset>)}
        {!spec.fixedLines && <button type="button" disabled={draft.lines.length >= 100} onClick={() => setDraft((old) => ({ ...old, lines: [...old.lines, {}] }))}>Thêm dòng</button>}
      </fieldset>}
      <div aria-live="polite">{failure && <p role="alert">{failure.message} {failure.code && `Mã lỗi: ${failure.code}.`} {failure.requestId && `Mã yêu cầu: ${failure.requestId}.`}</p>}
        {pending && <p>Khóa và dấu vân tay yêu cầu được giữ trong phiên trình duyệt, không lưu nội dung nghiệp vụ. Chỉ thử lại đúng dữ liệu cũ; đừng tạo lại ở tab khác.</p>}
        {completed && <p role="status">Yêu cầu đã được ghi nhận. Không gửi lại.</p>}
      </div>
      <div className="proc-actions"><button type="submit" disabled={busy || completed || conflict && !pending}>{busy ? "Đang xử lý…" : pending ? "Thử lại đúng yêu cầu" : "Xác nhận thao tác"}</button>
        <button type="button" disabled={busy} onClick={onCancel}>{pending ? "Đóng form (giữ nhật ký retry)" : completed ? "Đóng" : conflict ? "Đóng và tải lại dữ liệu" : "Hủy nhập"}</button></div>
    </form>
  </section>;
}
