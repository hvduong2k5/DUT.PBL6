"""Authorization acceptance against the real disposable IAM/Profile/Order stack."""

import json
import uuid


def verify(request, sql, identity, profile, order, staff, customer, principal):
    def call(base, path, token=staff, data=None, method=None, expected=200, headers=None):
        status, body, response_headers = request(base + path, data, token=token, method=method, headers=headers)
        assert status == expected, ("authorization", path, status, expected, body)
        return body

    def rows(query):
        return json.loads(sql(query, tuples=True))

    # Test assignments use their own roles; no existing Fulfillment grant changes.
    role, other_role = str(uuid.uuid4()), str(uuid.uuid4())
    assignment, other_assignment = str(uuid.uuid4()), str(uuid.uuid4())
    departments = ["AUTH_A", "AUTH_B"]
    orders = rows("SELECT json_agg(json_build_object('id',id,'warehouse',warehouse_id)) FROM orders.orders;")
    assert len(orders) >= 2
    warehouse = orders[0]["warehouse"]
    foreign_warehouse = "AUTH_OTHER_WAREHOUSE"
    # The full packing acceptance grants this employee ADMIN. Temporarily
    # remove pre-existing assignments so they cannot hide scope regressions.
    existing = rows(f"SELECT COALESCE(json_agg(assignment_id),'[]'::json) FROM identity.iam_role_assignments WHERE principal_id='{principal}' AND assignment_status='ACTIVE';")
    sql(f"""INSERT INTO identity.iam_roles(role_id,role_code,role_name) VALUES
        ('{role}','PROFILE_ORDER_AUTH_A','Authorization fixture A'),
        ('{other_role}','PROFILE_ORDER_AUTH_B','Authorization fixture B');
        INSERT INTO profile.departments(id,name) VALUES ('AUTH_A','Authorization A'),('AUTH_B','Authorization B');""")
    employee_ids = []
    try:
        for assigned in existing:
            sql(f"UPDATE identity.iam_role_assignments SET assignment_status='REVOKED' WHERE assignment_id='{uuid.UUID(assigned)}';")
        # A missing permission cannot use a signed realm role or forwarded header.
        call(profile, "/api/v1/admin/employees", expected=403)
        call(order, "/api/v1/admin/orders", expected=403)
        call(profile, "/api/v1/admin/employees", token=customer, expected=403)
        call(order, "/api/v1/admin/orders", token=customer, expected=403)
        sql(f"""INSERT INTO identity.iam_role_permissions(role_id,permission_code)
            SELECT '{role}',permission_code FROM identity.iam_permissions WHERE permission_code IN
            ('EMPLOYEE_PROFILE_VIEW','EMPLOYEE_PROFILE_CREATE','EMPLOYEE_COMPLIANCE_VIEW','ORDER_VIEW');
            INSERT INTO identity.iam_role_permissions(role_id,permission_code)
            SELECT '{other_role}',permission_code FROM identity.iam_permissions WHERE permission_code IN
            ('EMPLOYEE_PROFILE_SENSITIVE_VIEW','ORDER_SENSITIVE_VIEW','ORDER_HOLD_MANAGE');
            INSERT INTO identity.iam_role_assignments(assignment_id,principal_id,role_id,scope_type,scope_id) VALUES
            ('{assignment}','{principal}','{role}','DEPARTMENT','AUTH_A'),
            ('{other_assignment}','{principal}','{other_role}','DEPARTMENT','AUTH_B'),
            ('{uuid.uuid4()}','{principal}','{role}','WAREHOUSE','{warehouse}'),
            ('{uuid.uuid4()}','{principal}','{other_role}','WAREHOUSE','{foreign_warehouse}');""")
        me = call(identity, "/api/v1/users/me")["data"]
        assert any(g["permissionCode"] == "ORDER_VIEW" and g["scopeId"] == warehouse for g in me["permissionGrants"])
        # Customer routes stay closed even though this synthetic employee has a
        # legacy customer row in the main integration fixture.
        call(profile, "/api/v1/profile", expected=403)
        call(profile, "/api/v1/customer/capabilities", expected=403)
        call(order, "/api/v1/cart", expected=403)
        call(order, "/api/v1/checkout", data={}, expected=403)
        for department in departments:
            payload = {"employee_code": "AUTH_" + department, "full_name": "Authorization Fixture",
                       "phone_number": "0901234567", "id_card_number": "012345678901",
                       "department_id": department, "position": "TEST", "contract_type": "FULLTIME",
                       "contract_start_date": "2026-01-01T00:00:00Z", "food_safety_cert_no": "SENSITIVE-CERT",
                       "food_safety_cert_expiry": "2099-01-01T00:00:00Z"}
            if department == "AUTH_B":
                call(profile, "/api/v1/admin/employees", data=payload, expected=403)
                # Seed the foreign object using its own temporary valid grant.
                sql(f"UPDATE identity.iam_role_assignments SET scope_id='AUTH_B' WHERE assignment_id='{assignment}';")
            created = call(profile, "/api/v1/admin/employees", data=payload, expected=201)
            employee_ids.append(created["id"])
            if department == "AUTH_B":
                sql(f"UPDATE identity.iam_role_assignments SET scope_id='AUTH_A' WHERE assignment_id='{assignment}';")
            else:
                assert created["phone_number"] == "" and created["food_safety_cert_no"] == ""
        # Scope predicates must execute before LIMIT/OFFSET.
        listed = call(profile, "/api/v1/admin/employees?limit=1")
        assert [e["id"] for e in listed] == [employee_ids[0]]
        assert call(profile, "/api/v1/admin/employees?limit=1&offset=1") in ([], None)
        call(profile, "/api/v1/admin/employees?department_id=AUTH_B", expected=403)
        detail = call(profile, "/api/v1/admin/employees/" + employee_ids[0])
        assert detail["phone_number"] == "" and detail["id_card_number"] == "" and detail["food_safety_cert_no"] == ""
        call(profile, "/api/v1/admin/employees/" + employee_ids[1], expected=404)
        sql("UPDATE profile.employee_profiles SET food_safety_cert_expiry=CURRENT_DATE+1 WHERE employee_code LIKE 'AUTH_%';")
        compliance = call(profile, "/api/v1/admin/employees/compliance")
        assert [e["id"] for e in compliance] == [employee_ids[0]]
        assert compliance[0]["food_safety_cert_no"] == ""

        # Reuse real placed orders, and restore the foreign row even on failure.
        foreign = orders[1]
        sql(f"UPDATE orders.orders SET warehouse_id='{foreign_warehouse}' WHERE id='{foreign['id']}';")
        try:
            listed = call(order, "/api/v1/admin/orders?limit=100")["items"]
            assert all(o.get("order_id", o.get("id")) != foreign["id"] for o in listed)
            detail = call(order, "/api/v1/orders/" + orders[0]["id"])
            assert detail["snapshot"]["recipient"]["visibility"] == "MASKED"
            call(order, "/api/v1/orders/" + foreign["id"], expected=404)
            call(order, "/api/v1/admin/orders/" + orders[0]["id"] + "/hold", data={"active": True},
                 headers={"If-Match": '"1"', "Idempotency-Key": uuid.uuid4().hex}, expected=404)
            # Permission A on one assignment cannot borrow a sensitive scope B.
            sql(f"""INSERT INTO identity.iam_role_permissions(role_id,permission_code) VALUES
                ('{role}','EMPLOYEE_PROFILE_SENSITIVE_VIEW'),('{role}','ORDER_SENSITIVE_VIEW');""")
            full = call(profile, "/api/v1/admin/employees/" + employee_ids[0])
            assert full["id_card_number"] == "012345678901" and full["phone_number"] == "0901234567"
            full = call(order, "/api/v1/orders/" + orders[0]["id"])
            assert full["snapshot"]["recipient"]["visibility"] == "FULL"
        finally:
            sql(f"UPDATE orders.orders SET warehouse_id='{foreign['warehouse']}' WHERE id='{foreign['id']}';")

        # Authoritative grants are re-read on every request, without token refresh.
        sql(f"UPDATE identity.iam_roles SET is_enabled=false WHERE role_id='{role}';")
        call(profile, "/api/v1/admin/employees", expected=403)
        call(order, "/api/v1/admin/orders", expected=403)
        sql(f"UPDATE identity.iam_roles SET is_enabled=true WHERE role_id='{role}';")
        sql(f"UPDATE identity.iam_role_assignments SET valid_from=now()-interval '2 days',valid_until=now()-interval '1 day' WHERE role_id='{role}';")
        call(profile, "/api/v1/admin/employees", expected=403)
        call(order, "/api/v1/admin/orders", expected=403)
        sql(f"UPDATE identity.iam_role_assignments SET valid_until=NULL,assignment_status='REVOKED' WHERE role_id='{role}';")
        call(profile, "/api/v1/admin/employees", expected=403)
        call(order, "/api/v1/admin/orders", expected=403)
        print("PASS: Profile/Order IAM permissions, department/warehouse scopes before pagination, cross-role isolation, sensitive redaction, persona gates, role disable, expiry and revocation", flush=True)
    finally:
        sql(f"DELETE FROM identity.iam_role_assignments WHERE role_id IN ('{role}','{other_role}');"
            f"DELETE FROM identity.iam_role_permissions WHERE role_id IN ('{role}','{other_role}');"
            f"DELETE FROM identity.iam_roles WHERE role_id IN ('{role}','{other_role}');"
            "DELETE FROM profile.employee_profiles WHERE employee_code LIKE 'AUTH_%';"
            "DELETE FROM profile.departments WHERE id IN ('AUTH_A','AUTH_B');")
        for assigned in existing:
            sql(f"UPDATE identity.iam_role_assignments SET assignment_status='ACTIVE' WHERE assignment_id='{uuid.UUID(assigned)}';")
