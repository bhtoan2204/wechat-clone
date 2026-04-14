// CODE_GENERATOR - do not edit: response
package out

type SearchUsersResponse struct {
	Total  int64            `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Items  []SearchUserItem `json:"items"`
}

type SearchUserItem struct {
	ID              string `json:"id"`
	DisplayName     string `json:"display_name"`
	Username        string `json:"username"`
	AvatarObjectKey string `json:"avatar_object_key"`
	Status          string `json:"status"`
	EmailVerified   bool   `json:"email_verified"`
}
