# 修复清空日志后标签不更新的 Bug

## 问题分析

用户反馈：清空日志后，标签（tag）显示不会实时更新，只有手动 F5 刷新才会更新。

### 当前代码逻辑

1. `clearLogs()` 函数在清空成功后：
   - 调用 `showConfirm()` 显示成功提示
   - 调用 `loadLogs(1, false)` 刷新日志列表
   - 调用 `loadTags()` 刷新标签列表

2. `loadTags()` 函数：
   - 向 `/api/tags?title=xxx` 发送请求
   - 获取该标题的所有标签
   - 调用 `renderTags()` 渲染标签

### 可能的问题原因

1. **网络请求顺序问题**：`loadTags()` 可能被调用得太早，后端数据还未完全更新
2. **异步执行顺序**：`loadTags()` 和 `loadLogs()` 并行执行，可能标签获取在日志清空之前完成
3. **浏览器缓存**：虽然可能性较小，但可能存在缓存问题

## 修复方案

### 方案一：确保执行顺序（推荐）

在 `clearLogs()` 中，确保 `loadTags()` 在日志清空请求完成后再执行：

```javascript
function clearLogs() {
    // ...
    fetch('/api/logs?' + query.toString(), {
        method: 'DELETE'
    })
    .then(function(res) { return res.json(); })
    .then(function(data) {
        if (data.code !== 0) throw new Error(data.message || '清空失败');
        
        // 先显示成功提示
        showConfirm('操作成功', '日志已清空', null, null, 'success');
        
        // 然后刷新数据
        loadLogs(1, false);
        loadTags();
    })
    .catch(function(err) {
        showConfirm('清空失败', err.message, null, null, 'error');
    });
}
```

但查看当前代码，这部分逻辑已经是这样的。

### 方案二：添加延迟执行（备选）

如果方案一不起作用，可以给 `loadTags()` 添加短暂延迟，确保 DOM 更新完成：

```javascript
.then(function(data) {
    if (data.code !== 0) throw new Error(data.message || '清空失败');
    showConfirm('操作成功', '日志已清空', null, null, 'success');
    loadLogs(1, false);
    setTimeout(loadTags, 100); // 延迟 100ms 执行
})
```

### 方案三：清空前端标签缓存（彻底解决）

在 `clearLogs()` 成功后，直接清空前端标签数据并渲染：

```javascript
.then(function(data) {
    if (data.code !== 0) throw new Error(data.message || '清空失败');
    showConfirm('操作成功', '日志已清空', null, null, 'success');
    
    // 直接清空前端标签数据
    allTags = [];
    selectedTags = [];
    renderTags();
    
    loadLogs(1, false);
})
```

这样可以立即显示"暂无标签"的状态，不需要等待后端响应。

## 实现步骤

1. **修改 view.js 中的 clearLogs 函数**
   - 在成功回调中，先清空前端标签缓存
   - 然后调用 `renderTags()` 立即更新显示
   - 最后调用 `loadTags()` 从后端获取最新数据

2. **添加清空标签的辅助函数（可选）**
   - 如果需要，可以添加一个 `clearTags()` 函数

## 预期效果

- 清空日志后，标签区域立即显示"暂无标签"
- 同时从后端获取最新数据，确保数据一致性
- 用户体验：操作后立即看到效果，无需等待
