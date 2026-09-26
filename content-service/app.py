"""
校友之家 - 内容管理 API 服务
提供内容模块的 CRUD、图片上传、系统配置管理、通知发送、前端内容展示接口
"""
import os
import json
import time
import uuid
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from email.utils import formataddr
import pymysql
from flask import Flask, request, jsonify, send_from_directory
from flask_cors import CORS
from werkzeug.utils import secure_filename

app = Flask(__name__)
CORS(app)


import hashlib
import hmac
QUICK_REVIEW_SECRET = 'alumni_home_quick_review_secret_2026'
# 对外可访问的站点根地址。新订单通知由预约服务经 Docker 内网调用，
# request.host_url 会得到内网域名（手机无法解析），因此邮件内链接必须使用固定公网域名。
PUBLIC_BASE_URL = os.environ.get('PUBLIC_BASE_URL', 'https://www.szuedf.org.cn').rstrip('/')
def generate_quick_review_token(order_id, action):
    message = f"{order_id}:{action}:{QUICK_REVIEW_SECRET}"
    return hashlib.md5(message.encode()).hexdigest()
def verify_quick_review_token(order_id, action, token):
    expected = generate_quick_review_token(order_id, action)
    return hmac.compare_digest(expected, token)

# ============ 配置 ============
DB_CONFIG = {
    'host': os.environ.get('DB_HOST', 'mysql'),
    'port': int(os.environ.get('DB_PORT', 3306)),
    'user': os.environ.get('DB_USER', 'res_user'),
    'password': os.environ.get('DB_PASSWORD', 'xSIn34sU7qQl31kQ3TVfcQ=='),
    'database': os.environ.get('DB_NAME', 'home_xy'),
    'charset': 'utf8mb4',
    'cursorclass': pymysql.cursors.DictCursor,
}

ADMIN_USERNAME = os.environ.get('ADMIN_USERNAME', 'admin')
ADMIN_PASSWORD = os.environ.get('ADMIN_PASSWORD', 'admin123456')
UPLOAD_FOLDER = os.environ.get('UPLOAD_FOLDER', '/app/uploads')
ALLOWED_EXTENSIONS = {'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'}
MAX_CONTENT_LENGTH = 10 * 1024 * 1024  # 10MB

app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER
app.config['MAX_CONTENT_LENGTH'] = MAX_CONTENT_LENGTH

# 简单的 token 存储（生产环境建议用 Redis）
admin_tokens = {}

# 系统配置缓存（实时更新，不长期缓存）
_config_cache = {'data': None, 'time': 0}
CONFIG_CACHE_TTL = 5  # 5秒缓存，保证实时更新

# ============ 数据库工具 ============
def get_db():
    return pymysql.connect(**DB_CONFIG)

def query_db(sql, args=None, one=False):
    conn = get_db()
    try:
        with conn.cursor() as cur:
            cur.execute(sql, args or ())
            result = cur.fetchall()
            if one:
                return result[0] if result else None
            else:
                return result
    finally:
        conn.close()

def execute_db(sql, args=None):
    conn = get_db()
    try:
        with conn.cursor() as cur:
            cur.execute(sql, args or ())
            conn.commit()
            return cur.lastrowid
    finally:
        conn.close()

# ============ 认证工具 ============
def check_admin_token():
    token = request.headers.get('Authorization', '').replace('Bearer ', '')
    if not token or token not in admin_tokens:
        return False
    # 检查 token 是否过期（24小时）
    if time.time() - admin_tokens[token] > 86400:
        del admin_tokens[token]
        return False
    return True

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

# ============ 公开接口（前端调用） ============

@app.route('/api/content/modules', methods=['GET'])
def get_enabled_modules():
    """获取所有启用的内容模块（前端展示用）"""
    modules = query_db(
        "SELECT id, module_key, title, content_type, content, icon, sort_order, status, updated_at FROM content_modules WHERE status = 1 ORDER BY sort_order ASC"
    )
    # 解析 content 字段
    for m in modules:
        try:
            if m['content_type'] == 'image':
                m['content'] = json.loads(m['content']) if m['content'] else {'images': [], 'description': ''}
        except:
            pass
    return jsonify({'code': 0, 'data': modules, 'total': len(modules)})

@app.route('/api/content/modules/<module_key>', methods=['GET'])
def get_module_by_key(module_key):
    """根据 module_key 获取单个模块内容"""
    module = query_db(
        "SELECT id, module_key, title, content_type, content, icon, sort_order, status, updated_at FROM content_modules WHERE module_key = %s AND status = 1",
        (module_key,), one=True
    )
    if not module:
        return jsonify({'code': 404, 'msg': '模块不存在或已禁用'}), 404
    try:
        if module['content_type'] == 'image':
            module['content'] = json.loads(module['content']) if module['content'] else {'images': [], 'description': ''}
    except:
        pass
    return jsonify({'code': 0, 'data': module})

# ============ 管理接口（需认证） ============

@app.route('/api/content/admin/login', methods=['POST'])
def admin_login():
    """管理员登录"""
    data = request.get_json() or {}
    username = data.get('username', '')
    password = data.get('password', '')
    if username == ADMIN_USERNAME and password == ADMIN_PASSWORD:
        token = str(uuid.uuid4())
        admin_tokens[token] = time.time()
        return jsonify({'code': 0, 'data': {'token': token, 'username': username}})
    return jsonify({'code': 401, 'msg': '用户名或密码错误'}), 401

@app.route('/api/content/admin/modules', methods=['GET'])
def admin_get_all_modules():
    """获取所有模块（含禁用的，管理后台用）"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录或登录已过期'}), 401
    modules = query_db(
        "SELECT * FROM content_modules ORDER BY sort_order ASC, id DESC"
    )
    for m in modules:
        try:
            if m['content_type'] == 'image':
                m['content'] = json.loads(m['content']) if m['content'] else {'images': [], 'description': ''}
        except:
            pass
    return jsonify({'code': 0, 'data': modules, 'total': len(modules)})

@app.route('/api/content/admin/modules', methods=['POST'])
def admin_create_module():
    """创建模块"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    module_key = data.get('module_key', '').strip()
    title = data.get('title', '').strip()
    content_type = data.get('content_type', 'rich_text')
    content = data.get('content', '')
    icon = data.get('icon', '')
    sort_order = int(data.get('sort_order', 0))
    status = int(data.get('status', 1))

    if not module_key or not title:
        return jsonify({'code': 400, 'msg': '模块标识和标题不能为空'}), 400

    # 检查 module_key 是否重复
    existing = query_db("SELECT id FROM content_modules WHERE module_key = %s", (module_key,), one=True)
    if existing:
        return jsonify({'code': 400, 'msg': '模块标识已存在'}), 400

    # image 类型的 content 需要序列化为 JSON
    if content_type == 'image' and isinstance(content, dict):
        content = json.dumps(content, ensure_ascii=False)

    new_id = execute_db(
        "INSERT INTO content_modules (module_key, title, content_type, content, icon, sort_order, status) VALUES (%s, %s, %s, %s, %s, %s, %s)",
        (module_key, title, content_type, content, icon, sort_order, status)
    )
    return jsonify({'code': 0, 'data': {'id': new_id}, 'msg': '创建成功'})

@app.route('/api/content/admin/modules/<int:module_id>', methods=['PUT'])
def admin_update_module(module_id):
    """更新模块"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}

    existing = query_db("SELECT id FROM content_modules WHERE id = %s", (module_id,), one=True)
    if not existing:
        return jsonify({'code': 404, 'msg': '模块不存在'}), 404

    fields = []
    values = []
    for key in ['title', 'content_type', 'content', 'icon', 'sort_order', 'status', 'module_key']:
        if key in data:
            val = data[key]
            if key == 'content_type' and val == 'image' and isinstance(data.get('content'), dict):
                data['content'] = json.dumps(data['content'], ensure_ascii=False)
            fields.append(f"{key} = %s")
            values.append(val)

    if not fields:
        return jsonify({'code': 400, 'msg': '没有需要更新的字段'}), 400

    values.append(module_id)
    execute_db(f"UPDATE content_modules SET {', '.join(fields)} WHERE id = %s", tuple(values))
    return jsonify({'code': 0, 'msg': '更新成功'})

@app.route('/api/content/admin/modules/<int:module_id>', methods=['DELETE'])
def admin_delete_module(module_id):
    """删除模块"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    existing = query_db("SELECT id FROM content_modules WHERE id = %s", (module_id,), one=True)
    if not existing:
        return jsonify({'code': 404, 'msg': '模块不存在'}), 404
    execute_db("DELETE FROM content_modules WHERE id = %s", (module_id,))
    return jsonify({'code': 0, 'msg': '删除成功'})

