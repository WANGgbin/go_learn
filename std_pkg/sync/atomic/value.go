package main

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"
)

func getDefaultConfig() map[string]string {
	return make(map[string]string)
}

func printCfg(goroutineID int, cfg map[string]string) {
	var info string
	info += fmt.Sprintf("Goroutine: %d\n", goroutineID)
	for key, value := range cfg {
		info += fmt.Sprintf("key: %s, val: %s\n", key, value)
	}
	info += "\n"
	fmt.Print(info)
}

func main() {
	var config atomic.Value
	config.Store(getDefaultConfig())

	go func() {
		i := int64(0)
		for {

			originCfg := config.Load().(map[string]string)
			newCfg := make(map[string]string)

			for key, val := range originCfg {
				newCfg[key] = val
			}

			newCfg[strconv.FormatInt(i, 10)] = strconv.FormatInt(i, 10)
			i++
			config.Store(newCfg)
			time.Sleep(2 * time.Second)
		}
	}()

	for i := 1; i <= 10; i++ {
		go func(index int) {
			for{
				cfg := config.Load().(map[string]string)
				printCfg(index, cfg)
				time.Sleep(1 * time.Second)
			}
		}(i)
	}

	done := make(chan struct{})
	<-done
}
