package calendar

import (
	"strings"
	"time"

	"github.com/apognu/gocal"
	"github.com/go-resty/resty/v2"
	t "github.com/quesurifn/ics-calendar-tidbyt-server/types"
)

type Calendar struct {
}

func (c Calendar) DownloadCalendar(url string) (string, error) {
	client := resty.New()

	resp, err := client.R().Get(url)
	if err != nil {
		return "", err
	}

	return resp.String(), err
}

func (c Calendar) ParseCalendar(data string) ([]t.Event, error) {
	start, end := time.Now(), time.Now().Add(12*30*24*time.Hour)

	parser := gocal.NewParser(strings.NewReader(data))
	parser.Start, parser.End = &start, &end
	parser.Parse()

	var events []t.Event
	for _, e := range parser.Events {
		events = append(events, t.Event{
			Name:      e.Summary,
			StartTime: e.Start.Unix(),
			EndTime:   e.End.Unix(),
			Location:  &e.Location,
		})
	}

	return events, nil
}

func (c Calendar) NextEvent(events []t.Event) *t.Event {
	var next t.Event

	now := time.Now().Unix()

	for _, e := range events {
		if e.StartTime > now {
			next = e
		}
	}

	return &next
}