@app.route('/api/content/admin/sort', methods=['POST'])
def admin_update_sort():
    """批量更新排序"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    items = data.get('items', [])
    for item in items:
        if 'id' in item and 'sort_order' in item:
            execute_db("UPDATE content_modules SET sort_order = %s WHERE id = %s", (item['sort_order'], item['id']))
    return jsonify({'code': 0, 'msg': '排序更新成功'})

# ============ 图片上传 ============

@app.route('/api/content/admin/upload', methods=['POST'])
def admin_upload_image():
    """上传图片"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    if 'file' not in request.files:
        return jsonify({'code': 400, 'msg': '没有上传文件'}), 400
    file = request.files['file']
    if file.filename == '':
        return jsonify({'code': 400, 'msg': '没有选择文件'}), 400
    if file and allowed_file(file.filename):
        ext = file.filename.rsplit('.', 1)[1].lower()
        filename = f"{int(time.time())}_{uuid.uuid4().hex[:8]}.{ext}"
        filepath = os.path.join(app.config['UPLOAD_FOLDER'], filename)
        file.save(filepath)
        # 返回可访问的 URL
        url = f"/uploads/{filename}"
        return jsonify({'code': 0, 'data': {'url': url, 'filename': filename}})
    return jsonify({'code': 400, 'msg': '不支持的文件格式'}), 400

@app.route('/uploads/<filename>')
def uploaded_file(filename):
    """访问上传的图片"""
    return send_from_directory(app.config['UPLOAD_FOLDER'], filename)

# ============ 健康检查 ============

@app.route('/api/content/health', methods=['GET'])
def health():
    try:
        query_db("SELECT 1")
        return jsonify({'code': 0, 'msg': 'ok', 'db': 'connected'})
    except Exception as e:
        return jsonify({'code': 500, 'msg': 'db error', 'error': str(e)}), 500

# ============ 系统配置管理 ============

def get_config(key, default=''):
    """获取配置值"""
    result = query_db("SELECT config_value FROM system_config WHERE config_key = %s", (key,), one=True)
    return result['config_value'] if result else default

def set_config(key, value):
    """设置配置值"""
    existing = query_db("SELECT id FROM system_config WHERE config_key = %s", (key,), one=True)
    if existing:
        execute_db("UPDATE system_config SET config_value = %s WHERE config_key = %s", (value, key))
    else:
        execute_db("INSERT INTO system_config (config_key, config_value) VALUES (%s, %s)", (key, value))

@app.route('/api/content/admin/config', methods=['GET'])
def admin_get_config():
    """获取所有系统配置"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    configs = query_db("SELECT config_key, config_value, config_type, description FROM system_config ORDER BY id")
    return jsonify({'code': 0, 'data': configs})

@app.route('/api/content/admin/config', methods=['POST'])
def admin_update_config():
    """批量更新系统配置"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    configs = data.get('configs', {})
    for key, value in configs.items():
        set_config(key, str(value))
    return jsonify({'code': 0, 'msg': '配置更新成功，实时生效'})

# ============ 通知接收人管理 ============

@app.route('/api/content/admin/recipients', methods=['GET'])
def admin_get_recipients():
    """获取通知接收人列表"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    recipients = query_db("SELECT * FROM notify_recipients ORDER BY type, id")
    return jsonify({'code': 0, 'data': recipients, 'total': len(recipients)})

@app.route('/api/content/admin/recipients', methods=['POST'])
def admin_add_recipient():
    """添加通知接收人"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    rtype = data.get('type', 'wechat')
    value = data.get('value', '').strip()
    name = data.get('name', '')
    if not value:
        return jsonify({'code': 400, 'msg': '接收人值不能为空'}), 400
    new_id = execute_db(
        "INSERT INTO notify_recipients (type, value, name, enabled) VALUES (%s, %s, %s, 1)",
        (rtype, value, name)
    )
    return jsonify({'code': 0, 'data': {'id': new_id}, 'msg': '添加成功'})

@app.route('/api/content/admin/recipients/<int:rid>', methods=['DELETE'])
def admin_delete_recipient(rid):
    """删除通知接收人"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    execute_db("DELETE FROM notify_recipients WHERE id = %s", (rid,))
    return jsonify({'code': 0, 'msg': '删除成功'})

@app.route('/api/content/admin/recipients/<int:rid>/toggle', methods=['POST'])
def admin_toggle_recipient(rid):
    """切换接收人启用状态"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    execute_db("UPDATE notify_recipients SET enabled = NOT enabled WHERE id = %s", (rid,))
    return jsonify({'code': 0, 'msg': '状态已更新'})

# ============ 微信 access_token 管理 ============

wechat_token_cache = {'token': '', 'expires_at': 0}

def get_wechat_access_token():
    """获取微信 access_token（带缓存）"""
    if wechat_token_cache['token'] and time.time() < wechat_token_cache['expires_at']:
        return wechat_token_cache['token']

    appid = get_config('wechat_appid')
    appsecret = get_config('wechat_appsecret')
    if not appid or not appsecret:
        return None

    try:
        import urllib.request
        url = f"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid={appid}&secret={appsecret}"
        with urllib.request.urlopen(url, timeout=10) as resp:
            result = json.loads(resp.read().decode())
            if 'access_token' in result:
                wechat_token_cache['token'] = result['access_token']
                wechat_token_cache['expires_at'] = time.time() + result.get('expires_in', 7200) - 300
                return result['access_token']
    except Exception as e:
        print(f"获取access_token失败: {e}")
    return None

# ============ 公众号菜单管理 ============

@app.route('/api/content/admin/wechat/menu', methods=['GET'])
def admin_get_wechat_menu():
    """获取公众号当前菜单"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    access_token = get_wechat_access_token()
    if not access_token:
        return jsonify({'code': 400, 'msg': '请先配置公众号AppID和AppSecret'}), 400
    try:
        import urllib.request
        url = f"https://api.weixin.qq.com/cgi-bin/menu/get?access_token={access_token}"
        with urllib.request.urlopen(url, timeout=10) as resp:
            result = json.loads(resp.read().decode())
            return jsonify({'code': 0, 'data': result})
    except Exception as e:
        return jsonify({'code': 500, 'msg': f'获取菜单失败: {str(e)}'}), 500

@app.route('/api/content/admin/wechat/menu', methods=['POST'])
def admin_create_wechat_menu():
    """创建公众号菜单"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    access_token = get_wechat_access_token()
    if not access_token:
        return jsonify({'code': 400, 'msg': '请先配置公众号AppID和AppSecret'}), 400
    data = request.get_json() or {}
    menu_data = data.get('menu', {})
    try:
        import urllib.request
        url = f"https://api.weixin.qq.com/cgi-bin/menu/create?access_token={access_token}"
        req = urllib.request.Request(url, data=json.dumps(menu_data, ensure_ascii=False).encode('utf-8'), headers={'Content-Type': 'application/json'})
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read().decode())
            if result.get('errcode') == 0:
                return jsonify({'code': 0, 'msg': '菜单创建成功'})
            else:
                return jsonify({'code': 500, 'msg': result.get('errmsg', '创建失败')}), 500
    except Exception as e:
        return jsonify({'code': 500, 'msg': f'创建菜单失败: {str(e)}'}), 500

# ============ 通知发送 ============


# ============ 模板字段自动解析与智能适配 ============
_tfcache = {}
def get_tpl_fields(tid):
    if not tid: return {}
    if tid in _tfcache: return _tfcache[tid]
    try:
        tok = get_wechat_access_token()
        if not tok: return {}
        import urllib.request, json, re as _re
        url = f"https://api.weixin.qq.com/cgi-bin/template/get_all_private_template?access_token={tok}"
        r = json.loads(urllib.request.urlopen(url, timeout=10).read())
        flds = {}
        for t in r.get('template_list', []):
            if t.get('template_id') == tid:
                tc = t.get('content', '')
                for m in _re.finditer(r'([^\n{]+?)[：:]\s*\{\{([A-Za-z_]\w*)\.DATA\}\}', tc):
                    flds[m.group(2)] = m.group(1).strip()
                if '{{first.DATA}}' in tc: flds['first'] = 'first'
                if '{{remark.DATA}}' in tc: flds['remark'] = 'remark'
                break
        _tfcache[tid] = flds
        return flds
    except Exception as e:
        print(f"[WARN] 模板字段获取失败: {e}")
        return {}

