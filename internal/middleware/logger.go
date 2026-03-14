package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Logger returns a Fiber middleware that logs every request using zerolog —
// same style as ayws-gateway.
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start).Seconds()

		status := c.Response().StatusCode()
		evt := log.Info()
		if status >= 500 {
			evt = log.Error()
		} else if status >= 400 {
			evt = log.Warn()
		}

		evt.
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Float64("latency", latency).
			Str("ip", c.IP()).
			Str("user_agent", c.Get(fiber.HeaderUserAgent)).
			Msg("request")

		return err
	}
}
