package backend

import (
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"net/http"
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

func (m *MuxHandler) ListOperation() []string {
	var opNames = make([]string, 0)
	for opName := range m.opNameHandlerMap {
		opNames = append(opNames, opName)
	}
	return opNames
}
