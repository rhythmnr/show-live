package zhengzai

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"show-live/pkg/db"
	"show-live/pkg/http"
	"show-live/pkg/log"
)

type ZhengZaiGetter struct {
	d           db.DB
	url, adCode string
}

func NewZhengZaiGetterGetter(d db.DB, url, adCode string) *ZhengZaiGetter {
	return &ZhengZaiGetter{
		d:      d,
		url:    url,
		adCode: adCode,
	}
}

func (c ZhengZaiGetter) GetEventsToNotify() ([]string, error) {
	url := fmt.Sprintf("%s/kylin/performance/localList?adCode=%s&days=0&orderBy=timeStart&sort=ASC",
		c.url, c.adCode)
	var resp Resp
	if err := http.Request(url, "GET", nil, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "0" {
		msg := "invalid code resp from zhengzai"
		log.Logger.Error(msg)
		return nil, errors.New(msg)
	}
	nameToBeginTime := make(map[string]string, 0)
	for _, v := range resp.Data.List {
		if strconv.FormatInt(v.CityID, 10) != c.adCode {
			continue
		}
		title := v.Title + "___地点：" + v.FieldName
		nameToBeginTime[title] = v.TimeEnd
	}
	result := make([]string, 0, len(nameToBeginTime))
	for k, v := range nameToBeginTime {
		keyInDB := "zhengzai" + k
		exists, err := c.d.Exists(keyInDB)
		if err != nil {
			log.Logger.Errorf("check if %s exists in db error", keyInDB, err)
		}
		if !exists {
			result = append(result, fmt.Sprintf("%s___开始时间：%s", k, v))
			t, _ := time.Parse("2006-01-02 15:04:05", v)
			c.d.SetKey(keyInDB, t)
		}
	}
	return result, nil
}

type Resp struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"`
	Data    struct {
		Total     int `json:"total"`
		IsNative  int `json:"is_native"`
		Recommend int `json:"recommend"`
		List      []struct {
			Mid                 int         `json:"mid"`
			PerformancesID      string      `json:"performancesId"`
			Title               string      `json:"title"`
			ImgPoster           string      `json:"imgPoster"`
			PayCountdownMinute  int         `json:"payCountdownMinute"`
			ApprovalURL         string      `json:"approvalUrl"`
			Type                int         `json:"type"`
			TimeStart           string      `json:"timeStart"`
			TimeEnd             string      `json:"timeEnd"`
			StopSellTime        string      `json:"stopSellTime"`
			Price               string      `json:"price"`
			SellTime            string      `json:"sellTime"`
			SellMemberTime      string      `json:"sellMemberTime"`
			CityID              int64       `json:"cityId"`
			CityName            string      `json:"cityName"`
			FieldID             string      `json:"fieldId"`
			FieldName           string      `json:"fieldName"`
			Longitude           string      `json:"longitude"`
			Latitude            string      `json:"latitude"`
			DiffDistance        interface{} `json:"diffDistance"`
			ProjectID           string      `json:"projectId"`
			RoadShowID          string      `json:"roadShowId"`
			Details             interface{} `json:"details"`
			NoticeImage         interface{} `json:"noticeImage"`
			IsRecommend         int         `json:"isRecommend"`
			AppStatus           int         `json:"appStatus"`
			StatusSell          int         `json:"statusSell"`
			IsMember            int         `json:"isMember"`
			IsLackRegister      int         `json:"isLackRegister"`
			IsTrueName          int         `json:"isTrueName"`
			LimitCount          int         `json:"limitCount"`
			IDCount             int         `json:"idCount"`
			LimitCountMember    int         `json:"limitCountMember"`
			IsExclusive         int         `json:"isExclusive"`
			IsDiscount          int         `json:"isDiscount"`
			IsAdvance           int         `json:"isAdvance"`
			SysDamai            int         `json:"sysDamai"`
			Message             string      `json:"message"`
			Notice              string      `json:"notice"`
			IsShow              int         `json:"isShow"`
			TicketTimeList      interface{} `json:"ticketTimeList"`
			IsAgent             interface{} `json:"isAgent"`
			AgentName           interface{} `json:"agentName"`
			State               interface{} `json:"state"`
			CreatedAt           string      `json:"createdAt"`
			IsCanRefund         int         `json:"isCanRefund"`
			IsOpenRefundPresent int         `json:"isOpenRefundPresent"`
			RefundOpenTime      string      `json:"refundOpenTime"`
			RefundCloseTime     string      `json:"refundCloseTime"`
			IsTransfer          int         `json:"isTransfer"`
			TransferStartTime   interface{} `json:"transferStartTime"`
			TransferEndTime     interface{} `json:"transferEndTime"`
			IsRefundPoundage    int         `json:"isRefundPoundage"`
			IsRefundVoucher     int         `json:"isRefundVoucher"`
			IsRefundExpress     int         `json:"isRefundExpress"`
			IsBackPaperTicket   int         `json:"isBackPaperTicket"`
			IsRefundExpressNew  int         `json:"isRefundExpressNew"`
			AuditStatus         int         `json:"auditStatus"`
			RejectTxt           string      `json:"rejectTxt"`
			MerchantID          string      `json:"merchantId"`
			FieldAuditStatus    int         `json:"fieldAuditStatus"`
		} `json:"list"`
	} `json:"data"`
	Success bool `json:"success"`
}
