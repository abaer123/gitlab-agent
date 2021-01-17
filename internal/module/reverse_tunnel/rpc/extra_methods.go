package rpc

import (
	"google.golang.org/grpc/metadata"
)

func (x *RequestInfo) Metadata() metadata.MD {
	return ValuesMapToMeta(x.Meta)
}

func (x *Header) Metadata() metadata.MD {
	return ValuesMapToMeta(x.Meta)
}

func (x *Trailer) Metadata() metadata.MD {
	return ValuesMapToMeta(x.Meta)
}

func ValuesMapToMeta(vals map[string]*Values) metadata.MD {
	result := make(metadata.MD, len(vals))
	for k, v := range vals {
		val := make([]string, len(v.Value))
		copy(val, v.Value) // metadata may be mutated, so copy
		result[k] = val
	}
	return result
}

func MetaToValuesMap(meta metadata.MD) map[string]*Values {
	if len(meta) == 0 {
		return nil
	}
	result := make(map[string]*Values, len(meta))
	for k, v := range meta {
		val := make([]string, len(v))
		copy(val, v) // metadata may be mutated, so copy
		result[k] = &Values{
			Value: val,
		}
	}
	return result
}
