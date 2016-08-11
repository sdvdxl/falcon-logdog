package main

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/sdvdxl/falcon-logdog/config"
	"github.com/streamrail/concurrent-map"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	logTail  *tail.Tail
	workers  chan bool
	keywords cmap.ConcurrentMap
)

func main() {
	workers = make(chan bool, runtime.NumCPU()*2)
	keywords = cmap.New()
	runtime.GOMAXPROCS(runtime.NumCPU())

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(int64(config.Cfg.Timer)))
		for t := range ticker.C {
			fillData()

			log.Println("INFO: time to push data: ", keywords.Items(), t)
			postData(keywords)
		}
	}()

	go func() {
		file := getLogFile()
		if file != "" {
			logTail = readFile(file)
		}
	}()

	logFileWatcher()

}
func logFileWatcher() {
	c := config.Cfg
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("ERROR:", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if c.PathIsFile && event.Op == fsnotify.Create && event.Name == c.Path {
					log.Println("INFO: continue to watch file:", event.Name)
					if logTail != nil {
						logTail.Stop()
					}

					logTail = readFile(c.Path)
				} else {
					if event.Op == fsnotify.Create {
						newLogfile := event.Name
						log.Println("INFO: created file", event.Name)
						if strings.HasSuffix(newLogfile, config.Cfg.Suffix) && strings.HasPrefix(newLogfile, config.Cfg.Prefix) {
							if logTail != nil {
								logTail.Stop()
							}

							logTail = readFile(event.Name)

						}
					}
				}

			case err := <-watcher.Errors:
				log.Println("ERROR:", err)
			}
		}
	}()

	err = watcher.Add(filepath.Dir(c.Path))
	if err != nil {
		log.Fatal("ERROR:", err)

	}
	<-done
}

func readFile(filename string) *tail.Tail {

	log.Println("INFO: read file", filename)
	tail, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		log.Fatal("ERROR:", err)
	}

	go func() {
		for line := range tail.Lines {
			handleKeywords(line.Text)
		}
	}()

	return tail
}

func getLogFile() string {
	c := config.Cfg
	result := ""

	if config.Cfg.PathIsFile {
		return c.Path
	}

	filepath.Walk(c.Path, func(path string, info os.FileInfo, err error) error {
		cfgPath := config.Cfg.Path
		if strings.HasSuffix(cfgPath, "/") {
			cfgPath = string([]rune(cfgPath)[:len(cfgPath)-1])
		}

		//只读取root目录的log
		if filepath.Dir(path) != cfgPath {
			log.Println("DEBUG: ", path, "not in root path, ignoring , Dir:", filepath.Dir(path), "cofig path:", cfgPath)
			return err
		}

		if strings.HasPrefix(path, config.Cfg.Prefix) && strings.HasSuffix(path, config.Cfg.Suffix) && !info.IsDir() {
			result = path
			return err
		}

		return err
	})

	return result
}

// 查找关键词
func handleKeywords(line string) {
	c := config.Cfg
	for _, p := range c.Keywords {
		value := 0.0
		if p.Regex.MatchString(line) {
			value = 1.0
		}

		var data config.PushData
		if v, ok := keywords.Get(p.Exp); ok {
			d := v.(config.PushData)
			d.Value += value
			data = d
		} else {
			data = config.PushData{Metric: c.Metric,
				Endpoint:    c.Host,
				Timestamp:   time.Now().Unix(),
				Value:       value,
				Step:        c.Timer,
				CounterType: "GAUGE",
				Tags:        p.Tag + "=" + p.Exp,
			}
		}

		keywords.Set(p.Exp, data)

	}
}

func postData(m cmap.ConcurrentMap) {
	c := config.Cfg
	workers <- true

	go func() {
		if len(m.Items()) != 0 {
			data := make([]config.PushData, 0, 20)
			for k, v := range m.Items() {
				data = append(data, v.(config.PushData))
				m.Remove(k)
			}

			bytes, err := json.Marshal(data)
			if err != nil {
				log.Println("ERROR : marshal push data", data, err)
				return
			}

			resp, err := http.Post(c.Agent, "plain/text", strings.NewReader(string(bytes)))
			if err != nil {
				log.Println("ERROR: post data ", string(bytes), " to agent ", err)
			} else {
				defer resp.Body.Close()
				bytes, _ = ioutil.ReadAll(resp.Body)
				fmt.Println("INFO:", string(bytes))
			}
		}

		<-workers
	}()

}

func fillData() {
	c := config.Cfg
	for _, p := range c.Keywords {

		if _, ok := keywords.Get(p.Exp); ok {
			continue
		}

		//不存在要插入一个补全
		data := config.PushData{Metric: c.Metric,
			Endpoint:    c.Host,
			Timestamp:   time.Now().Unix(),
			Value:       0.0,
			Step:        c.Timer,
			CounterType: "GAUGE",
			Tags:        p.Tag + "=" + p.Exp,
		}

		keywords.Set(p.Exp, data)
	}

}
