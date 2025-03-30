<!-- TOC -->

* [VRCDancePreloader](#vrcdancepreloader)
    * [原理](#原理)
    * [使用方法](#使用方法)
        * [安装证书](#安装证书)
        * [设置代理](#设置代理)
    * [个性化](#个性化)
        * [配置文件](#配置文件)
        * [程序参数](#程序参数)
    * [设置代理规则](#设置代理规则)
        * [Clash Verge Rev (1.7及以上)](#clash-verge-rev-17及以上)
    * [VRChat ToS](#vrchat-tos)
    * [TODO](#todo)

<!-- TOC -->

# VRCDancePreloader

## 原理

通过监听PyPyDance在VRChat日志中打印的曲目队列提前下载即将播放的歌曲，从而在降低下载延迟的同时，不占用太多磁盘缓存。

## 使用方法

### 安装证书

程序使用goproxy代理视频的下载，需要安装根证书以便处理https请求，程序首次运行后，会在自身目录下输出根证书。

### 设置代理

需要使用系统代理或者网卡代理来代理VRChat的网络请求，考虑到用户代理的复杂性，本程序不会主动设置系统代理，建议使用Clash Verge
Rev或者Proxifier管理流量。

## 个性化

### 配置文件

程序首次运行后，会在自身目录下输出一个配置文件`config.yaml`，是一些比较长、变化少的配置。
**当程序以GUI模式运行时，可以在设置界面中修改这些配置，修改后会自动保存到配置文件。**

```yaml
# 本程序无视系统代理和环境变量，
# 需要通过以下配置程序自身下载视频、获取视频信息时使用的http代理，如果留空就不使用代理
proxy:
  # 访问PyPyDance获取视频和缩略图的代理，考虑到PyPyDance在内地有直连中华电信的线路，所以一般不需要配置代理
  "jd.pypy.moe": ""
  # 下载YouTube视频使用的代理，目前还没做，不用设置
  youtube-video: ""
  # 请求YouTube API获取YouTube视频信息（如标题）的代理，如果你没有YouTube API key，就不用设置
  youtube-api: ""
  # 请求i.ytimg.com获取YouTube视频缩略图的代理
  youtube-image: ""
keys:
  # YouTube API key，配置后程序可以补全跳舞房日志中空缺的YouTube视频标题
  youtube-api: ""
limits:
  # 提前加载的数量，比如设置为4，加载器会下载当前播放歌曲和后面4首歌
  max-preload-count: 4
  # 最大同时下载的视频数量，优先下载靠前的歌曲，其余歌曲会排队等待
  max-parallel-download-count: 2
  # 配置预加载器缓存可以使用的最大磁盘空间，预加载器退出时会清空较早的视频直到腾出足够的空间
  max-cache: 300
```

### 程序参数

|            参数名称            | 含义                         |
|:--------------------------:|----------------------------|
|       `--port`或`-p`        | 代理端口，默认为`7653`             |
|    `--vrchat-dir`或`-d`     | VRChat程序数据目录，用于搜索日志，一般无需设置 |
|        `--gui`或`-g`        | 是否显示GUI窗口，可以展示视频预加载状态      |
|        `--tui`或`-t`        | 是否在控制台显示TUI，以文字方式展示视频预加载状态 |
|    `--skip-client-test`    | 是否跳过启动时网络请求检查，加快启动速度       |
| `--disable-async-download` | 禁用边下边播，播放出现问题时可以试试禁用       |

## 设置代理规则

### Clash Verge Rev (1.7及以上)

进入“订阅”，右键点击当前使用的订阅，点击“编辑节点”，点击右上角的“高级”，在配置文件中输入：

```yaml
prepend:
  - name: 'vrcDancePreload'
    type: 'http'
    server: '127.0.0.1'
    port: 7653
append: [ ]
delete: [ ]
```

这样你就添加了一个名为`vrcDancePreload`的节点，如果你在程序参数里制定了端口，记得在里面改一下。

保存后，还是右键，点击“编辑规则”，点击右上角的“高级”，在配置文件中输入：

```yaml
prepend:
  - 'AND,((DOMAIN,jd.pypy.moe),(PROCESS-NAME-REGEX,VRChat)),vrcDancePreload'
  - 'AND,((DOMAIN,jd.pypy.moe),(PROCESS-NAME-REGEX,yt-dlp)),vrcDancePreload'
append: [ ]
delete: [ ]
```

这样你就添加了拦截VRChat和yt-dlp对jd.pypy.moe域名的所有请求的规则，并交给`vrcDancePreload`节点处理。

记得开Clash Verge的系统代理！

## VRChat ToS

本项目仅对VRChat的日志进行监听，并利用代理对PyPyDance域名提供视频缓存，不会对PyPyDance房间数据进行修改，不以任何方式对游戏进行修改。本项目不是模组或者修改器，不违反VRChat的服务条款。

## TODO

- [ ] 断点续传验证
- [ ] 完成临时加载歌曲的自动转移
- [ ] 稳定性优化，减少死锁
- [ ] 支持WannaDance
- [ ] YouTube视频预加载（需要和yt-dlp完美配合，还没想好怎么做）
- [ ] <del>换个GUI框架？（发现这个fyne吃的内存和webview2差不多诶）</del>
- [ ] 界面优化
- [ ] 实现播放历史查看、收藏管理
- [ ] 整合一下VRCX的API，实现PyPyDance的收藏同步
