package simullink

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"show-live/pkg/db"
	"show-live/pkg/http"
	"show-live/pkg/log"
)

type SimullinkGetter struct {
	d             db.DB
	tags          []string
	url, cityCode string
}

func NewSimullinkGetter(d db.DB, tags []string, url, cityCode string) *SimullinkGetter {
	return &SimullinkGetter{
		d:        d,
		tags:     tags,
		url:      url,
		cityCode: cityCode,
	}
}

type Params struct {
	CityCode      string `json:"cityCode"`
	FromTimestamp int64  `json:"fromTimestamp"`
	Orderby       string `json:"orderby"`
	ToTimestamp   int64  `json:"toTimestamp"`
}

type PageInfo struct {
	Size        int    `json:"size"`
	SortField   string `json:"sortField"`
	HasNextPage int    `json:"hasNextPage"`
	NextPageVal string `json:"nextPageVal"`
	Direction   string `json:"direction"`
}

func (c SimullinkGetter) GetEventsToNotify() ([]string, error) {
	now := time.Now()
	beginOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := beginOfDay.AddDate(1, 0, 0).Add(24 * time.Hour).Add(-time.Second)
	req := map[string]interface{}{
		"params": Params{
			CityCode:      "156310000",
			FromTimestamp: beginOfDay.Unix() * 1000,
			Orderby:       "RECENTPUBLISH",
			ToTimestamp:   end.Unix() * 1000,
		},
	}
	var resp Resp
	if err := http.Request(c.url+"/bigsearch/series", "POST", req, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		msg := "invalid code resp from simullink"
		log.Logger.Error(msg)
		return nil, errors.New(msg)
	}
	nameToBeginTime := make(map[string]int64, 0)
	for {
		for _, d := range resp.Data.Items {
			title := d.UI.Title.Text
			if len(c.tags) != 0 {
				foundTag := false
				tag := d.UI.Line1.Text
				for _, v := range c.tags {
					if strings.Contains(tag, v) {
						foundTag = true
					}
				}
				if !foundTag {
					continue
				}
			}
			if len(d.Extra.AllInstances) != 0 {
				venueName := d.Extra.AllInstances[0].VenueName
				title = title + "___地点：" + venueName
			}
			nameToBeginTime[title] = d.Extra.Series.BeginTime
		}
		if resp.Data.PageInfo.HasNextPage == 0 {
			break
		}
		req["pageInfo"] = PageInfo{
			Size:        resp.Data.PageInfo.Size,
			SortField:   resp.Data.PageInfo.SortField,
			HasNextPage: resp.Data.PageInfo.HasNextPage,
			NextPageVal: resp.Data.PageInfo.NextPageVal,
			Direction:   resp.Data.PageInfo.Direction,
		}
		resp = Resp{}
		if err := http.Request(c.url+"/bigsearch/series", "POST", req, &resp); err != nil {
			return nil, err
		}
		if resp.Code != 200 {
			msg := "invalid code resp from simullink"
			log.Logger.Error(msg)
			return nil, errors.New(msg)
		}
	}
	result := make([]string, 0, len(nameToBeginTime))
	for k, v := range nameToBeginTime {
		keyInDB := "simullink" + k
		exists, err := c.d.Exists(keyInDB)
		if err != nil {
			log.Logger.Errorf("check if %s exists in db error", keyInDB, err)
		}
		if !exists {
			result = append(result, fmt.Sprintf("%s___开始时间：%s", k, time.Unix(v/1000, 0).Format("2006-01-02 15:04:05")))
			c.d.SetKey(keyInDB, time.Unix(v, 0))
		}
	}
	return result, nil
}

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items []struct {
			UI struct {
				Title struct {
					Text string `json:"text"`
				} `json:"title"`
				Line1 struct {
					Text string `json:"text"`
				} `json:"line1"`
				Line2 struct {
					Text string `json:"text"`
				} `json:"line2"`
				Line3 struct {
					Text string `json:"text"`
				} `json:"line3"`
				Thumbnails []struct {
					URL         string `json:"url"`
					ResourceURL string `json:"resourceUrl"`
					Type        string `json:"type"`
				} `json:"thumbnails"`
			} `json:"ui"`
			NavigationSuite struct {
				ViewNavigation struct {
					Event       string `json:"event"`
					Action      string `json:"action"`
					ToPage      string `json:"toPage"`
					ContentID   string `json:"contentId"`
					ContentType string `json:"contentType"`
				} `json:"viewNavigation"`
			} `json:"navigationSuite"`
			Extra struct {
				Tickets struct {
				} `json:"tickets"`
				AllInstances []struct {
					ActivityID  string `json:"activityId"`
					BeginTime   int64  `json:"beginTime"`
					EndTime     int64  `json:"endTime"`
					City        string `json:"city"`
					VenueName   string `json:"venueName"`
					TimeStatus  string `json:"timeStatus"`
					VenueStatus string `json:"venueStatus"`
				} `json:"allInstances"`
				HitIndex int `json:"hitIndex"`
				Activity struct {
					SimulCoupon int `json:"simulCoupon"`
					SimulTicket int `json:"simulTicket"`
				} `json:"activity"`
				Series struct {
					ID        string `json:"id"`
					BeginTime int64  `json:"beginTime"`
					EndTime   int64  `json:"endTime"`
				} `json:"series"`
				Consulter struct {
					ID          string `json:"id"`
					AvatarURL   string `json:"avatarUrl"`
					Nickname    string `json:"nickname"`
					VipLevel    int    `json:"vipLevel"`
					VipTitle    string `json:"vipTitle"`
					Type        int    `json:"type"`
					EntityID    string `json:"entityId"`
					SignDisplay int    `json:"signDisplay"`
				} `json:"consulter"`
			} `json:"extra,omitempty"`
			Template struct {
				Type string `json:"type"`
			} `json:"template"`
		} `json:"items"`
		PageInfo struct {
			Size        int    `json:"size"`
			CurPageVal  string `json:"curPageVal"`
			NextPageVal string `json:"nextPageVal"`
			SortField   string `json:"sortField"`
			Direction   string `json:"direction"`
			HasNextPage int    `json:"hasNextPage"`
		} `json:"pageInfo"`
		Params struct {
			Keywords      string `json:"keywords"`
			TagID         string `json:"tagId"`
			CityCode      string `json:"cityCode"`
			FromTimestamp int64  `json:"fromTimestamp"`
			Orderby       string `json:"orderby"`
			SimulTicket   int    `json:"simulTicket"`
			ToTimestamp   int64  `json:"toTimestamp"`
		} `json:"params"`
	} `json:"data"`
}
