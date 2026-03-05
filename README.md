safeline
======

适用雷池硬件反代、软件反代、硬件透明桥、硬件透明代理等部署模式，支持 20.04、21.07 与 23 等版本。

支持命令行批量添加防护站点、更新防护站点、查看防护站点、添加封禁IP、自定义规则导入导出等功能。

## Web 页面（新增）

程序新增 `web` 子命令，可启动本地 Web 控制台。  
Web 页面通过调用原始 CLI 命令实现功能，因此可覆盖本文档中的所有能力（站点、策略、IP组、证书、ACL、全局/站点规则导入导出等）。

```shell
# 默认监听 127.0.0.1:28000
./safeline-darwin web

# 指定监听地址
./safeline-darwin web --listen 0.0.0.0:28000
```

打开浏览器访问：

```text
http://127.0.0.1:28000
```

页面提供：
- 全局参数输入（`config/addr/token/match/timeout/debug`）
- 一键命令模板（覆盖 README 常见操作）
- 自定义命令执行（可直接输入任意 safeline 子命令）
- 模板文件上传、导出文件下载（CSV/JSON）

## 一键构建与运行脚本（新增）

新增 `build/run.sh`，支持三类常见场景：
- 编译全部平台（darwin/linux/windows，amd64/arm64）
- 交互式数字选择运行方式
- 直接运行源码或本机二进制
- 未显式传 `--config` 时，自动使用 `build/token.csv`

```shell
# 交互菜单（按数字选择）
./build/run.sh

# 编译全部平台
./build/run.sh --build-all

# 编译指定平台架构（示例：linux/arm64）
./build/run.sh --build-target linux arm64

# macOS 也支持用 mac 别名
./build/run.sh --build-target mac arm64

# 直接运行源码
./build/run.sh --run-source ipgroup allip

# 运行本机二进制（不存在会先编译）
./build/run.sh --run-host web --listen 0.0.0.0:28000

# 编号选择已有二进制并运行
./build/run.sh --pick-run ipgroup ls
```

## 使用说明

在雷池上生成 api token （操作权限只需"站点和防护策略管理"即可），填写到 token.csv 模板。

```shell
./safeline-darwin -h                
NAME:
   safeline - 雷池命令行工具

USAGE:
   safeline [global options] command [command options] [arguments...]

VERSION:
   2024.11.11

COMMANDS:
   website      防护站点
   policy       防护策略
   ipgroup      IP组
   cert         SSL证书
   acl          频率控制
   global-rule  全局自定义规则
   custom-rule  站点自定义规则
   assetgroup   web资产组
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value   配置文件，默认加载当前目录下的 token.csv 文件 (default: "token.csv")
   --token value              雷池 openapi token
   --addr value               雷池管理节点地址，比如:"https://169.254.0.4:1443"
   --match value, -m value    当配置文件中有多个雷池管理节点 url，使用这个参数可匹配需要的地址，匹配关系为字符串包含
   --debug, -d                更新站点与创建站点时，输出解析 csv 模版后的数据 (default: false)
   --timeout value, -t value  单个请求超时时间，单位为 s (default: 5)
   --help, -h                 show help (default: false)
   --version, -v              print the version (default: false)
```

全局参数说明：  
`config`：默认加载当前目录下的 token.csv 文件配置，不在当前目录或者文件名称不一样需要指定此参数。  
`addr`：从命令行读取雷池管理节点地址，优先级大于从配置文件加载。  
`token`：从命令行读取雷池 openapi token，优先级大于从配置文件加载。  
`match`: 当 token.csv 模板中填写了多个地址时，使用此参数可以用于匹配本次操作需要的地址，匹配关系为字符串包含。  
`debug`: 默认关闭。更新站点与创建站点时，输出解析 csv 模版后的数据，用于调试。  
`timeout`: 单个请求超时时间，单位为秒，默认为5。站点数量达到几百个量级后，更新站点或者创建站点操作时，雷池响应时间会特别慢，可能大于5s，可适当增加。

### 防护站点管理

目前支持硬件反代、软件反代、硬件透明桥、硬件透明代理4种部署模式。

