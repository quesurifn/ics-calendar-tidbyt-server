package handlers

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func (h Handlers) RootHandler(c fiber.Ctx) error {
	h.Logger.Info("RootHandler", zap.String("ip", c.IP()))
	return c.SendString("Welcome to the Tidbyt ICS Server!")
}
