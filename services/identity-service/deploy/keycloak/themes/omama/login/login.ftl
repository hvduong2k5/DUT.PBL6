<#import "template.ftl" as layout>
<@layout.registrationLayout displayMessage=!messagesPerField.existsError('username','password') displayInfo=realm.password && realm.registrationAllowed && !registrationDisabled??; section>
    <#if section = "header">
        ${msg("loginAccountTitle")}
    <#elseif section = "form">

        <#-- ── Brand mark ── -->
        <div class="omama-brand-mark" aria-hidden="true">
            <svg width="44" height="44" viewBox="0 0 44 44" fill="none" xmlns="http://www.w3.org/2000/svg">
                <circle cx="22" cy="16" r="7" stroke="#563a26" stroke-width="2" fill="none"/>
                <path d="M10 36 Q22 24 34 36" stroke="#563a26" stroke-width="2" fill="none" stroke-linecap="round"/>
                <circle cx="22" cy="9" r="2.5" fill="#563a26"/>
                <path d="M15 16 Q16 20 22 22 Q28 20 29 16" stroke="#d4a24c" stroke-width="1.5" fill="none" stroke-linecap="round"/>
            </svg>
        </div>

        <#-- ── Heading ── -->
        <h1 class="omama-welcome-title">${msg("loginWelcomeBack")}</h1>
        <p class="omama-welcome-sub">${msg("loginWelcomeSub")}</p>

        <#-- ── Error alert ── -->
        <#if messagesPerField.existsError('username','password')>
            <div class="omama-alert omama-alert--error" role="alert">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <span>${kcSanitize(messagesPerField.getFirstError('username','password'))?no_esc}</span>
            </div>
        </#if>

        <form id="kc-form-login" class="${properties.kcFormClass!}" onsubmit="login.disabled = true; return true;" action="${url.loginAction}" method="post">

            <#-- ── Phone / Username field ── -->
            <div class="omama-field-group">
                <input
                    tabindex="1"
                    id="username"
                    name="username"
                    class="omama-input"
                    type="text"
                    autocomplete="username"
                    placeholder="${msg('usernameOrEmail')}"
                    value="${(login.username!'')?html}"
                    autofocus
                    aria-label="${msg('usernameOrEmail')}"
                    <#if messagesPerField.existsError('username','password')>aria-invalid="true"</#if>
                />
            </div>

            <#-- ── Password field with toggle ── -->
            <div class="omama-field-group omama-field-group--password">
                <span class="omama-input-icon omama-input-icon--left" aria-hidden="true">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                    </svg>
                </span>
                <input
                    tabindex="2"
                    id="password"
                    name="password"
                    class="omama-input omama-input--has-icon-left omama-input--has-icon-right"
                    type="password"
                    autocomplete="current-password"
                    placeholder="${msg('password')}"
                    aria-label="${msg('password')}"
                    <#if messagesPerField.existsError('username','password')>aria-invalid="true"</#if>
                />
                <button
                    type="button"
                    class="omama-password-toggle"
                    aria-label="${msg('showPassword')}"
                    onclick="togglePassword()"
                    tabindex="-1"
                >
                    <svg id="eye-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
                    </svg>
                    <svg id="eye-off-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none">
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
                        <line x1="1" y1="1" x2="23" y2="23"/>
                    </svg>
                </button>
            </div>

            <#-- ── Forgot password ── -->
            <#if realm.resetPasswordAllowed>
                <div class="omama-forgot-row">
                    <a tabindex="5" href="${url.loginResetCredentialsUrl}" class="omama-link omama-link--forgot">${msg("doForgotPassword")}</a>
                </div>
            </#if>

            <#-- ── Remember me (hidden by default, only shown if realm enables it) ── -->
            <#if realm.rememberMe && !usernameEditDisabled??>
                <div class="omama-remember-row">
                    <label class="omama-checkbox-label">
                        <input tabindex="3" id="rememberMe" name="rememberMe" type="checkbox" <#if login.rememberMe??>checked</#if>>
                        <span>${msg("rememberMe")}</span>
                    </label>
                </div>
            </#if>

            <#-- ── Submit ── -->
            <div class="omama-submit-group">
                <input
                    tabindex="4"
                    id="kc-login"
                    name="login"
                    class="omama-btn omama-btn--primary omama-btn--full"
                    type="submit"
                    value="${msg('doLogIn')}"
                />
            </div>

            <input type="hidden" id="id-hidden-input" name="credentialId" <#if auth.selectedCredential?has_content>value="${auth.selectedCredential}"</#if>/>
        </form>

        <#-- ── Register link ── -->
        <#if realm.password && realm.registrationAllowed && !registrationDisabled??>
            <p class="omama-register-row">
                ${msg("noAccount")} <a tabindex="6" href="${url.registrationUrl}" class="omama-link">${msg("doRegister")}</a>
            </p>
        </#if>

        <#-- ── Guest link ── -->
        <p class="omama-guest-row">
            <a href="${properties.guestUrl!'/'}" class="omama-link omama-link--muted">${msg("continueAsGuest")}</a>
        </p>

        <#-- ── Password toggle script ── -->
        <script>
            function togglePassword() {
                var input = document.getElementById('password');
                var eyeOn  = document.getElementById('eye-icon');
                var eyeOff = document.getElementById('eye-off-icon');
                if (input.type === 'password') {
                    input.type = 'text';
                    eyeOn.style.display  = 'none';
                    eyeOff.style.display = 'inline';
                } else {
                    input.type = 'password';
                    eyeOn.style.display  = 'inline';
                    eyeOff.style.display = 'none';
                }
            }
        </script>

    </#if>
</@layout.registrationLayout>
