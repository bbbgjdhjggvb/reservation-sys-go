================================
  测试环境部署步骤
================================

1. 上传到云服务器：
   scp reservation-sys-test.tar.gz user@your-server-ip:/root/reservation/

2. SSH 登录云服务器：
   ssh user@your-server-ip

3. 解压并导入镜像：
   cd /root/reservation
   tar -xzf reservation-sys-test.tar.gz
   docker load -i reservation-sys.tar

4. 修改配置（重要！）：
   vim configs/config_v1.test.yaml
   将 frontend_url 改为: http://YOUR_SERVER_IP/reserve

5. 修改菜单配置（重要！）：
   vim configs/menu.json
   将菜单 URL 中的服务器地址改为你的 IP 或域名

   提示: 可以使用工具生成新的菜单 URL
   ./scripts/generate_oauth_url.sh

   参考文档: doc/微信菜单配置快速参考.md

6. 启动服务：
   docker-compose -f docker-compose.test.yaml up -d

7. 验证服务：
   docker-compose -f docker-compose.test.yaml ps
   curl http://localhost/wx

8. 同步微信菜单：
   chmod +x sync_menu
   ./sync_menu -config configs/config_sync_menu.test.yaml

   注意: 菜单同步后，需要取消关注公众号后重新关注才能看到更新

9. 配置微信：
   URL: http://YOUR_SERVER_IP/wx
   Token: mytesttoken123

================================
  文件清单
================================

- reservation-sys.tar      # Docker 镜像
- sync_menu                # 菜单同步工具（可执行文件）
- docker-compose.test.yaml # Docker Compose 配置
- configs/                 # 配置文件目录
  - config_v1.test.yaml   # v1 服务配置
  - config_v2.test.yaml   # v2 服务配置
  - config_sync_menu.test.yaml  # 菜单同步配置
  - menu.json             # 微信菜单配置
- deploy/                  # 部署配置
  - mysql/init.sql        # 数据库初始化
  - nginx/nginx.config    # Nginx 配置
- internal/reservation/frontend/  # 前端页面
- scripts/                 # 工具脚本
  - generate_oauth_url.sh # OAuth URL 生成工具

================================
  菜单同步工具使用
================================

1. 生成新的菜单 URL：
   ./scripts/generate_oauth_url.sh

2. 修改菜单配置：
   vim configs/menu.json

3. 同步到微信：
   ./sync_menu -config configs/config_sync_menu.test.yaml

4. 重新关注公众号查看效果

注意：菜单同步工具需要 Redis 连接

================================
