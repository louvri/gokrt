package context

import (
	"context"

	"github.com/louvri/gokrt/sys_key"
)

type Property map[sys_key.SysKey]any

func InitiateProperty(ctx context.Context) Property {
	return Property{
		sys_key.FILE_KEY:        ctx.Value(sys_key.FILE_KEY),
		sys_key.FILE_OBJECT_KEY: ctx.Value(sys_key.FILE_OBJECT_KEY),
		sys_key.SOF:             ctx.Value(sys_key.SOF),
		sys_key.EOF:             ctx.Value(sys_key.EOF),
		sys_key.DATA_REF:        ctx.Value(sys_key.DATA_REF),
		sys_key.CACHE_KEY:       ctx.Value(sys_key.CACHE_KEY),
	}
}
func (p Property) Get(key sys_key.SysKey) any {
	switch key {
	case sys_key.FILE_KEY:
		return p[sys_key.FILE_KEY]
	case sys_key.FILE_OBJECT_KEY:
		return p[sys_key.FILE_OBJECT_KEY]
	case sys_key.SOF:
		return p[sys_key.SOF]
	case sys_key.EOF:
		return p[sys_key.EOF]
	case sys_key.DATA_REF:
		return p[sys_key.DATA_REF]
	case sys_key.CACHE_KEY:
		return p[sys_key.CACHE_KEY]
	case sys_key.GOKRT_CONTEXT:
		return p[sys_key.GOKRT_CONTEXT]
	default:
		return nil
	}
}

func (p Property) Set(key, value any) {
	switch key {
	case sys_key.FILE_KEY:
		p[sys_key.FILE_KEY] = value
	case sys_key.FILE_OBJECT_KEY:
		p[sys_key.FILE_OBJECT_KEY] = value
	case sys_key.SOF:
		p[sys_key.SOF] = value
	case sys_key.EOF:
		p[sys_key.EOF] = value
	case sys_key.DATA_REF:
		p[sys_key.DATA_REF] = value
	case sys_key.CACHE_KEY:
		p[sys_key.CACHE_KEY] = value
	case sys_key.GOKRT_CONTEXT:
		p[sys_key.GOKRT_CONTEXT] = value
	default:
	}
}
