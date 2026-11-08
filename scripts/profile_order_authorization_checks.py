"""Consumer acceptance against a disposable stack and TEST-only signed claims.

The fixture issuer deliberately stands in for task 1. It does not implement or
prove production IAM-to-Keycloak synchronization. Never reuse its client in dev.
"""

import json
import uuid


def fixture_issuer(request, admin_api, admin_token, issuer, account, password):
    def mint(grants):
        client_id = "test-staff-snapshot-" + uuid.uuid4().hex

        def claim(name, value, kind="String"):
            return {"name": name, "protocol": "openid-connect", "protocolMapper": "oidc-hardcoded-claim-mapper",
                    "config": {"claim.name": name, "claim.value": value, "jsonType.label": kind,
                               "access.token.claim": "true", "id.token.claim": "false", "userinfo.token.claim": "false"}}

        # Keycloak 24.0.5 HardcodedClaim + OIDCAttributeMapperHelper support JSON.
        # Only a dedicated client inside this disposable realm receives claims.
        client = {"clientId": client_id, "enabled": True, "publicClient": True,
                  "protocol": "openid-connect", "standardFlowEnabled": False,
                  "directAccessGrantsEnabled": True, "defaultClientScopes": ["roles", "profile", "email", "audience-scope", "order-customer"],
                  "protocolMappers": [claim("user_id", account["principal"]),
                                      claim("permission_grants", json.dumps(grants), "JSON")]}
        status, _, headers = request(admin_api + "/clients", client, token=admin_token, method="POST")
        assert status == 201, ("TEST-only staff client", status)
        location = headers["Location"]
        try:
            status, body, _ = request(issuer + "/protocol/openid-connect/token",
                                      {"grant_type": "password", "client_id": client_id,
                                       "username": account["email"], "password": password}, form=True)
            assert status == 200, ("TEST-only staff token", status, body.get("error"))
            return body["access_token"]
        finally:
            assert request(location, token=admin_token, method="DELETE")[0] == 204
    return mint


