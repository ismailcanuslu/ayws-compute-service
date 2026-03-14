package response

import "github.com/gofiber/fiber/v2"

// Envelope is the standard JSON envelope for all API responses.
type Envelope[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

// Meta holds pagination or extra information.
type Meta struct {
	Total  int `json:"total,omitempty"`
	Page   int `json:"page,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

// OK sends a 200 JSON response.
func OK[T any](c *fiber.Ctx, data T) error {
	return c.Status(fiber.StatusOK).JSON(Envelope[T]{Success: true, Data: data})
}

// Created sends a 201 JSON response.
func Created[T any](c *fiber.Ctx, data T) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope[T]{Success: true, Data: data})
}

// NoContent sends a 204 response with no body.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Fail sends an error JSON response with the given status code.
func Fail(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(Envelope[any]{Success: false, Error: msg})
}

// BadRequest sends a 400 error.
func BadRequest(c *fiber.Ctx, msg string) error { return Fail(c, fiber.StatusBadRequest, msg) }

// NotFound sends a 404 error.
func NotFound(c *fiber.Ctx, msg string) error { return Fail(c, fiber.StatusNotFound, msg) }

// Internal sends a 500 error.
func Internal(c *fiber.Ctx, msg string) error { return Fail(c, fiber.StatusInternalServerError, msg) }

// Unimplemented sends a 501 with a clear message.
func Unimplemented(c *fiber.Ctx) error {
	return Fail(c, fiber.StatusNotImplemented, "Bu özellik henüz yapılandırılmamış. Config dosyasını kontrol edin.")
}
