# Mysql常用命令
- 登入数据库：`mysql -u root -p`
- 查看数据库：`show databases;`

# 数据库管理
- 创建数据库
```sql
CREATE DATABASE home_xy DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```
- 查看所有数据库：`SHOW DATABASES;`
- 切换数据库 `USE home_xy;`

# 表格管理
- 查看所有表格：`SHOW TABLES;`
- 查看表结构：`DESCRIBE reservations;`
- 修改表格字段名字：`ALTER TABLE reservations RENAME COLUMN openid TO open_id`

# 账号管理
- 创建账号
```sql
-- 1. 创建新用户 (允许本机访问)
CREATE USER 'res_user'@'localhost' IDENTIFIED BY '12345678';

-- 2. 把我们之前建好的库的所有权限，单独赐予这个新用户
GRANT ALL PRIVILEGES ON home_xy.* TO 'res_user'@'localhost';

-- 3. 刷新权限让它生效
FLUSH PRIVILEGES;
exit;
```