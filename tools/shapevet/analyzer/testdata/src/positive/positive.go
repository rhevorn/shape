package positive

import (
	"context"
	"strings"
	"time"

	"github.com/rhevorn/shape"
	shapetypes "github.com/rhevorn/shape/types"
)

type Name string
type Count int16
type Small uint8
type Ratio float32
type Key int
type Names []Name
type Counts map[Key]Count

type Child struct {
	Code string `json:"code" shape:"trim,notempty"`
}

type Tagged struct {
	Name     Name                 `json:"name" shape:"LABEL='display name',IFZERO=guest,TRIM=' /',LTRIM=x,RTRIM=y,TOLOWER,TOUPPER,NOTEMPTY,MINLENGTH=0,MAXLENGTH=100,LEN=3,ONEOF=''"`
	Count    Count                `json:"count" shape:"ifzero=1,min=-10,max=10,between=-10|10,gt=-11,gte=-10,lt=11,lte=10,oneof=-1|0|1,positive,negative,nonnegative"`
	Small    Small                `json:"small" shape:"ifzero=1,max=255"`
	Ratio    Ratio                `json:"ratio" shape:"ifzero=1.5,min=-1e2,max=1e2"`
	Active   bool                 `json:"active" shape:"ifzero=true,label=active"`
	Created  time.Time            `json:"created" shape:"ifzero=2026-09-11T08:00:00Z"`
	Timeout  shapetypes.Duration  `json:"timeout" shape:"ifzero=30s,min=1s,max=5m,between=1s|5m,oneof=1s|30s"`
	Nanos    time.Duration        `json:"nanos" shape:"ifzero=1000000000,min=1"`
	Nickname *string              `json:"nickname" shape:"ifnull=guest,trim,notnull,notempty,email"`
	Enabled  *bool                `json:"enabled" shape:"ifnull=false,notnull"`
	Started  *time.Time           `json:"started" shape:"ifnull=2026-09-11T08:00:00Z,notnull"`
	Delay    *shapetypes.Duration `json:"delay" shape:"ifnull=1s,positive"`
	Child    Child                `json:"child" shape:"label=child"`
	ChildPtr *Child               `json:"childPtr" shape:"notnull,notempty"`
	Names    Names                `json:"names" shape:"notnull,notempty,min=1,max=5,len=2,unique"`
	Counts   Counts               `json:"counts" shape:"notnull,notempty,min=1,max=5,len=2"`
	Hidden   chan int             `json:"-"`
	private  any
}

type Explicit struct {
	Name    string
	Count   Count
	Active  bool
	Created time.Time
	Timeout shapetypes.Duration
	Nick    *string
	Names   []string
	Scores  map[string]int
	Child   Child
}

const nameField = "Name"

func build(data []byte) {
	_ = shape.Struct[Tagged]()
	var tagged Tagged
	_ = shape.BindJSON(&tagged, data)
	_ = shape.BindJSONContext(context.Background(), &tagged, data)
	_ = shape.BindJSONReader(&tagged, strings.NewReader("{}"))
	_ = shape.BindJSONReaderContext(context.Background(), &tagged, strings.NewReader("{}"))

	_ = shape.New[Explicit](
		shape.String(nameField).Trim().NotEmpty(),
		shape.Number[Count]("Count").Min(0),
		shape.Bool("Active"),
		shape.Time("Created"),
		shape.Duration("Timeout"),
		shape.Pointer("Nick", shape.String()),
		shape.Slice("Names", shape.String()),
		shape.Map("Scores", shape.String(), shape.Int()),
		shape.Field("Child", shape.Value[Child]()),
	)
	_ = shape.New[Explicit]()

	fieldName := "Name"
	_ = shape.New[Explicit](shape.String(fieldName))
	var dynamic shape.FieldSpec = shape.String("Name")
	_ = shape.New[Explicit](dynamic)
}

type Box[T any] struct {
	Value T `json:"value" shape:"min=1"`
}

func generic[T any]() {
	_ = shape.Struct[Box[T]]()
	_ = shape.New[Box[T]](shape.Value[T]("Value"))
}
