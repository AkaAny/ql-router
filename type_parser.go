package backend

import (
	"github.com/graphql-go/graphql"
	"reflect"
	"strings"
	"time"
)

func GetElem(t reflect.Type) (elemType reflect.Type) {
	elemType = t
	for {
		switch elemType.Kind() {
		case reflect.Pointer:
			fallthrough
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			elemType = elemType.Elem()
			continue
		}
		break
	}
	return elemType
}

func IsArrayOrSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Array || t.Kind() == reflect.Slice
}

type FieldToNameRule func(fieldInfo reflect.StructField) string

func DefaultFieldToNameRule(fieldInfo reflect.StructField) string {
	var fieldName = fieldInfo.Name
	if tagValue, ok := fieldInfo.Tag.Lookup("qlField"); ok {
		fieldName = tagValue
	} else {
		fieldName = strings.ToLower(string(fieldName[0])) + fieldName[1:]
	}
	return fieldName
}

type TypeMap map[reflect.Type]graphql.Output

func getDefaultTypeMap() TypeMap {
	var emptyString = ""
	var typeMap = map[reflect.Type]graphql.Output{
		reflect.TypeOf(int(0)):       graphql.Int,
		reflect.TypeOf(""):           graphql.String,
		reflect.TypeOf(&emptyString): graphql.String,
		reflect.TypeOf(time.Time{}):  graphql.DateTime,
		reflect.TypeOf(true):         graphql.Boolean,
		reflect.TypeOf(int64(0)):     graphql.Int,
	}
	return typeMap
}

func ParseObjectFromType(typeInfo reflect.Type, typeMap TypeMap, rule FieldToNameRule) graphql.Output {
	if rule == nil {
		rule = DefaultFieldToNameRule
	}
	if typeMap == nil || len(typeMap) == 0 {
		typeMap = getDefaultTypeMap()
	}
	if graphQLType, ok := typeMap[typeInfo]; ok {
		return graphQLType
	}
	var fields = graphql.Fields{}
	for fieldIndex := 0; fieldIndex < typeInfo.NumField(); fieldIndex++ {
		var goTypeFieldInfo = typeInfo.Field(fieldIndex)
		var fieldName = rule(goTypeFieldInfo)
		var fieldElemTypeInfo = GetElem(goTypeFieldInfo.Type)
		var qlFieldType graphql.Output = nil

		if fieldElemTypeInfo.Kind() == reflect.Struct {
			if cachedQlFieldType, ok := typeMap[fieldElemTypeInfo]; ok {
				qlFieldType = cachedQlFieldType
			} else {
				qlFieldType = ParseObjectFromType(fieldElemTypeInfo, typeMap, rule)
				//cache to make same go type ql obj have same address
				//Schema must contain unique named types but contains multiple types named "FavoriteModelNOJPage".
				typeMap[fieldElemTypeInfo] = qlFieldType
			}
		} else {
			if primitiveQLFieldType, ok := typeMap[fieldElemTypeInfo]; ok {
				qlFieldType = primitiveQLFieldType
			} else {
				if goTypeFieldInfo.Name == "Error" {
					qlFieldType = graphql.String
				}
			}
		}
		if IsArrayOrSlice(goTypeFieldInfo.Type) {
			qlFieldType = graphql.NewList(qlFieldType)
		}
		fields[fieldName] = &graphql.Field{
			Type:      qlFieldType,
			Args:      nil,
			Resolve:   nil,
			Subscribe: nil,
		}
	}
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   typeInfo.Name(),
		Fields: fields,
	})
}

func ParseObjectFromTypeWithGeneric[R any](rule FieldToNameRule) graphql.Output {
	var modelType = reflect.TypeOf(new(R)).Elem()
	return ParseObjectFromType(modelType, nil, rule)
}
