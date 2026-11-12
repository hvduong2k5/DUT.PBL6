"""Generate the provisional consumer handoff without editing legacy inferred examples."""
import copy
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[3]
S = {"type": "string", "maxLength": 4096}
B = {"type": "boolean"}
N = {"type": "integer", "minimum": 0, "maximum": 9007199254740991}
DATE = {"type": "string", "format": "date-time"}
CODE = {"type": "string", "pattern": "^[A-Za-z][A-Za-z0-9_.:-]{0,159}$"}

def enum(*values): return {"type": "string", "enum": list(values)}
def array(item, maximum=1000): return {"type": "array", "maxItems": maximum, "items": item}
def nullable(item): return {"anyOf": [item, {"type": "null"}]}
def obj(**fields): return {"type": "object", "required": list(fields), "properties": fields}
def ref(name): return {"$ref": "#/components/schemas/" + name}

account = enum("INVITED", "PENDING_ACTIVATION", "ACTIVE", "LOCKED", "DISABLED")
role_status = enum("ACTIVE", "INACTIVE", "ARCHIVED")
permission_status = enum("ACTIVE", "DEPRECATED", "ARCHIVED")
assignment = enum("SCHEDULED", "ACTIVE", "EXPIRED", "REVOKED")
revision = dict(N, minimum=1)
summary = dict(accountId=S, employeeId=S, employeeCode=S, displayName=S, email=S, department=S,
               jobTitle=S, loginIdentifier=S, status=account, statusLabel=S,
               invitationStatus=enum("NOT_SENT", "PENDING", "DELIVERED", "FAILED", "ACCEPTED"),
               roles=array(obj(roleCode=S, roleName=S, status=assignment)), effectivePermissionCodes=array(CODE),
               accessState=enum("ENABLED", "BLOCKED_ACCOUNT", "NO_ACTIVE_ROLE"), accessStateLabel=S, updatedAt=DATE, revision=revision)
role = dict(roleCode=S, roleName=S, description=S, scopeLabel=S, status=role_status, statusLabel=S,
            kind=enum("SYSTEM", "CUSTOM"), privileged=B, employeeCount=N, permissionCount=N, updatedAt=DATE, revision=revision)
permission = dict(permissionCode=CODE, permissionName=S, description=S, action=S, resource=S, scopeLabel=S,
                  status=permission_status, statusLabel=S, kind=enum("SYSTEM", "CONFIGURABLE"), roleCount=N, employeeCount=N, updatedAt=DATE)
def counts(status): return array(obj(status=enum("ALL", *status["enum"]), label=S, count=N))
schemas = {
    "EmployeeAccessList": obj(items=array(obj(**summary)), statusCounts=counts(account), departments=array(S),
                              roles=array(obj(roleCode=S, roleName=S)), permissions=array(obj(permissionCode=CODE, permissionName=S)), calculatedAt=DATE),
    "EmployeeAccessDetail": obj(**summary, profileReference=obj(employeeId=S, employeeCode=S, profileOwner={"const": "EPIC_02"}),
        identity=obj(emailVerified=B, lastAuthenticatedAt=nullable(DATE), credentialReadable={"const": False}),
        lock=nullable(obj(effectiveAt=DATE, reason=S, decidedByLabel=S)),
        roleAssignments=array(obj(assignmentId=S, roleCode=S, roleName=S, roleStatus=role_status, assignmentStatus=assignment,
                                  scopeLabel=S, effectiveFrom=DATE, effectiveUntil=nullable(DATE), assignedByLabel=S)),
        effectivePermissions=array(obj(permissionCode=CODE, permissionName=S, action=S, resource=S, scopeLabels=array(S),
                                       sources=array(obj(roleCode=S, roleName=S, assignmentId=S)))), calculatedAt=DATE),
    "AdminRoleList": obj(items=array(obj(**role)), statusCounts=counts(role_status), calculatedAt=DATE),
    "AdminRoleDetail": obj(**role, protectedRole=B, permissions=array(obj(permissionCode=CODE, permissionName=S, action=S, resource=S, scopeLabel=S)),
                           impact=obj(activeEmployees=N, scheduledAssignments=N, lockedAccounts=N)),
    "AdminPermissionList": obj(items=array(obj(**permission)), statusCounts=counts(permission_status), resources=array(S), calculatedAt=DATE),
    "AdminPermissionDetail": obj(**permission, protectedCode=B, usedByModules=array(S), rolesUsing=array(obj(roleCode=S, roleName=S, status=role_status, employeeCount=N))),
}
uuid = {"type": "string", "format": "uuid"}
order = obj(order_id=uuid, order_code=S, status=S, payment_status=S, payment_method=S, stock_status=S, warehouse_id=S,
            customer_id=nullable(uuid), task_id=nullable(S), shipment_id=nullable(S), final_amount=N, item_count=N,
            version=revision, operational_hold=B, has_active_case=B, sla_due_at=nullable(DATE), created_at=DATE, updated_at=DATE)