def match_val(label, v):
    if any(k in label for k in ['预约人', '申请人', '姓名', '用户']): return v.get('applicant_name', '-')
    if any(k in label for k in ['场地名称', '场地', '会议室', '地点', '场所']): return v.get('venue', '深圳大学校友之家')
    if any(k in label for k in ['预约时间', '时段', '时间', '日期', '使用时间']): return v.get('slots_text', '-')
    if any(k in label for k in ['开门密码', '密码', '门锁', '开锁']): return v.get('password', v.get('pwd_text', '无'))
    if any(k in label for k in ['对接人', '联系', '电话', '联系方式', '联系人']): return '-'  # 已删除对接人功能，避免信息泄露
    if any(k in label for k in ['审核状态', '审核结果', '状态', '结果']): return v.get('status_text', '-')
    if any(k in label for k in ['事由', '用途', '原因', '使用事由']): return v.get('reason', '-')
    if any(k in label for k in ['订单号', '订单', '编号', '流水号']): return v.get('order_no', '-')
    if any(k in label for k in ['人数', '参与人数', '参会人数']): return v.get('attendee_count', '-')
    return v.get(label, '-')

def build_tpl_data(tid, v, first_text='', remark_text='', first_color=None):
    flds = get_tpl_fields(tid)
    if not flds: return {}
    d = {}
    for kw, label in flds.items():
        if kw == 'first':
            d['first'] = {'value': first_text}
            if first_color: d['first']['color'] = first_color
        elif kw == 'remark':
            d['remark'] = {'value': remark_text}
        else:
            d[kw] = {'value': str(match_val(label, v))}
    return d

def send_wechat_template_message(openid, template_id, data):
    """发送微信模板消息"""
    access_token = get_wechat_access_token()
    if not access_token or not template_id:
        return False
    try:
        import urllib.request
        url = f"https://api.weixin.qq.com/cgi-bin/message/template/send?access_token={access_token}"
        payload = {
            'touser': openid,
            'template_id': template_id,
            'data': data
        }
        req = urllib.request.Request(url, data=json.dumps(payload, ensure_ascii=False).encode('utf-8'), headers={'Content-Type': 'application/json'})
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read().decode())
            if result.get('errcode') != 0:
                print(f"[WARN] 微信模板消息返回错误: errcode={result.get('errcode')} errmsg={result.get('errmsg')} openid={openid[:12]}... tpl={template_id[:14]}...")
            return result.get('errcode') == 0
    except Exception as e:
        print(f"发送模板消息失败: {e}")
        return False

def send_email(to_email, subject, body):
    """发送邮件"""
    smtp_host = get_config('email_smtp_host')
    smtp_port = int(get_config('email_smtp_port', '465'))
    username = get_config('email_username')
    password = get_config('email_password')
    from_name = get_config('email_from_name', '校友之家')

    if not smtp_host or not username or not password:
        return False

    try:
        import smtplib
        from email.mime.text import MIMEText
        from email.header import Header

        msg = MIMEText(body, 'html', 'utf-8')
        msg['From'] = formataddr((str(Header(from_name, 'utf-8')), username))
        msg['To'] = to_email
        msg['Subject'] = Header(subject, 'utf-8')

        if smtp_port == 465:
            server = smtplib.SMTP_SSL(smtp_host, smtp_port, timeout=10)
        else:
            server = smtplib.SMTP(smtp_host, smtp_port, timeout=10)
            server.starttls()
        server.login(username, password)
        server.sendmail(username, [to_email], msg.as_string())
        server.quit()
        return True
    except Exception as e:
        print(f"发送邮件失败: {e}")
        return False

@app.route('/api/content/admin/notify/test', methods=['POST'])
def admin_send_test_notify():
    """发送测试通知"""
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    notify_type = data.get('type', 'wechat')
    target = data.get('target', '')

    if notify_type == 'wechat':
        template_id = get_config('template_id_new_order')
        result = send_wechat_template_message(target, template_id, {
            'first': {'value': '测试通知', 'color': '#10B981'},
            'keyword1': {'value': '测试预约人'},
            'keyword2': {'value': '测试场地'},
            'keyword3': {'value': '2026-01-01 10:00-12:00'},
            'keyword4': {'value': '测试状态'},
            'remark': {'value': '\n这是一条测试通知'}
        })
        return jsonify({'code': 0 if result else 500, 'msg': '微信消息发送成功' if result else '发送失败'})
    else:
        result = send_email(target, '测试通知', '<h2>校友之家测试通知</h2><p>这是一条测试邮件，用于验证邮件通知功能是否正常。</p>')
        return jsonify({'code': 0 if result else 500, 'msg': '邮件发送成功' if result else '发送失败'})

@app.route('/api/content/notify/new-order', methods=['POST'])
def notify_new_order():
    """新预约通知（供预约系统回调）"""
    data = request.get_json() or {}
    order_no = data.get('order_no', '')
    applicant = data.get('applicant_name', '')
    phone = data.get('phone', '')
    slots = data.get('slots', [])
    reason = data.get('reason', '')

    # 获取启用的接收人
    recipients = query_db("SELECT * FROM notify_recipients WHERE enabled = 1")

    wechat_results = []
    email_results = []

    template_id = get_config('template_id_new_order')
    slots_text = '、'.join(filter(None, (_fmt_slot_cn(s.get('start_time', ''), s.get('end_time', '')) for s in slots))) or '-'

    for r in recipients:
        if r['type'] == 'wechat' and template_id:
            _v = {'applicant_name': applicant, 'venue': '深圳大学校友之家', 'slots_text': slots_text, 'status_text': status_text, 'order_no': order_no, 'phone': phone, 'reason': reason}
            _d = build_tpl_data(template_id, _v, first_text='新的场地预约申请，请及时审核', remark_text=f'\n订单号: {order_no}\n联系电话: {phone}\n使用事由: {reason}', first_color='#CE1A20')
            result = send_wechat_template_message(r['value'], template_id, _d)
            wechat_results.append({'name': r['name'], 'success': result})
        elif r['type'] == 'email':
            subject = f"新预约通知 - {applicant} - {slots_text}"
            approve_token = generate_quick_review_token(order_no, 'approve')
            reject_token = generate_quick_review_token(order_no, 'reject')
            # 必须使用公网域名：本接口由预约服务经内网调用，request.host_url 是内网地址，手机无法打开
            base_url = PUBLIC_BASE_URL
            approve_url = base_url + '/api/content/notify/quick-review?order_id=' + order_no + '&action=approve&token=' + approve_token
            reject_url = base_url + '/api/content/notify/quick-review?order_id=' + order_no + '&action=reject&token=' + reject_token
            body = f'''
            <div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif;">
                <div style="background:linear-gradient(135deg,#ce1a20,#a01818);color:#fff;padding:24px;border-radius:12px 12px 0 0;">
                    <h2 style="margin:0;font-size:22px;">📋 新的场地预约申请</h2>
                    <p style="margin:8px 0 0;opacity:0.9;font-size:14px;">请及时审核以下预约申请</p>
                </div>
                <div style="background:#fff;padding:24px;border:1px solid #e5e7eb;border-top:none;">
                    <p><strong>预约人：</strong>{applicant}</p>
                    <p><strong>联系电话：</strong>{phone}</p>
                    <p><strong>预约时段：</strong>{slots_text}</p>
                    <p><strong>使用事由：</strong>{reason}</p>
                    <p><strong>订单号：</strong>{order_no}</p>
                    <div style="margin-top:24px;">
                        <p style="color:#6b7280;font-size:14px;margin-bottom:16px;text-align:center;">点击下方按钮快速审核：</p>
                        <table width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:440px;margin:0 auto;">
                            <tr>
                                <td width="47%" style="text-align:center;">
                                    <a href="{approve_url}" style="display:block;background:#10b981;color:#fff;padding:14px 0;border-radius:10px;text-decoration:none;font-weight:700;font-size:15px;letter-spacing:1px;">✅ 审核通过</a>
                                </td>
                                <td width="6%"></td>
                                <td width="47%" style="text-align:center;">
                                    <a href="{reject_url}" style="display:block;background:#6b7280;color:#fff;padding:14px 0;border-radius:10px;text-decoration:none;font-weight:700;font-size:15px;letter-spacing:1px;">❌ 审核驳回</a>
                                </td>
                            </tr>
                        </table>
                        <p style="color:#9ca3af;font-size:12px;margin-top:14px;text-align:center;line-height:1.6;">点击按钮即完成审核，结果将自动通知预约人<br>无需额外填写密码或备注</p>
                    </div>
                </div>
                <div style="background:#f9fafb;padding:16px 24px;border-radius:0 0 12px 12px;border:1px solid #e5e7eb;border-top:none;">
                    <p style="color:#9ca3af;font-size:12px;margin:0;text-align:center;">此邮件由校友之家场地预约系统自动发送，请勿直接回复。</p>
                </div>
            </div>
            '''
            result = send_email(r['value'], subject, body)
            email_results.append({'name': r['name'], 'success': result})

    return jsonify({
        'code': 0,
        'msg': '通知发送完成',
        'data': {
            'wechat': wechat_results,
            'email': email_results,
            'total': len(wechat_results) + len(email_results)
        }
    })


