package service

import (
	"context"
	"errors"
	"strings"

	"github.com/awlsring/terraform-provider-headscale/internal/gen/client/headscale_service"
	"github.com/awlsring/terraform-provider-headscale/internal/gen/models"
	"github.com/go-openapi/strfmt"
)

func (h *HeadscaleService) ListUsers(ctx context.Context) ([]*models.V1User, error) {
	request := headscale_service.NewHeadscaleServiceListUsersParams()
	request.SetContext(ctx)

	resp, err := h.client.HeadscaleService.HeadscaleServiceListUsers(request)
	if err != nil {
		return nil, handleRequestError(err)
	}

	err = resp.Payload.Validate(strfmt.Default)
	if err != nil {
		return nil, err
	}

	return resp.Payload.Users, nil
}

// FindExternalUser looks up an OIDC user by provider_id (preferred) or name.
// CLI users (no provider_id) with a matching name are reported via cliConflict.
func (h *HeadscaleService) FindExternalUser(ctx context.Context, name, providerID string) (user *models.V1User, cliConflict bool, err error) {
	users, err := h.ListUsers(ctx)
	if err != nil {
		return nil, false, err
	}

	normalizedProviderID := strings.TrimSpace(providerID)
	normalizedName := strings.TrimSpace(strings.ToLower(name))

	if normalizedProviderID != "" {
		for _, u := range users {
			if u == nil {
				continue
			}
			if strings.TrimSpace(u.ProviderID) == normalizedProviderID {
				return u, false, nil
			}
		}
	}

	if normalizedName != "" {
		var oidcMatch *models.V1User
		for _, u := range users {
			if u == nil {
				continue
			}
			if strings.TrimSpace(strings.ToLower(u.Name)) != normalizedName {
				continue
			}
			if strings.TrimSpace(u.ProviderID) != "" || strings.EqualFold(strings.TrimSpace(u.Provider), "oidc") {
				if oidcMatch != nil {
					return nil, false, errors.New("multiple OIDC users found matching name")
				}
				oidcMatch = u
				continue
			}
			cliConflict = true
		}
		if oidcMatch != nil {
			return oidcMatch, false, nil
		}
	}

	return nil, cliConflict, nil
}

type GetUserInput struct {
	Name  *string
	ID    *string
	Email *string
}

func (h *HeadscaleService) GetUser(ctx context.Context, input GetUserInput) (*models.V1User, error) {
	request := headscale_service.NewHeadscaleServiceListUsersParams().
		WithContext(ctx).
		WithName(input.Name).
		WithID(input.ID).
		WithEmail(input.Email)

	resp, err := h.client.HeadscaleService.HeadscaleServiceListUsers(request)
	if err != nil {
		return nil, handleRequestError(err)
	}

	err = resp.Payload.Validate(strfmt.Default)
	if err != nil {
		return nil, err
	}

	if len(resp.Payload.Users) < 1 {
		return nil, errors.New("no user found matching criteria")
	}
	if len(resp.Payload.Users) > 1 {
		return nil, errors.New("multiple users found matching criteria")
	}

	return resp.Payload.Users[0], nil
}

type CreateUserInput struct {
	Name        string
	Email       string
	DisplayName string
	PictureURL  string
	ProviderID  string
}

func (h *HeadscaleService) CreateUser(ctx context.Context, input CreateUserInput) (*models.V1User, error) {
	request := headscale_service.NewHeadscaleServiceCreateUserParams()
	request.SetContext(ctx)
	request.SetBody(&models.V1CreateUserRequest{
		Name:        input.Name,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		PictureURL:  input.PictureURL,
		ProviderID:  input.ProviderID,
	})

	resp, err := h.client.HeadscaleService.HeadscaleServiceCreateUser(request)
	if err != nil {
		return nil, handleRequestError(err)
	}

	err = resp.Payload.Validate(strfmt.Default)
	if err != nil {
		return nil, err
	}

	return resp.Payload.User, nil
}

func (h *HeadscaleService) DeleteUser(ctx context.Context, userId string) error {
	request := headscale_service.NewHeadscaleServiceDeleteUserParams().
		WithContext(ctx).
		WithID(userId)

	_, err := h.client.HeadscaleService.HeadscaleServiceDeleteUser(request)
	if err != nil {
		return handleRequestError(err)
	}
	return nil
}

func (h *HeadscaleService) RenameUser(ctx context.Context, name string, userId string) (*models.V1User, error) {
	request := headscale_service.NewHeadscaleServiceRenameUserParams().
		WithContext(ctx).
		WithNewName(name).
		WithOldID(userId)

	resp, err := h.client.HeadscaleService.HeadscaleServiceRenameUser(request)
	if err != nil {
		return nil, handleRequestError(err)
	}

	err = resp.Payload.Validate(strfmt.Default)
	if err != nil {
		return nil, err
	}

	return resp.Payload.User, nil
}
