package main

import (
	"errors"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/quesurifn/tidbyt-ics-server/pkg/config"
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
			Next: func(c fiber.Ctx) bool {
				return c.IP() == "127.0.0.1"
			},
			Max:        20,
			Expiration: 30 * time.Second,
			KeyGenerator: func(c fiber.Ctx) string {
				return c.Get("x-forwarded-for")
			},
			LimitReached: func(c fiber.Ctx) error {
				return c.JSON(fiber.Map{
					"error": "Too many requests",
				})
			},
		})

		app.Use(fiberLimiter)
		app.Use(fiberLogger)

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
