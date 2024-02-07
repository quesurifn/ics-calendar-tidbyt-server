package types

type IcsRequest struct {
	ICSUrl string `json:"icsUrl"`
	TZ     string `json:"tz"`
}

type ICSResponse struct {
	EventName      string  `json:"eventName"`
	EventStartTime float64 `json:"eventStart"`
	EventEndTime   float64 `json:"eventEnd"`
	EventLocation  *string `json:"eventLocation"`
}
