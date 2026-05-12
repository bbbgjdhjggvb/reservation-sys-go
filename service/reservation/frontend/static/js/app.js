// ========== Global Error Handler ==========
window.addEventListener('error', function(e) {
    console.error('[JS Error]', e.message, 'at', e.filename, ':', e.lineno);
});

// ========== State ==========
let authToken = null;
let currentWeekStart = null;
let selectedSlots = [];
const MAX_SLOTS = 4;
let occupiedSlots = {};

// Time slots definition
const TIME_SLOTS = [
    { label: '8:00-10:00',  start: '08:00', end: '10:00' },
    { label: '10:00-12:00', start: '10:00', end: '12:00' },
    { label: '13:00-15:00', start: '13:00', end: '15:00' },
    { label: '15:00-17:00', start: '15:00', end: '17:00' },
];
const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];

// ========== 校友会选项数据 ==========
const ALUMNI_OPTIONS = [
    // 学院/学部校友会
    '计算机与软件学院校友会',
    '电子与信息工程学院校友会',
    '机电与控制工程学院校友会',
    '土木与交通工程学院校友会',
    '建筑与城市规划学院校友会',
    '管理科学学院校友会',
    '经济学院校友会',
    '法学院校友会',
    '师范学院（教育学部）校友会',
    '人文学院校友会',
    '外国语学院校友会',
    '传播学院校友会',
    '设计学院校友会',
    '数学与统计学院校友会',
    '物理与光电工程学院校友会',
    '生命与海洋科学学院校友会',
    '材料学院校友会',
    '化学与环境工程学院校友会',
    '医学部校友会',
    '体育学院校友会',
    '艺术学部校友会',
    // 其他类型
    '继续教育学院校友会',
    '国际交流学院校友会',
    '研究生院校友会',
];

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
            // 隐藏所有功能内容，只显示未授权提示
            const panel = document.querySelector('.bg-white.rounded-lg.shadow');
            if (panel) panel.classList.add('hidden');
            return;
        }

        const today = new Date();
        currentWeekStart = getMonday(today);
        renderCalendar();

        // Character counter
        const reasonEl = document.getElementById('f_reason');
        if (reasonEl) {
            reasonEl.addEventListener('input', function() {
                document.getElementById('reasonCount').textContent = this.value.length;
            });
        }

        // Form submission
        const form = document.getElementById('reserveForm');
        if (form) { form.addEventListener('submit', handleFormSubmit); }

        // Init alumni autocomplete
        initAlumniAutocomplete();
    } catch (err) {
        console.error('[Init] Fatal error:', err);
        const grid = document.getElementById('calendarGrid');
        if (grid) grid.innerHTML = '<div style="grid-column:1/-1;padding:20px;color:red;text-align:center;">页面初始化失败，请按 F12 查看控制台错误</div>';
    }
});

