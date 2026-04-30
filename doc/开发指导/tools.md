# openssl 生成密钥
```sh
# 生成 32 字节 (256位) 的随机数并进行 Base64 编码
openssl rand -base64 32
```

# 网络检测工具
```sh
# 检查端口占用
sudo ss -tulnp
sudo ss -tulnp | grep 8080
```

# curl 工具