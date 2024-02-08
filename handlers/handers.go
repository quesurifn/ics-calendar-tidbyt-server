package handlers

import (
	c "github.com/quesurifn/ics-calendar-tidbyt-server/calendar"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger   *zap.Logger
	Calendar *c.Calendar
}
