<!-- TOC -->
* [VRCDancePreloader](#vrcdancepreloader)
  * [原理](#原理)
  * [使用方法](#使用方法)
    * [安装证书（可选）](#安装证书可选)
    * [设置代理](#设置代理)
  * [个性化](#个性化)
    * [配置文件](#配置文件)
    * [程序参数](#程序参数)
  * [设置代理规则](#设置代理规则)
    * [Clash Verge Rev (1.7及以上)](#clash-verge-rev-17及以上)
    * [使用UU加速器](#使用uu加速器)
  * [VRChat ToS](#vrchat-tos)
  * [TODO](#todo)
<!-- TOC -->

# VRCDancePreloader

## 原理

通过监听PyPyDance在VRChat日志中打印的曲目队列提前下载即将播放的歌曲，从而在降低下载延迟的同时，不占用太多磁盘缓存。

## 使用方法

### 安装证书（可选）

程序使用goproxy代理视频的下载，如果需要支持处理https请求，需要安装根证书，程序首次运行后，会在自身目录下输出根证书`ca.crt`。
双击打开证书，点击“安装证书…”-“下一步”-“将所有证书都放入下列存储”-“浏览…”-“受信任的根证书颁发机构”。

### 设置代理

需要使用系统代理或者网卡代理来拦截VRChat的网络请求，考虑到用户代理的复杂性，本程序不会主动设置系统代理，建议使用Clash Verge
Rev或者Proxifier管理流量。详见“设置代理规则”。

## 个性化

### 配置文件

程序首次运行后，会在自身目录下输出一个配置文件`config.yaml`，是一些比较长、变化少的配置。

**当程序以GUI模式运行时，可以在设置界面中修改这些配置，修改后会自动保存到配置文件。**

```yaml
hijack:
  # 将软件设置为系统代理或者正确配置在其他代理工具中之后，本工具会拦截特定站点的视频请求
  # 作为代理服务器时使用的端口
  proxy-port: 7653
  # 本软件的拦截规则，设定哪些网站会被拦截
  intercepted-sites:
    - jd.pypy.moe
    - api.udon.dance
    - api.wannadance.online
    - www.bilibili.com
    - b23.tv
    - api.xin.moe
  # 是否启用HTTPS劫持，如果启用，需要将软件目录下的证书配置为“受信任的根证书颁发机构”
  enable-https: true
# 本程序无视系统代理和环境变量，
# 需要通过以下配置程序自身下载视频、获取视频信息时使用的http代理，如果留空就不使用代理
proxy:
  # 访问PyPyDance获取视频和缩略图的代理，考虑到PyPyDance在内地有直连中华电信的线路，所以一般不需要配置代理
  pypydance-api: ""
  # 访问WannaDance API获取视频和缩略图的代理，一般不需要配置代理
  wannadance-api: ""
  # 下载YouTube视频使用的代理，目前还没做，不用设置
  youtube-video: ""
  # 请求YouTube API获取YouTube视频信息（如标题）的代理，如果你没有YouTube API key，就不用设置
  youtube-api: ""
  # 请求i.ytimg.com获取YouTube视频缩略图的代理
  youtube-image: ""
keys:
  # YouTube API key，配置后程序可以补全跳舞房日志中空缺的YouTube视频标题
  youtube-api: ""
youtube:
  # 允许通过YouTube API补全YouTube视频信息
  enable-youtube-api: false
  # 允许通过i.ytimg.com加载YouTube视频缩略图
  enable-youtube-thumbnail: false
  # 允许预加载YouTube视频，但是还没做
  enable-youtube-video: false
preload:
  # 提前加载的数量，比如设置为4，加载器会下载当前播放歌曲和后面4首歌
  max-preload-count: 4
download:
  # 最大同时下载的视频数量，优先下载靠前的歌曲，其余歌曲会排队等待
  max-parallel-download-count: 2
cache:
  # 配置磁盘缓存文件夹
  path: "./cache"
  # 配置预加载器缓存可以使用的最大磁盘空间，预加载器退出时会清空较早的视频直到腾出足够的空间
  max-cache-size: 300
  # 清空视频时，是否保留在收藏夹中的视频
  keep-favorites: false
  # 缓存文件的格式
  file-format: 1
db:
  # 本地数据库（用于存储播放历史和乐曲偏好）的路径，启动时会自动创建
  path: ./data.db
live:
  # 是否启用H5网页渲染的直播套件
  enabled: false
  # 网页渲染的直播套件的地址，默认为127.0.0.1:7652
  port: 7652
  # 网页渲染的直播套件的设置，JSON格式，请在浏览器中打开直播套件来设置
  settings: '{}'
```

### 程序参数

|            参数名称            | 含义                         |
|:--------------------------:|----------------------------|
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
  - 'AND,((DOMAIN,api.pypy.dance),(PROCESS-NAME-REGEX,VRChat)),vrcDancePreload'
  - 'AND,((DOMAIN,api.pypy.dance),(PROCESS-NAME-REGEX,yt-dlp)),vrcDancePreload'
  - 'AND,((DOMAIN,api.udon.dance),(PROCESS-NAME-REGEX,VRChat)),vrcDancePreload'
  - 'AND,((DOMAIN,api.udon.dance),(PROCESS-NAME-REGEX,yt-dlp)),vrcDancePreload'
append: [ ]
delete: [ ]
```

**如果你同时使用加速器和Clash，可能需要增加放行VRChat流量的规则：**
```yaml
prepend:
  - 'DOMAIN-SUFFIX,vrchat.com,DIRECT'
  - 'DOMAIN-SUFFIX,vrchat.cloud,DIRECT'
  - 'AND,((DOMAIN,api.pypy.dance),(PROCESS-NAME-REGEX,VRChat)),vrcDancePreload'
  - 'AND,((DOMAIN,api.pypy.dance),(PROCESS-NAME-REGEX,yt-dlp)),vrcDancePreload'
  - 'AND,((DOMAIN,api.udon.dance),(PROCESS-NAME-REGEX,VRChat)),vrcDancePreload'
  - 'AND,((DOMAIN,api.udon.dance),(PROCESS-NAME-REGEX,yt-dlp)),vrcDancePreload'
append: [ ]
delete: [ ]
```

这样你就添加了拦截VRChat和yt-dlp对api.pypy.dance(PyPyDance)和api.udon.dance(WannaDance)域名的所有请求的规则，并交给`vrcDancePreload`节点处理。

记得开Clash Verge的系统代理！

### 使用UU加速器

由于UU加速器不提供规则转发，自身也不能配合Clash使用，因此提供一个有可能可行的方案：

使用路由模式的线路，然后在“网络和Internet”-“代理”-“使用代理服务器”中设置代理IP为`127.0.0.1`，端口为`7653`。

此时UU加速器会按自身规则拦截部分VRChat的请求，其余请求会走VRCDancePreloader，加载器只会处理和跳舞房相关的请求

## VRChat ToS

本项目仅对VRChat的日志进行监听，并利用代理对跳舞房的视频域名提供本地缓存，不会对房间数据进行修改，不以任何方式对游戏进行修改。本项目不是模组或者修改器，不违反VRChat的服务条款。

## TODO

- [x] 断点续传验证
- [x] 完成临时加载歌曲的自动转移
- [ ] 稳定性优化，减少死锁
- [x] 支持WannaDance
- [ ] YouTube视频预加载（需要和yt-dlp完美配合，还没想好怎么做）
- [x] b站视频预加载（准备走[bilibili-real-url](https://github.com/gizmo-ds/bilibili-real-url)）
- [ ] <del>整合一下VRCX的API，实现PyPyDance的收藏同步</del>
- [ ] 通过[OpenVROverlayPipe](https://github.com/BOLL7708/OpenVROverlayPipe)实现SteamVR内通知
- [ ] 自由控制各类来源歌曲是否预加载
- [ ] 欢迎屏幕
- [ ] 展示更多的内部状态（下载队列、冷却时间）
- [ ] 完善H5直播功能
- [ ] 添加心率传感器
