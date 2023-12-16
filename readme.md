动态切图服务器

GET `/r/{size}/{path}`

example:

- `/r/100/pic/cover/l/b4/4f/18692_E04qh.jpg`
- `/r/100x0/pic/cover/l/b4/4f/18692_E04qh.jpg`
- `/r/0x100/pic/cover/l/b4/4f/18692_E04qh.jpg`
- https://lain.bgm.tv/r/400/pic/cover/l/53/6a/1453_iZIOZ.jpg

size 应该是 `${width}` 或者 `${width}x${height}` 格式，如 `400`， `200x0`, `200x200` `0x200`。`200` 和 `200x0`含义相同。

width 或者 height 为 0 表示仅缩放，同时不为0的情况为缩放+裁剪。两种情况下图片的宽高比均会保持不变。

可用的 width height

- 100
- 200
- 400
- 600
- 800
- 1200

不合法的尺寸参数会直接返回 bad request

本仓库的代码仅仅是 http
gateway，实际的图片处理由 [imaginary](https://github.com/h2non/imaginary)
进行，处理后的图片储存在 s3 （minio）。

