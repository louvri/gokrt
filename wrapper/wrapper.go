package wrapper

import "context"

type Wrapper struct {
	Data any
	Ctx  context.Context
}
