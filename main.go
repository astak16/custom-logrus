package main

import (
	mylog "gin-demo/my-log"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	// 按照日期格式化输出
	mlog := mylog.NewDateLog(&mylog.DateLogConfig{
		Date: time.Now().Format("2006-01-02 15-04-05"),
		Path: "logrus_stydy/logs",
		Name: "uccs",
	})

	// 按照 level 格式化输出
	// mlog := mylog.NewLevelLog(&mylog.LevelConfig{
	// 	Date: time.Now().Format("2006-01-02 15-04-05"),
	// 	Path: "logrus_stydy/logs",
	// 	Name: "uccs",
	// })

	log := mlog.Init()
	log.SetLevel(logrus.DebugLevel)

	log.Errorln("error 你好")
	log.Warnln("warn 你好")
	log.Infof("info 你好")
	log.Debug("debug 你好")
}
