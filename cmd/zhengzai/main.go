package main

import (
	"flag"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"show-live/config"
	"show-live/internal/zhengzai"
	"show-live/pkg/db"
	"show-live/pkg/email"
	"show-live/pkg/log"
)

func main() {
	var config config.Zhengzai
	configFilePath := flag.String("config", "config-z.yml", "config file")
	if configFilePath != nil {
		configFile, err := ioutil.ReadFile(*configFilePath)
		if err != nil {
			log.Logger.Fatal(err)
		}
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			log.Logger.Fatal(err)
		}
	}
	log.InitLogger(config.Log.LogSuffix, config.Log.LogDir)
	d, err := db.InitCache(config.DBDir)
	if err != nil {
		log.Logger.Errorf("init cache error %v", err)
		return
	}
	defer func() {
		if err := d.Exit(); err != nil {
			log.Logger.Errorf("db exits error %v", err)
		}
	}()
	e := email.NewEmailSender(config.Email)
	c := zhengzai.NewZhengZaiGetterGetter(d, config.URL, config.AdCode)
	events, err := c.GetEventsToNotify()
	if err != nil {
		log.Logger.Errorf("get events to notify error %v", err)
		if err := e.Send("正在现场获取最新演出出错了", err.Error()); err != nil {
			log.Logger.Errorf("send email error %v", err)
		}
		return
	}
	if len(events) == 0 {
		log.Logger.Info("no new event to send")
		return
	}
	var content string
	for _, v := range events {
		content += v + "\n"
	}
	if err := e.Send("正在现场演出上新了", content); err != nil {
		log.Logger.Errorf("send email error %v", err)
	}
	log.Logger.Infof("%d event sent", len(events))
}
