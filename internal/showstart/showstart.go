package showstart

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"show-live/pkg/db"
	"show-live/pkg/log"
	"show-live/utils"
)

type ShowStart struct {
	d                    db.DB
	includeLabels        []string
	city                 []string
	otherCityInAfternoon []string
	initEventID          int64
	MaxNotFoundCount     int64
	Max404CountToCheck   int64
}

func NewShowStartGeter(d db.DB, tags []string, city, otherCityInAfternoon []string,
	initEventID, MaxNotFoundCount, Max404CountToCheck int64) *ShowStart {
	return &ShowStart{
		d:                    d,
		includeLabels:        tags,
		city:                 city,
		otherCityInAfternoon: otherCityInAfternoon,
		initEventID:          initEventID,
		MaxNotFoundCount:     MaxNotFoundCount,
		Max404CountToCheck:   Max404CountToCheck,
	}
}

const EventPushed = "已推送"
const EventNotInterested = "不感兴趣"
const Evenet404 = "404"
const EvenetErrorWhenRequest = "请求活动时报错"

func eventKeyInDB(id int64) string {
	return fmt.Sprintf("showstart_eventid_%d", id)
}

func eventURL(id int64) string {
	url := "https://www.showstart.com/event"
	return fmt.Sprintf("%s/%d", url, id)
}

func fillEventURL(id int64, e *utils.Event) {
	e.WebURL = fmt.Sprintf("https://www.showstart.com/event/%d", id)
	e.WebViewURL = fmt.Sprintf("https://wap.showstart.com/pages/activity/detail/detail?activityId=%d", id)
}

func (c *ShowStart) GetEventsToNotify() ([]*utils.Event, string, error) {
	initialEventID, err := c.d.GetValue(fmt.Sprintf("showstart_initial_eventid"))
	if err != nil {
		return nil, "", err
	}
	var initID int64
	if initialEventID != "" {
		initID, err = strconv.ParseInt(initialEventID, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("转换初试ID为int出错 %v", err)
		}
	}
	if initID > c.initEventID {
		c.initEventID = initID
	}
	events := make([]*utils.Event, 0)
	eventID := c.initEventID - 1
	consistentNonexistEventCount := int64(0)
	var errMsg string
	for {
		eventID++
		if consistentNonexistEventCount == c.MaxNotFoundCount {
			break
		}
		keyInDB := eventKeyInDB(eventID)
		value, err := c.d.GetValue(keyInDB)
		if err != nil {
			log.Logger.Errorf("检查键 %s 是否在数据库中存在时出错 %v", keyInDB, err)
			continue
		}
		if value == EventPushed || value == EventNotInterested {
			continue
		}
		e, err := c.requestEvent(eventURL(eventID), eventID)
		name := "未知"
		if e != nil {
			name = e.Name
		}
		if err != nil {
			if err == ErrorNotInterested {
				consistentNonexistEventCount = 0
				c.d.SetKey(keyInDB, name, EventNotInterested)
				continue
			}
			if err == Error404 {
				consistentNonexistEventCount++
				c.d.SetKey(keyInDB, name, Evenet404)
				continue
			}
			// 这个部分在出错的时候，返回错误内容，并在数据库里将活动标记为出错
			errMsg += fmt.Sprintf("请求演出报错，ID：%d，错误：%v\n", eventID, err)
			c.d.SetKey(keyInDB, name, EvenetErrorWhenRequest)
			continue
		}
		consistentNonexistEventCount = 0
		c.d.SetKey(keyInDB, name, EventPushed)
		fillEventURL(eventID, e)
		events = append(events, e)
	}
	initialID := eventID - int64(consistentNonexistEventCount) - 1
	if err := c.d.SetKey("showstart_initial_eventid", "占位", fmt.Sprintf("%d", initialID)); err != nil {
		return nil, "", err
	}
	events404, err := c.Check404AndErrorEvent()
	if err != nil {
		return nil, "", err
	}
	for _, e := range events404 {
		events = append(events, e)
	}
	lastID, _ := c.d.GetValue("showstart_initial_eventid")
	msg := errMsg + fmt.Sprintf("遍历开始ID：%d，遍历结束ID：%d，数据库存储ID：%d",
		c.initEventID, eventID-1, lastID)
	return events, msg, nil
}

