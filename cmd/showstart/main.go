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
	startTime := time.Now()
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

	c := showstart.NewShowStartGeter(d, config.TagsSelected, config.City, config.OtherCityInAfternoon,
		config.InitialEventID, config.MaxNotFoundCount, config.Max404CountToCheck)
	e := email.NewEmailSender(config.Email)
	events, msg, err := c.GetEventsToNotify()
	if err != nil {
		log.Logger.Errorf("get events to notify error %v", err)
		if err := trySendEmail(e, "秀动获取最新演出出错了", err.Error()); err != nil {
			log.Logger.Errorf("发送邮件失败 %v", err)
		}
		return
	}
	endTime := time.Now()
	if len(events) == 0 {
		log.Logger.Info("没有活动需要通知.........")
	}
	cont := content(startTime, endTime, events, msg)
	log.Logger.Infof("准备通知，通知内容为: %s", cont)
	if err := trySendEmail(e, fmt.Sprintf("秀动上新了%d个演出", len(events)), cont); err == nil {
		log.Logger.Infof("成功通知了 %d 个活动........", len(events))
	} else {
		log.Logger.Infof("通知活动时出错：%v", err)
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

func content(start, end time.Time, events []*utils.Event, msg string) string {
	time := fmt.Sprintf("<p>开始运行时间：%s，结束时间：%s</p>", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	if len(events) == 0 {
		return time + "<p>没有活动需要通知</p>" + fmt.Sprintf("<p>%s<p>", msg)
	}
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