@app.route('/api/content/notify/quick-review', methods=['GET', 'POST'])
def quick_review():
    if request.method == 'POST':
        order_id = request.form.get('order_id', '')
        action = request.form.get('action', '')
        token = request.form.get('token', '')
        door_password = request.form.get('door_password', '').strip()
        review_comment = request.form.get('review_comment', '').strip()
    else:
        order_id = request.args.get('order_id', '')
        action = request.args.get('action', '')
        token = request.args.get('token', '')
        door_password = ''
        review_comment = ''

    if not verify_quick_review_token(order_id, action, token):
        return _review_result_page('error', '审核链接无效', '签名验证失败，请登录管理后台进行审核。')
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("SELECT id, order_no, applicant_name, phone, reason, open_id, status FROM reservation_orders WHERE order_no = %s", (order_id,))
            order = cur.fetchone()
            if not order:
                conn.close()
                return _review_result_page('warn', '订单不存在', '订单号 %s 未找到。' % order_id)
            order_id_int = order['id']
            current_status = order['status']
            if current_status not in [1, 2]:
                conn.close()
                return _review_result_page('info', '订单已审核', '该预约已完成审核，无需重复操作。')

            cur.execute("SELECT start_time, end_time, password FROM reservation_slots WHERE order_id = %s ORDER BY start_time", (order_id_int,))
            slot_rows = cur.fetchall()

            # GET：渲染可编辑表单页（通过→填开门密码+审核意见；驳回→填驳回原因，均可自定义）
            if request.method == 'GET':
                conn.close()
                if action == 'approve':
                    return _render_review_form(order, slot_rows, token)
                elif action == 'reject':
                    return _render_reject_form(order, slot_rows, token)
                else:
                    return _review_result_page('error', '无效操作', '操作类型无效。')

            # 执行审核（POST 表单提交后生效）
            if action == 'approve':
                new_status = 5
                action_text = '审核通过'
            elif action == 'reject':
                new_status = 3
                action_text = '审核驳回'
            else:
                conn.close()
                return _review_result_page('error', '无效操作', '操作类型无效。')

            cur.execute("UPDATE reservation_orders SET status = %s WHERE id = %s", (new_status, order_id_int))
            comment_text = review_comment or ('邮件一键审核通过' if action == 'approve' else '邮件一键审核驳回')
            cur.execute("INSERT INTO review_records (order_id, reviewer_id, reviewer_role, action, comment, created_at) VALUES (%s, 0, 1, %s, %s, NOW())", (order_id_int, 1 if action == 'approve' else 2, comment_text))
            order['review_comment'] = comment_text
            order['comment'] = comment_text
            if action == 'reject':
                order['reject_reason'] = comment_text
            if action == 'approve' and door_password:
                cur.execute("UPDATE reservation_slots SET status = %s, password = %s WHERE order_id = %s", (new_status, door_password, order_id_int))
            else:
                cur.execute("UPDATE reservation_slots SET status = %s WHERE order_id = %s", (new_status, order_id_int))
        conn.commit()
        conn.close()

        # 重新查询slots（含最新密码）用于通知
        conn2 = get_reservation_db()
        with conn2.cursor() as cur2:
            cur2.execute("SELECT start_time, end_time, password FROM reservation_slots WHERE order_id = %s ORDER BY start_time", (order_id_int,))
            slot_rows_final = cur2.fetchall()
        conn2.close()

        notify_ok, notify_msg = _notify_applicant_after_review(order, slot_rows_final, action)

        if notify_ok:
            if action == 'approve':
                detail = '订单号：%s，已通过公众号发送开门密码及开锁说明给预约人。' % order_id
            else:
                detail = '订单号：%s，已通过公众号通知预约人审核驳回及原因。' % order_id
        elif order.get('open_id'):
            detail = '订单号：%s，状态已更新，但公众号通知未送达（%s），请登录后台手动通知。' % (order_id, notify_msg)
        else:
            detail = '订单号：%s，该预约人无 openid，未发送公众号通知，请人工告知。' % order_id

        page_type = 'success' if action == 'approve' else 'reject'
        return _review_result_page(page_type, '%s成功' % action_text, detail)
    except Exception as e:
        return _review_result_page('error', '审核失败', '错误信息：%s' % str(e))


def _render_review_form(order, slot_rows, token):
    """邮件审核通过 → 开门密码输入表单页面"""
    applicant = order.get('applicant_name', '') or ''
    phone = order.get('phone', '') or ''
    reason = order.get('reason', '') or ''
    order_no = order.get('order_no', '') or ''
    slot_parts = []
    for s in (slot_rows or []):
        txt = _fmt_slot_cn(s.get('start_time'), s.get('end_time'))
        if txt:
            slot_parts.append(txt)
    slots_text = '、'.join(slot_parts) if slot_parts else '-'
    contact_name = get_config('contact_name', '')
    contact_phone = get_config('contact_phone', '')
    contact_hint = ''
    if contact_name or contact_phone:
        contact_hint = '<p class="hint" style="color:#059669;">当前对接人：%s %s（可在后台系统配置中修改）</p>' % (contact_name, contact_phone)

    return '''<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<title>审核通过 - 设置开门密码</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Arial,sans-serif;background:#f3f4f6;min-height:100vh;padding:16px}
.card{max-width:480px;margin:0 auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 20px rgba(0,0,0,.08)}
.header{background:linear-gradient(135deg,#10b981,#059669);color:#fff;padding:24px 20px}
.header h1{font-size:20px;font-weight:700;margin-bottom:4px}
.header p{font-size:13px;opacity:.9}
.body{padding:20px}
.info-row{display:flex;padding:10px 0;border-bottom:1px solid #f0f0f0;font-size:14px}
.info-label{color:#6b7280;width:80px;flex-shrink:0}
.info-value{color:#111827;font-weight:500;flex:1;word-break:break-all}
.form-group{margin-top:20px}
.form-label{display:block;font-size:14px;font-weight:600;color:#374151;margin-bottom:8px}
.form-label .req{color:#ef4444}
.form-input{width:100%;padding:14px 16px;border:2px solid #e5e7eb;border-radius:12px;font-size:16px;outline:none;transition:border-color .2s}
.form-input:focus{border-color:#10b981}
.hint{font-size:12px;color:#9ca3af;margin-top:6px;line-height:1.5}
.btn-submit{width:100%;margin-top:24px;padding:16px;background:linear-gradient(135deg,#10b981,#059669);color:#fff;border:none;border-radius:12px;font-size:17px;font-weight:700;letter-spacing:2px;cursor:pointer}
.btn-submit:active{transform:scale(.98)}
.tip{margin-top:16px;padding:12px 14px;background:#fef3c7;border-radius:10px;font-size:12px;color:#92400e;line-height:1.6}
</style></head><body>
<div class="card">
  <div class="header"><h1>✅ 审核通过</h1><p>请设置开门密码，将通过公众号发送给预约人</p></div>
  <div class="body">
    <div class="info-row"><span class="info-label">预约人</span><span class="info-value">''' + applicant + '''</span></div>
    <div class="info-row"><span class="info-label">联系电话</span><span class="info-value">''' + phone + '''</span></div>
    <div class="info-row"><span class="info-label">预约时段</span><span class="info-value">''' + slots_text + '''</span></div>
    <div class="info-row"><span class="info-label">使用事由</span><span class="info-value">''' + reason + '''</span></div>
    <div class="info-row"><span class="info-label">订单号</span><span class="info-value">''' + order_no + '''</span></div>
    ''' + contact_hint + '''
    <form method="POST" action="/api/content/notify/quick-review">
      <input type="hidden" name="order_id" value="''' + order_no + '''">
      <input type="hidden" name="action" value="approve">
      <input type="hidden" name="token" value="''' + token + '''">
      <div class="form-group">
        <label class="form-label">开门密码 <span class="req">*</span></label>
        <input type="text" name="door_password" class="form-input" placeholder="请输入开门密码（支持任意长度）" required autocomplete="off">
        <p class="hint">密码将在审核通过后通过公众号发送给预约人，并附开锁使用说明。</p>
      </div>
      <div class="form-group">
        <label class="form-label">审核意见（选填）</label>
        <input type="text" name="review_comment" class="form-input" placeholder="可填写备注，如无则留空" autocomplete="off">
      </div>
      <button type="submit" class="btn-submit">确认通过并发送</button>
    </form>
    <div class="tip">如需驳回，请返回邮件点击「审核驳回」按钮；密码不区分位数，按门锁实际要求设置即可。</div>
  </div>
</div>
</body></html>'''


