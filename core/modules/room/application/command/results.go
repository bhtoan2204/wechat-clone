package command

const (
	CommandStatusCreated       = "created"
	CommandStatusUpdated       = "updated"
	CommandStatusDeleted       = "deleted"
	CommandStatusAlreadyExists = "already_exists"
	CommandStatusNoop          = "noop"
)

func commandStatus(changed bool) string {
	if changed {
		return CommandStatusUpdated
	}
	return CommandStatusNoop
}
