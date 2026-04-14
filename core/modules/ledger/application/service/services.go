package service

//go:generate mockgen -package=service -destination=services_mock.go -source=services.go
type Services interface {
}

type services struct {
}

func NewServices() Services {
	return &services{}
}
