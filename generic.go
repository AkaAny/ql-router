package backend

import (
	"github.com/graphql-go/graphql"
	"reflect"
)

type CustomFieldResolveFn[R any, A any] func(p graphql.ResolveParams, arg A) (*R, error)

type HandlerWithOperationWithGeneric[R any, A any] struct {
	OperationName string
	RootFieldName string
	ResolveFn     CustomFieldResolveFn[R, A]
	FieldRule     FieldToNameRule
	ArgRule       FieldToNameRule
}

func ConvertCustomFieldResolveFn[R any, A any](fn CustomFieldResolveFn[R, A], argRule FieldToNameRule) graphql.FieldResolveFn {
	if argRule == nil {
		argRule = DefaultFieldToNameRule
	}
	var argType = reflect.TypeOf(new(A)).Elem()
	var fieldCount = argType.NumField()

	var argNames = make([]string, 0)
	for i := 0; i < fieldCount; i++ {
		var fieldInfo = argType.Field(i)
		var argName = argRule(fieldInfo)
		argNames = append(argNames, argName)
	}
	return func(p graphql.ResolveParams) (interface{}, error) {
		var argValue = reflect.New(argType).Elem()
		for i := 0; i < fieldCount; i++ {
			var argName = argNames[i]
			var vValue = reflect.ValueOf(p.Args[argName])
			argValue.Field(i).Set(vValue)
		}
		var arg = argValue.Interface().(A)
		r, err := fn(p, arg)
		return r, err
	}
}

func PutHandlerWithOperationWithGeneric[R any, A any](m *MuxHandler,
	objType string, param HandlerWithOperationWithGeneric[R, A]) {
	var funcType = ParseObjectFromTypeWithGeneric[R](param.FieldRule)
	m.PutHandlerWithOperation(objType, HandlerWithOperation{
		OperationName: param.OperationName,
		RootFieldName: param.RootFieldName,
		FuncType:      funcType,
		Args:          BuildArgWithGeneric[A](param.ArgRule, param.FieldRule),
		ResolveFn:     ConvertCustomFieldResolveFn(param.ResolveFn, param.ArgRule),
	})
}
