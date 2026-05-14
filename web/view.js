var params = new URLSearchParams(window.location.search);
var currentTitle = params.has('title') ? params.get('title') : '';
var currentPage = 1;
var pageSize = 50;
var totalLogs = 0;
var autoRefresh = true;
var refreshTimer = null;
var eventSource = null;
var selectedTags = [];
var allTags = [];
var currentLogIds = [];
var currentLogs = [];

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    var div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

function formatTime(timestamp) {
    if (!timestamp) return '-';
    return new Date(timestamp).toLocaleString('zh-CN');
}

function levelClass(level) {
    if (level === 'warn') return 'warn';
    if (level === 'error') return 'error';
    if (level === 'debug') return 'debug';
    return 'info';
}

function buildLogQuery(page) {
    var query = new URLSearchParams();
    query.set('page', String(page));
    query.set('pageSize', String(pageSize));
    if (params.has('title')) query.set('title', currentTitle);

    var level = document.getElementById('levelFilter').value;
    if (level) query.set('level', level);
    if (selectedTags.length > 0) query.set('tag', selectedTags.join(','));
    return query.toString();
}

function renderTags() {
    var container = document.getElementById('tagFilters');
    if (!allTags || allTags.length === 0) {
        container.innerHTML = '<span class="subtitle">暂无标签</span>';
        return;
    }

    var html = '<button type="button" class="tag-btn ' + (selectedTags.length === 0 ? 'active' : '') + '" data-tag="">全部</button>';
    for (var i = 0; i < allTags.length; i++) {
        var tag = allTags[i];
        var active = selectedTags.indexOf(tag) !== -1 ? 'active' : '';
        html += '<button type="button" class="tag-btn ' + active + '" data-tag="' + escapeHtml(tag) + '">' + escapeHtml(tag) + '</button>';
    }
    container.innerHTML = html;
}

function loadTags() {
    var query = new URLSearchParams();
    query.set('title', currentTitle);
    fetch('/api/tags?' + query.toString())
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (data.code !== 0) throw new Error(data.message || '标签加载失败');
            allTags = data.data || [];
            renderTags();
        })
        .catch(function(err) {
            document.getElementById('tagFilters').innerHTML = '<span class="subtitle">' + escapeHtml(err.message) + '</span>';
        });
}

function renderLogs(logs) {
    var container = document.getElementById('logStream');
    if (!logs || logs.length === 0) {
        container.innerHTML = '<div class="empty">暂无日志数据</div>';
        return;
    }

    var html = '';
    for (var i = 0; i < logs.length; i++) {
        var entry = logs[i];
        var preview = entry.message || '';
        if (preview.length > 240) preview = preview.substring(0, 240) + '...';
        html += '<article class="log-card ' + levelClass(entry.level) + '" data-index="' + i + '">' +
            '<div class="log-meta">' +
                '<span class="log-level ' + levelClass(entry.level) + '">' + escapeHtml(String(entry.level || '').toUpperCase()) + '</span>' +
                '<span class="log-timestamp">' + formatTime(entry.timestamp) + '</span>' +
                (entry.tag ? '<span class="badge accent-badge">' + escapeHtml(entry.tag) + '</span>' : '') +
            '</div>' +
            '<div class="log-message">' + escapeHtml(preview) + '</div>' +
        '</article>';
    }
    container.innerHTML = html;
}

function renderPagination() {
    var totalPages = Math.ceil(totalLogs / pageSize);
    var container = document.getElementById('pagination');
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    var html = '';
    if (currentPage > 1) html += '<button class="page-btn" type="button" data-page="' + (currentPage - 1) + '">上一页</button>';
    html += '<span class="subtitle">第 ' + currentPage + ' / ' + totalPages + ' 页</span>';
    if (currentPage < totalPages) html += '<button class="page-btn" type="button" data-page="' + (currentPage + 1) + '">下一页</button>';
    container.innerHTML = html;
}

function loadLogs(page, isAutoRefresh) {
    if (!page) page = 1;
    currentPage = page;
    fetch('/api/logs?' + buildLogQuery(page))
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (data.code !== 0) throw new Error(data.message || '日志加载失败');
            var logs = data.data.logs || [];
            var newIds = logs.map(function(log) { return log.id; });
            if (isAutoRefresh && newIds.join('|') === currentLogIds.join('|')) return;

            currentLogIds = newIds;
            currentLogs = logs;
            totalLogs = data.data.total || 0;
            document.getElementById('logCount').textContent = String(totalLogs);
            renderLogs(logs);
            renderPagination();
        })
        .catch(function(err) {
            document.getElementById('logStream').innerHTML = '<div class="empty">加载失败：' + escapeHtml(err.message) + '</div>';
        });
}

