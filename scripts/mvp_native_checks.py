"""Native app wire acceptance against real Identity/Keycloak, without a device."""
import html
import http.cookiejar
import re
import urllib.parse
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def native_identity(request, gateway, issuer, profile_url, order_url, account, password, employee, retain=False):
    jar = http.cookiejar.CookieJar()
    app = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar), NoRedirect())
    browser = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()), NoRedirect())
    query = urllib.parse.urlencode({"format": "json", "redirect_uri": "omama://auth/callback"})
    status, start, _ = request(gateway + "/api/v1/public/auth/login?" + query, opener=app)
    assert status == 200, "Native login initialization failed"
    authorization = start["data"]["authorizationUrl"]
    parameters = urllib.parse.parse_qs(urllib.parse.urlparse(authorization).query)
    assert parameters["client_id"] == ["mobile-app"] and parameters["redirect_uri"] == ["omama://auth/callback"]
    assert parameters["code_challenge_method"] == ["S256"]
    state = parameters["state"][0]
    status, page, _ = request(authorization, opener=browser)
    assert status == 200, "Native Keycloak authorization page failed"
    form = re.search(r'<form\b[^>]*id="kc-form-login"[^>]*>', page)
    assert form, "Native authorization did not show the real Keycloak login form"
    action = html.unescape(re.search(r'action="([^"]+)"', form[0])[1])
    status, _, headers = request(action, {"username": account["email"], "password": password}, form=True, opener=browser)
    assert status in [302, 303], "Native authorization failed"
    callback = urllib.parse.urlparse(headers["Location"])
    assert callback.scheme == "omama" and callback.netloc == "auth" and callback.path == "/callback"
    values = urllib.parse.parse_qs(callback.query)
    assert values["state"] == [state] and "code" in values
    callback_url = gateway + "/api/v1/public/auth/callback?" + urllib.parse.urlencode({"format": "json", "code": values["code"][0], "state": state})
    assert request(callback_url)[0] == 400, "Another client stole the native authorization transaction"
    status, reply, _ = request(callback_url, opener=app)
    assert status == 200, "Native callback exchange failed"
    tokens = reply["data"]
    assert tokens["accessToken"] and tokens["refreshToken"]
    credential = next(cookie.value for cookie in jar if cookie.name == "OMAMA_SESSION")
    assert tokens["sessionId"] == credential
    status, me, _ = request(gateway + "/api/v1/users/me", token=tokens["accessToken"])
    assert status == 200 and me["data"]["principalId"] == account["principal"]
    assert me["data"]["email"] == account["email"], "Native verified profile is missing its email"
    assert me["data"]["principalType"] == ("EMPLOYEE" if employee else "CUSTOMER")
    for url in [profile_url + "/api/v1/profile", order_url + "/api/v1/cart"]:
        assert request(url, token=tokens["accessToken"], headers={"X-User-Role": "CUSTOMER"})[0] == (403 if employee else 200), "Direct bearer purchase persona gate failed"
    status, refreshed, _ = request(gateway + "/api/v1/public/auth/tokens/refresh", {}, opener=app)
    assert status == 200 and refreshed["data"]["accessToken"] and refreshed["data"]["sessionId"] == credential, "Native session refresh broke the credential"
    if retain:
        return {"accessToken":refreshed["data"]["accessToken"],"refreshToken":tokens["refreshToken"],"sessionId":credential}
    assert request(gateway + "/api/v1/public/auth/logout", {}, opener=app)[0] == 200
    status, session, _ = request(gateway + "/api/v1/public/auth/session", opener=app)
    assert status == 200 and not session["data"]["authenticated"], "Native logout retained a session"
