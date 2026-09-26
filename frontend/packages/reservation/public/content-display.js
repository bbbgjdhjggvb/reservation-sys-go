/**
 * 校友之家 - 内容模块展示组件
 * 注入预约首页，监听"场地平面指引""预约使用须知"等按钮点击，弹窗展示后端内容
 * 修复：自动隐藏原系统自带的弹窗，避免弹窗叠加
 * 增强：定时刷新模块列表 + 弹窗显示时实时获取最新内容，确保后台修改后前端实时同步
 */
(function() {
  'use strict';

  const API_BASE = '/api/content';
  const REFRESH_INTERVAL = 30000; // 30秒定时刷新模块列表
  let modulesCache = null;
  let modalEl = null;
  let hiddenOriginalModals = []; // 记录被隐藏的原系统弹窗

  // ============ 初始化 ============
  function init() {
    loadModules();
    createModal();
    bindGlobalClick(); // 事件委托：一次绑定到 document，不怕 Vue 重新渲染导致按钮替换
    // 模块列表若首次加载失败，自动重试
    setTimeout(function(){ if (!modulesCache) loadModules(); }, 1500);
    setTimeout(function(){ if (!modulesCache) loadModules(); }, 4000);
    setTimeout(function(){ if (!modulesCache) loadModules(); }, 8000);
    // 定时刷新模块列表，确保后台修改后前端实时同步
    setInterval(loadModules, REFRESH_INTERVAL);
  }

  // ============ 事件委托（核心修复：按钮何时渲染都能响应） ============
  function bindGlobalClick() {
    document.addEventListener('click', function(e) {
      if (!modulesCache) return; // 模块列表尚未加载，不拦截
      let btn = null;
      if (e.target.closest) {
        btn = e.target.closest('button, [role="button"], .btn, a, [onclick], [class*="btn"]');
      }
      if (!btn) return;
      const text = (btn.textContent || '').trim();
      for (let i = 0; i < modulesCache.length; i++) {
        const module = modulesCache[i];
        if (module.status !== 1 && module.status !== '1') continue;
        const title = (module.title || '').trim();
        if (title && (text === title || text.indexOf(title) !== -1)) {
          e.preventDefault();
          e.stopPropagation();
          if (e.stopImmediatePropagation) e.stopImmediatePropagation();
          showContent(module);
          return;
        }
      }
    }, true);
  }

  // ============ 加载内容模块（带时间戳防缓存） ============
  function loadModules() {
    const ts = Date.now();
    fetch(API_BASE + '/modules?_t=' + ts)
      .then(r => r.json())
      .then(res => {
        if (res.code === 0 && res.data) {
          modulesCache = res.data;
        }
      })
      .catch(() => { console.warn('[内容模块] 加载失败'); });
  }

  // ============ 绑定按钮事件 ============
  function bindButtons() {
    if (!modulesCache) {
      setTimeout(bindButtons, 1000);
      return;
    }

    modulesCache.forEach(module => {
      const buttons = findButtonsByText(module.title);
      buttons.forEach(btn => {
        if (!btn.dataset.contentBound) {
          btn.dataset.contentBound = '1';
          // 使用捕获阶段，确保我们的监听器先执行
          btn.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            e.stopImmediatePropagation();
            showContent(module);
          }, true);
        }
      });
    });
  }

  function findButtonsByText(text) {
    const results = [];
    const allElements = document.querySelectorAll('button, [role="button"], .btn, [class*="btn"], [class*="button"], div[onclick], span[onclick], a');
    allElements.forEach(el => {
      const elText = (el.textContent || '').trim();
      if (elText === text || elText.includes(text)) {
        results.push(el);
      }
    });
    if (results.length === 0) {
      const byKey = document.querySelectorAll('[data-module-key], [data-content-key]');
      byKey.forEach(el => {
        if (el.dataset.moduleKey === text || el.dataset.contentKey === text) {
          results.push(el);
        }
      });
    }
    return results;
  }

  // ============ 隐藏原系统弹窗（核心修复） ============
  function hideOriginalModals() {
    hiddenOriginalModals = [];
    // 延迟执行，等原系统弹窗完全渲染后再隐藏
    setTimeout(() => {
      // 查找所有可能是弹窗的元素（position: fixed 且有半透明背景）
      const allFixed = document.querySelectorAll('*');
      allFixed.forEach(el => {
        if (el.id === 'content-module-modal' || el.closest('#content-module-modal')) return;

        const style = window.getComputedStyle(el);
        if (style.position !== 'fixed') return;

        // 检查是否是弹窗遮罩（背景色半透明，且覆盖全屏）
        const bg = style.backgroundColor;
        const isOverlay = bg && (bg.includes('rgba') || bg.includes('rgb(')) &&
          parseFloat(style.opacity || '1') > 0.1 &&
          (parseInt(style.top) === 0 || style.top === '0px') &&
          (parseInt(style.left) === 0 || style.left === '0px');

        // 或者检查元素内是否包含弹窗相关文本
        const hasModalText = el.textContent && (
          el.textContent.includes('使用须知') ||
          el.textContent.includes('我已阅读并同意') ||
          el.textContent.includes('场地平面') ||
          el.textContent.includes('预约须知')
        );

        if (isOverlay || hasModalText) {
          // 找到最外层的遮罩元素
          let target = el;
          while (target.parentElement) {
            const parentStyle = window.getComputedStyle(target.parentElement);
            if (parentStyle.position === 'fixed') {
              target = target.parentElement;
            } else {
              break;
            }
          }
          if (!target.dataset.contentHidden) {
            target.dataset.contentHidden = '1';
            target.dataset.originalDisplay = target.style.display;
            target.dataset.originalVisibility = target.style.visibility;
            target.style.display = 'none';
            hiddenOriginalModals.push(target);
          }
        }
      });
    }, 150);
  }

  function restoreOriginalModals() {
    hiddenOriginalModals.forEach(el => {
      if (el.dataset.originalDisplay !== undefined) {
        el.style.display = el.dataset.originalDisplay;
      }
      if (el.dataset.originalVisibility !== undefined) {
        el.style.visibility = el.dataset.originalVisibility;
      }
      delete el.dataset.contentHidden;
    });
    hiddenOriginalModals = [];
  }

  // ============ 创建弹窗 ============
  function createModal() {
    modalEl = document.createElement('div');
    modalEl.id = 'content-module-modal';
    modalEl.style.cssText = `
      display: none;
      position: fixed;
      top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(0,0,0,0.65);
      z-index: 2147483647;
      justify-content: center;
      align-items: center;
      padding: 20px;
      animation: cmFadeIn 0.25s ease;
    `;
    modalEl.innerHTML = `
      <div id="content-modal-box" style="
        background: #fff;
        border-radius: 20px;
        width: 100%;
        max-width: 520px;
        max-height: 82vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        animation: cmSlideUp 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
        box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      ">
        <div id="content-modal-header" style="
          padding: 20px 24px;
          background: linear-gradient(135deg, #CE1A20 0%, #a0151a 100%);
          color: #fff;
          display: flex;
          justify-content: space-between;
          align-items: center;
          flex-shrink: 0;
          position: relative;
          overflow: hidden;
        ">
          <div style="display:flex;align-items:center;gap:10px;">
            <span style="font-size:22px;">📋</span>
            <span id="content-modal-title" style="font-size:18px;font-weight:700;letter-spacing:0.5px;"></span>
          </div>
          <span id="content-modal-close" style="
            cursor: pointer;
            font-size: 26px;
            width: 36px;
            height: 36px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 50%;
            transition: all 0.2s;
            line-height: 1;
          " onmouseover="this.style.background='rgba(255,255,255,0.25)';this.style.transform='rotate(90deg)'" onmouseout="this.style.background='transparent';this.style.transform='rotate(0deg)'">×</span>
        </div>
        <div id="content-modal-body" style="
          padding: 24px;
          overflow-y: auto;
          flex: 1;
          font-size: 15px;
          line-height: 1.85;
          color: #333;
          background: #fafafa;
        "></div>
        <div id="content-modal-footer" style="
          padding: 14px 24px;
          background: #fff;
          border-top: 1px solid #f0f0f0;
          text-align: center;
          flex-shrink: 0;
        ">
          <button id="content-modal-confirm" style="
            width: 100%;
            padding: 12px;
            background: linear-gradient(135deg, #CE1A20 0%, #a0151a 100%);
            color: #fff;
            border: none;
            border-radius: 12px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            letter-spacing: 1px;
          " onmouseover="this.style.transform='translateY(-2px)';this.style.boxShadow='0 6px 20px rgba(206,26,32,0.4)'" onmouseout="this.style.transform='translateY(0)';this.style.boxShadow='none'">我已阅读并同意</button>
        </div>
      </div>
    `;
    document.body.appendChild(modalEl);

    // 关闭事件
    modalEl.addEventListener('click', function(e) {
      if (e.target === modalEl || e.target.id === 'content-modal-close' || e.target.id === 'content-modal-confirm') {
        hideModal();
      }
    });

    // 添加动画样式
    const style = document.createElement('style');
    style.textContent = `
      @keyframes cmFadeIn { from { opacity: 0; } to { opacity: 1; } }
      @keyframes cmSlideUp { from { transform: translateY(40px) scale(0.95); opacity: 0; } to { transform: translateY(0) scale(1); opacity: 1; } }
      #content-modal-body img { max-width: 100%; border-radius: 10px; margin: 10px 0; box-shadow: 0 4px 12px rgba(0,0,0,0.1); }
      #content-modal-body p { margin-bottom: 14px; }
      #content-modal-body ul, #content-modal-body ol { margin: 10px 0; padding-left: 24px; }
      #content-modal-body li { margin-bottom: 8px; }
      #content-modal-body h1, #content-modal-body h2, #content-modal-body h3 { margin: 18px 0 10px; color: #CE1A20; font-weight: 700; }
      #content-modal-body strong { color: #CE1A20; }
      .cm-gallery { display: flex; flex-direction: column; gap: 14px; }
      .cm-gallery img { width: 100%; border-radius: 12px; }
      .cm-gallery-desc { color: #666; font-size: 14px; text-align: center; margin-top: 10px; padding: 10px; background: #fff; border-radius: 8px; }
      .cm-loading { text-align: center; padding: 40px; color: #999; }
      .cm-loading::after { content: ''; display: inline-block; width: 20px; height: 20px; border: 2px solid #CE1A20; border-top-color: transparent; border-radius: 50%; animation: cmSpin 0.8s linear infinite; margin-left: 10px; vertical-align: middle; }
      @keyframes cmSpin { to { transform: rotate(360deg); } }
    `;
    document.head.appendChild(style);
  }

  // ============ 展示内容（实时获取最新内容，确保后台修改后前端同步） ============
  function showContent(module) {
    if (!module) return;

    document.getElementById('content-modal-title').textContent = module.title;
    const body = document.getElementById('content-modal-body');
    body.innerHTML = '<div class="cm-loading">加载中</div>';
    modalEl.style.display = 'flex';

    // 隐藏原系统弹窗（核心修复）
    hideOriginalModals();

    // 实时获取最新内容（所有类型都重新获取，确保后台修改后前端同步）
    const ts = Date.now();
    fetch(API_BASE + '/modules/' + module.module_key + '?_t=' + ts)
      .then(r => r.json())
      .then(res => {
        if (res.code === 0 && res.data) {
          const data = res.data;
          if (data.content_type === 'rich_text') {
            body.innerHTML = data.content || '<p style="color:#999;text-align:center;">暂无内容</p>';
          } else if (data.content_type === 'text') {
            body.innerHTML = '<p>' + (data.content || '暂无内容').replace(/\n/g, '<br>') + '</p>';
          } else if (data.content_type === 'image' || data.content_type === 'image_gallery') {
            renderImageContent(data.content);
          }
        } else {
          // 获取失败时，降级使用缓存内容
          renderFallbackContent(module);
        }
      })
      .catch(() => {
        // 网络错误时，降级使用缓存内容
        renderFallbackContent(module);
      });
  }

  // 降级渲染（获取失败时使用缓存内容）
  function renderFallbackContent(module) {
    const body = document.getElementById('content-modal-body');
    if (module.content_type === 'rich_text') {
      body.innerHTML = module.content || '<p style="color:#999;text-align:center;">暂无内容</p>';
    } else if (module.content_type === 'text') {
      body.innerHTML = '<p>' + (module.content || '暂无内容').replace(/\n/g, '<br>') + '</p>';
    } else if (module.content_type === 'image' || module.content_type === 'image_gallery') {
      renderImageContent(module.content);
    }
  }

  function renderImageContent(content) {
    const body = document.getElementById('content-modal-body');
    let imgData = {};
    try {
      imgData = typeof content === 'string' ? JSON.parse(content || '{}') : (content || {});
    } catch (e) {
      console.warn('[内容模块] 图片数据解析失败:', e);
      imgData = { images: [], description: '数据格式错误，请重新上传' };
    }
    const images = imgData.images || [];
    if (images.length === 0) {
      let emptyHtml = '<div style="text-align:center;padding:40px 20px;">';
      emptyHtml += '<div style="font-size:48px;margin-bottom:16px;opacity:0.5;">🖼️</div>';
      emptyHtml += '<p style="color:#999;font-size:15px;margin-bottom:8px;">暂无场地图片</p>';
      if (imgData.description) {
        emptyHtml += '<p style="color:#666;font-size:14px;background:#f5f5f5;padding:12px;border-radius:8px;margin-top:12px;">' + imgData.description + '</p>';
      }
      emptyHtml += '<p style="color:#bbb;font-size:12px;margin-top:16px;">管理员可在后台上传场地平面图</p>';
      emptyHtml += '</div>';
      body.innerHTML = emptyHtml;
      return;
    }
    let html = '<div class="cm-gallery">';
    images.forEach((url, idx) => {
      // 给图片URL添加时间戳，防止浏览器缓存
      const cacheBust = url.indexOf('?') !== -1 ? '&_t=' + Date.now() : '?_t=' + Date.now();
      html += '<img src="' + url + cacheBust + '" alt="场地图片' + (idx + 1) + '" loading="lazy">';
    });
    html += '</div>';
    if (imgData.description) {
      html += '<p class="cm-gallery-desc">' + imgData.description + '</p>';
    }
    body.innerHTML = html;
  }

  function hideModal() {
    modalEl.style.display = 'none';
    // 恢复原系统弹窗
    restoreOriginalModals();
  }

  // ESC 关闭
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && modalEl && modalEl.style.display === 'flex') {
      hideModal();
    }
  });

  // 启动
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
