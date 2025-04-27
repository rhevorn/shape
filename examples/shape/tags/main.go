package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type Address struct {
	City string `json:"city" shape:"trim,notempty,label=city"`
}

type TaggedRequest struct {
	DefaultName string `json:"default_name" shape:"ifzero=guest"`
	Name        string `json:"name" shape:"trim,notempty,minlength=5,maxlength=5,len=5,oneof=shape,pattern=^sha,startswith=sh,endswith=pe,contains=hap,label=name"`
	Left        string `json:"left" shape:"ltrim=_"`
	Right       string `json:"right" shape:"rtrim=_"`
	Quoted      string `json:"quoted" shape:"trim=' /'"`
	Lower       string `json:"lower" shape:"tolower"`
	Upper       string `json:"upper" shape:"toupper"`
	Email       string `json:"email" shape:"email"`
	URL         string `json:"url" shape:"url"`
	UUID        string `json:"uuid" shape:"uuid"`
	IP          string `json:"ip" shape:"ip"`

	Count    int     `json:"count" shape:"min=1,max=10,gt=1,gte=2,lt=10,lte=9,between=2|9,positive,nonnegative,oneof=5|7"`
	Delta    int     `json:"delta" shape:"negative"`
	Optional *string `json:"optional" shape:"ifnull=anonymous,notnull,notempty"`

	Tags       []string       `json:"tags" shape:"notnull,notempty,min=2,max=2,len=2,unique"`
	Attributes map[string]int `json:"attributes" shape:"notnull,notempty,min=1,max=1,len=1"`
	Address    Address        `json:"address"`
}

var taggedSchema = shape.Struct[TaggedRequest]().
	Apply(func(value TaggedRequest) (TaggedRequest, error) { return value, nil }).
	ApplyContext(func(ctx context.Context, value TaggedRequest) (TaggedRequest, error) {
		return value, ctx.Err()
	}).
	Refine(func(value TaggedRequest) error {
		if value.Name == value.DefaultName {
			return errors.New("name must differ from default_name")
		}
		return nil
	}).
	RefineContext(func(ctx context.Context, _ TaggedRequest) error { return ctx.Err() })

func main() {
	data := []byte(`{
		"name":" shape ",
		"left":"__left",
		"right":"right__",
		"quoted":" /quoted/ ",
		"lower":"SHAPE",
		"upper":"shape",
		"email":"pong@example.com",
		"url":"https://example.com",
		"uuid":"123e4567-e89b-12d3-a456-426614174000",
		"ip":"127.0.0.1",
		"count":5,
		"delta":-1,
		"tags":["go","shape"],
		"attributes":{"score":1},
		"address":{"city":" Shanghai "}
	}`)

	request, err := taggedSchema.ParseJSON(data)
	fmt.Printf("parse: default=%q name=%q left=%q right=%q quoted=%q lower=%q upper=%q optional=%q city=%q error=%v\n",
		request.DefaultName,
		request.Name,
		request.Left,
		request.Right,
		request.Quoted,
		request.Lower,
		request.Upper,
		dereference(request.Optional),
		request.Address.City,
		err,
	)

	var bound TaggedRequest
	err = shape.BindJSON(&bound, data)
	fmt.Printf("bind: name=%q count=%d tags=%v error=%v\n", bound.Name, bound.Count, bound.Tags, err)

	_, err = taggedSchema.ParseJSON([]byte(`{
		"name":"bad",
		"email":"not-email",
		"url":"bad",
		"uuid":"bad",
		"ip":"bad",
		"count":0,
		"delta":1,
		"tags":[],
		"attributes":{},
		"address":{"city":""}
	}`))
	var validationError *validate.Error
	if errors.As(err, &validationError) {
		fmt.Printf("invalid: %d issues, first=%s at %s\n",
			len(validationError.Issues),
			validationError.Issues[0].Code,
			validationError.Issues[0].Path,
		)
	}
}

func dereference(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
