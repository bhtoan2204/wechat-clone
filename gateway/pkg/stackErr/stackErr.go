package stackErr

import (
	"reflect"

	pkgErr "github.com/pkg/errors"
)

func errorWithStack(err error) error {
	if err == nil {
		return nil
	}

	stackTrace := reflect.ValueOf(err).MethodByName("StackTrace")
	if stackTrace.IsValid() {
		return Error(err)
	}

	return pkgErr.WithStack(err)
}

func Error(err error) error {
	if err == nil {
		return nil
	}
	return errorWithStack(err)
}
