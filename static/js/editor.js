// Doc-Manager form editor JavaScript
// Interactive field placement editor with drag/drop/resize, version history, and font configuration.

(function() {
    'use strict';

    // ==================== EDITOR STATE ====================

    var fields = [];
    var selectedField = null;
    var isDragging = false;
    var isResizing = false;
    var hasUnsavedChanges = false;
    var formFontFamily = 'Helvetica';
    var formFontSize = 12;
    var currentPage = 1;
    var totalPages = 1;
    var dragState = null;
    var resizeState = null;

    // ==================== INITIALIZATION ====================

    window.initEditor = function(fieldsData, imageUrl, fontFamily, fontSize) {
        fields = fieldsData || [];
        formFontFamily = fontFamily || 'Helvetica';
        formFontSize = fontSize || 12;

        // Load form image
        var img = document.getElementById('form-image');
        if (imageUrl) {
            img.src = imageUrl;
            img.onload = function() {
                renderFields();
                updatePageIndicator();
            };
            img.onerror = function() {
                // If image fails to load, still render fields
                renderFields();
                updatePageIndicator();
            };
        } else {
            renderFields();
            updatePageIndicator();
        }

        // Calculate total pages from fields
        totalPages = 1;
        for (var i = 0; i < fields.length; i++) {
            if (fields[i].page > totalPages) {
                totalPages = fields[i].page;
            }
        }
        updatePageIndicator();

        // Set up global mouse event listeners for drag/resize
        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);

        // Click on canvas background deselects field
        var canvasContainer = document.getElementById('canvas-container');
        if (canvasContainer) {
            canvasContainer.addEventListener('click', function(e) {
                if (e.target === canvasContainer || e.target.id === 'form-canvas' || e.target.id === 'form-image') {
                    deselectField();
                }
            });
        }
    };

    // ==================== RENDERING ====================

    function renderFields() {
        var canvas = document.getElementById('field-canvas');
        if (!canvas) return;
        canvas.innerHTML = '';

        for (var i = 0; i < fields.length; i++) {
            // Only show fields for the current page
            if (fields[i].page !== currentPage) continue;

            var overlay = createFieldOverlay(fields[i], i);
            canvas.appendChild(overlay);
        }

        updateFieldsList();
        updateStatusBar();
    }

    function createFieldOverlay(field, index) {
        var div = document.createElement('div');
        div.className = 'field-overlay absolute cursor-move';
        div.style.left = field.x + '%';
        div.style.top = field.y + '%';
        div.style.width = field.width + '%';
        div.style.height = field.height + '%';
        div.style.zIndex = selectedField === index ? '10' : '1';
        div.setAttribute('data-field-index', index);

        // Inner container for border and background
        var inner = document.createElement('div');
        inner.className = 'absolute inset-0 rounded-sm';

        // Color based on confidence
        var confidence = field.confidence || 0;
        if (confidence >= 0.8) {
            inner.className += ' border-2 border-blue-500';
            if (selectedField === index) {
                inner.className += ' bg-blue-500/15';
            } else {
                inner.className += ' hover:bg-blue-500/5';
                inner.style.borderColor = 'rgba(59, 130, 246, 0.6)';
            }
        } else if (confidence >= 0.5) {
            inner.className += ' border-2 border-amber-400';
            if (selectedField === index) {
                inner.className += ' bg-amber-500/15';
            } else {
                inner.className += ' hover:bg-amber-500/5';
                inner.style.borderColor = 'rgba(251, 191, 36, 0.7)';
            }
        } else {
            inner.className += ' border-2 border-red-400';
            if (selectedField === index) {
                inner.className += ' bg-red-500/15';
            } else {
                inner.className += ' hover:bg-red-500/5';
                inner.style.borderColor = 'rgba(248, 113, 113, 0.7)';
            }
        }

        // Selected state
        if (selectedField === index) {
            inner.style.borderColor = '';
            if (confidence >= 0.8) {
                inner.className = inner.className.replace('hover:bg-blue-500/5', '');
            } else if (confidence >= 0.5) {
                inner.className = inner.className.replace('hover:bg-amber-500/5', '');
            } else {
                inner.className = inner.className.replace('hover:bg-red-500/5', '');
            }
        }

        div.appendChild(inner);

        // Label
        var label = document.createElement('span');
        label.className = 'absolute text-xs font-medium px-1.5 py-0.5 rounded shadow-sm whitespace-nowrap';
        label.style.top = '-20px';
        label.style.left = '0';
        if (confidence >= 0.8) {
            label.className += ' text-blue-600 bg-blue-50';
        } else if (confidence >= 0.5) {
            label.className += ' text-amber-600 bg-amber-50';
        } else {
            label.className += ' text-red-600 bg-red-50';
        }
        label.textContent = field.fieldName;
        div.appendChild(label);

        // Resize handle (bottom-right corner, only for selected field)
        if (selectedField === index) {
            var handle = document.createElement('div');
            handle.className = 'resize-handle se';
            handle.addEventListener('mousedown', function(e) {
                e.stopPropagation();
                e.preventDefault();
                startResize(e, index);
            });
            div.appendChild(handle);
        }

        // Drag event
        div.addEventListener('mousedown', function(e) {
            if (e.target.classList.contains('resize-handle')) return;
            e.preventDefault();
            selectField(index);
            startDrag(e, index);
        });

        // Click to select
        div.addEventListener('click', function(e) {
            e.stopPropagation();
            selectField(index);
        });

        return div;
    }

    // ==================== FIELDS LIST (LEFT PANEL) ====================

    function updateFieldsList() {
        var ul = document.getElementById('fields-list-ul');
        if (!ul) return;
        ul.innerHTML = '';

        for (var i = 0; i < fields.length; i++) {
            var li = createFieldListItem(fields[i], i);
            ul.appendChild(li);
        }

        var countEl = document.getElementById('field-count');
        if (countEl) {
            countEl.textContent = fields.length + ' field' + (fields.length !== 1 ? 's' : '') + ' total';
        }
    }

    function createFieldListItem(field, index) {
        var li = document.createElement('li');
        li.className = 'px-3 py-2.5 mx-2 my-0.5 rounded-md cursor-pointer transition-colors';

        if (selectedField === index) {
            li.className += ' bg-indigo-50 border border-indigo-600/20';
        } else {
            li.className += ' hover:bg-gray-50';
        }

        var container = document.createElement('div');
        container.className = 'flex items-center justify-between';

        var leftSide = document.createElement('div');
        leftSide.className = 'flex items-center gap-2.5';

        var checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.checked = true;
        checkbox.className = 'rounded border-gray-300 text-indigo-600 focus:ring-indigo-600 h-3.5 w-3.5';
        checkbox.addEventListener('click', function(e) {
            e.stopPropagation();
        });

        var nameSpan = document.createElement('span');
        nameSpan.className = 'text-sm';
        if (selectedField === index) {
            nameSpan.className += ' font-medium text-indigo-700';
        } else {
            nameSpan.className += ' text-gray-700';
        }
        nameSpan.textContent = field.fieldName;

        leftSide.appendChild(checkbox);
        leftSide.appendChild(nameSpan);
        container.appendChild(leftSide);

        // Confidence dot
        var dot = document.createElement('span');
        dot.className = 'w-2.5 h-2.5 rounded-full flex-shrink-0';
        var confidence = field.confidence || 0;
        if (confidence >= 0.8) {
            dot.className += ' bg-green-500';
            dot.title = 'High confidence';
        } else if (confidence >= 0.5) {
            dot.className += ' bg-yellow-400';
            dot.title = 'Medium confidence';
        } else {
            dot.className += ' bg-red-500';
            dot.title = 'Low confidence';
        }
        container.appendChild(dot);

        li.appendChild(container);

        li.addEventListener('click', function() {
            selectField(index);
            // Scroll to page if field is on a different page
            if (field.page !== currentPage) {
                currentPage = field.page;
                updatePageIndicator();
                renderFields();
            }
        });

        return li;
    }

    // ==================== PROPERTIES PANEL (RIGHT PANEL) ====================

    function updatePropertiesPanel() {
        var panel = document.getElementById('properties-panel');
        if (!panel) return;

        if (selectedField === null || selectedField >= fields.length) {
            panel.innerHTML = '<p class="text-sm text-gray-400 italic">Select a field to view properties</p>';
            return;
        }

        var field = fields[selectedField];
        var confidence = field.confidence || 0;

        // Confidence color and label
        var confColor, confBgColor, confLabel;
        if (confidence >= 0.8) {
            confColor = 'bg-green-500';
            confBgColor = 'bg-green-100 text-green-700';
            confLabel = 'High';
        } else if (confidence >= 0.5) {
            confColor = 'bg-yellow-400';
            confBgColor = 'bg-yellow-100 text-yellow-700';
            confLabel = 'Medium';
        } else {
            confColor = 'bg-red-500';
            confBgColor = 'bg-red-100 text-red-700';
            confLabel = 'Low';
        }

        var confPercent = Math.round(confidence * 100);

        // Font override display
        var currentFontFamily = field.fontFamily || null;
        var currentFontSize = field.fontSize || null;

        var fontFamilyOptions = ['Helvetica', 'Courier', 'Times-Roman'];
        var fontSizeOptions = [8, 9, 10, 11, 12, 14, 16, 18, 20];

        var fontFamilySelect = '<option value=""' + (!currentFontFamily ? ' selected' : '') + '>Inherit (' + formFontFamily + ')</option>';
        for (var i = 0; i < fontFamilyOptions.length; i++) {
            var ff = fontFamilyOptions[i];
            fontFamilySelect += '<option value="' + ff + '"' + (currentFontFamily === ff ? ' selected' : '') + '>' + ff + '</option>';
        }

        var fontSizeSelect = '<option value=""' + (!currentFontSize ? ' selected' : '') + '>Inherit (' + formFontSize + 'pt)</option>';
        for (var j = 0; j < fontSizeOptions.length; j++) {
            var fs = fontSizeOptions[j];
            fontSizeSelect += '<option value="' + fs + '"' + (currentFontSize === fs ? ' selected' : '') + '>' + fs + 'pt</option>';
        }

        panel.innerHTML =
            '<div>' +
            '  <label class="block text-xs font-medium text-gray-500 mb-1">Field Name</label>' +
            '  <div class="text-sm font-medium text-gray-900 bg-gray-50 px-3 py-2 rounded-md border border-gray-200">' + escapeHtml(field.fieldName) + '</div>' +
            '</div>' +
            '<div class="grid grid-cols-2 gap-3">' +
            '  <div>' +
            '    <label class="block text-xs font-medium text-gray-500 mb-1">X Position</label>' +
            '    <input type="number" step="0.1" value="' + field.x.toFixed(1) + '" onchange="updateFieldPosition(\'x\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-indigo-600 focus:border-transparent">' +
            '  </div>' +
            '  <div>' +
            '    <label class="block text-xs font-medium text-gray-500 mb-1">Y Position</label>' +
            '    <input type="number" step="0.1" value="' + field.y.toFixed(1) + '" onchange="updateFieldPosition(\'y\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-indigo-600 focus:border-transparent">' +
            '  </div>' +
            '</div>' +
            '<div class="grid grid-cols-2 gap-3">' +
            '  <div>' +
            '    <label class="block text-xs font-medium text-gray-500 mb-1">Width</label>' +
            '    <input type="number" step="0.1" value="' + field.width.toFixed(1) + '" onchange="updateFieldPosition(\'width\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-indigo-600 focus:border-transparent">' +
            '  </div>' +
            '  <div>' +
            '    <label class="block text-xs font-medium text-gray-500 mb-1">Height</label>' +
            '    <input type="number" step="0.1" value="' + field.height.toFixed(1) + '" onchange="updateFieldPosition(\'height\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-indigo-600 focus:border-transparent">' +
            '  </div>' +
            '</div>' +
            '<div>' +
            '  <label class="block text-xs font-medium text-gray-500 mb-1">Confidence</label>' +
            '  <div class="flex items-center gap-2">' +
            '    <div class="flex-1 bg-gray-200 rounded-full h-1.5">' +
            '      <div class="' + confColor + ' h-1.5 rounded-full" style="width: ' + confPercent + '%"></div>' +
            '    </div>' +
            '    <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ' + confBgColor + '">' + confidence.toFixed(2) + '</span>' +
            '  </div>' +
            '</div>' +
            (field.reasoning ? '<div><label class="block text-xs font-medium text-gray-500 mb-1">AI Reasoning</label><p class="text-xs text-gray-600 bg-gray-50 p-2 rounded border border-gray-200">' + escapeHtml(field.reasoning) + '</p></div>' : '') +
            '<div class="pt-3 border-t border-gray-100">' +
            '  <h3 class="text-xs font-semibold text-gray-700 mb-3">Font Override</h3>' +
            '  <div class="space-y-3">' +
            '    <div>' +
            '      <label class="block text-xs font-medium text-gray-500 mb-1">Font Family</label>' +
            '      <select onchange="updateFieldFont(\'fontFamily\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 bg-white text-gray-700 focus:outline-none focus:ring-1 focus:ring-indigo-600">' +
            fontFamilySelect +
            '      </select>' +
            '    </div>' +
            '    <div>' +
            '      <label class="block text-xs font-medium text-gray-500 mb-1">Font Size</label>' +
            '      <select onchange="updateFieldFont(\'fontSize\', this.value)" class="w-full text-sm border border-gray-300 rounded-md px-3 py-1.5 bg-white text-gray-700 focus:outline-none focus:ring-1 focus:ring-indigo-600">' +
            fontSizeSelect +
            '      </select>' +
            '    </div>' +
            '  </div>' +
            '</div>' +
            '<div class="pt-3 border-t border-gray-100">' +
            '  <button onclick="removeField()" class="w-full px-3 py-2 text-xs font-medium text-red-600 border border-red-300 rounded-md hover:bg-red-50 transition-colors flex items-center justify-center gap-1.5">' +
            '    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>' +
            '    Remove Field' +
            '  </button>' +
            '</div>';
    }

    // ==================== FIELD SELECTION ====================

    function selectField(index) {
        selectedField = index;
        renderFields();
        updatePropertiesPanel();
    }
    window.selectField = selectField;

    function deselectField() {
        selectedField = null;
        renderFields();
        updatePropertiesPanel();
    }

    // ==================== DRAG FUNCTIONALITY ====================

    function startDrag(e, index) {
        isDragging = true;
        var canvas = document.getElementById('field-canvas');
        var rect = canvas.getBoundingClientRect();
        dragState = {
            index: index,
            startMouseX: e.clientX,
            startMouseY: e.clientY,
            startFieldX: fields[index].x,
            startFieldY: fields[index].y,
            canvasWidth: rect.width,
            canvasHeight: rect.height
        };
    }

    function startResize(e, index) {
        isResizing = true;
        var canvas = document.getElementById('field-canvas');
        var rect = canvas.getBoundingClientRect();
        resizeState = {
            index: index,
            startMouseX: e.clientX,
            startMouseY: e.clientY,
            startWidth: fields[index].width,
            startHeight: fields[index].height,
            canvasWidth: rect.width,
            canvasHeight: rect.height
        };
    }

    function onMouseMove(e) {
        if (isDragging && dragState) {
            var dx = e.clientX - dragState.startMouseX;
            var dy = e.clientY - dragState.startMouseY;

            // Convert pixel delta to percentage
            var dxPercent = (dx / dragState.canvasWidth) * 100;
            var dyPercent = (dy / dragState.canvasHeight) * 100;

            var field = fields[dragState.index];
            var newX = dragState.startFieldX + dxPercent;
            var newY = dragState.startFieldY + dyPercent;

            // Clamp to canvas bounds
            newX = Math.max(0, Math.min(100 - field.width, newX));
            newY = Math.max(0, Math.min(100 - field.height, newY));

            field.x = Math.round(newX * 10) / 10;
            field.y = Math.round(newY * 10) / 10;

            markUnsaved();
            renderFields();
            updatePropertiesPanel();
        }

        if (isResizing && resizeState) {
            var rdx = e.clientX - resizeState.startMouseX;
            var rdy = e.clientY - resizeState.startMouseY;

            var rdxPercent = (rdx / resizeState.canvasWidth) * 100;
            var rdyPercent = (rdy / resizeState.canvasHeight) * 100;

            var rField = fields[resizeState.index];
            var newWidth = resizeState.startWidth + rdxPercent;
            var newHeight = resizeState.startHeight + rdyPercent;

            // Minimum size and clamp to canvas
            newWidth = Math.max(2, Math.min(100 - rField.x, newWidth));
            newHeight = Math.max(1, Math.min(100 - rField.y, newHeight));

            rField.width = Math.round(newWidth * 10) / 10;
            rField.height = Math.round(newHeight * 10) / 10;

            markUnsaved();
            renderFields();
            updatePropertiesPanel();
        }
    }

    function onMouseUp() {
        isDragging = false;
        isResizing = false;
        dragState = null;
        resizeState = null;
    }

    // ==================== FIELD MANIPULATION ====================

    window.updateFieldPosition = function(prop, value) {
        if (selectedField === null || selectedField >= fields.length) return;
        var num = parseFloat(value);
        if (isNaN(num)) return;
        num = Math.max(0, Math.min(100, num));
        fields[selectedField][prop] = num;
        markUnsaved();
        renderFields();
    };

    window.updateFieldFont = function(prop, value) {
        if (selectedField === null || selectedField >= fields.length) return;
        if (prop === 'fontFamily') {
            fields[selectedField].fontFamily = value || null;
        } else if (prop === 'fontSize') {
            fields[selectedField].fontSize = value ? parseInt(value) : null;
        }
        markUnsaved();
    };

    window.updateFormFont = function() {
        var familyEl = document.getElementById('form-font-family');
        var sizeEl = document.getElementById('form-font-size');
        if (familyEl) formFontFamily = familyEl.value;
        if (sizeEl) formFontSize = parseInt(sizeEl.value);
        markUnsaved();
        // Update properties panel to reflect new inherit values
        updatePropertiesPanel();
    };

    // ==================== ADD / REMOVE FIELDS ====================

    window.showAddFieldDialog = function() {
        var dialog = document.getElementById('add-field-dialog');
        if (dialog) {
            dialog.classList.remove('hidden');
            var input = document.getElementById('new-field-name');
            if (input) {
                input.value = '';
                input.focus();
            }
        }
    };

    window.hideAddFieldDialog = function() {
        var dialog = document.getElementById('add-field-dialog');
        if (dialog) {
            dialog.classList.add('hidden');
        }
    };

    window.confirmAddField = function() {
        var input = document.getElementById('new-field-name');
        if (!input) return;
        var name = input.value.trim();
        if (!name) {
            alert('Please enter a field name.');
            return;
        }
        addField(name);
        hideAddFieldDialog();
    };

    function addField(fieldName) {
        fields.push({
            fieldName: fieldName,
            page: currentPage,
            x: 10,
            y: 10,
            width: 20,
            height: 3,
            fontFamily: null,
            fontSize: null,
            confidence: 1.0,
            reasoning: ''
        });
        markUnsaved();
        selectField(fields.length - 1);
    }

    window.removeField = function() {
        if (selectedField !== null && selectedField < fields.length) {
            if (!confirm('Remove field "' + fields[selectedField].fieldName + '"?')) return;
            fields.splice(selectedField, 1);
            selectedField = null;
            markUnsaved();
            renderFields();
            updatePropertiesPanel();
        }
    };

    // ==================== PAGE NAVIGATION ====================

    window.prevPage = function() {
        if (currentPage > 1) {
            currentPage--;
            updatePageIndicator();
            renderFields();
        }
    };

    window.nextPage = function() {
        if (currentPage < totalPages) {
            currentPage++;
            updatePageIndicator();
            renderFields();
        }
    };

    function updatePageIndicator() {
        var indicator = document.getElementById('page-indicator');
        if (indicator) {
            indicator.textContent = 'Page ' + currentPage + ' of ' + totalPages;
        }
    }

    // ==================== STATUS BAR ====================

    function updateStatusBar() {
        var fieldCountEl = document.getElementById('status-field-count');
        if (fieldCountEl) {
            fieldCountEl.textContent = fields.length + ' field' + (fields.length !== 1 ? 's' : '') + ' placed';
        }
    }

    function markUnsaved() {
        hasUnsavedChanges = true;
        var statusEl = document.getElementById('status-changes');
        if (statusEl) {
            statusEl.className = 'text-amber-400 flex items-center gap-1';
            statusEl.innerHTML = '<span class="w-1.5 h-1.5 rounded-full bg-amber-400 inline-block"></span> Unsaved changes';
        }
        var saveStatusEl = document.getElementById('save-status');
        if (saveStatusEl) {
            saveStatusEl.textContent = 'Unsaved changes';
            saveStatusEl.className = 'text-xs text-amber-500 hidden md:inline mr-2';
        }
    }

    function markSaved() {
        hasUnsavedChanges = false;
        var statusEl = document.getElementById('status-changes');
        if (statusEl) {
            statusEl.className = 'text-green-400 flex items-center gap-1';
            statusEl.innerHTML = '<span class="w-1.5 h-1.5 rounded-full bg-green-400 inline-block"></span> Saved';
        }
        var saveStatusEl = document.getElementById('save-status');
        if (saveStatusEl) {
            saveStatusEl.textContent = 'Saved just now';
            saveStatusEl.className = 'text-xs text-green-500 hidden md:inline mr-2';
        }
    }

    // ==================== SAVE ====================

    window.saveFields = function() {
        var payload = {
            fontFamily: formFontFamily,
            fontSize: formFontSize,
            fields: fields,
            source: 'manual_edit'
        };

        var formId = document.getElementById('form-id').value;
        fetch('/api/forms/' + formId + '/fields', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        })
        .then(function(response) {
            if (!response.ok) {
                throw new Error('Save failed: ' + response.statusText);
            }
            return response.json();
        })
        .then(function(result) {
            markSaved();
            // Update version display
            var versionEl = document.getElementById('status-version');
            if (versionEl && result.version) {
                versionEl.textContent = 'Version ' + result.version;
            }
            loadVersionHistory();
        })
        .catch(function(err) {
            alert('Failed to save: ' + err.message);
        });
    };

    // ==================== ANALYZE WITH AI ====================

    window.analyzeForm = function() {
        var formId = document.getElementById('form-id').value;
        if (hasUnsavedChanges) {
            if (!confirm('You have unsaved changes. Analyzing will create a new version. Continue?')) {
                return;
            }
        }

        fetch('/api/forms/' + formId + '/analyze', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        })
        .then(function(response) {
            if (!response.ok) {
                throw new Error('Analysis failed: ' + response.statusText);
            }
            return response.json();
        })
        .then(function() {
            // Reload the page to show the new AI analysis results
            window.location.reload();
        })
        .catch(function(err) {
            alert('Analysis failed: ' + err.message);
        });
    };

    // ==================== VERSION HISTORY ====================

    function loadVersionHistory() {
        var formId = document.getElementById('form-id').value;
        fetch('/api/forms/' + formId + '/fields/versions')
        .then(function(response) {
            if (!response.ok) {
                throw new Error('Failed to load versions');
            }
            return response.json();
        })
        .then(function(versions) {
            renderVersionHistory(versions);
        })
        .catch(function() {
            // Silently fail - versions panel will just not update
        });
    }

    window.renderVersionHistory = function(versions) {
        var panel = document.getElementById('version-history');
        if (!panel) return;
        panel.innerHTML = '';

        if (!versions || versions.length === 0) {
            var emptyMsg = document.createElement('li');
            emptyMsg.className = 'text-sm text-gray-400 italic p-2';
            emptyMsg.textContent = 'No versions yet';
            panel.appendChild(emptyMsg);
            return;
        }

        // Update status bar with latest version
        var versionEl = document.getElementById('status-version');
        if (versionEl) {
            versionEl.textContent = 'Version ' + versions[0].version;
        }

        for (var i = 0; i < versions.length; i++) {
            var v = versions[i];
            var item = document.createElement('li');

            if (i === 0) {
                item.className = 'p-2.5 rounded-md bg-indigo-50 border border-indigo-600/15';
            } else {
                item.className = 'p-2.5 rounded-md hover:bg-gray-50 cursor-pointer transition-colors group';
            }

            var sourceLabel;
            if (v.source === 'ai_analysis') {
                sourceLabel = '<span class="flex items-center gap-1"><svg class="w-3 h-3 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/></svg> AI Analysis</span>';
            } else if (v.source === 'manual_edit') {
                sourceLabel = 'Manual Edit';
            } else if (v.source && v.source.indexOf('revert_from') === 0) {
                sourceLabel = 'Reverted';
            } else {
                sourceLabel = v.source || 'Unknown';
            }

            var createdAt = v.createdAt ? new Date(v.createdAt).toLocaleString() : '';

            var revertBtn = '';
            if (i > 0) {
                revertBtn = '<button onclick="revertToVersion(' + v.version + ')" class="text-[10px] text-indigo-600 font-medium opacity-0 group-hover:opacity-100 transition-opacity hover:underline">Revert</button>';
            } else {
                revertBtn = '<span class="text-[10px] text-indigo-600 font-medium bg-indigo-100 px-1.5 py-0.5 rounded">Current</span>';
            }

            item.innerHTML =
                '<div class="flex items-center justify-between mb-1">' +
                '  <span class="text-xs font-' + (i === 0 ? 'semibold text-indigo-700' : 'medium text-gray-700') + '">v' + v.version + '</span>' +
                '  ' + revertBtn +
                '</div>' +
                '<div class="text-[11px] text-gray-600">' + sourceLabel + '</div>' +
                '<div class="text-[10px] text-gray-400 mt-0.5">' + escapeHtml(createdAt) + '</div>';

            panel.appendChild(item);
        }
    };

    // ==================== REVERT ====================

    window.revertToVersion = function(version) {
        if (!confirm('Revert to version ' + version + '? This will create a new version.')) return;

        var formId = document.getElementById('form-id').value;
        fetch('/api/forms/' + formId + '/fields/revert/' + version, { method: 'POST' })
        .then(function(response) {
            if (!response.ok) {
                throw new Error('Revert failed: ' + response.statusText);
            }
            return response.json();
        })
        .then(function() {
            window.location.reload();
        })
        .catch(function(err) {
            alert('Revert failed: ' + err.message);
        });
    };

    // ==================== UTILITY ====================

    function escapeHtml(text) {
        if (!text) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(text));
        return div.innerHTML;
    }

    // Warn before leaving with unsaved changes
    window.addEventListener('beforeunload', function(e) {
        if (hasUnsavedChanges) {
            e.preventDefault();
            e.returnValue = '';
        }
    });

})();
