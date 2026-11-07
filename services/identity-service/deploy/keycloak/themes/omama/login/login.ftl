<#import "template.ftl" as layout>
<#if client?? && client.baseUrl?has_content && client.baseUrl != "/">
    <#assign omamaGuestUrl = client.baseUrl>
<#elseif properties.guestUrl?has_content && !properties.guestUrl?starts_with("$") && properties.guestUrl != "/">
    <#assign omamaGuestUrl = properties.guestUrl>
<#else>
    <#assign omamaGuestUrl = "">
</#if>
<@layout.registrationLayout displayMessage=false displayInfo=false; section>
    <#if section = "header">
        ${msg("loginAccountTitle")}
    <#elseif section = "form" && message?has_content && (message.type!'') == 'success' && (message.summary!'') == msg("emailSentMessage")>

        <#-- ── Top navigation bar ── -->
        <div class="omama-top-nav">
            <a href="${url.loginUrl}" class="omama-top-nav__back" aria-label="${msg('backToLogin')}">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="15 18 9 12 15 6"></polyline>
                </svg>
            </a>
            <span class="omama-top-nav__title">${msg("resetTopTitle")}</span>
        </div>

        <div class="omama-page-icon" aria-hidden="true">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 12.5V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8"/>
                <path d="m3 6 9 6 9-6"/>
                <path d="m16 19 2 2 4-4"/>
            </svg>
        </div>

        <h1 class="omama-section-title">${msg("checkEmailHeading")}</h1>
        <#assign attemptedEmail = (auth.attemptedUsername!'')>
        <p class="omama-section-text" id="omama-check-email-with-addr" <#if !attemptedEmail?has_content>style="display:none"</#if>>
            ${msg("checkEmailPrefix")} <strong id="omama-sent-email">${attemptedEmail}</strong>. ${msg("checkEmailSpamNote")}
        </p>
        <p class="omama-section-text" id="omama-check-email-generic" <#if attemptedEmail?has_content>style="display:none"</#if>>
            ${msg("checkEmailGeneric")} ${msg("checkEmailSpamNote")}
        </p>

        <div class="omama-submit-group">
            <a href="${url.loginUrl}" class="omama-btn omama-btn--primary omama-btn--full">${msg("backToLoginTriKy")}</a>
        </div>

        <p class="omama-back-row">
            ${msg("checkEmailWrongAddress")} <a href="${url.loginResetCredentialsUrl}" class="omama-link omama-link--bold">${msg("checkEmailTryAgain")}</a>
        </p>

    <#elseif section = "form">

        <#-- ── Top Navigation Bar (Mobile Header) ── -->
        <div class="omama-top-nav">
            <a id="omama-login-back" href="${omamaGuestUrl?has_content?then(omamaGuestUrl, '#')}" class="omama-top-nav__back" aria-label="${msg('backToLogin')}">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="15 18 9 12 15 6"></polyline>
                </svg>
            </a>
            <span class="omama-top-nav__title">${msg("loginTopTitle")}</span>
        </div>

        <#-- ── Brand mark (Hoa văn truyền thống Ô Mạ) ── -->
        <div class="omama-brand-mark" aria-hidden="true">
            <svg width="44" height="44" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                <!-- Center circle -->
                <circle cx="24" cy="18" r="3.5" stroke="#563a26" stroke-width="2.4" fill="none"/>
                <!-- 8 Petals radiating -->
                <circle cx="24" cy="9.5" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="24" cy="26.5" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="15.5" cy="18" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="32.5" cy="18" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="18" cy="12" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="30" cy="12" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="18" cy="24" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <circle cx="30" cy="24" r="2.8" stroke="#563a26" stroke-width="2.2" fill="none"/>
                <!-- 2 Leaves cradling the base -->
                <path d="M14 29 C 17 35, 21 37.5, 24 37.5 C 27 37.5, 31 35, 34 29" stroke="#563a26" stroke-width="2.4" stroke-linecap="round" fill="none"/>
                <path d="M11 31.5 C 15 39, 20 41.5, 24 41.5 C 28 41.5, 33 39, 37 31.5" stroke="#563a26" stroke-width="2.4" stroke-linecap="round" fill="none"/>
            </svg>
        </div>

        <#-- ── Heading & Subtitle ── -->
        <h1 class="omama-welcome-title">${msg("loginWelcomeBack")}</h1>
        <p class="omama-welcome-sub">${msg("loginWelcomeSub")}</p>

        <#-- ── Error alert ── -->
        <#if messagesPerField.existsError('username','password')>
            <div class="omama-alert omama-alert--error" role="alert">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <span>${kcSanitize(messagesPerField.getFirstError('username','password'))?no_esc}</span>
            </div>
        <#elseif message?has_content>
            <#-- Non-field messages (session expired, account disabled, ...) -->
            <div class="omama-alert omama-alert--${message.type}" role="alert">
                <span>${kcSanitize(message.summary)?no_esc}</span>
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
                    value="${(login.username!'')}"
                    autofocus
                    aria-label="${msg('usernameOrEmail')}"
                    <#if messagesPerField.existsError('username','password')>aria-invalid="true"</#if>
                />
            </div>

            <#-- ── Password field with lock icon & eye toggle ── -->
            <div class="omama-field-group omama-field-group--password">
                <span class="omama-input-icon omama-input-icon--left" aria-hidden="true">
                    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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
                    <svg id="eye-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
                    </svg>
                    <svg id="eye-off-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none">
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
                        <line x1="1" y1="1" x2="23" y2="23"/>
                    </svg>
                </button>
            </div>

            <#-- ── Forgot password (aligned right) ── -->
            <#if realm.resetPasswordAllowed>
                <div class="omama-forgot-row">
                    <a tabindex="5" href="${url.loginResetCredentialsUrl}" class="omama-link omama-link--forgot">${msg("doForgotPassword")}</a>
                </div>
            </#if>

            <#-- ── Remember me (optional) ── -->
            <#if realm.rememberMe && !usernameEditDisabled??>
                <div class="omama-remember-row">
                    <label class="omama-checkbox-label">
                        <input tabindex="3" id="rememberMe" name="rememberMe" type="checkbox" <#if login.rememberMe??>checked</#if>>
                        <span>${msg("rememberMe")}</span>
                    </label>
                </div>
            </#if>

            <#-- ── Submit Button ── -->
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

            <input type="hidden" id="id-hidden-input" name="credentialId" <#if auth?has_content && auth.selectedCredential?has_content>value="${auth.selectedCredential}"</#if>/>
        </form>

        <#-- ── Register link ── -->
        <#if realm.password && realm.registrationAllowed && !registrationDisabled??>
            <p class="omama-register-row">
                ${msg("noAccount")} <a tabindex="6" href="${url.registrationUrl}" class="omama-link omama-link--bold">${msg("doRegister")}</a>
            </p>
        </#if>

        <#-- ── Continue as guest link ── -->
        <p class="omama-guest-row">
            <a id="omama-guest-link" href="${omamaGuestUrl?has_content?then(omamaGuestUrl, '#')}" class="omama-link omama-link--guest">${msg("continueAsGuest")}</a>
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