def _render_reject_form(order, slot_rows, token):
    """邮件审核驳回 → 填写驳回原因/审核意见的表单页面"""
    applicant = order.get('applicant_name', '') or ''
    phone = order.get('phone', '') or ''
    reason = order.get('reason', '') or ''
    order_no = order.get('order_no', '') or ''
    slot_parts = []
    for s in (slot_rows or []):
        txt = _fmt_slot_cn(s.get('start_time'), s.get('end_time'))
        if txt:
            slot_parts.append(txt)
    slots_text = '、'.join(slot_parts) if slot_parts else '-'

    return '''<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<title>审核驳回 - 填写审核意见</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Arial,sans-serif;background:#f3f4f6;min-height:100vh;padding:16px}
.card{max-width:480px;margin:0 auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 20px rgba(0,0,0,.08)}
.header{background:linear-gradient(135deg,#ef4444,#b91c1c);color:#fff;padding:24px 20px}
.header h1{font-size:20px;font-weight:700;margin-bottom:4px}
.header p{font-size:13px;opacity:.9}
.body{padding:20px}
.info-row{display:flex;padding:10px 0;border-bottom:1px solid #f0f0f0;font-size:14px}
.info-label{color:#6b7280;width:80px;flex-shrink:0}
.info-value{color:#111827;font-weight:500;flex:1;word-break:break-all}
.form-group{margin-top:20px}
.form-label{display:block;font-size:14px;font-weight:600;color:#374151;margin-bottom:8px}
.form-input{width:100%;padding:14px 16px;border:2px solid #e5e7eb;border-radius:12px;font-size:16px;outline:none;transition:border-color .2s}
.form-input:focus{border-color:#ef4444}
.hint{font-size:12px;color:#9ca3af;margin-top:6px;line-height:1.5}
.btn-submit{width:100%;margin-top:24px;padding:16px;background:linear-gradient(135deg,#ef4444,#b91c1c);color:#fff;border:none;border-radius:12px;font-size:17px;font-weight:700;letter-spacing:2px;cursor:pointer}
.btn-submit:active{transform:scale(.98)}
.tip{margin-top:16px;padding:12px 14px;background:#fef3c7;border-radius:10px;font-size:12px;color:#92400e;line-height:1.6}
</style></head><body>
<div class="card">
  <div class="header"><h1>❌ 审核驳回</h1><p>请填写驳回原因，将通过公众号通知预约人</p></div>
  <div class="body">
    <div class="info-row"><span class="info-label">预约人</span><span class="info-value">''' + applicant + '''</span></div>
    <div class="info-row"><span class="info-label">联系电话</span><span class="info-value">''' + phone + '''</span></div>
    <div class="info-row"><span class="info-label">预约时段</span><span class="info-value">''' + slots_text + '''</span></div>
    <div class="info-row"><span class="info-label">使用事由</span><span class="info-value">''' + reason + '''</span></div>
    <div class="info-row"><span class="info-label">订单号</span><span class="info-value">''' + order_no + '''</span></div>
    <form method="POST" action="/api/content/notify/quick-review">
      <input type="hidden" name="order_id" value="''' + order_no + '''">
      <input type="hidden" name="action" value="reject">
      <input type="hidden" name="token" value="''' + token + '''">
      <div class="form-group">
        <label class="form-label">驳回原因 / 审核意见</label>
        <textarea name="review_comment" class="form-input" rows="3" placeholder="请输入驳回原因，将展示给预约人" required autocomplete="off"></textarea>
        <p class="hint">驳回原因将显示在公众号通知中，让预约人了解驳回缘由。</p>
      </div>
      <button type="submit" class="btn-submit">确认驳回并通知</button>
    </form>
  </div>
</div>
</body></html>'''



def _review_result_page(page_type, title, detail):
    """统一的审核结果页面"""
    styles = {
        'success': ('#d1fae5', '#065f46', '✅'),
        'reject': ('#fee2e2', '#991b1b', '❌'),
        'error': ('#fee2e2', '#991b1b', '❌'),
        'warn': ('#fef3c7', '#92400e', '⚠️'),
        'info': ('#dbeafe', '#1e40af', 'ℹ️'),
    }
    bg, color, icon = styles.get(page_type, styles['info'])
    return '''<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>''' + title + '''</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Arial,sans-serif;background:#f3f4f6;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:20px}
.card{max-width:480px;width:100%;background:''' + bg + ''';color:''' + color + ''';padding:40px 28px;border-radius:16px;text-align:center;box-shadow:0 4px 20px rgba(0,0,0,.1)}
.card h1{font-size:22px;font-weight:700;margin-bottom:12px}
.card p{font-size:14px;line-height:1.7;margin-bottom:8px;word-break:break-all}
.card a{color:#2563eb;text-decoration:none;font-weight:600}
</style></head><body>
<div class="card">
  <h1>''' + icon + ' ' + title + '''</h1>
  <p>''' + detail + '''</p>
  <p style="margin-top:16px;"><a href="https://www.szuedf.org.cn/admin/">前往管理后台查看 →</a></p>
</div>
</body></html>'''


@app.route('/api/content/notify/review-result', methods=['POST'])
def notify_review_result():
    """后台(Go admin)审核完成后回调：统一由 content 发送 6 字段微信给预约人 + 审核结果邮件给接收人。
    收口 access_token，避免 Go 与 content 各自刷新全局 token 互相覆盖(errcode 40001)。本接口不修改订单状态。"""
    data = request.get_json(silent=True) or {}
    order_no = (data.get('order_no') or '').strip()
    action = (data.get('action') or '').strip()
    reason = (data.get('reason') or '').strip()
    if not order_no or action not in ('approve', 'reject'):
        return jsonify({'code': 400, 'msg': '参数错误：需要 order_no 与 action(approve/reject)'}), 400
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("SELECT id, order_no, applicant_name, phone, reason, open_id, status FROM reservation_orders WHERE order_no=%s", (order_no,))
            order = cur.fetchone()
            if not order:
                conn.close()
                return jsonify({'code': 404, 'msg': '订单不存在'}), 404
            oid = order['id']
            cur.execute("SELECT start_time,end_time,password FROM reservation_slots WHERE order_id=%s ORDER BY start_time", (oid,))
            slot_rows = cur.fetchall()
            # 从review_records获取最新审核备注（通过和驳回都获取）
            action_val = 1 if action == 'approve' else 2
            cur.execute("SELECT comment FROM review_records WHERE order_id=%s AND action=%s ORDER BY created_at DESC LIMIT 1", (oid, action_val))
            rr = cur.fetchone()
            if rr and rr.get('comment'):
                order['review_comment'] = rr['comment']
                order['comment'] = rr['comment']
            if action == 'reject':
                if rr and rr.get('comment'):
                    order['reject_reason'] = rr['comment']
                if reason and not order.get('reject_reason'):
                    order['reject_reason'] = reason
                    order['comment'] = reason
        conn.close()

        # 1) 给预约人发微信模板（自动适配字段数）
        wx_ok, wx_msg = _notify_applicant_after_review(order, slot_rows, action)

        # 2) 给启用的邮件接收人发审核结果邮件（内容与微信模板「场地预订结果通知」对齐：预约时间/场地名称/预订结果/原因）
        applicant = order.get('applicant_name', '') or ''
        phone = order.get('phone', '') or ''
        venue_name = '深圳大学校友之家'
        passwords = []
        for s in slot_rows:
            if s.get('password'):
                passwords.append(s['password'])
        slots_text = '、'.join(filter(None, (_fmt_slot_cn(s.get('start_time'), s.get('end_time')) for s in slot_rows))) or '-'
        pwd_text = '、'.join(passwords)
        action_text = '审核通过' if action == 'approve' else '审核驳回'
        result_word = '通过' if action == 'approve' else '不通过'
        color = '#10b981' if action == 'approve' else '#ef4444'
        if action == 'approve':
            if pwd_text:
                reason_text = '开门密码：%s，以#号结束' % pwd_text
            else:
                reason_text = order.get('review_comment') or order.get('comment') or '预约成功，请按时到场'
        else:
            reason_text = order.get('reject_reason') or '场地预约信息不完善'
        email_results = []
        recipients = query_db("SELECT * FROM notify_recipients WHERE enabled=1 AND type='email'")
        for r in recipients:
            body = '''
            <div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif;">
              <div style="background:%s;color:#fff;padding:22px 24px;border-radius:12px 12px 0 0;">
                <h2 style="margin:0;font-size:20px;">%s</h2>
                <p style="margin:6px 0 0;opacity:.9;font-size:13px;">一条预约已在管理后台完成审核</p>
              </div>
              <div style="background:#fff;padding:24px;border:1px solid #e5e7eb;border-top:none;">
                <p><strong>预约时间：</strong>%s</p>
                <p><strong>场地名称：</strong>%s</p>
                <p><strong>预订结果：</strong>%s</p>
                <p><strong>原因：</strong>%s</p>
                <p><strong>预约人：</strong>%s</p>
                <p><strong>联系电话：</strong>%s</p>
                <p><strong>订单号：</strong>%s</p>
              </div>
              <div style="background:#f9fafb;padding:14px 24px;border-radius:0 0 12px 12px;border:1px solid #e5e7eb;border-top:none;">
                <p style="color:#9ca3af;font-size:12px;margin:0;text-align:center;">此邮件由校友之家场地预约系统自动发送，请勿直接回复。</p>
              </div>
            </div>''' % (color, action_text, slots_text, venue_name, result_word, reason_text, applicant, phone, order_no)
            ok = send_email(r['value'], '%s - %s - %s' % (action_text, applicant, slots_text), body)
            email_results.append({'name': r['name'], 'success': ok})

        return jsonify({'code': 0, 'msg': 'ok', 'data': {
            'wechat': {'success': wx_ok, 'detail': wx_msg},
            'email': email_results
        }})
    except Exception as e:
        import traceback
        traceback.print_exc()
        return jsonify({'code': 500, 'msg': str(e)}), 500


