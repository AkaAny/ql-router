package example

import (
	"context"
	"errors"
	"fmt"
	backend "github.com/AkaAny/ql-router"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"strings"
	"time"
)

type ArticleEntity struct {
	Name         string
	BelongToUser string
	Date         time.Time
	Content      string
}

type UserIDArticleMap map[string][]*ArticleEntity

type ArticleResponse struct {
	Name         string
	BelongToUser string
	Date         time.Time
	Content      string
}

type ListArticle struct {
	ArticleList []*ArticleResponse
}

type ListArticleArg struct {
	NameLike   string
	DateBefore time.Time
	DateAfter  time.Time
}

const ContextKeyUserID = "userID"

func NewListArticleByDateResolveFn(articleMap UserIDArticleMap) backend.CustomFieldResolveFn[ListArticle, ListArticleArg] {
	return func(p graphql.ResolveParams, arg ListArticleArg) (*ListArticle, error) {
		var userID = p.Context.Value(ContextKeyUserID).(string)
		fmt.Println(userID)
		articles, ok := articleMap[userID]
		if !ok {
			return &ListArticle{ArticleList: nil}, errors.New("user does not exist")
		}
		var dataList = make([]*ArticleResponse, 0)
		for _, articleItem := range articles {
			if !strings.Contains(articleItem.Name, arg.NameLike) {
				continue
			}
			if !articleItem.Date.Before(arg.DateBefore) {
				continue
			}
			if !articleItem.Date.After(arg.DateAfter) {
				continue
			}
			dataList = append(dataList, &ArticleResponse{
				Name:         articleItem.Name,
				BelongToUser: articleItem.BelongToUser,
				Date:         articleItem.Date,
				Content:      articleItem.Content,
			})
		}
		return &ListArticle{ArticleList: dataList}, nil
	}
}

func QueryUserByNew(articleMap UserIDArticleMap) {
	var muxHandler = backend.NewMuxHandler()
	backend.PutHandlerWithOperationWithGeneric(muxHandler,
		graphql.DirectiveLocationQuery, backend.HandlerWithOperationWithGeneric[ListArticle, ListArticleArg]{
			OperationName: "listArticle",
			RootFieldName: "", //default means same with operation name
			ResolveFn:     NewListArticleByDateResolveFn(articleMap),
			FieldRule:     nil, //default means use DefaultFieldToNameRule
			ArgRule:       nil, //default means use DefaultFieldToNameRule
		})
	var engine = gin.Default()
	engine.POST("/graphql/", func(c *gin.Context) {
		var userID = c.GetHeader("X-UserID")
		var contextWithIdentity = context.WithValue(c.Request.Context(), ContextKeyUserID, userID)
		var req = c.Request.Clone(contextWithIdentity)
		muxHandler.ServeHTTP(c.Writer, req)
	})
	err := engine.Run(":10000")
	if err != nil {
		panic(err)
	}
}

func QueryUserByTraditional(articleMap UserIDArticleMap) {
	var articleResponseType = graphql.NewObject(graphql.ObjectConfig{
		Name: "ArticleResponse",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"belongToUser": &graphql.Field{
				Type: graphql.String,
			},
			"date": &graphql.Field{
				Type: graphql.DateTime,
			},
			"content": &graphql.Field{
				Type: graphql.String,
			},
		},
	})
	var funcType = graphql.NewObject(graphql.ObjectConfig{
		Name: graphql.DirectiveLocationQuery,
		Fields: graphql.Fields{
			"listArticle": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "ListArticle",
					Fields: graphql.Fields{
						"articleList": &graphql.Field{
							Type: graphql.NewList(articleResponseType),
						},
					},
				}),
				Args: graphql.FieldConfigArgument{
					"nameLike": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"dateBefore": &graphql.ArgumentConfig{
						Type: graphql.DateTime,
					},
					"dateAfter": &graphql.ArgumentConfig{
						Type: graphql.DateTime,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var nameLike = p.Args["nameLike"].(string)
					var dateBefore = p.Args["dateBefore"].(time.Time)
					var dateAfter = p.Args["dateAfter"].(time.Time)
					var customResolveFn = NewListArticleByDateResolveFn(articleMap)
					return customResolveFn(p, ListArticleArg{
						NameLike:   nameLike,
						DateBefore: dateBefore,
						DateAfter:  dateAfter,
					})
				},
			},
		},
	})
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: funcType,
	})
	if err != nil {
		panic(err)
	}
	var qlHandler = handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	var engine = gin.Default()
	engine.POST("/graphql/", func(c *gin.Context) {
		var userID = c.GetHeader("X-UserID")
		var contextWithIdentity = context.WithValue(c.Request.Context(), ContextKeyUserID, userID)
		var req = c.Request.Clone(contextWithIdentity)
		qlHandler.ServeHTTP(c.Writer, req)
	})
	err = engine.Run(":20000")
	if err != nil {
		panic(err)
	}
}
