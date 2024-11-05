package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	e, err := requestEventList(30, 20, 21)
	fmt.Println(len(e))
	fmt.Println(err)
}

func requestEventList(page, pageSize, cityCode int) ([]int64, error) {
	url := fmt.Sprintf("https://www.showstart.com/event/list?pageNo=%d&pageSize=%d&cityCode=%d",
		page, pageSize, cityCode)
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
				// TODO: log...............
				return
			}
			re := regexp.MustCompile(`/event/(\d+)`)
			// 提取匹配的数字部分
			matches := re.FindStringSubmatch(href)
			if len(matches) > 1 {
				r, err := strconv.ParseInt(matches[1], 10, 64)
				if err != nil {
					// TODO: log...............
				} else {
					eventIDs = append(eventIDs, r)
				}
			} else {
				// TODO: log...............
			}
		}
	})
	return eventIDs, nil

}