_WEEKDAY_CN = ['一', '二', '三', '四', '五', '六', '日']

def _parse_dt(v):
    """兼容 datetime 与 'YYYY-MM-DD HH:MM[:SS]' 字符串，返回 datetime 或 None"""
    if v is None:
        return None
    if isinstance(v, str):
        try:
            import datetime as _dt
            return _dt.datetime.strptime(v[:16], '%Y-%m-%d %H:%M')
        except Exception:
            return None
    return v

def _fmt_slot_cn(st, et):
    """预约时段完整格式：2026年9月7日10:00-12:00（起止完整，与预约系统实际时段一致）"""
    st_dt = _parse_dt(st)
    if not st_dt:
        return ''
    et_dt = _parse_dt(et)
    base = '%d年%d月%d日%s' % (st_dt.year, st_dt.month, st_dt.day, st_dt.strftime('%H:%M'))
    if et_dt:
        return '%s-%s' % (base, et_dt.strftime('%H:%M'))
    return base

def _fmt_tpl_time(slot_rows):
    """微信模板 time8 字段：2026年9月7日 10:00~12:00（时间范围）。
    实测：微信 time 字段仅接受波浪号 ~ 连接的时间范围；短横线 - / 至 / 纯时间等写法均被 47003 拒绝。"""
    parts = []
    for s in (slot_rows or []):
        st = _parse_dt(s.get('start_time'))
        if not st:
            continue
        et = _parse_dt(s.get('end_time'))
        if et:
            parts.append('%d年%d月%d日 %s~%s' % (st.year, st.month, st.day, st.strftime('%H:%M'), et.strftime('%H:%M')))
        else:
            parts.append('%d年%d月%d日 %s' % (st.year, st.month, st.day, st.strftime('%H:%M')))
    if not parts:
        return ''
    if len(parts) > 1:
        return parts[0] + ' 等%d个时段' % len(parts)
    return parts[0]


def _notify_applicant_after_review(order, slot_rows, action):
    """审核后给预约人发送微信模板消息（模板：场地预订结果通知，4字段）。
    字段映射：time8=预约时间 / thing3=场地名称 / short_thing2=预订结果 / thing5=原因。
    返回 (是否成功, 说明)。"""
    try:
        openid = order.get('open_id') or ''
        if not openid:
            return False, '预约人缺少openid'
        applicant = order.get('applicant_name', '') or ''
        order_no = order.get('order_no', '') or ''
        venue_name = '深圳大学校友之家'

        # 组装密码（time8 用 _fmt_tpl_time：2026年9月5日 10:00，微信time字段实测兼容中文年月日）
        passwords = []
        for s in (slot_rows or []):
            if s.get('password'):
                passwords.append(s['password'])
        pwd_text = '、'.join(passwords)
        # time8 时间字段仅展示首个时段开始时间，避免格式校验失败
        tpl_time_text = _fmt_tpl_time(slot_rows)

        if action == 'approve':
            tpl = get_config('template_id_approval')
            result_word = '通过'
            if pwd_text:
                reason_text = '开门密码：%s，以#号结束' % pwd_text
            else:
                reason_text = order.get('review_comment') or order.get('comment') or '预约成功，请按时到场'
        else:
            tpl = get_config('template_id_rejection') or get_config('template_id_approval')
            result_word = '不通过'
            reason_text = order.get('reject_reason') or '场地预约信息不完善'
        # thing5 原因字段限 20 字符，超长截断
        if len(reason_text) > 20:
            reason_text = reason_text[:20]
        _v = {'applicant_name': applicant or '-', 'venue': venue_name, 'slots_text': tpl_time_text, 'password': pwd_text or '无', 'contact_text': '-', 'status_text': result_word, 'order_no': order_no, 'reason': reason_text, 'review_comment': order.get('review_comment') or ''}
        first_text = '您的场地预约已通过！\n' if action == 'approve' else '您的场地预约未通过审核。\n'
        remark_text = '订单号: %s' % order_no
        first_color = '#10B981' if action == 'approve' else '#EF4444'
        data = build_tpl_data(tpl, _v, first_text=first_text, remark_text=remark_text, first_color=first_color)

        if not tpl:
            return False, '未配置模板ID'
        # 发送，失败时隔1.5秒重试一次（应对微信瞬时限流/网络抖动）
        ok = send_wechat_template_message(openid, tpl, data)
        if not ok:
            import time
            time.sleep(1.5)
            ok = send_wechat_template_message(openid, tpl, data)
        # 驳回场景兜底：若驳回模板无效/未配置，自动用通过模板补发（预订结果仍显示"不通过"）
        if not ok and action == 'reject':
            backup_tpl = get_config('template_id_approval')
            if backup_tpl and backup_tpl != tpl:
                import time as _t
                _t.sleep(1)
                ok = send_wechat_template_message(openid, backup_tpl, data)
        return (True, 'ok') if ok else (False, '微信接口返回失败')
    except Exception as e:
        print(f"快速审核通知异常: {e}")
        return False, str(e)



# ============ 新预约自动检测通知（每分钟检查） ============
import threading, time as _time
_notified_order_ids = set()

def check_and_notify_new_orders():
    while True:
        try:
            conn = get_reservation_db()
            with conn.cursor() as cur:
                cur.execute("SELECT id, order_no, applicant_name, phone, reason, open_id, status, created_at FROM reservation_orders WHERE created_at >= NOW() - INTERVAL 5 MINUTE ORDER BY created_at DESC")
                orders = cur.fetchall()
            conn.close()
            for order in orders:
                oid = order['id']
                if oid in _notified_order_ids:
                    continue
                _notified_order_ids.add(oid)
                try:
                    conn2 = get_reservation_db()
                    with conn2.cursor() as cur2:
                        cur2.execute("SELECT start_time,end_time FROM reservation_slots WHERE order_id=%s ORDER BY start_time", (oid,))
                        slots = cur2.fetchall()
                    conn2.close()
                    slot_parts = []
                    for s in slots:
                        st, et = s.get('start_time'), s.get('end_time')
                        st_s = st.strftime('%m-%d %H:%M') if st else ''
                        et_s = et.strftime('%H:%M') if et else ''
                        slot_parts.append('%s~%s' % (st_s, et_s))
                    slots_text = '、'.join(slot_parts) if slot_parts else '-'
                except:
                    slots_text = '-'
                try:
                    _send_new_order_notification(order['order_no'], order['applicant_name'], order['phone'], order['reason'], slots_text, order.get('status', 1))
                    print(f"[新预约通知] 已发送 订单{order['order_no']}")
                except Exception as e:
                    print(f"[新预约通知] 失败: {e}")
        except Exception as e:
            print(f"[新预约检测] 异常: {e}")
        _time.sleep(60)

