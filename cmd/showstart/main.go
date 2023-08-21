package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"show-live/config"
	"show-live/internal/showstart"
	"show-live/pkg/db"
	"show-live/pkg/email"
	"show-live/pkg/log"
	"show-live/utils"
)

const (
	emailTryTimes = 10
)

func main() {
	var config config.ShowStart
	configFilePath := flag.String("config", "config-showstart.yml", "config file")
	if configFilePath != nil {
		configFile, err := os.ReadFile(*configFilePath)
		if err != nil {
			log.Logger.Fatal(err)
		}
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			log.Logger.Fatal(err)
		}
	}
	log.InitLogger(config.Log.LogSuffix, config.Log.LogDir)
	log.Logger.Info("服务准备运行，启动中.........")
	d, err := db.InitSqlite(config.DBFile)
	if err != nil {
		log.Logger.Errorf("初始化数据库错误 %v", err)
		return
	}
	defer func() {
		if err := d.Exit(); err != nil {
			log.Logger.Errorf("数据库退出过程中出错 %v", err)
		}
	}()

	e := email.NewEmailSender(config.Email)
	c := showstart.NewShowStartGeter(d, config.TagsSelected, config.City, config.OtherCityInAfternoon,
		config.InitialEventID, config.MaxNotFoundCount, config.Max404CountToCheck)
	events, msg, err := c.GetEventsToNotify()
	if err != nil {
		log.Logger.Errorf("get events to notify error %v", err)
		if err := trySendEmail(e, "秀动获取最新演出出错了", err.Error()); err != nil {
			log.Logger.Errorf("发送邮件失败 %v", err)
		}
		return
	}
	if len(events) == 0 {
		log.Logger.Info("没有活动需要通知，程序返回.........")
		return
	}
	log.Logger.Infof("准备通知，通知内容为: %s", content(events, msg))
	if err := trySendEmail(e, fmt.Sprintf("秀动上新了%d个演出", len(events)), content(events, msg)); err == nil {
		log.Logger.Infof("成功通知了 %d 个活动........", len(events))
	}
}

func trySendEmail(e *email.EmailSender, title string, content string) error {
	var errToReturn error
	for i := 0; i < emailTryTimes; i++ {
		err := e.Send(title, content)
		if err == nil {
			break
		}
		if err != nil {
			log.Logger.Errorf("发送邮件失败 %v, 第 %d 次失败", err, i+1)
		}
		if i == emailTryTimes-1 {
			errToReturn = err
		}
		time.Sleep(time.Second)
	}
	return errToReturn
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
