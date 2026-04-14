package utils

import "github.com/samber/lo"

func ClonePtr[T any](p *T) *T {
	if p == nil {
		return lo.Nil[T]()
	}
	return lo.ToPtr(*p)
}