def _send_new_order_notification(order_no, applicant, phone, reason, slots_text, status=1):
    # 状态文本映射
    status_map = {1: '待审核', 2: '待二级审核', 3: '已驳回', 5: '已通过', 6: '已取消'}
    status_text = status_map.get(status, '待审核')
    try:
        template_id = get_config('template_id_new_order')
        if template_id:
            recipients = query_db("SELECT * FROM notify_recipients WHERE enabled=1 AND type='wechat'")
            for r in recipients:
                _v = {'applicant_name': applicant, 'venue': '深圳大学校友之家', 'slots_text': slots_text, 'status_text': status_text, 'order_no': order_no, 'phone': phone, 'reason': reason}
                _d = build_tpl_data(template_id, _v, first_text='新的场地预约申请，请及时审核', remark_text='\n订单号: %s\n联系电话: %s\n使用事由: %s' % (order_no, phone, reason), first_color='#CE1A20')
                send_wechat_template_message(r['value'], template_id, _d)
    except Exception as e:
        print(f"[新预约微信] 失败: {e}")
    try:
        recipients = query_db("SELECT * FROM notify_recipients WHERE enabled=1 AND type='email'")
        for r in recipients:
            subject = '新预约通知 - %s - %s（%s）' % (applicant, slots_text, status_text)
            body = ('<div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif;">'
                    '<div style="background:#CE1A20;color:#fff;padding:22px 24px;border-radius:12px 12px 0 0;">'
                    '<h2 style="margin:0;font-size:20px;">新预约通知</h2>'
                    '<p style="margin:6px 0 0;opacity:.9;font-size:13px;">有新的场地预约申请，请及时审核</p></div>'
                    '<div style="background:#fff;padding:24px;border:1px solid #e5e7eb;border-top:none;">'
                    '<p><strong>预约人：</strong>%s</p>'
                    '<p><strong>联系电话：</strong>%s</p>'
                    '<p><strong>预约时段：</strong>%s</p>'
                    '<p><strong>使用事由：</strong>%s</p>'
                    '<p><strong>订单号：</strong>%s</p>'
                    '<p><strong>状态：</strong>%s</p></div>'
                    '<div style="background:#f9fafb;padding:14px 24px;border-radius:0 0 12px 12px;border:1px solid #e5e7eb;border-top:none;">'
                    '<p style="color:#9ca3af;font-size:12px;margin:0;text-align:center;">此邮件由校友之家场地预约系统自动发送，请勿直接回复。</p></div>'
                    '</div>') % (applicant, phone, slots_text, reason, order_no, status_text)
            send_email(r['value'], subject, body)
    except Exception as e:
        print(f"[新预约邮件] 失败: {e}")


if __name__ == '__main__':
    os.makedirs(UPLOAD_FOLDER, exist_ok=True)
    app.run(host='0.0.0.0', port=5000, debug=False)

# ============ 订单管理（关闭二级审核） ============

def get_reservation_db():
    """获取预约数据库连接（home_res）"""
    config = DB_CONFIG.copy()
    config['database'] = 'home_res'
    return pymysql.connect(**config)

@app.route('/api/content/orders/auto-approve-level2', methods=['POST'])
def auto_approve_level2():
    """
    自动通过二级审核
    将指定订单或所有 status=2 的订单改为 status=5（审核通过）
    用于关闭二级审核功能
    """
    data = request.get_json() or {}
    order_id = data.get('order_id')
    
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            if order_id:
                # 只更新指定订单
                cur.execute(
                    "UPDATE reservation_orders SET status=5 WHERE id=%s AND status=2",
                    (order_id,)
                )
                affected = cur.rowcount
            else:
                # 更新所有待二级审核的订单
                cur.execute("UPDATE reservation_orders SET status=5 WHERE status=2")
                affected = cur.rowcount
            conn.commit()
        conn.close()
        
        return jsonify({
            'code': 0,
            'msg': '自动通过二级审核成功',
            'data': {'affected_rows': affected}
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': f'自动通过二级审核失败: {str(e)}'
        }), 500