schemas.update({
    "OrderSummary": order,
    "OrderPage": obj(items=array(ref("OrderSummary"),100), next_cursor=nullable({"type":"string","maxLength":2048}), calculated_at=DATE),
    "OrderDetail": obj(order=ref("OrderSummary"), subtotal=N, shipping_fee=N, discount=N, calculated_at=DATE,
                       recipient=nullable(obj(name=S, phone=S, address=S)),
                       lines=array(obj(sku_code=S, product_name=S, quantity=revision, unit_price=N, line_total=N)),
                       history=array(obj(event_id=uuid, from_status=nullable(S), to_status=S, occurred_at=DATE),50)),
    "Error": obj(code=S, message=S, requestId=S, errors=array(obj(field=S,message=S))),
    "LegacyOrderError": obj(error=obj(code=S,retryable=B)),
})
paths = {}
errors = {str(status): {"description": description, "content": {"application/json": {"schema": ref("Error")}}}
          for status, description in [(400,"Invalid query"),(401,"Missing or invalid authentication"),(403,"Insufficient principal/permission/scope"),
                                      (404,"Resource hidden or missing; IAM producer includes X-IAM-Read-Contract"),(429,"Rate limited"),
                                      (502,"Invalid or oversized upstream projection"),(503,"Dependency or projection capability unavailable")]}
def operation(name, schema, permission, cookie=False, **extra):
    return dict(operationId=name, summary=name, tags=["Consumer read handoff"],
                **{"x-lifecycle":"draft","x-required-permission":permission,
                   "security":[{"adminSession" if cookie else "bearerAuth": []}]},
                responses={"200":{"description":"Read projection; no-store. Empty collections use empty arrays.",
                                  "content":{"application/json":{"schema":schema}}}, **copy.deepcopy(errors)}, **extra)

for segment, list_name, detail_name, permission_code in [
    ("employees","EmployeeAccessList","EmployeeAccessDetail","EMPLOYEE_ACCOUNT_VIEW"),
    ("roles","AdminRoleList","AdminRoleDetail","ROLE_VIEW"),
    ("permissions","AdminPermissionList","AdminPermissionDetail","PERMISSION_VIEW")]:
    for detail, data_name in [(False,list_name),(True,detail_name)]:
        path = "/api/v1/admin/access/"+segment+("/{id}" if detail else "")
        envelope = obj(contractVersion={"const":"iam-admin-read.v1"}, capabilities=obj(accessReview=B), data=ref(data_name))
        extra = {"parameters":[{"name":"id","in":"path","required":True,"schema":CODE}]} if detail else {}
        op = operation("read"+data_name,envelope,permission_code,cookie=True,**extra)
        op["description"] = "Pending task 2 producer. Accept application/vnd.omama.iam-admin-read.v1+json. Current configured access, not live JWT inventory. No query parameters or mutations."
        if segment=="employees" and detail: op["x-required-permission"] += " + ACCESS_REVIEW_VIEW"
        paths[path]={"get":op}
for suffix, schema_name in [("","OrderPage"),("/queue","OrderPage"),("/{id}","OrderDetail")]:
    params = [{"name":"id","in":"path","required":True,"schema":uuid}] if suffix=="/{id}" else [
        {"name":"limit","in":"query","schema":{"type":"integer","minimum":1,"maximum":100,"default":25}},
        {"name":"status","in":"query","schema":{"type":"string","maxLength":64}},
        {"name":"cursor","in":"query","schema":{"type":"string","maxLength":2048}}]
    op=operation("readOrder"+("Queue" if suffix=="/queue" else schema_name),ref(schema_name),"ORDER_VIEW",parameters=params)
    op["description"]="Provisional staff JWT v1. GLOBAL/WAREHOUSE permission scope. Accept application/vnd.omama.admin-order+json selects projection; legacy JSON list unchanged. Recipient requires ORDER_SENSITIVE_VIEW for the same warehouse. Issuer rollout pending task 1."
    for status in errors:
        op["responses"][status]["content"]["application/json"]["schema"] = ref("LegacyOrderError")
    paths["/api/v1/admin/orders"+suffix]={"get":op}
    proxy=copy.deepcopy(op);proxy["operationId"]+="ViaIdentity";proxy["security"]=[{"adminSession":[]}]
    proxy["description"]="Read-only cookie-to-server-bearer proxy; fixed Order target and Accept. Identity retains its existing ACTIVE employee gate. No browser bearer or internal headers forwarded."
    for status in errors:
        proxy["responses"][status]["content"]["application/json"]["schema"] = {"oneOf":[ref("Error"),ref("LegacyOrderError")]}
    paths["/api/v1/admin/order-api/orders"+suffix]={"get":proxy}

doc={"openapi":"3.1.0","info":{"title":"Profile/Order and IAM consumer read handoff","version":"0.1.0",
      "description":"Generated by packages/contracts/iam/generate_read_contract.py. Provisional integration contract; producer tasks 1–2 pending. Not evidence of deployed dev capability.",
      "license":{"name":"Proprietary","identifier":"LicenseRef-Proprietary"}},
     "servers":[{"url":"https://api.omama.vn"}],"tags":[{"name":"Consumer read handoff","description":"JWT and read projection integration"}],
     "paths":paths,"components":{"securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer","bearerFormat":"JWT"},
          "adminSession":{"type":"apiKey","in":"cookie","name":"OMAMA_SESSION"}},"schemas":schemas}}
(ROOT/"docs/03_api_specs/iam-consumer-read.openapi.yaml").write_text(yaml.safe_dump(doc,sort_keys=False,allow_unicode=True),encoding="utf-8")
