package backend

import (
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"net/http"
	"reflect"
)

type MuxHandler struct {
	opNameHandlerMap map[string]*handler.Handler
}

func NewMuxHandler() *MuxHandler {
	return &MuxHandler{
		opNameHandlerMap: map[string]*handler.Handler{},
	}
}

type HandlerWithOperation struct {
	OperationName string
	RootFieldName string
	FuncType      graphql.Output
	Args          graphql.FieldConfigArgument
	ResolveFn     graphql.FieldResolveFn
}

func (m *MuxHandler) PutHandlerWithOperation(objType string, param HandlerWithOperation) {
	var opName = param.OperationName
	var rootFieldName = opName
	if param.RootFieldName != "" {
		rootFieldName = param.RootFieldName
	}
	var queryObj = graphql.NewObject(graphql.ObjectConfig{
		Name: objType,
		Fields: graphql.Fields{
			rootFieldName: &graphql.Field{
				Type:    param.FuncType,
				Args:    param.Args,
				Resolve: param.ResolveFn,
			},
		},
	})
	var schemaConfig = graphql.SchemaConfig{
		Query: queryObj,
	}
	switch objType {
	case graphql.DirectiveLocationMutation:
		schemaConfig.Mutation = queryObj
	}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		panic(err)
	}
	var qlHandler = handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	m.opNameHandlerMap[opName] = qlHandler
}

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

func (m *MuxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var opName = req.Header.Get("x-operation-name")
	fmt.Println("operation:", opName)
	qlHandler, ok := m.opNameHandlerMap[opName]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	qlHandler.ServeHTTP(w, req)
}
