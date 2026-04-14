package modruntime

//go:generate mockgen -package=modruntime -destination=module_mock.go -source=module.go
type Module interface {
	Start() error
	Stop() error
}
