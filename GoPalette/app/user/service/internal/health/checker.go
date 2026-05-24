package health

import "context"

type Checker interface {
	Name() string
	Check(context.Context) error
}
