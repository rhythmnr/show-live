package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"show-live/config"
	"show-live/internal/showstart"
	"show-live/pkg/db"
	"show-live/pkg/email"
	"show-live/pkg/log"
	"show-live/utils"
)

func main() {
	var config config.ShowStart
	configFilePath := flag.String("config", "config-showstart.yml", "config file")
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
	d, err := db.InitSqlite(config.DBFile)
	if err != nil {
		log.Logger.Errorf("init cache error %v", err)
		return
	}
	defer func() {
		if err := d.Exit(); err != nil {
			log.Logger.Errorf("db exits error %v", err)
		}
	}()

	c := showstart.NewShowStartGeter(d, config.TagsSelected, config.City, config.InitialEventID, config.MaxNotFoundCount)
	events, msg, err := c.GetEventsToNotify()
	e := email.NewEmailSender(config.Email)
	if err != nil {
		log.Logger.Errorf("get events to notify error %v", err)
		if err := e.Send("秀动获取最新演出出错了", err.Error()); err != nil {
			log.Logger.Errorf("send email error %v", err)
		}
		return
	}
	if len(events) == 0 {
		log.Logger.Info("no new event to send")
		return
	}

	if err := e.Send(fmt.Sprintf("秀动上新了%d个演出", len(events)), content(events, msg)); err != nil {
		log.Logger.Errorf("send email error %v", err)
	}
	log.Logger.Infof("%d event sent", len(events))
}

func content(events []*utils.Event, msg string) string {
	r := "<p>购票前务必先看大麦与确认是否有空观看，即使显示独家也要确认大麦！</p>"
	for _, e := range events {
		r += fmt.Sprintf("<p>🤜<a href=\"%s\"><font color=green></strong>%s<strong></font></a>，<strong>演出时间</strong>：%s，"+
			"<strong>艺人</strong>： %s，<strong>场地</strong>：%s，<strong>票价</strong>：%s，<a href=\"%s\">App内查看详情</a></p>",
			e.WebURL, e.Name, e.Time, e.Artist, e.Site, e.Price, e.WebViewURL,
		)
	}
	r += fmt.Sprintf("<p>%s<p>", msg)
	return r
}
