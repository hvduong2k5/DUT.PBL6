"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { companyRequest, nextCompanyStatuses, type CompanyDetail, type CompanyList } from "./company-delivery";

export function CompanyWorkbench() {
  const [list,setList]=useState<CompanyList|null>(null);
  const [detail,setDetail]=useState<CompanyDetail|null>(null);
  const [offset,setOffset]=useState(0);
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState("");
  const [reason,setReason]=useState("");
  const pending=useRef<{signature:string;key:string}|null>(null);
  const load=useCallback(async()=>{
    setBusy(true);setError("");setDetail(null);
    try{setList(await companyRequest<CompanyList>(`?limit=50&offset=${offset}`));}catch(e){setError(e instanceof Error?e.message:"Không thể tải chuyến giao");}finally{setBusy(false);}
  },[offset]);
  useEffect(()=>{void Promise.resolve().then(load);},[load]);
  async function open(id:string){setBusy(true);setError("");setDetail(null);setReason("");try{setDetail(await companyRequest<CompanyDetail>(`/${id}`));}catch(e){setError(e instanceof Error?e.message:"Không thể tải chi tiết");}finally{setBusy(false);}}
  async function act(status?:string){
    if(!detail || busy)return;
    const path=`/${detail.shipment_id}/${status?"progress":"accept"}`;
    const body=status?{assignment_version:detail.assignment_version,expected_revision:detail.revision,status,reason}:{expected_version:detail.assignment_version};
    const signature=path+JSON.stringify(body);
    if(pending.current?.signature!==signature)pending.current={signature,key:crypto.randomUUID()};
    setBusy(true);setError("");
    try{
      await companyRequest(path,body,pending.current.key);
      pending.current=null;
      const updated=await companyRequest<CompanyDetail>(`/${detail.shipment_id}`);
      setDetail(updated);setReason("");
      setList(await companyRequest<CompanyList>(`?limit=50&offset=${offset}`));
    }catch(e){setError(e instanceof Error?e.message:"Không thể cập nhật. Thử lại giữ nguyên dữ liệu; nếu 409 hãy làm mới.");}finally{setBusy(false);}
  }
  return <div className="admin-page">
    <header className="admin-page-header"><div><h1>Giao hàng nội bộ</h1><p>Chỉ hiển thị chuyến được phân công cho bạn. Không áp dụng cho GHN hoặc thu COD.</p></div><button disabled={busy} onClick={()=>void load()}>Làm mới</button></header>
    {error?<p role="alert">{error}</p>:null}
    {busy?<p role="status">Đang xử lý…</p>:null}
    <section aria-label="Chuyến được phân công"><ul>{list?.items.map(item=><li key={item.shipment_id}><button disabled={busy} onClick={()=>void open(item.shipment_id)}>{item.shipment_number} — {item.status} — {item.accepted?"Đã nhận":"Chưa nhận"}</button></li>)}</ul>{list?.items.length===0?<p>Không có chuyến ở trang này.</p>:null}
      <button disabled={busy||offset===0} onClick={()=>setOffset(Math.max(0,offset-50))}>Trang trước</button><span> {list?.items.length?offset+1:0} / {list?.total??0} </span><button disabled={busy||!list||offset>=10000||offset+list.items.length>=list.total} onClick={()=>setOffset(offset+50)}>Trang sau</button>
    </section>
    {detail?<section aria-label="Chi tiết chuyến"><h2>{detail.shipment_number}</h2><p>{detail.status} · Revision {detail.revision} · Phân công {detail.assignment_version}</p>
      <p>{detail.recipient.name} · {detail.recipient.phone}</p><p>{[detail.recipient.street,detail.recipient.ward,detail.recipient.district,detail.recipient.province].filter(Boolean).join(", ")}</p><p>{detail.parcel.weight_grams} g · {detail.tracking_code}</p>
      {!detail.accepted&&["CREATED","READY_FOR_HANDOVER"].includes(detail.status)?<button disabled={busy} onClick={()=>void act()}>Nhận chuyến</button>:null}
      {!detail.accepted&&!['CREATED','READY_FOR_HANDOVER'].includes(detail.status)?<p>Chuyến chưa được nhận trước bàn giao. Liên hệ điều phối viên.</p>:null}
      {detail.accepted&&nextCompanyStatuses(detail.status).length>0?<><label>Ghi nhận kết quả/lý do (không phải bằng chứng ảnh đã xác minh)<textarea maxLength={500} disabled={busy} value={reason} onChange={e=>setReason(e.target.value)}/></label>{nextCompanyStatuses(detail.status).map(status=><button key={status} disabled={busy||!reason.trim()} onClick={()=>void act(status)}>{status}</button>)}</>:null}
      {detail.accepted&&["CREATED","READY_FOR_HANDOVER"].includes(detail.status)?<p>Đang chờ kho bàn giao. Tài xế không tự xác nhận bước này.</p>:null}
      <p>Giao thành công/hoàn hàng là xác nhận thủ công; không tự động thu tiền hoặc hoàn tiền. Khi lỗi mạng hãy thử lại cùng thao tác và nội dung. Khi 409 hãy làm mới trước.</p>
    </section>:null}
  </div>;
}
