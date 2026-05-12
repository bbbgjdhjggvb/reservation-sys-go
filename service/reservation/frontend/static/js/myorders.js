// ========== Global Error Handler ==========
window.addEventListener('error', function(e) {
    console.error('[JS Error]', e.message, 'at', e.filename, ':', e.lineno);
});

// ========== State ==========
let authToken = null;
let ordersData = [];

// ========== Init ==========
document.addEventListener('DOMContentLoaded', function() {
    try {
        const urlParams = new URLSearchParams(window.location.search);
        authToken = urlParams.get('token') || localStorage.getItem('auth_token') || null;

        // 保存token到localStorage（用于跨页面传递）
        if (authToken && !localStorage.getItem('auth_token')) {
            localStorage.setItem('auth_token', authToken);
        }

        if (!authToken) {
            document.getElementById('tokenError').classList.remove('hidden');
            // 隐藏 header 和订单面板，只显示未授权提示
            const appHeader = document.getElementById('appHeader');
            if (appHeader) appHeader.classList.add('hidden');
            const ordersPanel = document.getElementById('ordersPanel');
            if (ordersPanel) ordersPanel.classList.add('hidden');
            return;
        }
        loadMyOrders();
    } catch (err) {
        console.error('[Init] Fatal error:', err);
    }
});

// ========== My Orders ==========
async function loadMyOrders() {
    const container = document.getElementById('ordersContainer');
    const emptyEl = document.getElementById('ordersEmpty');
    const loadingEl = document.getElementById('ordersLoading');

    container.innerHTML = '';
    emptyEl.classList.add('hidden');
    loadingEl.classList.remove('hidden');

    try {
        const d = await apiRequest('/reservation/my');
        loadingEl.classList.add('hidden');

        if (d.code !== 200 || !d.data) {
            emptyEl.classList.remove('hidden');
            return;
        }

        ordersData = Array.isArray(d.data) ? d.data : [];

        if (ordersData.length === 0) {
            emptyEl.classList.remove('hidden');
            return;
        }

        // Sort by created_at desc (newest first)
        ordersData.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

        container.innerHTML = ordersData.map(order => renderOrderCard(order)).join('');
    } catch (e) {
        loadingEl.classList.add('hidden');
        if (e.message !== 'Unauthorized') {
            container.innerHTML = '<div class="text-center py-8 text-sm text-red-500">加载失败，请稍后重试</div>';
        }
    }
}

function renderOrderCard(order) {
    const dn = ['日', '一', '二', '三', '四', '五', '六'];
    const statusMap = { 0: '待审核', 1: '已通过', 2: '已拒绝', 3: '已完成', 4: '已取消', 5: '审核中', 6: '一级审核驳回', 7: '二级审核驳回' };
    const canCancel = order.status === 0 || order.status === 5;

    const slotsHtml = (order.slots || []).map(slot => {
        const sd = new Date(slot.start_time);
        const timePart = slot.start_time.split(' ')[1] + '-' + slot.end_time.split(' ')[1];
        const dateStr = `${sd.getMonth() + 1}月${sd.getDate()}日(周${dn[sd.getDay()]})`;
        const dotClass = slot.status === 1 ? 'slot-dot-approved' :
            slot.status === 2 ? 'slot-dot-rejected' :
                slot.status === 3 ? 'slot-dot-completed' :
                    slot.status === 4 ? 'slot-dot-cancelled' : 'slot-dot-pending';
        return `<span class="order-slot-tag"><span class="slot-status-dot ${dotClass}"></span>${dateStr} ${timePart}</span>`;
    }).join('');

    const cancelBtn = canCancel
        ? `<button class="btn-cancel-order" onclick="showCancelModal(${order.id}, '${order.order_no}')">取消预约</button>`
        : '';

    return `<div class="order-card">
        <div class="order-header">
            <div>
                <div class="order-no">${escapeHtml(order.order_no)}</div>
                <div style="font-size:13px;color:var(--gray-700);margin-top:2px;">${escapeHtml(order.applicant_name)}</div>
            </div>
            <span class="order-status status-${order.status}">${statusMap[order.status] || '未知'}</span>
        </div>
        <div class="order-slots">${slotsHtml || '<span class="text-xs text-gray-400">无时段信息</span>'}</div>
        <div class="order-meta">
            <span>学院：${escapeHtml(order.alumni_association)}</span>
            <span>专业：${escapeHtml(order.major)}</span>
            <span>手机：${order.phone}</span>
            <span>创建时间：${order.created_at || '-'}</span>
        </div>
        ${order.reason ? `<div class="mt-2 text-xs text-gray-400" style="word-break:break-all">事由：${escapeHtml(order.reason)}</div>` : ''}
        ${cancelBtn ? `<div class="order-actions">${cancelBtn}</div>` : ''}
    </div>`;
}

