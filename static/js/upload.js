// Doc-Manager file upload JavaScript
// Handles the pre-signed URL upload flow: file selection, validation, S3 upload, and completion.

(function () {
    'use strict';

    // ==================== CONSTANTS ====================

    var ALLOWED_TYPES = [
        'application/pdf',
        'image/png',
        'image/jpeg',
        'image/tiff'
    ];

    var ALLOWED_EXTENSIONS = ['.pdf', '.png', '.jpg', '.jpeg', '.tiff', '.tif'];

    var MAX_FILE_SIZE = 10 * 1024 * 1024; // 10 MB

    // ==================== STATE ====================

    var selectedFile = null;
    var formId = null;
    var uploadAbortController = null;

    // ==================== INITIALIZATION ====================

    function initUpload() {
        var dropZone = document.getElementById('drop-zone');
        var fileInput = document.getElementById('file-input');

        if (!dropZone || !fileInput) {
            return; // Not on the upload page; silently skip.
        }

        // --- Drag-and-drop events ---
        dropZone.addEventListener('dragover', function (e) {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.add('border-indigo-500', 'bg-indigo-50');
        });

        dropZone.addEventListener('dragenter', function (e) {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.add('border-indigo-500', 'bg-indigo-50');
        });

        dropZone.addEventListener('dragleave', function (e) {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.remove('border-indigo-500', 'bg-indigo-50');
        });

        dropZone.addEventListener('drop', function (e) {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.remove('border-indigo-500', 'bg-indigo-50');
            var files = e.dataTransfer.files;
            if (files.length > 0) {
                handleFileSelected(files[0]);
            }
        });

        // --- Click to browse ---
        dropZone.addEventListener('click', function () {
            fileInput.click();
        });

        fileInput.addEventListener('change', function (e) {
            if (e.target.files.length > 0) {
                handleFileSelected(e.target.files[0]);
            }
        });

        // --- Upload button ---
        var uploadBtn = document.getElementById('upload-btn');
        if (uploadBtn) {
            uploadBtn.addEventListener('click', function () {
                startUpload();
            });
        }

        // --- Cancel button ---
        var cancelBtn = document.getElementById('cancel-upload-btn');
        if (cancelBtn) {
            cancelBtn.addEventListener('click', function () {
                cancelUpload();
            });
        }
    }

    // ==================== FILE VALIDATION ====================

    /**
     * Validates the selected file's MIME type. Falls back to extension check
     * when the browser reports an empty type (common on some OS/browser combos).
     */
    function isAllowedType(file) {
        if (file.type && ALLOWED_TYPES.indexOf(file.type) !== -1) {
            return true;
        }
        // Fallback: check extension
        var name = file.name.toLowerCase();
        for (var i = 0; i < ALLOWED_EXTENSIONS.length; i++) {
            if (name.lastIndexOf(ALLOWED_EXTENSIONS[i]) === name.length - ALLOWED_EXTENSIONS[i].length) {
                return true;
            }
        }
        return false;
    }

    /**
     * Infers MIME type from the file object. Uses the browser-reported type when
     * available; otherwise maps from the file extension.
     */
    function inferContentType(file) {
        if (file.type) return file.type;
        var name = file.name.toLowerCase();
        if (name.indexOf('.pdf') === name.length - 4) return 'application/pdf';
        if (name.indexOf('.png') === name.length - 4) return 'image/png';
        if (name.indexOf('.jpg') === name.length - 4) return 'image/jpeg';
        if (name.indexOf('.jpeg') === name.length - 5) return 'image/jpeg';
        if (name.indexOf('.tiff') === name.length - 5) return 'image/tiff';
        if (name.indexOf('.tif') === name.length - 4) return 'image/tiff';
        return 'application/octet-stream';
    }

    // ==================== FILE SELECTION ====================

    function handleFileSelected(file) {
        // Validate type
        if (!isAllowedType(file)) {
            var ext = file.name.split('.').pop();
            showUploadError(
                '".' + ext + '" files are not supported. Please upload a PDF, PNG, JPG, or TIFF file.'
            );
            return;
        }

        // Validate size
        if (file.size > MAX_FILE_SIZE) {
            showUploadError('The file exceeds the maximum size of 10 MB. Please choose a smaller file.');
            return;
        }

        selectedFile = file;
        showFileSelected(file);
    }

    /**
     * Transitions the UI to show the selected file info, including a form-name
     * input that defaults to the filename (sans extension, with separators replaced).
     */
    function showFileSelected(file) {
        showState('state-uploading');

        var filenameEl = document.getElementById('upload-filename');
        var filesizeEl = document.getElementById('upload-filesize');
        var formNameInput = document.getElementById('form-name');

        if (filenameEl) filenameEl.textContent = file.name;
        if (filesizeEl) filesizeEl.textContent = formatFileSize(file.size);

        // Auto-fill form name from filename
        if (formNameInput) {
            var formName = file.name
                .replace(/\.[^/.]+$/, '') // strip extension
                .replace(/[-_]/g, ' ');   // replace separators with spaces
            formNameInput.value = formName;
        }

        // Reset progress bar to zero
        updateProgress(0, 'Ready to upload');

        // Show the upload button, hide the progress label row if separate
        var uploadBtn = document.getElementById('upload-btn');
        if (uploadBtn) uploadBtn.classList.remove('hidden');
    }

    // ==================== UPLOAD FLOW ====================

    /**
     * Drives the three-step pre-signed URL upload flow:
     *   1. POST /api/forms/upload-url  -> get pre-signed PUT URL + formId + s3Key
     *   2. PUT  <uploadUrl>           -> upload file bytes directly to S3
     *   3. POST /api/forms/:id/upload-complete -> notify backend
     */
    function startUpload() {
        if (!selectedFile) return;

        var formNameInput = document.getElementById('form-name');
        var formName = formNameInput ? formNameInput.value.trim() : '';
        if (!formName) {
            showUploadError('Please enter a form name.');
            return;
        }

        // Hide the upload button so it cannot be clicked twice
        var uploadBtn = document.getElementById('upload-btn');
        if (uploadBtn) uploadBtn.classList.add('hidden');

        var contentType = inferContentType(selectedFile);
        uploadAbortController = new AbortController();

        updateProgress(10, 'Requesting upload URL...');

        // Step 1: Request pre-signed URL from the backend
        fetch('/api/forms/upload-url', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            signal: uploadAbortController.signal,
            body: JSON.stringify({
                filename: selectedFile.name,
                contentType: contentType,
                name: formName
            })
        })
            .then(function (response) {
                if (!response.ok) {
                    return response.json().then(function (body) {
                        throw new Error(body.error || 'Failed to get upload URL');
                    });
                }
                return response.json();
            })
            .then(function (data) {
                formId = data.formId;
                var uploadUrl = data.uploadUrl;
                var s3Key = data.s3Key;

                // Step 2: Upload file directly to S3
                updateProgress(30, 'Uploading file...');
                return fetch(uploadUrl, {
                    method: 'PUT',
                    headers: { 'Content-Type': contentType },
                    signal: uploadAbortController.signal,
                    body: selectedFile
                }).then(function (uploadResponse) {
                    if (!uploadResponse.ok) {
                        throw new Error('Failed to upload file to storage');
                    }
                    return s3Key;
                });
            })
            .then(function (s3Key) {
                // Step 3: Notify backend upload is complete
                updateProgress(80, 'Finalizing...');
                return fetch('/api/forms/' + formId + '/upload-complete', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    signal: uploadAbortController.signal,
                    body: JSON.stringify({ s3Key: s3Key })
                });
            })
            .then(function (completeResponse) {
                if (!completeResponse.ok) {
                    throw new Error('Failed to finalize upload');
                }
                // Step 4: Success
                updateProgress(100, 'Upload complete!');
                showUploadComplete();
            })
            .catch(function (err) {
                if (err.name === 'AbortError') {
                    // User cancelled — reset quietly
                    resetUpload();
                    return;
                }
                console.error('Upload failed:', err);
                showUploadError(err.message || 'Upload failed. Please try again.');
            });
    }

    // ==================== UI STATE MANAGEMENT ====================

    var STATE_IDS = ['state-default', 'state-uploading', 'state-complete', 'state-error'];

    function showState(stateId) {
        for (var i = 0; i < STATE_IDS.length; i++) {
            var el = document.getElementById(STATE_IDS[i]);
            if (el) {
                if (STATE_IDS[i] === stateId) {
                    el.classList.remove('hidden');
                } else {
                    el.classList.add('hidden');
                }
            }
        }
    }

    function updateProgress(percent, message) {
        var bar = document.getElementById('progress-bar');
        var text = document.getElementById('progress-text');
        var percentLabel = document.getElementById('upload-percent');

        if (bar) bar.style.width = percent + '%';
        if (text) text.textContent = message;
        if (percentLabel) percentLabel.textContent = Math.round(percent) + '%';
    }

    function showUploadComplete() {
        showState('state-complete');

        // Set the editor link if we have a formId
        var editorLink = document.getElementById('editor-link');
        if (editorLink && formId) {
            editorLink.href = '/forms/' + formId + '/edit';
        }

        // Auto-redirect after a brief delay so the user sees the success state
        if (formId) {
            setTimeout(function () {
                window.location.href = '/forms/' + formId + '/edit';
            }, 1500);
        }
    }

    function showUploadError(message) {
        showState('state-error');

        var errorEl = document.getElementById('error-message');
        if (errorEl) errorEl.textContent = message;
    }

    // ==================== RESET / CANCEL ====================

    function cancelUpload() {
        if (uploadAbortController) {
            uploadAbortController.abort();
            uploadAbortController = null;
        }
        resetUpload();
    }

    function resetUpload() {
        selectedFile = null;
        formId = null;
        uploadAbortController = null;

        var fileInput = document.getElementById('file-input');
        if (fileInput) fileInput.value = '';

        showState('state-default');
    }

    // ==================== UTILITIES ====================

    function formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    // ==================== PUBLIC API ====================
    // Expose functions so inline onclick handlers (in templates) can call them.

    window.startUpload = startUpload;
    window.cancelUpload = cancelUpload;
    window.resetUpload = resetUpload;

    // ==================== BOOTSTRAP ====================

    document.addEventListener('DOMContentLoaded', initUpload);
})();
