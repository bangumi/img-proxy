动态切图服务端

GET /r/{size}/{path}

example:

- `/r/100/pic/cover/l/b4/4f/18692_E04qh.jpg`
- `/r/100x0/pic/cover/l/b4/4f/18692_E04qh.jpg`
- `/r/0x100/pic/cover/l/b4/4f/18692_E04qh.jpg`

size 应该是 width x height 格式，如 `200x0`, `200x200` `0x200`，width 或者 height
为 0 表示缩放，同时指定的情况为缩放+裁剪

可用的 width height

- 100
- 200
- 400
- 600
- 800
- 1200

不合法的尺寸参数会直接返回 bad request
