package backend

import (
	"github.com/graphql-go/graphql"
	"reflect"
)

func BuildArg(modelType reflect.Type, argRule FieldToNameRule,
	fieldRule FieldToNameRule) graphql.FieldConfigArgument {
	//var argModel=new(A)
	//var modelType=reflect.TypeOf(argModel).Elem()
	if fieldRule == nil {
		fieldRule = DefaultFieldToNameRule
	}
	if argRule == nil {
		argRule = DefaultFieldToNameRule
	}
	var argCount = modelType.NumField()
	if argCount == 0 {
		return nil
	}
	var result = make(graphql.FieldConfigArgument)
	for i := 0; i < argCount; i++ {
		var fieldInfo = modelType.Field(i)
		var argName = argRule(fieldInfo)
		result[argName] = &graphql.ArgumentConfig{
			Type: ParseObjectFromType(fieldInfo.Type, nil, fieldRule),
		}
	}
	return result
}

func BuildArgWithGeneric[A any](argRule FieldToNameRule, fieldRule FieldToNameRule) graphql.FieldConfigArgument {
	var argModel = new(A)
	var modelType = reflect.TypeOf(argModel).Elem()
	return BuildArg(modelType, argRule, fieldRule)
}
