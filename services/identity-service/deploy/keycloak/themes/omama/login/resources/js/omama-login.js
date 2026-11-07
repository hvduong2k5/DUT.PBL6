/**
 * omama-login.js — Progressive enhancement for the Keycloak login theme.
 * Note: Keycloak's template.ftl loads scripts inside <head>, so we wait for
 * DOMContentLoaded before querying elements.
 */
(function () {
  'use strict';

  var SENT_EMAIL_KEY = 'omama.reset.email';
  var GUEST_URL_KEY = 'omama.guest.returnUrl';

  /**
   * Resolve where "Continue as guest" / Back should take the user:
   * 1. Mobile custom scheme redirect_uri (e.g. omama://auth/callback)
   *    -> redirect_uri?error=access_denied&state=... so the mobile auth sheet closes
   * 2. Identity BFF callback (/public/auth/callback)
   *    -> redirect_uri?error=access_denied&state=... so Identity BFF redirects back
   *       to the frontend or mobile app that initiated login
   * 3. Direct web frontend redirect_uri (http/https)
   *    -> origin of redirect_uri (e.g. https://omama.vn/ or http://localhost:3000/)
   * 4. Persisted value in sessionStorage (survives login error / reset-password hops)
   * 5. Cross-origin document.referrer (when arriving from web frontend)
   */
  function resolveGuestReturnUrl() {
    var params = new URLSearchParams(window.location.search);
    var redirectUri = params.get('redirect_uri');
    var state = params.get('state');

    if (redirectUri) {
      var resolved = null;
      if (!/^https?:\/\//i.test(redirectUri)) {
        /* Custom mobile scheme (e.g. omama://auth/callback) */
        var sep = redirectUri.indexOf('?') === -1 ? '?' : '&';
        resolved =
          redirectUri +
          sep +
          'error=access_denied' +
          (state ? '&state=' + encodeURIComponent(state) : '');
      } else {
        try {
          var parsed = new URL(redirectUri);
          if (/\/public\/auth\/callback\/?$/i.test(parsed.pathname)) {
            /* Identity BFF callback: let BFF resolve state -> frontend/mobile URL */
            parsed.searchParams.set('error', 'access_denied');
            if (state) parsed.searchParams.set('state', state);
            resolved = parsed.toString();
          } else if (parsed.origin !== window.location.origin) {
            /* Direct web OIDC callback: return to the web frontend root */
            resolved = parsed.origin + '/';
          }
        } catch (ignored) {
          /* ignore malformed redirect_uri */
        }
      }

      if (resolved) {
        try {
          window.sessionStorage.setItem(GUEST_URL_KEY, resolved);
        } catch (ignored) {}
        return resolved;
      }
    }

    try {
      var cached = window.sessionStorage.getItem(GUEST_URL_KEY);
      if (cached) return cached;
    } catch (ignored) {}

    if (document.referrer) {
      try {
        var refUrl = new URL(document.referrer);
        if (refUrl.origin !== window.location.origin) {
          var refRoot = refUrl.origin + '/';
          try {
            window.sessionStorage.setItem(GUEST_URL_KEY, refRoot);
          } catch (ignored) {}
          return refRoot;
        }
      } catch (ignored) {}
    }

    return null;
  }

  function init() {
    /* ── Wire guest & back links to return to frontend / mobile app ───────── */
    var guestReturnUrl = resolveGuestReturnUrl();
    var guestLink = document.getElementById('omama-guest-link');
    var backLink = document.getElementById('omama-login-back');

    function wireReturnLink(linkEl) {
      if (!linkEl) return;
      if (guestReturnUrl) {
        linkEl.setAttribute('href', guestReturnUrl);
        return;
      }
      var currentHref = linkEl.getAttribute('href');
      if (!currentHref || currentHref === '#') {
        linkEl.addEventListener('click', function (event) {
          event.preventDefault();
          window.history.back();
        });
      }
    }

    wireReturnLink(guestLink);
    wireReturnLink(backLink);

    /* ── Login page ───────────────────────────────────────────────────────── */
    var loginForm = document.getElementById('kc-form-login');
    var usernameInput = loginForm ? document.getElementById('username') : null;

    if (usernameInput) {
      if (!usernameInput.value) usernameInput.focus();

      /* Phone number input: allow digits, +, spaces, dashes and parentheses only */
      usernameInput.addEventListener('input', function () {
        this.value = this.value.replace(/[^\d\s+\-().]/g, '');
      });
    }

    if (loginForm) {
      /* Disable submit button after first click to prevent double-submit */
      loginForm.addEventListener('submit', function () {
        var btn = document.getElementById('kc-login');
        if (btn) {
          btn.disabled = true;
          btn.style.opacity = '0.7';
        }
      });
    }

    /* ── Forgot password page ─────────────────────────────────────────────── */
    var resetForm = document.getElementById('omama-reset-form');
    if (resetForm) {
      var emailInput = document.getElementById('username');
      var errorEl = document.getElementById('omama-email-error');
      var submitBtn = document.getElementById('kc-reset-password-submit');
      var emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

      var setError = function (hasError) {
        emailInput.classList.toggle('omama-input--error', hasError);
        if (hasError) {
          emailInput.setAttribute('aria-invalid', 'true');
          errorEl.textContent = errorEl.getAttribute('data-invalid-message');
        } else {
          emailInput.removeAttribute('aria-invalid');
          errorEl.textContent = '';
        }
      };

      emailInput.addEventListener('input', function () {
        if (emailInput.classList.contains('omama-input--error')) {
          setError(!emailPattern.test(emailInput.value.trim()));
        }
      });

      resetForm.addEventListener('submit', function (event) {
        var value = emailInput.value.trim();
        if (!emailPattern.test(value)) {
          event.preventDefault();
          setError(true);
          emailInput.focus();
          return;
        }
        emailInput.value = value;
        try {
          window.sessionStorage.setItem(SENT_EMAIL_KEY, value);
        } catch (ignored) {
          /* Storage disabled: fallback to generic copy */
        }
        if (submitBtn) {
          submitBtn.disabled = true;
          submitBtn.style.opacity = '0.7';
        }
      });
    }

    /* ── Check email page ─────────────────────────────────────────────────── */
    var sentEmailEl = document.getElementById('omama-sent-email');
    var withAddrEl = document.getElementById('omama-check-email-with-addr');
    var genericEl = document.getElementById('omama-check-email-generic');
    if (sentEmailEl && !sentEmailEl.textContent.trim()) {
      try {
        var sentEmail = window.sessionStorage.getItem(SENT_EMAIL_KEY);
        if (sentEmail) {
          sentEmailEl.textContent = sentEmail;
          if (withAddrEl) withAddrEl.style.display = '';
          if (genericEl) genericEl.style.display = 'none';
        }
      } catch (ignored) {
        /* keep generic paragraph visible */
      }
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
