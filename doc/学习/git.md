# 远程仓库管理

```sh
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

---

# 代码推送

```sh
# 推送分支到远程仓库
git push <name> <branch>
git push server master

# 首次推送并设置上游分支
git push -u server main
```

---

# 拉取代码

```sh
# 拉取server仓库的主分支
git pull server master

# 查看远程分支
git branch -r

# 拉取远程分支到本地
git checkout -b feature server/feature
```

---

# 分支操作

```sh
# 查看本地分支
git branch

# 查看所有分支（远程和本地）
git branch -a

# 创建新的分支
git switch -c workspace

# 切换分支
git switch workspace

# 重命名当前分支
git branch -M main

# 删除本地分支
git branch -d workspace

# 删除远程分支
git push server --delete workspace
```

# 分支合并
在合并之前先在`master`分支进行同步，`git pull server master`
```sh
# 合并之前要进行提交，在 workspace 分支
git add <需要提交的文件>
git commit -m "提交介绍"

# 切换到主分支
git switch master
git merge workspace
```

---

# 工作流程
1. 在本地 workspace 分支进行修改
2. 修改完后 commit 
3. 切换到 master 分支，先进行 pull，执行合并
4. push 到远程仓库