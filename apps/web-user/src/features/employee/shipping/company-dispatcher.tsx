"use client";

import {useRef,useState} from "react";
import {companyRequest,CompanyApiError} from "./company-delivery";
import {assignmentPayload,loadDispatchSnapshot,validDispatchUuid,type CompanyAssignment,type CompanyAssignmentHistory} from "./company-dispatch";

export function CompanyDispatcher(){
  const [shipment,setShipment]=useState("");const [loadedId,setLoadedId]=useState("");
  const [current,setCurrent]=useState<CompanyAssignment|null>(null);const [history,setHistory]=useState<CompanyAssignmentHistory|null>(null);
  const [driver,setDriver]=useState("");const [reason,setReason]=useState("");const [busy,setBusy]=useState(false);const [error,setError]=useState("");const [notice,setNotice]=useState("");
  const pending=useRef<{signature:string;key:string}|null>(null);
  async function read(id:string){
    // History succeeds for a real unassigned INTERNAL shipment; a detail 404 alone
    // cannot establish an initial version for an unknown/non-INTERNAL resource.
    const snapshot=await loadDispatchSnapshot(id);
    setCurrent(snapshot.current);setHistory(snapshot.history);setLoadedId(id);
  }
  async function load(){
    if(busy)return;const id=shipment.trim();if(!validDispatchUuid(id)){setError("Nhập UUID vận đơn hợp lệ.");return;}
    setBusy(true);setError("");setNotice("");setLoadedId("");setCurrent(null);setHistory(null);
    try{await read(id);}catch(e){setError(e instanceof Error?e.message:"Không thể tải phân công.");}finally{setBusy(false);}
  }
  async function save(){
    if(busy||!loadedId||shipment.trim()!==loadedId)return;
    let body;try{body=assignmentPayload(driver.trim(),current?.version??0,reason);}catch(e){setError(e instanceof Error?e.message:"Dữ liệu không hợp lệ.");return;}
    const signature=loadedId+JSON.stringify(body);if(pending.current?.signature!==signature)pending.current={signature,key:crypto.randomUUID()};
    setBusy(true);setError("");setNotice("");
    try{await companyRequest<CompanyAssignment>(`/${loadedId}/assignment`,body,pending.current.key);pending.current=null;setNotice("Đã lưu phân công. Tài xế mới cần nhận chuyến.");setReason("");await read(loadedId);}
    catch(e){setError(e instanceof Error?e.message:"Không thể lưu phân công.");if(e instanceof CompanyApiError&&e.status===409){setLoadedId("");setNotice("Có xung đột: tải lại phân công trước khi tạo yêu cầu mới.");}}
    finally{setBusy(false);}
  }
  return <div className="admin-page"><header className="admin-page-header"><div><h1>Điều phối giao hàng nội bộ</h1><p>Chỉ phân công vận đơn INTERNAL trả trước, trước khi kho bàn giao. Không áp dụng cho GHN.</p></div></header>
    <section className="admin-list-card"><form onSubmit={e=>{e.preventDefault();void load();}}><label>UUID vận đơn<input value={shipment} disabled={busy} onChange={e=>setShipment(e.target.value)} placeholder="UUID từ Order/Fulfillment/Shipping" required/></label><button type="submit" disabled={busy}>Tải phân công</button></form>
      {busy?<p role="status">Đang xử lý…</p>:null}{error?<p role="alert">{error}</p>:null}{notice?<p role="status">{notice}</p>:null}
      {loadedId&&shipment.trim()===loadedId?<><h2>Phân công hiện tại</h2>{current?<dl><dt>UUID tài xế</dt><dd>{current.driver_id}</dd><dt>Phiên bản phân công</dt><dd>{current.version}</dd><dt>Lý do</dt><dd>{current.reason}</dd><dt>Người phân công</dt><dd>{current.assigned_by}</dd></dl>:<p>Chưa có tài xế. Yêu cầu đầu tiên dùng phiên bản 0.</p>}
        <form onSubmit={e=>{e.preventDefault();void save();}}><label>UUID nhân viên nhận chuyến<input disabled={busy} required value={driver} onChange={e=>setDriver(e.target.value)} placeholder="Identity principalId, không phải Keycloak sub"/></label><label>Lý do phân công/đổi tài xế<textarea required disabled={busy} maxLength={500} value={reason} onChange={e=>setReason(e.target.value)}/></label><button disabled={busy||!driver.trim()||!reason.trim()} type="submit">{current?"Đổi tài xế":"Phân công tài xế"}</button></form>
        <p>Cần SHIPMENT_ASSIGN#GLOBAL; thao tác lưu còn cần USER_VIEW#GLOBAL. Backend kiểm tra nhân viên ACTIVE. UUID không tự chứng minh đủ điều kiện. Khi lỗi mạng thử lại cùng dữ liệu; khi 409 phải tải lại.</p>
        <h2>Lịch sử phân công</h2>{history?.items.length?<ol>{history.items.map(item=><li key={item.event_id}>Phiên bản {item.version}: {item.previous_driver_id??"Chưa phân công"} → {item.driver_id} · {item.assigned_at}<p>{item.reason} · Người thao tác: {item.assigned_by}</p></li>)}</ol>:<p>Chưa có lịch sử phân công.</p>}
      </>:null}
    </section><p>Chưa có danh bạ tài xế chuyên dụng; dùng đúng principalId từ Identity. Không gán quyền, thu COD hoặc sửa trạng thái vận đơn tại màn hình này.</p></div>;
}