```shell
./safeline-darwin website -h

NAME:
   safeline website - 防护站点

USAGE:
   safeline website command [command options] [arguments...]

COMMANDS:
   ls       list website
   create   创建站点
   update   更新站点
   export   导出模版
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --mode value, -m value  部署模式[TransparentBridge TransparentProxy HardwareReverseProxy SoftwareReverseProxy]
   --help, -h              show help (default: false)
```

#### 创建站点

填写对应模式部署 csv 模版文件。

示例：
```shell
./safeline-darwin website create -f template.csv
```

#### 更新站点

根据实际使用经验，创建站点后，一般很少需要去全量更新所有站点，安全起见，目前只支持修改站点是否启用、防护策略、证书修改、工作组等字段。
程序会解析 csv 模版中的所有字段，但是只会更新选择的字段。

```shell
./safeline-darwin website  update -h
NAME:
   safeline website update - 更新站点

USAGE:
   safeline website update [command options] [arguments...]

OPTIONS:
   --filename value, -f value  模板文件
   --select value, -s value    选择更新的字段 e.g. "0,1,2"
                               0：站点是否启用（21及以上版本支持）
                               1：防护策略ID
                               2：源IP获取方式
                               3：SSL证书ID
                               4：工作组（透明代理与硬件反代模式）
                               5：web资产组ID
                               10：访问日志设置 (透明代理与软、硬件反代模式）
                               
                               以下仅支持软件反代与硬件反代模式：
                               6：站点生效URL路径（23版本开始支持）
                               7：站点生效的检测节点（23版本开始支持）
                               8：用户识别方式
                               9：回源IP配置
                               11：HTTP头配置
                               12：保持连接配置
                               13：业务服务器获取源IP XFF 配置
   --help, -h                  show help (default: false)
````

导出当前防护站点数据为模版：
```shell
./safeline-darwin website export -f template.csv
```

获取到最新的模版数据后，可对需要的字段进行修改，更新站点是通过`防护站点名称`去定位的，如果多个站点名一样，会修改多个防护站点。

示例，只更新源IP获取方式：
```shell
./safeline-darwin website update -f template.csv -s "2"
```

也可以一次性更改多个字段，用逗号分隔：
```shell
./safeline-darwin website update -f template.csv -s "2,3"
```

#### 查看站点


标准输出：
```shell
./safeline-darwin website ls
```

输出至 csv 中：
```shell
./safeline-darwin website ls -o website 
```

输出原始 json 数据：
```shell
./safeline-darwin website ls -o website -r
```

### 封禁IP

适用所有模式，支持从频率控制规则中添加IP，也可以添加封禁ip至ip组中。

#### 频率控制规则操作

```shell
./safeline-darwin acl -h

NAME:
   safeline acl - 频率控制

USAGE:
   safeline acl command [command options] [arguments...]

COMMANDS:
   ls       list acl
   addip    添加IP
   getip    查询IP
   delip    删除IP
   clearip  清空所有IP
   create   创建频率控制规则
   delete   删除频率控制规则
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

#### IP组操作

```shell
./safeline-darwin ipgroup -h

NAME:
   safeline ipgroup - IP组

USAGE:
   safeline ipgroup command [command options] [arguments...]

COMMANDS:
   ls       list ipgroup
   getip    查询IP组内IP
   allip    联合查询所有IP组及IP
   exportip 导出IP组及IP明细
   addip    添加IP
   delip    删除IP
   create   创建IP组
   delete   删除IP组
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

### 自定义规则导入导出

全局自定义规则

```shell
# 导出
./safeline-darwin global-rule export -f test

# 导入
./safeline-darwin global-rule import -f test.json
```

站定自定义规则

```shell
# 导出
./safeline-darwin custom-rule export -f test

# 导入
./safeline-darwin custom-rule import -f test.json
```

### 其它子命令

有时候需要查看一些资源id，比如工作组id、防护策略id、证书id、web资产组id等等  
`ls` 命令对所有资源适用，也可使用 --help/-h 查看帮助。

### 模版参数约束

参考：https://wiki.chaitin.net/pages/viewpage.action?pageId=415469587&focusedCommentId=555254924#comment-555254924
