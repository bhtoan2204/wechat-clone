package support

import (
	"time"

	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/utils"
)

func ToGetProfileResponse(account *entity.Account) *out.GetProfileResponse {
	if account == nil {
		return nil
	}

	return &out.GetProfileResponse{
		ID:                account.ID,
		DisplayName:       account.DisplayName,
		Email:             account.Email.Value(),
		Username:          utils.StringValue(account.Username),
		AvatarObjectKey:   utils.StringValue(account.AvatarObjectKey),
		Status:            account.Status.String(),
		EmailVerified:     account.EmailVerifiedAt != nil,
		EmailVerifiedAt:   utils.FormatOptionalTime(account.EmailVerifiedAt),
		LastLoginAt:       utils.FormatOptionalTime(account.LastLoginAt),
		PasswordChangedAt: utils.FormatOptionalTime(account.PasswordChangedAt),
		CreatedAt:         account.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func ToUpdateProfileResponse(account *entity.Account) *out.UpdateProfileResponse {
	if account == nil {
		return nil
	}

	return &out.UpdateProfileResponse{
		ID:                account.ID,
		DisplayName:       account.DisplayName,
		Email:             account.Email.Value(),
		Username:          utils.StringValue(account.Username),
		AvatarObjectKey:   utils.StringValue(account.AvatarObjectKey),
		Status:            account.Status.String(),
		EmailVerified:     account.EmailVerifiedAt != nil,
		EmailVerifiedAt:   utils.FormatOptionalTime(account.EmailVerifiedAt),
		LastLoginAt:       utils.FormatOptionalTime(account.LastLoginAt),
		PasswordChangedAt: utils.FormatOptionalTime(account.PasswordChangedAt),
		CreatedAt:         account.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
