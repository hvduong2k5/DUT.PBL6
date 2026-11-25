import {companyRequest,CompanyApiError} from "./company-delivery";

export interface CompanyAssignment {
  shipment_id:string;driver_id:string;assigned_by:string;version:number;reason:string;assigned_at:string;
}
export interface CompanyAssignmentEvent extends CompanyAssignment {event_id:string;previous_driver_id:string|null}
export interface CompanyAssignmentHistory {items:CompanyAssignmentEvent[];total:number}
export async function loadDispatchSnapshot(id:string){
  const history=await companyRequest<CompanyAssignmentHistory>(`/${id}/assignment/history`);
  let current:CompanyAssignment|null=null;
  try{current=await companyRequest<CompanyAssignment>(`/${id}/assignment`);}catch(e){if(!(e instanceof CompanyApiError)||e.status!==404)throw e;}
  if(!current&&history.total>0)throw new Error("Dữ liệu phân công chưa nhất quán. Hãy tải lại.");
  return {current,history};
}
export function validDispatchUuid(value:string){return /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(value)&&value!=="00000000-0000-0000-0000-000000000000";}
export function assignmentPayload(driver:string,version:number,reason:string){
  if(!validDispatchUuid(driver)||!Number.isSafeInteger(version)||version<0||version>=Number.MAX_SAFE_INTEGER||!reason.trim()||Array.from(reason).length>500||/[\u0000-\u001f\u007f-\u009f]/u.test(reason))throw new Error("UUID, phiên bản hoặc lý do phân công không hợp lệ.");
  return {driver_id:driver,expected_version:version,reason};
}

