<#import "template.ftl" as layout>
<@layout.registrationLayout displayInfo=false displayMessage=false; section>
    <#if section = "header">
        ${msg("emailForgotTitle")}
    <#elseif section = "form">

        <#-- ── Top navigation bar ── -->
        <div class="omama-top-nav">
            <a href="${url.loginUrl}" class="omama-top-nav__back" aria-label="${msg('backToLogin')}">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="15 18 9 12 15 6"></polyline>
                </svg>
            </a>
            <span class="omama-top-nav__title">${msg("resetTopTitle")}</span>
        </div>

        <#-- ── Mail icon ── -->
        <div class="omama-page-icon" aria-hidden="true">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#563a26" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 12.5V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8"/>
                <path d="m3 6 9 6 9-6"/>
                <path d="m16 19 2 2 4-4"/>
            </svg>
        </div>

        <h1 class="omama-section-title">${msg("resetHeading")}</h1>
        <p class="omama-section-text">${msg("resetInstruction")}</p>

        <form id="omama-reset-form" action="${url.loginAction}" method="post" novalidate>
            <div class="omama-field-group">
                <input
                    type="email"
                    id="username"
                    name="username"
                    class="omama-input<#if messagesPerField.existsError('username')> omama-input--error</#if>"
                    autocomplete="email"
                    autofocus
                    value="${(auth.attemptedUsername!'')}"
                    placeholder="${msg('resetEmailPlaceholder')}"
                    aria-label="${msg('resetEmailPlaceholder')}"
                    aria-describedby="omama-email-error"
                    <#if messagesPerField.existsError('username')>aria-invalid="true"</#if>
                />
                <span id="omama-email-error" class="omama-field-error" aria-live="polite"
                      data-invalid-message="${msg('resetEmailInvalid')}"><#if messagesPerField.existsError('username')>${kcSanitize(messagesPerField.get('username'))?no_esc}</#if></span>
            </div>

            <div class="omama-submit-group">
                <input id="kc-reset-password-submit" class="omama-btn omama-btn--primary omama-btn--full" type="submit" value="${msg('resetDoSubmit')}"/>
            </div>
        </form>

        <p class="omama-back-row">
            <a href="${url.loginUrl}" class="omama-link omama-link--guest">${msg("backToLoginTriKy")}</a>
        </p>

    </#if>
</@layout.registrationLayout>