// ========== Date Helpers ==========
function getMonday(d) {
    const date = new Date(d); const day = date.getDay();
    const diff = date.getDate() - day + (day === 0 ? -6 : 1);
    date.setDate(diff); date.setHours(0,0,0,0); return date;
}
function addDays(d, n) { const r = new Date(d); r.setDate(r.getDate() + n); return r; }
function formatDate(d) { return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`; }
function formatDateShort(d) { return `${d.getMonth()+1}/${d.getDate()}`; }
function isSameDay(d1, d2) { return d1.getFullYear()===d2.getFullYear() && d1.getMonth()===d2.getMonth() && d1.getDate()===d2.getDate(); }
function isToday(d) { return isSameDay(d, new Date()); }
function isPastDay(d) { const t=new Date();t.setHours(0,0,0,0);const c=new Date(d);c.setHours(0,0,0,0);return c<t; }
function isBeyondBookable(d) { const m=addDays(new Date(),14);m.setHours(23,59,59,999);return d>m; }
function isSlotInPast(dateStr, endTime) { return new Date(`${dateStr}T${endTime}:00`) <= new Date(); }

// ========== Week Navigation ==========
function changeWeek(delta) { currentWeekStart = addDays(currentWeekStart, delta*7); selectedSlots=[]; renderCalendar(); }

// ========== Render Calendar ==========
async function renderCalendar() {
    try {
        const grid = document.getElementById('calendarGrid'); if (!grid) return;
        const today = new Date(); today.setHours(0,0,0,0);

        const weekEnd = addDays(currentWeekStart, 6);
        document.getElementById('weekLabel').textContent =
            `${currentWeekStart.getFullYear()}年${currentWeekStart.getMonth()+1}月${currentWeekStart.getDate()}日 — ${weekEnd.getMonth()+1}月${weekEnd.getDate()}日`;

        const prevMonday = getMonday(today);
        document.getElementById('prevWeekBtn').disabled = currentWeekStart <= prevMonday;
        document.getElementById('prevWeekBtn').style.opacity = currentWeekStart <= prevMonday ? '0.3' : '1';
        const maxMonday = getMonday(addDays(today, 13));
        document.getElementById('nextWeekBtn').disabled = currentWeekStart >= maxMonday;
        document.getElementById('nextWeekBtn').style.opacity = currentWeekStart >= maxMonday ? '0.3' : '1';

        if (authToken) await loadOccupiedSlotsForWeek();

        let html = '';
        html += '<div class="calendar-header" style="background:#f9fafb;"></div>';
        for (let i=0;i<7;i++) {
            const day = addDays(currentWeekStart, i);
            html += `<div class="calendar-header ${isToday(day)?'today-col':''}">
                <div>周${WEEKDAYS[i]}</div>
                <div class="calendar-date ${isToday(day)?'is-today':(isPastDay(day)?'is-past':'')}">${formatDateShort(day)}</div>
            </div>`;
        }

        TIME_SLOTS.forEach(slot => {
            html += `<div class="time-label">${slot.label}</div>`;
            for (let i=0;i<14;i++) {
                const day = addDays(currentWeekStart, i); const ds=formatDate(day);
                const inPast=isSlotInPast(ds,slot.end), beyond=isBeyondBookable(day), pastD=isPastDay(day);
                const occ=getSlotStatus(ds, slot.start, slot.end);
                const sel=selectedSlots.some(s=>s.date===ds && s.startTime===slot.start && s.endTime===slot.end);
                let cls='time-slot', text='可选';
                if(pastD||inPast){cls+=' slot-past';text='已过'}
                else if(beyond){cls+=' slot-past';text='超出'}
                else if(occ==='approved'){cls+=' slot-occupied';text='已占用'}
                else if(occ==='pending'){cls+=' slot-pending';text='待审核'}
                else if(sel){cls+=' slot-selected';text='已选'}
                const click=!pastD&&!inPast&&!beyond&&!occ;
                html+=`<div class="${cls}"${click?` onclick="selectSlot('${ds}','${slot.start}','${slot.end}')"`:''}>${text}</div>`;
            }
        });

        grid.innerHTML = html;
        updateSelectedInfo();
    } catch(err) {
        console.error('[renderCalendar] Error:', err);
        const g=document.getElementById('calendarGrid');
        if(g)g.innerHTML='<div style="grid-column:1/-1;padding:20px;color:red;text-align:center;">日历渲染失败: '+err.message+'</div>';
    }
}

// ========== Occupied Slots ==========
async function loadOccupiedSlotsForWeek() {
    const ps=[];
    for(let i=0;i<7;i++){
        const ds=formatDate(addDays(currentWeekStart,i));
        if(!occupiedSlots[ds]) ps.push(loadOccupiedSlotsForDate(ds));
    }
    await Promise.allSettled(ps);
}
async function loadOccupiedSlotsForDate(ds){
    try{
        const d=await apiRequest(`/reservation/occupied?date=${ds}`);
        occupiedSlots[ds]=(d.code===200&&d.data)?d.data:[];
    }catch(e){occupiedSlots[ds]=[]}
}

function getSlotStatus(dateStr, startTime, endTime) {
    const slots=occupiedSlots[dateStr]||[];
    for(const s of slots){
        const sE=extractTime(s.end_time), sS=extractTime(s.start_time);
        if(startTime<sE&&endTime>sS){
            // /reservation/occupied 接口返回 status 为字符串: "pending" / "approved"
            // 只有待审核和已通过会占用时段；已拒绝/已完成/已取消的时段不会出现在结果中（后端已过滤）
            if(s.status==='approved')return 'approved';
            if(s.status==='pending')return 'pending';
        }
    }return null;
}
function extractTime(dt){return dt.includes(' ')?dt.split(' ')[1].substring(0,5):dt.substring(0,5);}

// ========== Slot Selection ==========
function selectSlot(date, startTime, endTime) {
    const idx=selectedSlots.findIndex(s=>s.date===date&&s.startTime===startTime&&s.endTime===endTime);
    if(idx>=0){selectedSlots.splice(idx,1)}
    else{if(selectedSlots.length>=MAX_SLOTS){showToast(`最多可选择${MAX_SLOTS}个时间段`,'warning');return}
        selectedSlots.push({date,startTime,endTime})}
    renderCalendar();
}
function formatSlotDisplay(slot){const dn=['日','一','二','三','四','五','六'];const d=new Date(slot.date);return`${d.getMonth()+1}月${d.getDate()}日(周${dn[d.getDay()]}) ${slot.startTime}-${slot.endTime}`;}

function updateSelectedInfo(){
    const info=document.getElementById('selectedInfo'),txt=document.getElementById('selectedText'),btn=document.getElementById('nextStepBtn');
    if(selectedSlots.length>0){info.classList.remove('hidden');txt.innerHTML=selectedSlots.map(s=>`<div>${formatSlotDisplay(s)}</div>`).join('');btn.classList.remove('btn-disabled');btn.style.opacity='1';btn.style.cursor='pointer';btn.textContent='下一步：填写信息（已选 '+selectedSlots.length+'/'+MAX_SLOTS+'）'}
    else{info.classList.add('hidden');btn.classList.add('btn-disabled');btn.style.opacity='0.7';btn.style.cursor='not-allowed';btn.textContent='下一步：填写信息'}
}

// ========== Step Navigation ==========
function goToForm(){
    if(selectedSlots.length===0){showToast('请先选择预约时间段','warning');return}
    document.getElementById('step-calendar').classList.add('hidden');
    document.getElementById('step-form').classList.remove('hidden');
    document.getElementById('formTimeDisplay').innerHTML=selectedSlots.map(s=>`<div class="py-1">${formatSlotDisplay(s)}</div>`).join('')
}
function goToCalendar(){document.getElementById('step-form').classList.add('hidden');document.getElementById('step-calendar').classList.remove('hidden')}

// ========== Form Submit ==========
function handleFormSubmit(e){
    e.preventDefault();
    if(!authToken){showToast('未授权，无法提交','error');return}
    const f=e.target;
    const name=f.applicant_name.value.trim(),year=f.year.value,major=f.major.value.trim(),
          phone=f.phone.value.trim(),reason=f.reason.value.trim();
    const alumniValue = document.getElementById('f_alumni_value').value;

    if(!name||!year||!major||!phone||!reason){showToast('请填写所有必填项','error');return}
    if(!/^\d{4}$/.test(year)){showToast('入学年份请填写4位数字','error');return}
    if(!/^\d{11}$/.test(phone)){showToast('手机号码必须为11位数字','error');return}
    if(!alumniValue||!ALUMNI_OPTIONS.includes(alumniValue)){
        showToast('请从下拉列表中选择所属学院校友会','error');
        closeAlumniDropdown(); return;
    }

    // 构建单次批量提交的请求体
    pendingFormData={
        applicant_name:name, year:parseInt(year), alumni_association:alumniValue,
        major:major, phone:phone, reason:reason,
        slots:selectedSlots.map(s=>({
            start_time:`${s.date} ${s.startTime}:00`,
            end_time:`${s.date} ${s.endTime}:00`
        }))
    };
    showConfirmModal();
}

// ========== Confirm Modal ==========
let pendingFormData = null;  // 单次请求体（含slots数组）

function showConfirmModal(){
    const details=document.getElementById('confirmDetails'), first=pendingFormData, dn=['日','一','二','三','四','五','六'];
    let sl=first.slots.map((slot,i)=>{
        const d=new Date(slot.start_time),ts=`${d.getMonth()+1}月${d.getDate()}日 周${dn[d.getDay()]} ${slot.start_time.split(' ')[1]}-${slot.end_time.split(' ')[1]}`;
        return`<div class="flex justify-between"><span style="color:var(--gray-500)">时段${i+1}</span><span class="font-medium text-gray-800">${ts}</span></div>`;
    }).join('');
    details.innerHTML=`<div class="font-medium mb-2 pb-2 border-b" style="color:var(--primary-color)">共 ${first.slots.length} 个时段</div>${sl}<div class="mt-3 pt-2 border-t space-y-2">
        <div class="flex justify-between"><span class="text-gray-500">申请人</span><span class="font-medium text-gray-800">${first.applicant_name}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">入学年份</span><span class="font-medium text-gray-800">${first.year}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">校友会</span><span class="font-medium text-gray-800">${first.alumni_association}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">专业</span><span class="font-medium text-gray-800">${first.major}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">手机号</span><span class="font-medium text-gray-800">${first.phone}</span></div>
        <div class="flex justify-start gap-2"><span class="text-gray-500 shrink-0">会议内容</span><span class="font-medium text-gray-800">${first.reason}</span></div>
    </div>`;
    document.getElementById('confirmModal').classList.remove('hidden');
}
function closeConfirmModal(){document.getElementById('confirmModal').classList.add('hidden')}

async function doSubmit(){
    const btn=document.getElementById('confirmSubmitBtn'),tx=document.getElementById('confirmSubmitText'),sp=document.getElementById('confirmSubmitSpinner');
    btn.disabled=true; tx.textContent='提交中...'; sp.classList.remove('hidden');
    try{
        const data=await apiRequest('/reservation/submit',{method:'POST',body:JSON.stringify(pendingFormData)});
        if(data.code===200){
            closeConfirmModal();
            showToast(`预约提交成功，共${pendingFormData.slots.length}个时段，请等待审核`,'success');
            // 清除已选时段的缓存
            pendingFormData.slots.forEach(s=>{
                const dateKey=s.start_time.split(' ')[0];
                delete occupiedSlots[dateKey];
            });
            document.getElementById('reserveForm').reset();document.getElementById('reasonCount').textContent='0';
            document.getElementById('f_alumni_value').value='';
            selectedSlots=[];pendingFormData=null;
            goToCalendar();renderCalendar();
        }else{
            showToast(data.msg||'提交失败','error');
        }
    }catch(e){if(e.message!=='Unauthorized')showToast('网络错误，请稍后重试','error')}
    finally{btn.disabled=false;tx.textContent='确认提交';sp.classList.add('hidden')}
}

// ========== API Helper ==========
async function apiRequest(ep,opt={}){
    if(!authToken)throw new Error('NoAuthToken');
    const def={headers:{'Content-Type':'application/json','Authorization':`Bearer ${authToken}`}};
    const r=await fetch(`/api/v2${ep}`,{...def,...opt,headers:{...def.headers,...opt.headers}});
    const d=await r.json();
    if(r.status===401){localStorage.removeItem('auth_token');showToast('登录已过期，请重新进入','error');setTimeout(()=>document.getElementById('tokenError').classList.remove('hidden'),1500);throw new Error('Unauthorized')}
    return d;
}

// ========== Toast ==========
function showToast(msg,type='info'){
    const t=document.getElementById('toast'),c=document.getElementById('toastContent');
    c.className=`px-4 py-3 rounded-lg shadow-lg toast-${type}`;c.textContent=msg;t.classList.remove('hidden');
    setTimeout(()=>t.classList.add('hidden'),3000)
}

// ========== Alumni Association Autocomplete ==========
let alumniActiveIdx = -1;

function initAlumniAutocomplete() {
    const input = document.getElementById('f_alumni');
    const wrapper = document.getElementById('alumniWrapper');
    const dropdown = document.getElementById('alumniDropdown');
    const hidden = document.getElementById('f_alumni_value');

    input.addEventListener('focus', () => { wrapper.classList.add('focused'); filterAlumni(input.value); });
    input.addEventListener('input', () => { hidden.value = ''; filterAlumni(input.value); });
    input.addEventListener('keydown', handleAlumniKeydown);
    input.addEventListener('blur', () => {
        // Delay so click on item can fire before blur hides dropdown
        setTimeout(() => { wrapper.classList.remove('focused'); closeAlumniDropdown(); }, 150);
    });
    // Click outside to close
    document.addEventListener('click', (e) => {
        if (!wrapper.contains(e.target)) closeAlumniDropdown();
    });
}

function handleAlumniKeydown(e) {
    const dd = document.getElementById('alumniDropdown');
    if (!dd.classList.contains('show')) return;
    const items = dd.querySelectorAll('.autocomplete-item');

    switch(e.key) {
        case 'ArrowDown':
            e.preventDefault();
            alumniActiveIdx = Math.min(alumniActiveIdx+1, items.length-1);
            highlightAlumniItem(items); break;
        case 'ArrowUp':
            e.preventDefault();
            alumniActiveIdx = Math.max(alumniActiveIdx-1, 0);
            highlightAlumniItem(items); break;
        case 'Enter':
            e.preventDefault();
            if (alumniActiveIdx>=0 && items[alumniActiveIdx])
                selectAlumniItem(items[alumniActiveIdx].dataset.value);
            break;
        case 'Escape':
            e.preventDefault();
            closeAlumniDropdown(); break;
    }
}

function highlightAlumniItem(items) {
    items.forEach((el,i)=>el.classList.toggle('active',i===alumniActiveIdx));
    if(items[alumniActiveIdx]) items[alumniActiveIdx].scrollIntoView({block:'nearest'});
}

function filterAlumni(query) {
    const dd = document.getElementById('alumniDropdown');
    const q = query.trim().toLowerCase();

    if (!q) {
        // Show all options when empty (or top N)
        showAlumniResults(ALUMNI_OPTIONS.slice(0, 12), '');
        return;
    }

    // Fuzzy match: split query into chars/pinyin-like segments, score each option
    const scored = ALUMNI_OPTIONS.map(opt => ({
        value: opt,
        score: fuzzyScore(q, opt.toLowerCase())
    })).filter(x => x.score > 0).sort((a,b)=>b.score-a.score);

    showAlumniResults(scored.slice(0, 10).map(x=>x.value), q);
}

/**
 * Fuzzy scoring algorithm:
 * - Full prefix match of any segment → high score (100)
 * - Each consecutive char match in a segment → medium score per char (10)
 * - Any char found anywhere → low score (1)
 */
function fuzzyScore(query, target) {
    let score = 0;
    const segs = ['校友会', '学院', '学部'];
    // Remove common suffixes from both for fair comparison
    let t = target;
    segs.forEach(s => { if(t.startsWith(s)) {score += 5;t=t.substring(s.length)}});
    let qi = 0;
    for(let ti=0;ti<t.length&&qi<query.length;){
        if(t[ti]===query[qi]){qi++;score += (qi===query.length && ti===t.length-1)?50:(qi===1?10:5)}
        ti++;
    }
    // Bonus: query as substring anywhere
    if(target.includes(query)) score = Math.max(score, 80);
    // Bonus: each matched word segment
    const qWords = query.replace(/[a-zA-Z]/g,'');
    const tWords = target.replace(/校友会|学院|学部/g,'');
    let wi=0;
    for(let ti=0;ti<tWords.length&&wi<qWords.length;){
        if(tWords[ti]===qWords[wi]){wi++;}
        ti++;
    }
    if(wi===qWords.length) score = Math.max(score, 60 + qWords.length*3);

    return score;
}

function showAlumniResults(options, query) {
    const dd = document.getElementById('alumniDropdown');
    if (options.length === 0) {
        dd.innerHTML = '<div class="autocomplete-empty">无匹配结果</div>';
        dd.classList.add('show');
        alumniActiveIdx = -1;
        return;
    }

    dd.innerHTML = options.map((opt, i) => {
        const label = highlightMatch(opt, query);
        return `<div class="autocomplete-item" data-value="${escapeHtml(opt)}" data-index="${i}" onclick="selectAlumniItem('${escapeHtml(opt)}')">${label}</div>`;
    }).join('');
    dd.classList.add('show');
    alumniActiveIdx = -1;
}

function highlightMatch(text, query) {
    if (!query) return escapeHtml(text);
    const escaped = escapeHtml(text);
    const re = new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'), 'gi');
    return escaped.replace(re, m => `<span class="ac-match">${m}</span>`);
}

function escapeHtml(str) { return str.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }

function selectAlumniItem(value) {
    const input = document.getElementById('f_alumni');
    const hidden = document.getElementById('f_alumni_value');
    input.value = value.replace(/校友会$/, '');
    hidden.value = value;
    closeAlumniDropdown();
}

function closeAlumniDropdown() {
    const dd = document.getElementById('alumniDropdown');
    dd.classList.remove('show'); dd.innerHTML = '';
    alumniActiveIdx = -1;
}
