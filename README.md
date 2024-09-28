# PyPyDancePreloader

## 悲报

PyPyDance更新后不再在日志输出曲目队列（是可以边下边放了，但请求依旧很慢，还是要等很久，为什么要搞那么多重定向），所以这个项目寄了

等等看后面有没有其他办法吧

## 原理

通过监听PyPyDance在VRChat日志中打印的曲目队列提前下载即将播放的歌曲，从而在降低下载延迟的同时，不占用太多磁盘缓存。

## 使用方法

使用goproxy代理视频的下载，需要安装根证书。

需要使用全局代理来代理VRChat的网络请求，建议使用Clash Verge或者Proxifier管理流量。

### 参数

- `--port` 代理端口，默认为`7653`
- `--cache` 缓存目录，默认为`./cache`


### Clash Verge

## VRChat ToS

本项目仅对VRChat的日志进行监听，并利用代理对PyPyDance域名提供视频缓存，不会对PyPyDance房间数据进行修改，不以任何方式对游戏进行修改。本项目不是模组或者修改器，不违反VRChat的服务条款。