// ========== Cancel Order ==========
let pendingCancelId = null;
let pendingCancelOrderNo = '';

function showCancelModal(orderId, orderNo) {
    pendingCancelId = orderId;
    pendingCancelOrderNo = orderNo;

    const order = ordersData.find(o => o.id === orderId);
    const info = document.getElementById('cancelOrderInfo');
    if (order) {
        const slotsText = (order.slots || []).map(s =>
            `${s.start_time.split(' ')[0]} ${s.start_time.split(' ')[1]}-${s.end_time.split(' ')[1]}`
        ).join('<br>');
        info.innerHTML = `<div><strong>订单号：</strong>${escapeHtml(orderNo)}</div><div class="mt-1"><strong>时段：</strong>${slotsText || '-'}</div>`;
    } else {
        info.innerHTML = `<div><strong>订单号：</strong>${escapeHtml(orderNo)}</div>`;
    }

    document.getElementById('cancelModal').classList.remove('hidden');
}

function closeCancelModal() {
    document.getElementById('cancelModal').classList.add('hidden');
    pendingCancelId = null;
    pendingCancelOrderNo = '';
}

async function doCancel() {
    if (!pendingCancelId) return;
    const btn = document.getElementById('confirmCancelBtn');
    const tx = document.getElementById('confirmCancelText');
    const sp = document.getElementById('confirmCancelSpinner');

    btn.disabled = true;
    tx.textContent = '取消中...';
    sp.classList.remove('hidden');

    try {
        const d = await apiRequest(`/reservation/${pendingCancelId}`, { method: 'DELETE' });
        if (d.code === 200) {
            closeCancelModal();
            showToast('预约已取消', 'success');
            loadMyOrders();
        } else {
            showToast(d.msg || '取消失败', 'error');
        }
    } catch (e) {
        if (e.message !== 'Unauthorized') showToast('网络错误，请稍后重试', 'error');
    } finally {
        btn.disabled = false;
        tx.textContent = '确认取消';
        sp.classList.add('hidden');
    }
}

// ========== API Helper ==========
async function apiRequest(ep, opt = {}) {
    if (!authToken) throw new Error('NoAuthToken');
    const def = { headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` } };
    const r = await fetch(`/api/reservation${ep}`, { ...def, ...opt, headers: { ...def.headers, ...(opt.headers || {}) } });
    const d = await r.json();
    if (r.status === 401) {
        localStorage.removeItem('auth_token');
        showToast('登录已过期，请重新进入', 'error');
        setTimeout(() => document.getElementById('tokenError').classList.remove('hidden'), 1500);
        throw new Error('Unauthorized');
    }
    return d;
}

// ========== Toast ==========
function showToast(msg, type = 'info') {
    const t = document.getElementById('toast'), c = document.getElementById('toastContent');
    c.className = `px-4 py-3 rounded-lg shadow-lg toast-${type}`;
    c.textContent = msg;
    t.classList.remove('hidden');
    setTimeout(() => t.classList.add('hidden'), 3000);
}

// ========== Utils ==========
function escapeHtml(str) {
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
