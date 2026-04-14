// CODE_GENERATOR - do not edit: response
package out

type ListRoomsResponse struct {
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Rooms []RoomResponse `json:"rooms"`
}

type RoomResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RoomType    string `json:"room_type"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