function showModal(index) {
    var entry = currentLogs[index];
    if (!entry) return;
    document.getElementById('modalContent').innerHTML =
        '<div class="log-meta"><span class="log-level ' + levelClass(entry.level) + '">' + escapeHtml(String(entry.level || '').toUpperCase()) + '</span><span class="log-timestamp">' + formatTime(entry.timestamp) + '</span></div>' +
        '<p><span class="subtitle">ID：</span>' + escapeHtml(entry.id) + '</p>' +
        '<p><span class="subtitle">标题：</span>' + escapeHtml(entry.title || '未知') + '</p>' +
        (entry.tag ? '<p><span class="subtitle">标签：</span><span class="badge accent-badge">' + escapeHtml(entry.tag) + '</span></p>' : '') +
        '<p class="subtitle">消息：</p><pre>' + escapeHtml(entry.message) + '</pre>';
    document.getElementById('modal').classList.add('show');
}

function closeModal() {
    document.getElementById('modal').classList.remove('show');
}

var confirmCallback = null;

function showConfirm(title, message, info, onConfirm, type) {
    if (!type) type = 'confirm';
    
    var modal = document.getElementById('confirmModal');
    var iconEl = document.getElementById('confirmIcon');
    var cancelBtn = document.getElementById('confirmCancelBtn');
    var confirmBtn = document.getElementById('confirmBtn');
    
    document.getElementById('confirmTitle').textContent = title;
    document.getElementById('confirmMessage').textContent = message;
    
    var infoEl = document.getElementById('confirmInfo');
    if (info) {
        infoEl.textContent = info;
        infoEl.style.display = 'block';
    } else {
        infoEl.style.display = 'none';
    }
    
    iconEl.className = 'confirm-icon';
    cancelBtn.style.display = 'none';
    confirmBtn.className = 'btn';
    confirmBtn.textContent = '关闭';
    
    if (type === 'confirm') {
        iconEl.innerHTML = '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
        iconEl.style.color = 'var(--error)';
        iconEl.style.background = 'var(--error-bg)';
        cancelBtn.style.display = 'inline-flex';
        cancelBtn.textContent = '取消';
        confirmBtn.className = 'btn danger';
        confirmBtn.textContent = '确认删除';
    } else if (type === 'success') {
        iconEl.innerHTML = '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>';
        iconEl.style.color = 'var(--info)';
        iconEl.style.background = 'var(--info-bg)';
        setTimeout(function() { hideConfirm(); }, 1500);
    } else if (type === 'error') {
        iconEl.innerHTML = '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>';
        iconEl.style.color = 'var(--error)';
        iconEl.style.background = 'var(--error-bg)';
    } else if (type === 'info') {
        iconEl.innerHTML = '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>';
        iconEl.style.color = 'var(--accent)';
        iconEl.style.background = 'var(--accent-bg)';
    }
    
    confirmCallback = onConfirm;
    modal.classList.add('show');
}

function hideConfirm() {
    document.getElementById('confirmModal').classList.remove('show');
    confirmCallback = null;
}

function initConfirmModal() {
    document.getElementById('confirmCancelBtn').addEventListener('click', hideConfirm);
    document.getElementById('confirmBtn').addEventListener('click', function() {
        if (confirmCallback) {
            var callback = confirmCallback;
            hideConfirm();
            callback();
        } else {
            hideConfirm();
        }
    });
    document.querySelector('#confirmModal .confirm-overlay').addEventListener('click', hideConfirm);
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && document.getElementById('confirmModal').classList.contains('show')) {
            hideConfirm();
        }
    });
}

function buildStreamUrl() {
    var query = new URLSearchParams();
    if (params.has('title')) query.set('title', currentTitle);

    var level = document.getElementById('levelFilter').value;
    if (level) query.set('level', level);
    if (selectedTags.length > 0) query.set('tag', selectedTags.join(','));
    return '/api/logs/stream?' + query.toString();
}

function closeEventSource() {
    if (eventSource) {
        eventSource.close();
        eventSource = null;
    }
}

