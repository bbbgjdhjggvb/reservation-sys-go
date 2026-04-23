# 不能在本地构建 Docker 镜像
```
# 将代码从本地同步云端服务器上
git push ssh://root@106.52.23.213/root/git/reservation_sys_go.git master

# 运行脚本构建 Docker 镜像，并进行部署
sh deploy-local.sh
```

# 在本地进行 Docker 镜像构建然后上传镜像