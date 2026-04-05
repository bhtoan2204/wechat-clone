package types

type RoomResult struct {
	ID          string
	Name        string
	Description string
	RoomType    string
	OwnerID     string
	CreatedAt   string
	UpdatedAt   string
}

type ListRoomsResult struct {
	Page  int
	Limit int
	Rooms []RoomResult
}
