package handlers

import (
	"github.com/gofiber/fiber/v2"
	t "github.com/quesurifn/ics-calendar-tidbyt-server/types"
	"go.uber.org/zap"
)

func (h Handlers) NextEventHandler(c *fiber.Ctx) error {
	var icsRequest t.IcsRequest

	if err := c.BodyParser(&icsRequest); err != nil {
		return c.Status(400).SendString(err.Error())
	}

	h.Logger.Info("NextEventHandler", zap.String("url", icsRequest.ICSUrl))

	calString, err := h.Calendar.DownloadCalendar(icsRequest.ICSUrl)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}

	h.Logger.Info("NextEventHandler", zap.String("calString", calString))

	events, err := h.Calendar.ParseCalendar(calString, icsRequest.TZ)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}

	h.Logger.Info("NextEventHandler", zap.Any("events", events))

	nextEvent := h.Calendar.NextEvent(events)
	if nextEvent == nil {
		return c.Status(404).SendString("No upcoming events")
	}

	h.Logger.Info("NextEventHandler", zap.Any("nextEvent", nextEvent))

	return c.JSON(nextEvent)
}
