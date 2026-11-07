/**
 * omama-login.js — Progressive enhancement for the Keycloak login theme.
 * Loaded at the end of the page by Keycloak (theme.properties: scripts=js/omama-login.js).
 */
(function () {
  'use strict';

  var SENT_EMAIL_KEY = 'omama.reset.email';

  /* ── Login page ─────────────────────────────────────────────────────────── */
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

  /* ── Forgot password page ───────────────────────────────────────────────── */
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
        /* Storage disabled: the check-email page falls back to generic text */
      }
      if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.style.opacity = '0.7';
      }
    });
  }

  /* ── Check email page ───────────────────────────────────────────────────── */
  var sentEmailEl = document.getElementById('omama-sent-email');
  if (sentEmailEl) {
    try {
      var sentEmail = window.sessionStorage.getItem(SENT_EMAIL_KEY);
      if (sentEmail) sentEmailEl.textContent = sentEmail; // textContent: no HTML injection
    } catch (ignored) {
      /* keep fallback text */
    }
  }
})();
