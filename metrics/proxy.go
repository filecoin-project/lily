package metrics

import (
	"context"
	"reflect"

	"go.opencensus.io/tag"
)
//
//func MetricedGatewayAPI(a api.GatewayAPI) api.GatewayAPI {
//	var out apistruct.GatewayStruct
//	proxy(a, &out.Internal)
//	return &out
//}

func Proxy(in interface{}, out interface{}) {
	rint := reflect.ValueOf(out).Elem()
	ra := reflect.ValueOf(in)

	for f := 0; f < rint.NumMethod(); f++ {
		field := rint.Type().Field(f)
		fn := ra.MethodByName(field.Name)

		rint.Field(f).Set(reflect.MakeFunc(field.Type, func(args []reflect.Value) (results []reflect.Value) {
			ctx := args[0].Interface().(context.Context)
			// upsert function name into context
			ctx, _ = tag.New(ctx, tag.Upsert(API, field.Name))
			stop := Timer(ctx, LensRequestDuration)
			defer stop()
			// pass tagged ctx back into function call
			args[0] = reflect.ValueOf(ctx)
			return fn.Call(args)
		}))
	}
}
