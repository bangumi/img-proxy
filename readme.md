动态切图服务端

GET /r/{height}/{path}

example: `/r/100/pic/cover/l/b4/4f/18692_E04qh.jpg`

可用的 height

- 100
- 200
- 400
- 600
- 800
- 1200

不合法的 height 会直接返回 401
