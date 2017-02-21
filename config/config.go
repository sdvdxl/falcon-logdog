package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-errors/errors"
	"github.com/hpcloud/tail"
)

// SeekInfoFile 文件读取信息
const SeekInfoFile = ".config/fileseek.info"

// Config 配置信息
type Config struct {
	Metric     string      //度量名称,比如log.console 或者log
	Timer      int         // 每隔多长时间（秒）上报
	Host       string      //主机名称
	Agent      string      //agent api url
	WatchFiles []WatchFile `json:"files"`
	LogLevel   string
}

// SeekInfo 文件读取信息
type SeekInfo struct {
	Offset int
	Whence int
}

// Write 将读取的信息写入文件保存
func (info SeekInfo) Write(offset int, whence int) {
	//info.SeekFile.WriteString(fmt.Sprintf("%s\n%v\n%v\n", info.FileName, offset, whence))
}

type resultFile struct {
	FileName string
	ModTime  time.Time
	LogTail  *tail.Tail
}

// WatchFile 要监控的文件
type WatchFile struct {
	Path       string //路径
	Prefix     string //log前缀
	Suffix     string //log后缀
	Keywords   []keyWord
	PathIsFile bool       //path 是否是文件
	ResultFile resultFile `json:"-"`
	SeekInfo   SeekInfo   // 文件定位，写入到 .config/fileseek.info 文件
}

type keyWord struct {
	Exp      string
	Tag      string
	FixedExp string         `json:"-"` //替换
	Regex    *regexp.Regexp `json:"-"`
}

// PushData 说明：这7个字段都是必须指定
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

	var err error
	Cfg, err = ReadConfig(configFile)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	if err = checkConfig(Cfg); err != nil {
		log.Fatal(err)
	}

	go func() {
		FileWatcher()
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

	fmt.Println(config.LogLevel)

	// 检查配置项目
	if err := checkConfig(config); err != nil {
		return nil, err
	}

	// 读取文件seek信息
	// if seekFile, err := os.Open(SeekInfoFile); err != nil {
	// 	if err == os.ErrNotExist {
	// 		// 创建文件
	// 		if config.SeekInfo.SeekFile, err = os.Create(SeekInfoFile); err != nil {
	// 			log.Fatal("can not create config file", SeekInfoFile)
	// 		}
	// 	}
	// } else {
	// 	if bytes, err := ioutil.ReadAll(seekFile); err != nil {
	// 		log.Println("WARN", "file seek info has error, will reread")
	// 	} else {
	// 		config.SeekInfo.SeekFile = seekFile
	// 		info := strings.Split(string(bytes), "\n")
	// 		config.SeekInfo.FileName = info[0]
	// 		config.SeekInfo.Offset, _ = strconv.Atoi(info[1])
	// 		config.SeekInfo.Whence, _ = strconv.Atoi(info[2])
	// 	}
	// }

	// if config.SeekInfo.FileName != "" {
	// 	config.WatchFiles
	// }

	log.Println("config init success, start to work ...")
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

		log.Println("host not set will use system's name:", config.Host)

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
			log.Println("file pre ", config.WatchFiles[i].Path, "suffix is no set, will use .log")
			config.WatchFiles[i].Suffix = ".log"
		}

		//agent不检查,可能后启动agent
		//检查keywords
		if len(v.Keywords) == 0 {
			return errors.New("ERROR: keyword list not set")
		}

		for _, keyword := range v.Keywords {
			if keyword.Exp == "" || keyword.Tag == "" {
				return errors.New("ERROR: keyword's exp and tag are requierd")
			}
		}

		// 设置正则表达式
		for j, keyword := range v.Keywords {

			if config.WatchFiles[i].Keywords[j].Regex, err = regexp.Compile(keyword.Exp); err != nil {
				return err
			}

			log.Println("INFO: tag:", keyword.Tag, "regex", config.WatchFiles[i].Keywords[j].Regex.String())

			config.WatchFiles[i].Keywords[j].FixedExp = string(fixExpRegex.ReplaceAll([]byte(keyword.Exp), []byte(".")))
		}
	}

	return nil
}

// FileWatcher 配置文件监控,可以实现热更新
func FileWatcher() {
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
				if event.Name == configFile && event.Op != fsnotify.Chmod { //&& (event.Op == fsnotify.Chmod || event.Op == fsnotify.Rename || event.Op == fsnotify.Write || event.Op == fsnotify.Create)
					log.Println("modified config file", event.Name, "will reaload config")
					if cfg, err := ReadConfig(configFile); err != nil {
						log.Println("ERROR: config has error, will not use old config", err)
					} else if checkConfig(Cfg) != nil {
						log.Println("ERROR: config has error, will not use old config", err)
					} else {
						log.Println("config reload success")
						Cfg = cfg
					}

				}
			case err := <-watcher.Errors:
				log.Fatal(err)
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
