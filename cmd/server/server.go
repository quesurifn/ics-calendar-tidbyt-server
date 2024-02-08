package main

import (
	"errors"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	c "github.com/quesurifn/ics-calendar-tidbyt-server/calendar"
	h "github.com/quesurifn/ics-calendar-tidbyt-server/handlers"
	"github.com/quesurifn/ics-calendar-tidbyt-server/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cfg *config.Config

var serverCmd = &cobra.Command{
	Use:   "isc-srv",
	Short: "Run the ICS server",
	Run: func(cmd *cobra.Command, args []string) {
		app := fiber.New()
		logger, _ := zap.NewProduction()
		fiberLogger := fiberzap.New(fiberzap.Config{
			Logger: logger,
		})
		fiberLimiter := limiter.New(limiter.Config{
			Next: func(c *fiber.Ctx) bool {
				return c.IP() == "127.0.0.1"
			},
			Max:        20,
			Expiration: 30 * time.Second,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.Get("x-forwarded-for")
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{
					"error": "Too many requests",
				})
			},
		})

		app.Use(fiberLimiter)
		app.Use(fiberLogger)

		cal := c.Calendar{
			Logger: logger,
			TZMap: map[string]string{
				"Hawaii Standard Time":     "Pacific/Honolulu",
				"Alaskan Standard Time":    "America/Anchorage",
				"Alaskan Daylight Time":    "America/Anchorage",
				"SA Pacific Standard Time": "America/Bogota",
				"Pacific Standard Time":    "America/Los_Angeles",
				"Pacific Daylight Time":    "America/Los_Angeles",
				"Central Standard Time":    "America/Chicago",
				"Central Daylight Time":    "America/Chicago",
				"Mountain Standard Time":   "America/Denver",
				"Mountain Daylight Time":   "America/Denver",
				"Eastern Standard Time":    "America/New_York",
				"Eastern Daylight Time":    "America/New_York",
			},
		}
		h := h.Handlers{
			Logger:   logger,
			Calendar: &cal,
		}

		app.Get("/", h.RootHandler)
		app.Post("/ics/next-event", h.NextEventHandler)

		defer func() {
			err := logger.Sync()
			if err != nil && !errors.Is(err, syscall.ENOTTY) {
				logger.Fatal(err.Error())
			}
		}()

		log.Fatal(app.Listen(":3000"))
	},
}

func init() {
	cfg = config.New(&config.Settings{ENVPrefix: "ICS_SRV"})

	serverCmd.Flags().StringVarP(&appConfig.Port, "port", "p", appConfig.Port, "app server port")
	serverCmd.Flags().BoolVarP(&cfg.Debug, "debug", "d", cfg.Debug, "Debug Mode")
}

func main() {
	if err := cfg.Load(&appConfig, "config.yml"); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(-1)
	}

	if err := serverCmd.Execute(); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(-1)
	}
}
