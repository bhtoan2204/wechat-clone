// CODE_GENERATOR: response
package out

type CreateMessageResponse struct {
	ID        string `json:"id"`
	RoomID    string `json:"room_id"`
	SenderID  string `json:"sender_id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}
