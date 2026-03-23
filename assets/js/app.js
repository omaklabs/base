// Omakase Go — minimal JS entry point
// Alpine.js and HTMX are loaded via separate script tags in base.templ

// Allow HTMX to swap 422 (validation error) responses so that
// server-rendered form errors are displayed to the user.
document.addEventListener("DOMContentLoaded", function () {
  if (typeof htmx !== "undefined") {
    htmx.config.responseHandling = [
      { code: "204", swap: false },
      { code: "[23]..", swap: true },
      { code: "422", swap: true },
      { code: "[45]..", swap: false, error: true },
    ];
  }
});

// Include the CSRF token in every HTMX AJAX request header.
// gorilla/csrf accepts the token from the X-CSRF-Token header.
// The token is available from either the <meta name="csrf-token"> tag
// (present on every page) or a form hidden input.
document.addEventListener("htmx:configRequest", function (e) {
  var meta = document.querySelector('meta[name="csrf-token"]');
  if (meta && meta.content) {
    e.detail.headers["X-CSRF-Token"] = meta.content;
    return;
  }
  var input = document.querySelector('input[name="gorilla.csrf.Token"]');
  if (input) {
    e.detail.headers["X-CSRF-Token"] = input.value;
  }
});

// Reinitialize Alpine.js components after HTMX content swaps.
// This ensures dynamically loaded content with x-data works correctly.
document.addEventListener("htmx:afterSettle", function (e) {
  if (typeof Alpine !== "undefined") {
    Alpine.initTree(e.detail.target);
  }
});
