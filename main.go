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
	workers = make(chan bool, runtime.NumCPU() * 2)
	keywords = cmap.New()
	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg := config.ReadConfig("cfg.json")

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(int64(cfg.Timer)))
		for t := range ticker.C {
			log.Println("INFO: time to push data: ", keywords.Items(), t)
			postData(keywords, &cfg)
		}
	}()

	go func() {
		file := getLogFile(&cfg)
		if file != "" {
			logTail = readFile(file, &cfg)
		}
	}()

	logFileWatcher(&cfg)

}
func logFileWatcher(cfg *config.Config) {
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
				if event.Op & fsnotify.Create == fsnotify.Create {
					newLogfile := event.Name
					log.Println("INFO: created file", event.Name)
					if strings.HasSuffix(newLogfile, cfg.Suffix) && strings.HasPrefix(newLogfile, cfg.Prefix) {
						if logTail != nil {
							logTail.Stop()
						}

						logTail = readFile(event.Name, cfg)

					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(cfg.Path)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func readFile(filename string, c *config.Config) *tail.Tail {

	log.Println("INFO: read file", filename)
	tail, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for line := range tail.Lines {
			handleKeywords(line.Text, c)
		}
	}()

	return tail
}

func getLogFile(cfg *config.Config) string {
	result := ""
	filepath.Walk(cfg.Path, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, cfg.Prefix) && strings.HasSuffix(path, cfg.Suffix) && !info.IsDir() {
			result = path
			return nil
		}

		return err
	})

	return result
}

// 查找关键词
func handleKeywords(line string, c *config.Config) {
	for _, p := range c.Regs {
		tags := ""
		for _, foundStr := range p.FindAllString(line, -1) {
			tags += p.String() + "=" + foundStr + ","
		}

		value := 1.0
		if tags == "" {
			value = 0.0
		}

		if v, ok := keywords.Get(p.String()); ok {
			d := v.(config.PushData)
			d.Value += value
			keywords.Set(p.String(), d)
		} else {
			d := config.PushData{Metric: c.Metric,
				Endpoint:    c.Host,
				Timestamp:   time.Now().Unix(),
				Value:       value,
				Step:        c.Timer,
				CounterType: "GAUGE",
				Tags:        tags,
			}
			keywords.Set(p.String(), d)

		}

	}
}

func postData(m cmap.ConcurrentMap, c *config.Config) {
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
