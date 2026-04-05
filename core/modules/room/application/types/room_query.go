package types

type GetRoomQuery struct {
	ID string
}

type ListRoomsQuery struct {
	Page  int
	Limit int
}
