// Doc-Manager application JavaScript
// Global utilities: toast notifications, confirmation dialogs, delete helpers, mobile nav.

(function () {
    'use strict';

    // ==================== TOAST NOTIFICATION SYSTEM ====================

    var TOAST_DURATION = 4000; // ms before auto-dismiss
    var TOAST_SLIDE_OUT = 300; // ms for the slide-out animation

    var TOAST_COLORS = {
        success: 'bg-green-600',
        error: 'bg-red-600',
        info: 'bg-indigo-600',
        warning: 'bg-amber-500'
    };

    /**
     * Show a brief toast notification in the top-right corner.
     * @param {string} message  Text to display.
     * @param {string} [type]   One of 'success' | 'error' | 'info' | 'warning'. Defaults to 'success'.
     */
    function showToast(message, type) {
        type = type || 'success';
        var colorClass = TOAST_COLORS[type] || TOAST_COLORS.info;

        var toast = document.createElement('div');
        toast.className =
            'fixed top-4 right-4 ' + colorClass +
            ' text-white px-6 py-3 rounded-lg shadow-lg z-50' +
            ' flex items-center gap-3' +
            ' transform transition-all duration-300 translate-x-full';

        toast.innerHTML =
            '<span>' + escapeHtml(message) + '</span>' +
            '<button class="text-white/80 hover:text-white flex-shrink-0" aria-label="Dismiss">' +
            '  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
            '    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>' +
            '  </svg>' +
            '</button>';

        // Dismiss on close button click
        var closeBtn = toast.querySelector('button');
        closeBtn.addEventListener('click', function () {
            dismissToast(toast);
        });

        document.body.appendChild(toast);

        // Trigger slide-in on next frame so the browser registers the initial translate-x-full
        requestAnimationFrame(function () {
            toast.classList.remove('translate-x-full');
        });

        // Auto-dismiss
        var timer = setTimeout(function () {
            dismissToast(toast);
        }, TOAST_DURATION);

        // Store the timer so we can cancel if manually dismissed
        toast._toastTimer = timer;
    }

    function dismissToast(toast) {
        if (!toast || !toast.parentNode) return;
        clearTimeout(toast._toastTimer);
        toast.classList.add('translate-x-full');
        setTimeout(function () {
            if (toast.parentNode) toast.parentNode.removeChild(toast);
        }, TOAST_SLIDE_OUT);
    }

    // ==================== CONFIRMATION DIALOG ====================

    /**
     * Show a modal confirmation dialog. Returns a Promise that resolves to
     * `true` (confirm) or `false` (cancel / backdrop click).
     * @param {string} message  The question / warning to display.
     * @returns {Promise<boolean>}
     */
    function confirmAction(message) {
        return new Promise(function (resolve) {
            var overlay = document.createElement('div');
            overlay.className = 'fixed inset-0 bg-black/50 flex items-center justify-center z-50';

            overlay.innerHTML =
                '<div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">' +
                '  <p class="text-gray-700 mb-6">' + escapeHtml(message) + '</p>' +
                '  <div class="flex justify-end gap-3">' +
                '    <button id="confirm-cancel" class="px-4 py-2 text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">Cancel</button>' +
                '    <button id="confirm-ok" class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors">Confirm</button>' +
                '  </div>' +
                '</div>';

            document.body.appendChild(overlay);

            function cleanup(result) {
                if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
                resolve(result);
            }

            overlay.querySelector('#confirm-cancel').addEventListener('click', function () {
                cleanup(false);
            });

            overlay.querySelector('#confirm-ok').addEventListener('click', function () {
                cleanup(true);
            });

            // Dismiss on backdrop click
            overlay.addEventListener('click', function (e) {
                if (e.target === overlay) cleanup(false);
            });

            // Dismiss on Escape key
            function onKeyDown(e) {
                if (e.key === 'Escape') {
                    document.removeEventListener('keydown', onKeyDown);
                    cleanup(false);
                }
            }
            document.addEventListener('keydown', onKeyDown);
        });
    }

    // ==================== DELETE HELPERS ====================

    /**
     * Delete a form after user confirmation. Removes the card from the DOM on
     * success, or falls back to a full page reload.
     * @param {string} id    Form ID
     * @param {string} [name] Form name for the confirmation prompt
     */
    function deleteForm(id, name) {
        var prompt = name
            ? 'Delete "' + name + '"? This cannot be undone.'
            : 'Are you sure you want to delete this form? This cannot be undone.';

        confirmAction(prompt).then(function (confirmed) {
            if (!confirmed) return;

            fetch('/api/forms/' + id, { method: 'DELETE' })
                .then(function (response) {
                    if (!response.ok) throw new Error('Delete failed');
                    showToast('Form deleted successfully');
                    // Try to remove the card in-place; fall back to reload
                    var card = document.querySelector('[data-form-id="' + id + '"]');
                    if (card) {
                        card.remove();
                        // If the grid is now empty, reload to show the empty state
                        var grid = document.querySelector('[data-form-grid]');
                        if (grid && grid.children.length === 0) {
                            window.location.reload();
                        }
                    } else {
                        window.location.reload();
                    }
                })
                .catch(function () {
                    showToast('Failed to delete form', 'error');
                });
        });
    }

    /**
     * Delete a customer after user confirmation.
     * @param {string} id    Customer ID
     * @param {string} [name] Customer name for the confirmation prompt
     */
    function deleteCustomer(id, name) {
        var prompt = name
            ? 'Delete "' + name + '"? This cannot be undone.'
            : 'Are you sure you want to delete this customer? This cannot be undone.';

        confirmAction(prompt).then(function (confirmed) {
            if (!confirmed) return;

            fetch('/api/customers/' + id, { method: 'DELETE' })
                .then(function (response) {
                    if (!response.ok) throw new Error('Delete failed');
                    showToast('Customer deleted successfully');
                    window.location.reload();
                })
                .catch(function () {
                    showToast('Failed to delete customer', 'error');
                });
        });
    }

    // ==================== MOBILE NAV TOGGLE ====================

    function toggleMobileNav() {
        var nav = document.getElementById('mobile-nav');
        if (nav) nav.classList.toggle('hidden');
    }

    // ==================== UTILITIES ====================

    /**
     * Simple HTML-entity escaping to prevent XSS when injecting user-supplied
     * text into innerHTML.
     */
    function escapeHtml(text) {
        if (!text) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(text));
        return div.innerHTML;
    }

    // ==================== PUBLIC API ====================

    window.showToast = showToast;
    window.confirmAction = confirmAction;
    window.deleteForm = deleteForm;
    window.deleteCustomer = deleteCustomer;
    window.toggleMobileNav = toggleMobileNav;

})();