@app.route('/api/content/orders/pending-level2', methods=['GET'])
def get_pending_level2():
    """获取所有待二级审核的订单"""
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("""
                SELECT id, order_no, applicant_name, phone, status, created_at
                FROM reservation_orders
                WHERE status=2
                ORDER BY created_at DESC
            """)
            orders = cur.fetchall()
        conn.close()
        
        return jsonify({
            'code': 0,
            'msg': 'success',
            'data': {'orders': orders, 'count': len(orders)}
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500


# ============ 时段管理 API ============

@app.route('/api/content/admin/time-slots', methods=['GET'])
def get_time_slots():
    """获取所有时段配置"""
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("""
                SELECT id, day_of_week, start_time, end_time, status, display_label, sort_order
                FROM venue_time_slots
                ORDER BY day_of_week, sort_order, start_time
            """)
            slots = cur.fetchall()
        conn.close()
        
        # 按星期分组
        result = {}
        day_names = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
        for i in range(7):
            result[i] = {'day': i, 'day_name': day_names[i], 'slots': []}
        
        for slot in slots:
            day = slot['day_of_week']
            if day in result:
                result[day]['slots'].append({
                    'id': slot['id'],
                    'start_time': str(slot['start_time']),
                    'end_time': str(slot['end_time']),
                    'status': slot['status'],
                    'display_label': slot['display_label'],
                    'sort_order': slot['sort_order']
                })
        
        return jsonify({
            'code': 0,
            'msg': 'success',
            'data': list(result.values())
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

@app.route('/api/content/admin/time-slots', methods=['POST'])
def add_time_slot():
    """添加时段配置"""
    data = request.get_json() or {}
    day_of_week = data.get('day_of_week')
    start_time = data.get('start_time')
    end_time = data.get('end_time')
    status = data.get('status', 1)
    display_label = data.get('display_label', '')
    sort_order = data.get('sort_order', 0)
    
    if day_of_week is None or not start_time or not end_time:
        return jsonify({'code': 400, 'msg': '缺少必要参数'}), 400
    
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO venue_time_slots (day_of_week, start_time, end_time, status, display_label, sort_order)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (day_of_week, start_time, end_time, status, display_label, sort_order))
            new_id = cur.lastrowid
        conn.commit()
        conn.close()
        
        return jsonify({
            'code': 0,
            'msg': '添加成功',
            'data': {'id': new_id}
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

@app.route('/api/content/admin/time-slots/<int:slot_id>', methods=['PUT'])
def update_time_slot(slot_id):
    """更新时段配置"""
    data = request.get_json() or {}
    
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            # 检查是否存在
            cur.execute("SELECT id FROM venue_time_slots WHERE id=%s", (slot_id,))
            if not cur.fetchone():
                conn.close()
                return jsonify({'code': 404, 'msg': '时段不存在'}), 404
            
            # 构建更新语句
            updates = []
            params = []
            for field in ['day_of_week', 'start_time', 'end_time', 'status', 'display_label', 'sort_order']:
                if field in data:
                    updates.append(f"{field}=%s")
                    params.append(data[field])
            
            if updates:
                params.append(slot_id)
                cur.execute(f"UPDATE venue_time_slots SET {', '.join(updates)} WHERE id=%s", params)
                conn.commit()
        
        conn.close()
        
        return jsonify({
            'code': 0,
            'msg': '更新成功'
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

@app.route('/api/content/admin/time-slots/<int:slot_id>', methods=['DELETE'])
def delete_time_slot(slot_id):
    """删除时段配置"""
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("DELETE FROM venue_time_slots WHERE id=%s", (slot_id,))
            affected = cur.rowcount
        conn.commit()
        conn.close()
        
        if affected == 0:
            return jsonify({'code': 404, 'msg': '时段不存在'}), 404
        
        return jsonify({
            'code': 0,
            'msg': '删除成功'
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

@app.route('/api/content/time-slots', methods=['GET'])
def get_public_time_slots():
    """公开获取时段配置（用于预约前端和日历）"""
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            cur.execute("""
                SELECT id, day_of_week, start_time, end_time, status, display_label, sort_order
                FROM venue_time_slots
                WHERE status = 1
                ORDER BY day_of_week, sort_order, start_time
            """)
            slots = cur.fetchall()
        conn.close()
        
        # 按星期分组
        result = {}
        day_names = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
        for i in range(7):
            result[i] = {'day': i, 'day_name': day_names[i], 'slots': []}
        
        for slot in slots:
            day = slot['day_of_week']
            if day in result:
                result[day]['slots'].append({
                    'id': slot['id'],
                    'start_time': str(slot['start_time']),
                    'end_time': str(slot['end_time']),
                    'display_label': slot['display_label']
                })
        
        return jsonify({
            'code': 0,
            'msg': 'success',
            'data': list(result.values())
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

@app.route('/api/content/admin/time-slots/batch', methods=['POST'])
def batch_update_time_slots():
    """批量更新时段配置（用于快速复制某天的配置到其他天）"""
    data = request.get_json() or {}
    source_day = data.get('source_day')
    target_days = data.get('target_days', [])
    
    if source_day is None or not target_days:
        return jsonify({'code': 400, 'msg': '缺少必要参数'}), 400
    
    try:
        conn = get_reservation_db()
        with conn.cursor() as cur:
            # 获取源天的时段配置
            cur.execute("""
                SELECT start_time, end_time, status, display_label, sort_order
                FROM venue_time_slots
                WHERE day_of_week = %s
                ORDER BY sort_order, start_time
            """, (source_day,))
            source_slots = cur.fetchall()
            
            if not source_slots:
                conn.close()
                return jsonify({'code': 404, 'msg': '源天没有时段配置'}), 404
            
            # 删除目标天的现有配置
            for target_day in target_days:
                cur.execute("DELETE FROM venue_time_slots WHERE day_of_week = %s", (target_day,))
            
            # 复制源天的配置到目标天
            for target_day in target_days:
                for slot in source_slots:
                    cur.execute("""
                        INSERT INTO venue_time_slots (day_of_week, start_time, end_time, status, display_label, sort_order)
                        VALUES (%s, %s, %s, %s, %s, %s)
                    """, (target_day, slot['start_time'], slot['end_time'], slot['status'], slot['display_label'], slot['sort_order']))
            
        conn.commit()
        conn.close()
        
        return jsonify({
            'code': 0,
            'msg': f'已将周{source_day}的配置复制到 {len(target_days)} 天'
        })
    except Exception as e:
        return jsonify({
            'code': 500,
            'msg': str(e)
        }), 500

# ============ 微信分享配置（标题/描述/缩略图） ============
SHARE_HTML_DIR = os.environ.get('SHARE_HTML_DIR', '/app/static_html')
SHARE_DEFAULT = {
    'share_title': '深圳大学校友之家 · 场地预约',
    'share_description': '家门常开，等你归来。\n点击链接进行团体返校预约。',
    'share_image_url': 'https://www.szuedf.org.cn/share-icon.jpg',
    'share_url': 'https://www.szuedf.org.cn/',
}

def _get_share_config():
    cfg = {}
    for k in SHARE_DEFAULT:
        v = get_config(k, SHARE_DEFAULT[k])
        cfg[k] = v
    return cfg

def _render_share_index():
    """根据当前分享配置生成 index.html 并写入静态目录"""
    cfg = _get_share_config()
    title = (cfg.get('share_title') or SHARE_DEFAULT['share_title']).replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;')
    desc = (cfg.get('share_description') or '').replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;')
    img = (cfg.get('share_image_url') or '').replace('&', '&amp;').replace('"', '&quot;')
    url = (cfg.get('share_url') or SHARE_DEFAULT['share_url']).replace('&', '&amp;').replace('"', '&quot;')
    html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover" />
  <meta name="theme-color" content="#CE1A20" />
  <meta name="format-detection" content="telephone=no" />
  <link rel="icon" type="image/svg+xml" href="/vite.svg" />
  <title>{title}</title>
  <meta property="og:title" content="{title}" />
  <meta property="og:description" content="{desc}" />
  <meta property="og:image" content="{img}" />
  <meta property="og:url" content="{url}" />
  <script type="module" crossorigin src="/assets/index-Bg5ll6SL.js"></script>
  <link rel="stylesheet" crossorigin href="/assets/index-DhIQQ6Pe.css">
</head>
<body class="bg-gray-50 min-h-screen">
  <div id="app"></div>
  <script src="/content-display.js"></script>
</body>
</html>
"""
    os.makedirs(SHARE_HTML_DIR, exist_ok=True)
    out = os.path.join(SHARE_HTML_DIR, 'index.html')
    with open(out, 'w', encoding='utf-8') as f:
        f.write(html)
    return out

@app.route('/api/content/admin/share-config', methods=['GET'])
def admin_get_share_config():
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    return jsonify({'code': 0, 'data': _get_share_config()})

@app.route('/api/content/admin/share-config', methods=['POST'])
def admin_save_share_config():
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    data = request.get_json() or {}
    for k in SHARE_DEFAULT:
        if k in data and data[k] is not None:
            set_config(k, str(data[k]))
    try:
        path = _render_share_index()
        return jsonify({'code': 0, 'msg': '保存成功', 'index_path': path})
    except Exception as e:
        return jsonify({'code': 500, 'msg': f'配置已保存，但生成 index.html 失败: {e}'}), 500

@app.route('/api/content/admin/share-config/image', methods=['POST'])
def admin_upload_share_image():
    if not check_admin_token():
        return jsonify({'code': 401, 'msg': '未登录'}), 401
    if 'file' not in request.files:
        return jsonify({'code': 400, 'msg': '没有上传文件'}), 400
    file = request.files['file']
    if file.filename == '':
        return jsonify({'code': 400, 'msg': '没有选择文件'}), 400
    ext = os.path.splitext(file.filename)[1].lower().lstrip('.')
    if ext not in ALLOWED_EXTENSIONS:
        return jsonify({'code': 400, 'msg': f'不支持的格式: {ext}'}), 400
    filename = f'share_{uuid.uuid4().hex[:12]}.{ext}'
    save_path = os.path.join(app.config['UPLOAD_FOLDER'], filename)
    file.save(save_path)
    url = f'https://www.szuedf.org.cn/uploads/{filename}'
    return jsonify({'code': 0, 'url': url, 'path': f'/uploads/{filename}'})

@app.route('/share-admin', methods=['GET'])
def share_admin_page():
    html = """<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>分享配置</title>
<style>
body{font-family:-apple-system,"PingFang SC",sans-serif;background:#f5f5f5;margin:0;padding:20px;}
.box{max-width:640px;margin:0 auto;background:#fff;border-radius:12px;padding:24px;box-shadow:0 1px 4px rgba(0,0,0,.08);}
h2{margin:0 0 16px;color:#A6192E;}
label{display:block;font-size:13px;color:#666;margin:14px 0 6px;}
input,textarea{width:100%;box-sizing:border-box;padding:8px 10px;border:1px solid #ddd;border-radius:6px;font-size:14px;}
textarea{min-height:70px;resize:vertical;}
button{margin-top:16px;width:100%;padding:10px;background:#A6192E;color:#fff;border:0;border-radius:6px;font-size:15px;cursor:pointer;}
#preview{margin-top:16px;padding:12px;background:#fafafa;border-radius:6px;font-size:12px;color:#888;}
#status{margin-top:10px;font-size:13px;}
img{max-width:100%;margin-top:8px;border-radius:6px;}
</style></head><body>
<div class="box">
<h2>微信分享配置</h2>
<label>标题</label><input id="title">
<label>描述（可换行）</label><textarea id="desc"></textarea>
<label>缩略图 URL</label><input id="img">
<label>上传新缩略图</label><input type="file" id="file" accept="image/*">
<button onclick="save()">保存并生效</button>
<div id="status"></div>
<div id="preview"></div>
</div>
<script>
const TOKEN=localStorage.getItem('share_token')||'';
async function api(p){const r=await fetch(p.url,{method:p.m||'GET',headers:{'Authorization':TOKEN,...(p.d?{'Content-Type':'application/json'}:{})},body:p.d?JSON.stringify(p.d):undefined});return r.json();}
async function load(){
  if(!TOKEN){const u=prompt('管理员用户名：');const p=prompt('密码：');
    const r=await fetch('/api/content/admin/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p})});
    const j=await r.json();if(j.code!==0){alert('登录失败');return;}
    localStorage.setItem('share_token',j.data.token);location.reload();return;}
  const j=await api({url:'/api/content/admin/share-config'});
  if(j.code===401){localStorage.removeItem('share_token');location.reload();return;}
  document.getElementById('title').value=j.data.share_title||'';
  document.getElementById('desc').value=j.data.share_description||'';
  document.getElementById('img').value=j.data.share_image_url||'';
  document.getElementById('preview').innerHTML='当前缩略图：<br><img src="'+(j.data.share_image_url||'')+'">';
}
document.getElementById('file').onchange=async function(e){
  const f=e.target.files[0];if(!f)return;
  const fd=new FormData();fd.append('file',f);
  const r=await fetch('/api/content/admin/share-config/image',{method:'POST',headers:{'Authorization':TOKEN},body:fd});
  const j=await r.json();if(j.code===0){document.getElementById('img').value=j.url;}
  else{alert(j.msg||'上传失败');}
};
async function save(){
  const d={share_title:title.value,share_description:desc.value,share_image_url:img.value};
  const j=await api({url:'/api/content/admin/share-config',m:'POST',d:d});
  document.getElementById('status').textContent=j.code===0?'已保存并生效 ✓':'失败：'+(j.msg||'');
  if(j.code===0)load();
}
load();
</script></body></html>"""
    return html

# ============ 启动新预约检测后台线程（所有函数定义之后） ============
try:
    import threading as _th
    _t = _th.Thread(target=check_and_notify_new_orders, daemon=True)
    _t.start()
    print("[新预约检测] 后台线程已启动（延迟启动）")
except Exception as _e:
    print(f"[新预约检测] 启动失败: {_e}")
