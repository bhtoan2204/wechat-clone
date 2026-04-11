// CODE_GENERATOR: application-handler
package query

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	"go-socket/core/modules/account/domain/entity"
	repos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	"strings"

	"github.com/samber/lo"
)

type searchUsersHandler struct {
	accountRepos repos.AccountRepository
}

func NewSearchUsers(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	services service.Services,
) cqrs.Handler[*in.SearchUsersRequest, *out.SearchUsersResponse] {
	return &searchUsersHandler{
		accountRepos: baseRepo.AccountRepository(),
	}
}

func (u *searchUsersHandler) Handle(ctx context.Context, req *in.SearchUsersRequest) (*out.SearchUsersResponse, error) {
	if req == nil {
		return nil, stackErr.Error(fmt.Errorf("request is required"))
	}

	q := strings.TrimSpace(req.Q)
	if q == "" {
		return nil, stackErr.Error(fmt.Errorf("q is required"))
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	accounts, total, err := u.accountRepos.SearchUsers(ctx, q, limit, offset)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("search users: %w", err))
	}

	return &out.SearchUsersResponse{
		Items: lo.Map(accounts, func(account *entity.Account, _ int) out.SearchUserItem {
			return out.SearchUserItem{
				ID:          account.ID,
				DisplayName: account.DisplayName,
				Username: func(data *string) string {
					return lo.Ternary(data != nil, *data, "")
				}(account.Username),
				AvatarObjectKey: func(data *string) string {
					return lo.Ternary(data != nil, *data, "")
				}(account.AvatarObjectKey),
				Status:        account.Status.String(),
				EmailVerified: account.EmailVerifiedAt != nil,
			}
		}),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
