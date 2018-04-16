# falcon-logdog
open-falcon日志关键词监控

## 下载
可以下载2进制包进行安装，也可以clone本库到机器上安装。clone本库的话，请自行编译。`control ` 脚本默认执行的是 `falcon-logdog` 文件。

## 配置

配置文件名字为 `cfg.json` ，而且必须是这个名字。各个字段说明如下：

名字 | 默认值 | 必填 | 说明
---- | ----|----|----
metric | 无 | 是 | 统计度量，比如叫做 log
path | 无 | 是 | 要监控的日志目录或者文件,如果是目录则会寻找其中一个匹配的日志文件,如果是文件,则会直接监控这个文件,但是不管如何,启动程序时候路径都要存在
timer | 无 | 是 | 要同步数据间隔时间和上报数据的step值，api接口貌似最小30，保持 60为好
agent | 无 | 是 | agent api url，比如 http://localhost:1988/v1/push
host | hostname 命令查看的值 | 否 | 主机名字，根据hostname设定，不要使用localhost，可能导致查询不到数据
prefix | 空字符串 | 否 | 要监控的日志文件名字前缀
suffix | .log | 否 | 要监控的日志文件后缀，如果不填则用默认值
keywords | 无 | 是 | 是 keyword对象数组

keyword 对象说明

名字 | 默认值 | 必填 | 说明
---- | ----|----|----
exp | 无 | 是 | 正则表达式,tag的value
tag|无|是| 对应于监控中tag的key

### 配置热更新

组件支持配置热更新，即不需要重启即可让最新配置生效。注意，配置文件中timer不支持热更新，其余参数都是支持的。同时，如果修改配置文件导致配置错误，新的配置不会生效，会继续使用旧的配置，直到配置内容正确为止。

## 上报数据
格式如下：

```json
{
	"metrix":"log", 
	"endpoint":"10-10-128-53",
	"timestamp":1470827020,
	"value":5,
	"step":60,
	"counterType":"GAUGE",
	"tags":"tag=exp"
}
```

其中，tags 格式为 `keywords` 中 'tag' + '=' + 'FixedExp', `FixedExp` 是用`.`替换 `exp` 之后的并将`.`去重字符串。

## 启动脚本
使用 `control` 脚本来操作:
./control option

option 可以为

- start 启动
- stop 停止
- status 查看运行状态
- restart 重启
- tail 类似tail 查看日志

## 日志操作

- 如果所监控的文件夹有多个 `.log` 结尾日志文件，那么只会选择其中一个，可能会选择最后创建的文件（具体还未测试观察）
- 如果有新的文件创建，且符合上面 `prefix` 和 `suffix` 过滤规则，那么会切换到这个新文件上进行监控。

### 日志格式
如果有数据要push上去，那么可能会观察到有如下日志产生：

```
time to push data:  map[key2:{log 10-10-128-53 1470827020 5 60 GAUGE key2=tag2,} key1:{log 10-10-128-53 1470827020 5 60 GAUGE key1=tag,}] 2016-08-10 19:04:40.850295266 +0800 CST
```
