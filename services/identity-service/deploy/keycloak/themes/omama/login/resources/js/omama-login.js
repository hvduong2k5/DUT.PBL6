/**
 * omama-login.js — Progressive enhancement for Keycloak login page.
 * Loaded after DOM is ready (Keycloak places scripts at end of body).
 */
(function () {
  'use strict';

  /* Auto-focus username unless it already has a value (error state) */
  var usernameInput = document.getElementById('username');
  if (usernameInput && !usernameInput.value) {
    usernameInput.focus();
  }

  /* Phone number input: strip non-numeric chars except leading + */
  if (usernameInput) {
    usernameInput.addEventListener('input', function () {
      var val = this.value;
      // Allow digits, +, spaces and dashes (common phone formats)
      this.value = val.replace(/[^\d\s+\-().]/g, '');
    });
  }

  /* Disable submit button after first click to prevent double-submit */
  var form = document.getElementById('kc-form-login');
  if (form) {
    form.addEventListener('submit', function () {
      var btn = document.getElementById('kc-login');
      if (btn) {
        btn.disabled = true;
        btn.style.opacity = '0.7';
      }
    });
  }
})();
