# 远程仓库管理

```
#  添加远程仓库
git remote add <name> <url>
git remote add server ssh://root@106.52.23.213/root/git/reservation_sys_go.git

# 查看所有远程仓库
git remote -v

# 修改远程仓库地址
git remote set-url <name> <new_url>
git remote set-url server ssh://root@106.52.23.213/root/git/reservation_sys_go.git

# 删除远程仓库
git remote remove <name>
git remote remove server
```

# 代码推送

```
# 推送分支到远程仓库
git push <name> <branch>
git push server master

# 首次推送并设置上游分支
git push -u server main
```

# 拉取代码

```
# 拉取server仓库的主分支
git pull server master

# 查看远程分支
git branch -r

# 拉取远程分支到本地
git checkout -b feature server/feature
```

# 分支操作

```
# 查看本地分支
git branch

# 查看所有分支（远程和本地）
git branch -a

# 创建新的分支
git checkout -b feature

# 切换分支
git checkout feature

# 重命名当前分支
git branch -M main

# 删除本地分支
git branch -d feature

# 删除远程分支
git push server --delete feature
```

# 工作流程

```
# 本地
git add .
git commit -m ""
git push server master

# 服务器部署
cd /root/workspace/reservation_sys_go

# orgin 是在执行 git clone 时自动给远程仓库起的名字
git pull orgin master
```
