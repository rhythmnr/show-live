package showstart

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"show-live/pkg/db"
	"show-live/pkg/log"
	"show-live/utils"
)

type ShowStart struct {
	d                    db.DB
	includeLabels        []string
	cityCode             []int
	otherCityInAfternoon []string
	MaxNotFoundCount     int64
	Max404CountToCheck   int64
}

func NewShowStartGeter(d db.DB, tags []string, city []int) *ShowStart {
	return &ShowStart{
		d:             d,
		includeLabels: tags,
		cityCode:      city,
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

func (c *ShowStart) GetEventsToNotify() ([]*utils.Event, error) {
	pageSize := 20
	events := make([]*utils.Event, 0)
	page := 0
	var errMsg string
	fmt.Println(".........c.cityCode.......", len(c.cityCode))
	for _, city := range c.cityCode {
		for {
			page++
			eventIDs, err := c.requestEventList(page, pageSize, city)
			if err != nil {
				log.Logger.Errorf("请求城市 %d 的第 %d 出错 %v", city, page, err)
				continue
			}
			if len(eventIDs) == 0 {
				break
			}
			for _, eventID := range eventIDs {
				keyInDB := eventKeyInDB(eventID)
				value, err := c.d.GetValue(keyInDB)
				if err != nil {
					log.Logger.Errorf("检查键 %s 是否在数据库中存在时出错 %v", keyInDB, err)
					continue
				}

				if value == EventPushed || value == EventNotInterested {
					continue
				}
				e, err := c.requestEvent(eventURL(eventID))
				name := "未知"
				if e != nil {
					name = e.Name
				}
				if err != nil {
					if err == ErrorNotInterested {
						c.d.SetKey(keyInDB, name, EventNotInterested)
						continue
					}
					if err == Error404 {
						c.d.SetKey(keyInDB, name, Evenet404)
						continue
					}
					// 这个部分在出错的时候，返回错误内容，并在数据库里将活动标记为出错
					errMsg += fmt.Sprintf("请求演出报错，ID：%d，错误：%v\n", eventID, err)
					c.d.SetKey(keyInDB, name, EvenetErrorWhenRequest)
					continue
				}
				c.d.SetKey(keyInDB, name, EventPushed)
				fillEventURL(eventID, e)
				events = append(events, e)
			}

		}
	}

	return events, nil
}

var Error404 = errors.New("404")
var ErrorNotInterested = errors.New("event is not interested")

var maxRetryTimes = 5

func (c *ShowStart) requestEvent(url string) (*utils.Event, error) {
	var retryTimes = 0
	var res *http.Response
	for {
		// time.Sleep(time.Second)
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
	labels := doc.Find(prefix + "div.label").Text()
	for _, v := range c.includeLabels {
		if strings.Contains(labels, v) {
			found = true
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

func (c *ShowStart) requestEventList(page, pageSize, cityCode int) ([]int64, error) {
	url := fmt.Sprintf("https://www.showstart.com/event/list?pageNo=%d&pageSize=%d&cityCode=%d",
		page, pageSize, cityCode)
	fmt.Println("..........................", url)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%s return %d code", url, res.StatusCode))
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("new document of query error: %v", err)
	}
	eventIDs := []int64{}
	doc.Find("#__layout > section > main > div > div.list-box.clearfix").Children().Each(func(i int, s *goquery.Selection) {
		// 检查是否是 a 标签
		if goquery.NodeName(s) == "a" {
			// 获取 href 属性
			href, exists := s.Attr("href")
			if !exists {
				log.Logger.Errorf("%s中的第 %d 个元素的href不存在", url, i)
				return
			}
			re := regexp.MustCompile(`/event/(\d+)`)
			// 提取匹配的数字部分
			matches := re.FindStringSubmatch(href)
			if len(matches) > 1 {
				r, err := strconv.ParseInt(matches[1], 10, 64)
				if err != nil {
					log.Logger.Errorf("转换 %s 为 int 失败 %v", matches[1], err)
				} else {
					eventIDs = append(eventIDs, r)
				}
			} else {
				log.Logger.Errorf("正则匹配 %s 不符合预期，找不到活动ID", href)
			}
		}
	})
	return eventIDs, nil
}
