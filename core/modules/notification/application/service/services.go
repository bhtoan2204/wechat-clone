package service

type Services interface {
}

type services struct {
}

func NewServices() Services {
	return &services{}
}
