/**
 * CopyButton — copies target element's text to clipboard.
 *
 * Data attributes:
 *   [data-copy-button]       — The button container
 *   [data-copy-target-id]    — ID of element to copy from
 *   [data-copy-icon-default] — Icon shown by default (clipboard)
 *   [data-copy-icon-success] — Icon shown after copy (check)
 */
document.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-copy-button]");
  if (!btn) return;

  const targetId = btn.dataset.copyTargetId;
  if (!targetId) return;

  const target = document.getElementById(targetId);
  if (!target) return;

  const text = target.textContent || target.value || "";
  navigator.clipboard.writeText(text.trim()).then(() => {
    const iconDefault = btn.querySelector("[data-copy-icon-default]");
    const iconSuccess = btn.querySelector("[data-copy-icon-success]");

    if (iconDefault) iconDefault.classList.add("hidden");
    if (iconSuccess) iconSuccess.classList.remove("hidden");

    setTimeout(() => {
      if (iconDefault) iconDefault.classList.remove("hidden");
      if (iconSuccess) iconSuccess.classList.add("hidden");
    }, 2000);
  });
});