def verify(request, sql, identity, profile, order, legacy_staff, customer, principal, mint):
    staff = mint([])

    def call(base, path, token=None, data=None, method=None, expected=200, headers=None):
        status, body, _ = request(base + path, data, token=staff if token is None else token, method=method, headers=headers)
        assert status == expected, ("authorization", path, status, expected, body)
        return body

    def grant(code, kind="GLOBAL", scope=None):
        # Same wire format Identity syncs onto the Keycloak user attribute:
        # PERMISSION#SCOPE_TYPE, plus #SCOPE_ID for non-GLOBAL grants.
        return code + "#" + kind if scope is None else code + "#" + kind + "#" + scope

    orders = json.loads(sql("SELECT json_agg(json_build_object('id',id,'warehouse',warehouse_id)) FROM orders.orders;", tuples=True))
    assert len(orders) >= 2
    warehouse, foreign_warehouse = orders[0]["warehouse"], "AUTH_OTHER_WAREHOUSE"
    sql("INSERT INTO profile.departments(id,name) VALUES ('AUTH_A','Authorization A'),('AUTH_B','Authorization B');")
    employee_ids = []
    foreign = orders[1]
    try:
        for token in [legacy_staff, customer, staff]:
            call(profile, "/api/v1/admin/employees", token=token, expected=403)
            call(order, "/api/v1/admin/orders", token=token, expected=403)
        staff = mint([grant("EMPLOYEE_PROFILE_VIEW", "DEPARTMENT", "AUTH_A")])
        call(profile, "/api/v1/admin/employees", expected=403)
        grants = [grant(code) for code in ["EMPLOYEE_PROFILE_VIEW", "EMPLOYEE_PROFILE_CREATE", "EMPLOYEE_COMPLIANCE_VIEW"]]
        grants += [grant("ORDER_VIEW", "WAREHOUSE", warehouse),
                   grant("EMPLOYEE_PROFILE_SENSITIVE_VIEW", "DEPARTMENT", "AUTH_B")]
        grants += [grant(code, "WAREHOUSE", foreign_warehouse) for code in ["ORDER_SENSITIVE_VIEW", "ORDER_HOLD_MANAGE", "ORDER_CANCEL_REQUEST"]]
        staff = mint(grants)
        # Customer routes stay closed even if the employee has a legacy customer row.
        for path in ["/api/v1/profile", "/api/v1/customer/capabilities"]:
            call(profile, path, expected=403)
        call(order, "/api/v1/cart", expected=403)
        # A bodyless POST avoids Python 3.14/Windows reset-on-early-denial behavior.
        call(order, "/api/v1/checkout", method="POST", expected=403)
        for department in ["AUTH_A", "AUTH_B"]:
            payload = {"employee_code": "AUTH_" + department, "full_name": "Authorization Fixture",
                       "phone_number": "0901234567", "id_card_number": "012345678901",
                       "department_id": department, "position": "TEST", "contract_type": "FULLTIME",
                       "contract_start_date": "2026-01-01T00:00:00Z", "food_safety_cert_no": "SENSITIVE-CERT",
                       "food_safety_cert_expiry": "2099-01-01T00:00:00Z"}
            created = call(profile, "/api/v1/admin/employees", data=payload, expected=201)
            employee_ids.append(created["id"])
            assert created["phone_number"] == "" and created["food_safety_cert_no"] == ""
        for department, employee in zip(["AUTH_A", "AUTH_B"], employee_ids):
            listed = call(profile, "/api/v1/admin/employees?department_id=" + department + "&limit=1")
            assert [e["id"] for e in listed] == [employee]
            detail = call(profile, "/api/v1/admin/employees/" + employee)
            assert detail["id_card_number"] == "" and detail["phone_number"] == ""
        sql("UPDATE profile.employee_profiles SET food_safety_cert_expiry=CURRENT_DATE+1 WHERE employee_code LIKE 'AUTH_%';")
        compliance = call(profile, "/api/v1/admin/employees/compliance")
        assert all(e["food_safety_cert_no"] == "" for e in compliance if e["id"] in employee_ids)

        sql(f"UPDATE orders.orders SET warehouse_id='{foreign_warehouse}' WHERE id='{uuid.UUID(foreign['id'])}';")
        listed = call(order, "/api/v1/admin/orders?limit=100")["items"]
        assert all(o.get("order_id", o.get("id")) != foreign["id"] for o in listed)
        call(order, "/api/v1/orders/" + foreign["id"], expected=404)
        detail = call(order, "/api/v1/orders/" + orders[0]["id"])
        assert detail["snapshot"]["recipient"]["visibility"] == "MASKED"
        projection_headers = {"Accept": "application/vnd.omama.admin-order+json"}
        projected = call(order, "/api/v1/admin/orders/" + orders[0]["id"], headers=projection_headers)
        assert projected["recipient"] is None and projected["order"]["order_id"] == orders[0]["id"]
        old_staff = staff
        staff = mint(grants + [grant("EMPLOYEE_PROFILE_SENSITIVE_VIEW"), grant("ORDER_SENSITIVE_VIEW", "WAREHOUSE", warehouse)])
        full = call(profile, "/api/v1/admin/employees/" + employee_ids[1])
        assert full["id_card_number"] == "012345678901" and full["phone_number"] == "0901234567"
        assert call(order, "/api/v1/orders/" + orders[0]["id"])["snapshot"]["recipient"]["visibility"] == "FULL"
        assert call(order, "/api/v1/orders/" + orders[0]["id"], token=old_staff)["snapshot"]["recipient"]["visibility"] == "MASKED"
        revoked_snapshot = mint([])
        call(profile, "/api/v1/admin/employees", token=revoked_snapshot, expected=403)
        call(order, "/api/v1/admin/orders", token=revoked_snapshot, expected=403)
        # Old signed snapshot remains valid. This is NOT proof of producer sync.
        call(profile, "/api/v1/admin/employees", token=old_staff)
        call(order, "/api/v1/admin/orders", token=old_staff)
        print("PASS: TEST-issued JWT snapshots, HR GLOBAL across departments, warehouse isolation, masked projections, old/new token semantics; production issuer sync NOT tested", flush=True)
    finally:
        sql(f"UPDATE orders.orders SET warehouse_id='{foreign['warehouse']}' WHERE id='{uuid.UUID(foreign['id'])}';"
            "DELETE FROM profile.employee_profiles WHERE employee_code LIKE 'AUTH_%';"
            "DELETE FROM profile.departments WHERE id IN ('AUTH_A','AUTH_B');")
