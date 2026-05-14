function escapeHtml(text) {
    if (!text) return '';
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTime(timestamp) {
    if (!timestamp) return '-';
    return new Date(timestamp).toLocaleString('zh-CN');
}

function renderLevelBadges(levelDist) {
    if (!levelDist) return '';
    var levels = [
        ['info', 'INFO'],
        ['warn', 'WARN'],
        ['error', 'ERROR'],
        ['debug', 'DEBUG']
    ];
    var html = '';
    for (var i = 0; i < levels.length; i++) {
        var key = levels[i][0];
        var label = levels[i][1];
        var count = levelDist[key] || 0;
        if (count > 0) {
            html += '<span class="badge ' + key + '-badge">' + label + ' ' + count + '</span>';
        }
    }
    return html ? '<div class="badges">' + html + '</div>' : '';
}

function renderTitles(titles) {
    var container = document.getElementById('titleList');
    if (!titles || titles.length === 0) {
        container.innerHTML = '<div class="empty">暂无日志数据</div>';
        return;
    }

    var html = '';
    for (var i = 0; i < titles.length; i++) {
        var title = titles[i];
        var viewTitle = title.name === '未知' ? '' : title.name;
        html += '<a class="title-card" href="/view?title=' + encodeURIComponent(viewTitle) + '">' +
            '<div class="title-row">' +
                '<div class="title-name">' + escapeHtml(title.name) + '</div>' +
                '<div class="title-count">' + title.count + '</div>' +
            '</div>' +
            '<div class="title-time">' + formatTime(title.lastTime) + '</div>' +
            renderLevelBadges(title.levelDist) +
        '</a>';
    }
    container.innerHTML = html;
}

function loadTitles() {
    fetch('/api/titles')
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (data.code !== 0) throw new Error(data.message || '加载失败');
            document.getElementById('totalCount').textContent = data.data.total;
            renderTitles(data.data.titles);
        })
        .catch(function(err) {
            document.getElementById('titleList').innerHTML = '<div class="empty">加载失败：' + escapeHtml(err.message) + '</div>';
        });
}

loadTitles();
setInterval(loadTitles, 3000);
