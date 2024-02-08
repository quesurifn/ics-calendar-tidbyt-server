package types

type Event struct {
	Name      string
	StartTime int64
	EndTime   int64
	Location  *string
}

type BaseResponse[t any] struct {
	Data    t      `json:"data"`
	Message string `json:"message"`
}

type IcsRequest struct {
	ICSUrl string `json:"icsUrl"`
	TZ     string `json:"tz"`
}

type IcsResponse struct {
	EventName      string  `json:"eventName"`
	EventStartTime int64   `json:"eventStart"`
	EventEndTime   int64   `json:"eventEnd"`
	EventLocation  *string `json:"eventLocation"`
}
