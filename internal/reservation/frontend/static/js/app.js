// Token Management
let authToken = null;

// Initialize
document.addEventListener('DOMContentLoaded', function() {
    // Get token from URL
    const urlParams = new URLSearchParams(window.location.search);
    authToken = urlParams.get('token');

    if (!authToken) {
        // Check localStorage
        authToken = localStorage.getItem('auth_token');
    } else {
        // Save to localStorage
        localStorage.setItem('auth_token', authToken);
    }

    if (!authToken) {
        document.getElementById('tokenError').classList.remove('hidden');
        return;
    }

    // Set minimum date to today
    const dateInput = document.querySelector('input[name="date"]');
    const today = new Date().toISOString().split('T')[0];
    dateInput.min = today;
    dateInput.value = today;

    // Load occupied slots for today
    loadOccupiedSlots();

    // Character counter for reason
    const reasonTextarea = document.querySelector('textarea[name="reason"]');
    reasonTextarea.addEventListener('input', function() {
        document.getElementById('reasonCount').textContent = this.value.length;
    });

    // Form submission
    document.getElementById('reserveForm').addEventListener('submit', handleSubmit);
});

// API Request Helper
async function apiRequest(endpoint, options = {}) {
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authToken}`
        }
    };

    const response = await fetch(`/api/v2${endpoint}`, {
        ...defaultOptions,
        ...options,
        headers: {
            ...defaultOptions.headers,
            ...options.headers
        }
    });

    const data = await response.json();

    if (response.status === 401) {
        localStorage.removeItem('auth_token');
        showToast('登录已过期，请重新进入', 'error');
        setTimeout(() => {
            document.getElementById('tokenError').classList.remove('hidden');
        }, 1500);
        throw new Error('Unauthorized');
    }

    return data;
}

// Load Occupied Slots
async function loadOccupiedSlots() {
    const dateInput = document.querySelector('input[name="date"]');
    const date = dateInput.value;

    if (!date) return;

    try {
        const data = await apiRequest(`/reservation/occupied?date=${date}`);

        if (data.code === 200 && data.data && data.data.length > 0) {
            const slotsContainer = document.getElementById('occupiedSlots');
            const slotsList = document.getElementById('slotsList');

            slotsList.innerHTML = data.data.map(slot => {
                const startTime = new Date(slot.start_time).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
                const endTime = new Date(slot.end_time).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
                const statusClass = slot.status === 'approved' ? 'slot-approved' : 'slot-pending';
                const statusText = slot.status === 'approved' ? '已通过' : '待审核';
                return `<span class="slot-tag ${statusClass}">${startTime} - ${endTime} (${statusText})</span>`;
            }).join('');

            slotsContainer.classList.remove('hidden');
        } else {
            document.getElementById('occupiedSlots').classList.add('hidden');
        }
    } catch (error) {
        console.error('Failed to load occupied slots:', error);
    }
}

// Handle Form Submit
async function handleSubmit(e) {
    e.preventDefault();

    const form = e.target;
    const formData = new FormData(form);
    const date = formData.get('date');
    const startTime = formData.get('start_time');
    const endTime = formData.get('end_time');

    // Validate times
    if (startTime >= endTime) {
        showToast('结束时间必须晚于开始时间', 'error');
        return;
    }

    const submitBtn = document.getElementById('submitBtn');
    const submitText = document.getElementById('submitText');
    const submitSpinner = document.getElementById('submitSpinner');

    // Show loading state
    submitBtn.disabled = true;
    submitText.textContent = '提交中...';
    submitSpinner.classList.remove('hidden');

    try {
        const data = await apiRequest('/reservation/submit', {
            method: 'POST',
            body: JSON.stringify({
                applicant_name: formData.get('applicant_name'),
                alumni_association: formData.get('alumni_association'),
                phone: formData.get('phone'),
                reason: formData.get('reason'),
                start_time: `${date} ${startTime}:00`,
                end_time: `${date} ${endTime}:00`
            })
        });

        if (data.code === 200) {
            showToast('预约提交成功，请等待审核', 'success');
            form.reset();
            document.getElementById('reasonCount').textContent = '0';

            // Reset date to today
            const today = new Date().toISOString().split('T')[0];
            document.querySelector('input[name="date"]').value = today;
            loadOccupiedSlots();
        } else {
            showToast(data.msg || '提交失败', 'error');
        }
    } catch (error) {
        console.error('Submit error:', error);
        if (error.message !== 'Unauthorized') {
            showToast('网络错误，请稍后重试', 'error');
        }
    } finally {
        submitBtn.disabled = false;
        submitText.textContent = '提交预约';
        submitSpinner.classList.add('hidden');
    }
}

// Switch Tab
function switchTab(tab) {
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active', 'border-blue-500', 'text-blue-600');
        btn.classList.add('border-transparent', 'text-gray-500');
    });

    const activeBtn = document.getElementById(`tab-${tab}`);
    activeBtn.classList.add('active', 'border-blue-500', 'text-blue-600');
    activeBtn.classList.remove('border-transparent', 'text-gray-500');

    // Update panels
    document.querySelectorAll('.tab-panel').forEach(panel => {
        panel.classList.add('hidden');
    });

    document.getElementById(`panel-${tab}`).classList.remove('hidden');

    // Load my reservations when switching to my tab
    if (tab === 'my') {
        loadMyReservations();
    }
}

// Load My Reservations
async function loadMyReservations() {
    const loadingState = document.getElementById('loadingState');
    const emptyState = document.getElementById('emptyState');
    const itemsContainer = document.getElementById('reservationItems');

    loadingState.classList.remove('hidden');
    emptyState.classList.add('hidden');
    itemsContainer.innerHTML = '';

    try {
        const data = await apiRequest('/reservation/my');

        loadingState.classList.add('hidden');

        if (data.code === 200 && data.data && data.data.length > 0) {
            itemsContainer.innerHTML = data.data.map(item => createReservationCard(item)).join('');
        } else {
            emptyState.classList.remove('hidden');
        }
    } catch (error) {
        console.error('Failed to load reservations:', error);
        loadingState.classList.add('hidden');
        if (error.message !== 'Unauthorized') {
            showToast('加载失败，请稍后重试', 'error');
        }
    }
}

// Create Reservation Card HTML
function createReservationCard(item) {
    const canCancel = item.status === 0 || item.status === 1;

    return `
        <div class="reservation-card">
            <div class="flex justify-between items-start mb-3">
                <div>
                    <span class="text-sm text-gray-500">订单号：${item.order_no}</span>
                    <h3 class="font-medium text-gray-800 mt-1">${item.applicant_name}</h3>
                </div>
                <span class="status-badge status-${item.status}">${item.status_text}</span>
            </div>
            <div class="space-y-2 text-sm text-gray-600">
                <div class="flex items-center">
                    <svg class="w-4 h-4 mr-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                    </svg>
                    <span>${item.start_time} - ${item.end_time}</span>
                </div>
                <div class="flex items-center">
                    <svg class="w-4 h-4 mr-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z"></path>
                    </svg>
                    <span>${item.phone}</span>
                </div>
                <div class="flex items-start">
                    <svg class="w-4 h-4 mr-2 mt-0.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                    </svg>
                    <span class="flex-1">${item.reason}</span>
                </div>
            </div>
            ${canCancel ? `
                <div class="mt-4 pt-3 border-t border-gray-100 flex justify-end">
                    <button class="btn-cancel" onclick="cancelReservation(${item.id})">取消预约</button>
                </div>
            ` : ''}
        </div>
    `;
}

// Cancel Reservation
async function cancelReservation(id) {
    if (!confirm('确定要取消这个预约吗？')) {
        return;
    }

    try {
        const data = await apiRequest(`/reservation/${id}`, {
            method: 'DELETE'
        });

        if (data.code === 200) {
            showToast('取消成功', 'success');
            loadMyReservations();
        } else {
            showToast(data.msg || '取消失败', 'error');
        }
    } catch (error) {
        console.error('Cancel error:', error);
        if (error.message !== 'Unauthorized') {
            showToast('网络错误，请稍后重试', 'error');
        }
    }
}

// Show Toast
function showToast(message, type = 'info') {
    const toast = document.getElementById('toast');
    const toastContent = document.getElementById('toastContent');

    toastContent.className = `px-4 py-3 rounded-lg shadow-lg toast-${type}`;
    toastContent.textContent = message;

    toast.classList.remove('hidden');

    setTimeout(() => {
        toast.classList.add('hidden');
    }, 3000);
}
