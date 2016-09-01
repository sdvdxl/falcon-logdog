package config

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/go-errors/errors"
	"github.com/hpcloud/tail"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Metric     string      //度量名称,比如log.console 或者log
	Timer      int         // 每隔多长时间（秒）上报
	Host       string      //主机名称
	Agent      string      //agent api url
	WatchFiles []WatchFile `json:"files"`
}

type resultFile struct {
	FileName string
	ModTime  time.Time
	LogTail  *tail.Tail
}

type WatchFile struct {
	Path       string //路径
	Prefix     string //log前缀
	Suffix     string //log后缀
	Keywords   []keyWord
	PathIsFile bool       //path 是否是文件
	ResultFile resultFile `json:"-"`
}

type keyWord struct {
	Exp      string
	Tag      string
	FixedExp string         `json:"-"` //替换
	Regex    *regexp.Regexp `json:"-"`
}

//说明：这7个字段都是必须指定
type PushData struct {
	Metric    string  `json:"metric"`    //统计纬度
	Endpoint  string  `json:"endpoint"`  //主机
	Timestamp int64   `json:"timestamp"` //unix时间戳,秒
	Value     float64 `json:"value"`     // 代表该metric在当前时间点的值
	Step      int     `json:"step"`      //  表示该数据采集项的汇报周期，这对于后续的配置监控策略很重要，必须明确指定。
	//COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔
	//COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔

	CounterType string `json:"counterType"` //只能是COUNTER或者GAUGE二选一，前者表示该数据采集项为计时器类型，后者表示其为原值 (注意大小写)
	//GAUGE：即用户上传什么样的值，就原封不动的存储
	//COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔
	Tags string `json:"tags"` //一组逗号分割的键值对, 对metric进一步描述和细化, 可以是空字符串. 比如idc=lg，比如service=xbox等，多个tag之间用逗号分割
}

const configFile = "cfg.json"

var (
	Cfg         *Config
	fixExpRegex = regexp.MustCompile(`[\W]+`)
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
	var err error
	Cfg, err = ReadConfig(configFile)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	if err = checkConfig(Cfg); err != nil {
		log.Fatal(err)
	}

	go func() {
		ConfigFileWatcher()
	}()

	fmt.Println("INFO: config:", Cfg)
}

func ReadConfig(configFile string) (*Config, error) {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config *Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	// 检查配置项目
	if err := checkConfig(config); err != nil {
		return nil, err
	}

	log.Println("INFO: config init success, start to work ...")
	return config, nil
}

// 检查配置项目是否正确
func checkConfig(config *Config) error {
	var err error

	//检查 host
	if config.Host == "" {
		if config.Host, err = os.Hostname(); err != nil {
			return err
		}

		log.Println("INFO: host not set will use system's name:", config.Host)

	}

	for i, v := range config.WatchFiles {
		//检查路径
		fInfo, err := os.Stat(v.Path)
		if err != nil {
			return err
		}

		if !fInfo.IsDir() {
			config.WatchFiles[i].PathIsFile = true
		}

		//检查后缀,如果没有,则默认为.log
		config.WatchFiles[i].Prefix = strings.TrimSpace(v.Prefix)
		config.WatchFiles[i].Suffix = strings.TrimSpace(v.Suffix)
		if config.WatchFiles[i].Suffix == "" {
			log.Println("INFO: file pre ", config.WatchFiles[i].Path, "suffix is no set, will use .log")
			config.WatchFiles[i].Suffix = ".log"
		}

		//agent不检查,可能后启动agent

		//检查keywords
		if len(v.Keywords) == 0 {
			return errors.New("ERROR: keyword list not set")
		}

		for _, v := range v.Keywords {
			if v.Exp == "" || v.Tag == "" {
				return errors.New("ERROR: keyword's exp and tag are requierd")
			}
		}

		// 设置正则表达式
		for i, v := range v.Keywords {

			if config.WatchFiles[i].Keywords[i].Regex, err = regexp.Compile(v.Exp); err != nil {
				return err
			}

			config.WatchFiles[i].Keywords[i].FixedExp = string(fixExpRegex.ReplaceAll([]byte(v.Exp), []byte(".")))
		}
	}

	return nil
}

//配置文件监控,可以实现热更新
func ConfigFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Name == configFile && (event.Op == fsnotify.Chmod || event.Op == fsnotify.Rename || event.Op == fsnotify.Write || event.Op == fsnotify.Create) {
					log.Println("INFO: modified config file", event.Name, "will reaload config")
					if cfg, err := ReadConfig(configFile); err != nil {
						log.Println("ERROR: config has error, will not use old config", err)
					} else if checkConfig(Cfg) != nil {
						log.Println("ERROR: config has error, will not use old config", err)
					} else {
						log.Println("INFO: config reload success")
						Cfg = cfg
					}

				}
			case err := <-watcher.Errors:
				log.Println("ERROR:", err)
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
