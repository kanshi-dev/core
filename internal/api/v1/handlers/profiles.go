package handlers

import (
	"errors"
	"mime"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/service"
)

func CreateProfile(svc *service.ProfilesService) fiber.Handler {
	return func(c fiber.Ctx) error {
		var in service.ProfileCaptureInput
		if err := c.Bind().Body(&in); err != nil {
			return badRequest(c, errors.New("invalid request body"))
		}
		capture, err := svc.Create(c.Context(), c.Params("agentId"), in)
		if err != nil {
			return profileError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusCreated, "success", capture)
	}
}

func ListProfiles(svc *service.ProfilesService) fiber.Handler {
	return func(c fiber.Ctx) error {
		limit := 50
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return badRequest(c, errors.New("invalid limit"))
			}
			limit = int(parsed)
		}
		captures, err := svc.List(c.Context(), c.Params("agentId"), int32(limit))
		if err != nil {
			return profileError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", captures)
	}
}

func GetProfile(svc *service.ProfilesService) fiber.Handler {
	return func(c fiber.Ctx) error {
		capture, err := svc.Get(c.Context(), c.Params("id"))
		if err != nil {
			return profileError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", capture)
	}
}

func GetProfileFlamegraph(svc *service.ProfilesService) fiber.Handler {
	return func(c fiber.Ctx) error {
		graph, err := svc.Flamegraph(c.Context(), c.Params("id"), c.Query("sampleType"))
		if err != nil {
			return profileError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", graph)
	}
}

func DownloadProfile(svc *service.ProfilesService) fiber.Handler {
	return func(c fiber.Ctx) error {
		artifact, err := svc.Download(c.Context(), c.Params("id"))
		if err != nil {
			return profileError(c, err)
		}
		c.Set(fiber.HeaderContentDisposition, mime.FormatMediaType("attachment", map[string]string{"filename": artifact.Filename}))
		c.Set(fiber.HeaderContentType, artifact.ContentType)
		return c.Send(artifact.Data)
	}
}

func profileError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidProfile):
		return badRequest(c, err)
	case errors.Is(err, service.ErrTargetNotFound), errors.Is(err, service.ErrProfileNotFound):
		return response.CustomResponse(c, fiber.StatusNotFound, "not found", err.Error())
	case errors.Is(err, service.ErrProfileConflict):
		return response.CustomResponse(c, fiber.StatusConflict, "conflict", err.Error())
	default:
		return response.CustomResponse(c, fiber.StatusInternalServerError, "profile operation failed", err.Error())
	}
}
