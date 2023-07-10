package showstart

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"show-live/pkg/db"
	"show-live/pkg/log"
	"show-live/utils"
)

type ShowStart struct {
	d                db.DB
	includeLabels    []string
	city             []string
	initEventID      int64
	MaxNotFoundCount int64
}

func NewShowStartGeter(d db.DB, tags []string, city []string, initEventID, MaxNotFoundCount int64) *ShowStart {
	return &ShowStart{
		d:                d,
		includeLabels:    tags,
		city:             city,
		initEventID:      initEventID,
		MaxNotFoundCount: MaxNotFoundCount,
	}
}

const EventPushed = int64(1)
const EventNotInterested = int64(2)
const Evenet404 = int64(3)

func eventKeyInDB(id int64) string {
	return fmt.Sprintf("showstart_eventid_%d", id)
}

func eventURL(id int64) string {
	url := "https://www.showstart.com/event"
	return fmt.Sprintf("%s/%d", url, id)
}

func filEventURL(id int64, e *utils.Event) {
	e.WebURL = fmt.Sprintf("https://www.showstart.com/event/%d", id)
	e.WebViewURL = fmt.Sprintf("https://wap.showstart.com/pages/activity/detail/detail?activityId=%d", id)
}

func (c *ShowStart) GetEventsToNotify() ([]*utils.Event, string, error) {
	initialEventID, err := c.d.GetValue(fmt.Sprintf("showstart_initial_eventid"))
	if err != nil {
		return nil, "", err
	}
	if initialEventID.(int64) > c.initEventID {
		c.initEventID = initialEventID.(int64)
	}
	events := make([]*utils.Event, 0)
	eventID := c.initEventID - 1
	consistentNonexistEventCount := int64(0)
	for {
		eventID++
		if consistentNonexistEventCount == c.MaxNotFoundCount {
			break
		}
		keyInDB := eventKeyInDB(eventID)
		value, err := c.d.GetValue(keyInDB)
		if err != nil {
			log.Logger.Errorf("check if %s exists in db error %v", keyInDB, err)
			continue
		}
		if value == EventPushed || value == EventNotInterested {
			continue
		}
		e, err := c.requestEvent(eventURL(eventID), eventID)
		if err != nil {
			if err == ErrorNotInterested {
				consistentNonexistEventCount = 0
				c.d.SetKey(keyInDB, EventNotInterested)
				continue
			}
			if err == Error404 {
				consistentNonexistEventCount++
				c.d.SetKey(keyInDB, Evenet404)
				continue
			}
			return nil, "", fmt.Errorf("请求演出报错，ID：%d，错误：%v", eventID, err)
		}
		consistentNonexistEventCount = 0
		c.d.SetKey(keyInDB, EventPushed)
		filEventURL(eventID, e)
		events = append(events, e)
	}
	initialID := eventID - int64(consistentNonexistEventCount) - 1
	if err := c.d.SetKey("showstart_initial_eventid", initialID); err != nil {
		return nil, "", err
	}
	events404, err := c.Check404Event()
	if err != nil {
		return nil, "", err
	}
	for _, e := range events404 {
		events = append(events, e)
	}
	lastID, _ := c.d.GetValue("showstart_initial_eventid")
	msg := fmt.Sprintf("遍历开始ID：%d，遍历结束ID：%d，数据库存储ID：%d",
		c.initEventID, eventID-1, lastID)
	return events, msg, nil
}

func (c *ShowStart) Check404Event() ([]*utils.Event, error) {
	events := make([]*utils.Event, 0)
	eventStr, err := c.d.GetEventByValue(Evenet404)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\d+`)
	for _, e := range eventStr {
		match := re.FindString(e)
		eventID, err := strconv.ParseInt(match, 10, 64)
		if err != nil {
			return nil, err
		}
		keyInDB := fmt.Sprintf("showstart_eventid_%d", eventID)
		e, err := c.requestEvent(eventURL(eventID), eventID)
		if err != nil {
			if err == ErrorNotInterested {
				c.d.SetKey(keyInDB, EventNotInterested)
				// 正常标记到数据库
				continue
			}
			if err == Error404 {
				continue
			}
			return nil, err
		}
		c.d.SetKey(keyInDB, EventPushed)
		filEventURL(eventID, e)
		events = append(events, e)
	}
	return events, nil
}

var Error404 = errors.New("404")
var ErrorNotInterested = errors.New("event is not interested")

func (c *ShowStart) requestEvent(url string, eventID int64) (*utils.Event, error) {
	time.Sleep(3 * time.Second)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 404 {
			return nil, Error404
		}
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
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
	if !found {
		return &utils.Event{
			Time: time}, ErrorNotInterested
	}
	title := doc.Find(prefix + "div.title").Text()
	if strings.Contains(title, "夜猫俱乐部") {
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
