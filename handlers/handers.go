package handlers

import "github.com/gofiber/fiber/v3"

func RootHandler(c fiber.Ctx) error {
	return c.SendString("Welcome to the Tidbyt ICS Server!")
}

func IcsHandler(c fiber.Ctx) error {
	return c.SendString("ICS")
}