func (c *ShowStart) Check404AndErrorEvent() ([]*utils.Event, error) {
	events := make([]*utils.Event, 0)
	eventStr, err := c.d.GetEventByValue(Evenet404)
	if err != nil {
		return nil, err
	}
	eventErrStr, err := c.d.GetEventByValue(EvenetErrorWhenRequest)
	if err != nil {
		return nil, err
	}
	eventStr = append(eventStr, eventErrStr...)
	eventInt := make([]int64, 0, len(eventStr))
	re := regexp.MustCompile(`\d+`)
	for _, e := range eventStr {
		match := re.FindString(e)
		eventID, err := strconv.ParseInt(match, 10, 64)
		if err != nil {
			return nil, err
		}
		eventInt = append(eventInt, eventID)
	}
	sort.Slice(eventInt, func(i, j int) bool {
		return eventInt[i] < eventInt[j]
	})
	if len(eventInt) > int(c.Max404CountToCheck) {
		eventInt = eventInt[len(eventInt)-int(c.Max404CountToCheck):]
		// eventInt = eventInt[:c.Max404CountToCheck]
	}
	// if len(eventInt) < int(c.Max404CountToCheck) {
	// 	firstItem := eventInt[0]
	// 	for i := int64(0); i < c.Max404CountToCheck-int64(len(eventInt)); i++ {
	// 		eventInt = append([]int64{firstItem - 1 - i}, eventInt...)
	// 	}
	// } else {
	// 	eventInt = eventInt[:c.Max404CountToCheck]
	// }
	for _, eventID := range eventInt {
		keyInDB := fmt.Sprintf("showstart_eventid_%d", eventID)
		e, err := c.requestEvent(eventURL(eventID), eventID)
		name := "未知"
		if e != nil {
			name = e.Name
		}
		if err != nil {
			if err == ErrorNotInterested {
				c.d.SetKey(keyInDB, name, EventNotInterested)
				// 正常标记到数据库
				continue
			}
			if err == Error404 {
				continue
			}
			c.d.SetKey(keyInDB, name, EvenetErrorWhenRequest)
			continue
		}
		c.d.SetKey(keyInDB, name, EventPushed)
		fillEventURL(eventID, e)
		events = append(events, e)
	}
	return events, nil
}

var Error404 = errors.New("404")
var ErrorNotInterested = errors.New("event is not interested")

var maxRetryTimes = 5

func (c *ShowStart) requestEvent(url string, eventID int64) (*utils.Event, error) {
	var retryTimes = 0
	var res *http.Response
	for {
		time.Sleep(time.Second)
		retryTimes++
		var err error
		res, err = http.Get(url)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode == 200 {
			break
		}
		if res.StatusCode != 200 {
			if res.StatusCode == 404 {
				return nil, Error404
			}
			if retryTimes >= maxRetryTimes {
				return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
			}
		}
		if retryTimes >= maxRetryTimes {
			break
		}
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("new document of query error: %v", err)
	}
	prefix := "#__layout > section > main > div > div.product > div > div.describe > "
	site := doc.Find(prefix + "p:nth-child(4) > a").Text()
	time := strings.TrimPrefix(doc.Find(prefix+"p:nth-child(2)").Text(), "演出时间：")
	var found = false
	for _, v := range c.city {
		if strings.HasPrefix(site, v) {
			labels := doc.Find(prefix + "div.label").Text()
			for _, v := range c.includeLabels {
				if strings.Contains(labels, v) {
					found = true
				}
			}
		}
	}
	for _, v := range c.otherCityInAfternoon {
		if !found && strings.HasPrefix(site, v) {
			labels := doc.Find(prefix + "div.label").Text()
			for _, v := range c.includeLabels {
				if strings.Contains(labels, v) {
					// 对于市外演出，只关注下午开场的
					if strings.Contains(time, "12:00") || strings.Contains(time, "12:30") ||
						strings.Contains(time, "13:00") || strings.Contains(time, "13:30") ||
						strings.Contains(time, "14:00") || strings.Contains(time, "14:30") ||
						strings.Contains(time, "15:00") || strings.Contains(time, "15:30") ||
						strings.Contains(time, "16:00") || strings.Contains(time, "16:30") ||
						strings.Contains(time, "17:00") || strings.Contains(time, "17:30") ||
						strings.Contains(time, "18:00") || strings.Contains(time, "18:30") {
						found = true
					}
				}
			}
		}
	}
	if !found {
		return &utils.Event{
			Time: time}, ErrorNotInterested
	}
	title := doc.Find(prefix + "div.title").Text()
	if strings.Contains(title, "夜猫俱乐部") || strings.Contains(title, "【JZ Club】") {
		return &utils.Event{
			Time: time}, ErrorNotInterested
	}
	artist := doc.Find(prefix + "p:nth-child(3) > a").Text()
	price := doc.Find("#__layout > section > main > div > div.product > div > div.buy > div.price-tags").Text()
	return &utils.Event{
		Name:   title,
		Time:   time,
		Artist: artist,
		Site:   site,
		Price:  price}, nil
}
