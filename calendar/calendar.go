package calendar

import (
	"strings"
	"time"

	"github.com/apognu/gocal"
	"github.com/go-resty/resty/v2"
	t "github.com/quesurifn/ics-calendar-tidbyt-server/types"
	"go.uber.org/zap"
)

type Calendar struct {
	Logger *zap.Logger
	TZMap  map[string]string
}

func (c Calendar) DownloadCalendar(url string) (string, error) {
	client := resty.New()

	resp, err := client.R().Get(url)
	if err != nil {
		return "", err
	}

	return resp.String(), err
}

func (c Calendar) ParseCalendar(data string, tz string) ([]t.Event, error) {
	gocal.SetTZMapper(func(s string) (*time.Location, error) {
		override := ""
		if val, ok := c.TZMap[s]; ok {
			override = val
		}
		if override != "" {
			loc, err := time.LoadLocation(override)
			if err != nil {
				c.Logger.Error("Error", zap.Any("err", err))
				return nil, err
			}
			return loc, nil
		}

		loc, err := time.LoadLocation(s)
		if err != nil {
			c.Logger.Error("Error", zap.Any("err", err))
			return nil, err
		}
		return loc, nil
	})

	usersLoc, err := time.LoadLocation(tz)
	if err != nil {
		c.Logger.Error("Error", zap.Any("err", err))
		return nil, err
	}

	parser := gocal.NewParser(strings.NewReader(data))
	start, end := time.Now().In(usersLoc), time.Now().AddDate(0, 0, 7).In(usersLoc)
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

	c.Logger.Info("ParseCalendar", zap.Any("events", events))

	return events, nil
}

func (c Calendar) NextEvent(events []t.Event) *t.Event {
	var next t.Event

	now := time.Now().Unix()

	// TODO: Google and others doesn't come back sorted where office does....
	next = events[0]

	fiveMinutesFromStart := next.StartTime - 5*60
	tenMinutesFromStart := next.StartTime - 10*60
	oneMinuteFromStart := next.StartTime - 60

	next.FiveMinuteWarning = now >= fiveMinutesFromStart && now < tenMinutesFromStart
	next.TenMinuteWarning = now >= tenMinutesFromStart && now < fiveMinutesFromStart
	next.OneMinuteWarning = now >= oneMinuteFromStart && now < tenMinutesFromStart
	next.InProgress = now >= next.StartTime

	c.Logger.Info("NextEvent", zap.Any("nextEvent", next))
	c.Logger.Info("Now", zap.Any("now", now))

	return &next
}