function openEventSource() {
    closeEventSource();
    eventSource = new EventSource(buildStreamUrl());
    eventSource.addEventListener('log', function(e) {
        var entry = JSON.parse(e.data);
        if (currentLogIds.indexOf(entry.id) !== -1) return;

        if (entry.tag && allTags.indexOf(entry.tag) === -1) {
            allTags.push(entry.tag);
            allTags.sort();
            renderTags();
        }

        currentLogs.unshift(entry);
        currentLogIds.unshift(entry.id);
        totalLogs++;

        if (currentLogs.length > pageSize) currentLogs.pop();
        if (currentLogIds.length > pageSize) currentLogIds.pop();

        document.getElementById('logCount').textContent = String(totalLogs);
        renderLogs(currentLogs);
        renderPagination();
    });
    eventSource.onerror = function() {
        closeEventSource();
        if (autoRefresh) {
            refreshTimer = setTimeout(openEventSource, 3000);
        }
    };
}

function restartAutoRefresh() {
    if (!autoRefresh) return;
    if (refreshTimer) clearTimeout(refreshTimer);
    openEventSource();
}

function setAutoRefresh(enabled) {
    autoRefresh = enabled;
    var btn = document.getElementById('autoRefreshBtn');
    btn.textContent = enabled ? '开启' : '关闭';
    btn.className = enabled ? 'btn primary' : 'btn';
    if (refreshTimer) clearTimeout(refreshTimer);
    refreshTimer = null;
    if (enabled) openEventSource();
    else closeEventSource();
}

function exportLogs() {
    var query = new URLSearchParams();
    if (params.has('title')) query.set('title', currentTitle);
    var level = document.getElementById('levelFilter').value;
    if (level) query.set('level', level);
    if (selectedTags.length > 0) query.set('tag', selectedTags.join(','));
    window.location.href = '/api/export?' + query.toString();
}

function clearLogs() {
    if (!currentTitle) {
        showConfirm('无法清空', '当前没有标题，无法清空日志', null, null, 'error');
        return;
    }
    
    showConfirm(
        '确认清空日志',
        '确定要清空以下标题的所有日志吗？',
        '标题：' + currentTitle + '\n此操作不可恢复！',
        function() {
            var query = new URLSearchParams();
            query.set('title', currentTitle);
            fetch('/api/logs?' + query.toString(), {
                method: 'DELETE'
            })
            .then(function(res) { return res.json(); })
            .then(function(data) {
                if (data.code !== 0) throw new Error(data.message || '清空失败');
                showConfirm('操作成功', '日志已清空', null, null, 'success');
                loadLogs(1, false);
                loadTags();
            })
            .catch(function(err) {
                showConfirm('清空失败', err.message, null, null, 'error');
            });
        },
        'confirm'
    );
}

document.title = '日志详情 - ' + (currentTitle || '未知');
document.getElementById('pageTitle').textContent = currentTitle || '未知';
document.getElementById('levelFilter').addEventListener('change', function() {
    loadLogs(1, false);
    restartAutoRefresh();
});
document.getElementById('autoRefreshBtn').addEventListener('click', function() { setAutoRefresh(!autoRefresh); });
document.getElementById('exportBtn').addEventListener('click', exportLogs);
document.getElementById('clearBtn').addEventListener('click', clearLogs);
document.getElementById('closeModalBtn').addEventListener('click', closeModal);
document.getElementById('modal').addEventListener('click', function(e) { if (e.target.id === 'modal') closeModal(); });
document.getElementById('tagFilters').addEventListener('click', function(e) {
    if (!e.target.classList.contains('tag-btn')) return;
    var tag = e.target.getAttribute('data-tag') || '';
    if (!tag) {
        selectedTags = [];
    } else {
        var index = selectedTags.indexOf(tag);
        if (index === -1) selectedTags.push(tag);
        else selectedTags.splice(index, 1);
    }
    renderTags();
    loadLogs(1, false);
    restartAutoRefresh();
});
document.getElementById('logStream').addEventListener('click', function(e) {
    var card = e.target.closest('.log-card');
    if (card) showModal(Number(card.getAttribute('data-index')));
});
document.getElementById('pagination').addEventListener('click', function(e) {
    var page = e.target.getAttribute('data-page');
    if (page) loadLogs(Number(page), false);
});

loadTags();
loadLogs(1, false);
setAutoRefresh(true);
initConfirmModal();
