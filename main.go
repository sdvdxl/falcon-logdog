package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/sdvdxl/falcon-logdog/config"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	logTail *tail.Tail
)

func main() {
	cfg := config.ReadConfig("cfg.json")

	logFile := getLogFile(cfg)
	if logFile != "" {
		logTail = readFile(logFile, cfg)
	}

	logFileWatcher(cfg)

}
func logFileWatcher(cfg config.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		config.Logger.Fatal(err)
	}

	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					newLogfile := event.Name
					if strings.HasSuffix(newLogfile, cfg.Suffix) && strings.HasPrefix(newLogfile, cfg.Prefix) {
						logTail.Stop()
						logTail = readFile(event.Name, cfg)
						log.Println("created file:", event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("var")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func readFile(filename string, c *config.Config) *tail.Tail {
	tail, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		config.Logger.Fatal(err)
	}

	go func() {
		for line := range tail.Lines {
			handleKeywords(line.Text, c)
		}
	}()

	return tail
}

func getLogFile(cfg config.Config) string {
	result := ""
	filepath.Walk(cfg.Path, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, cfg.Prefix) && strings.HasSuffix(path, cfg.Suffix) && !info.IsDir() {
			result = path
			return nil
		}

		return err
	})

	config.Logger.Println("read log file:", result)

	return result
}

// 查找关键词
func handleKeywords(line, c *config.Config) {
	for _, p := range c.Regs {
		for _, foundStr := range p.FindAllString(line, -1) {
			// 上报agent
			fmt.Println("========= found:", foundStr)
		}

	}
}
