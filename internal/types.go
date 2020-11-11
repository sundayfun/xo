package internal

import (
	"strings"

	"github.com/xo/xo/models"
)

// TemplateType represents a template type.
type TemplateType uint

// the order here will be the alter the output order per file.
const (
	EnumTemplate TemplateType = iota
	ProcTemplate
	TypeTemplate
	ForeignKeyTemplate
	IndexTemplate
	MapTemplate
	QueryTypeTemplate
	QueryTemplate
	OptionalTemplate
	TypeProtoTemplate

	// always last
	XOTemplate
)

// String returns the name for the associated template type.
func (tt TemplateType) String() string {
	var s string
	switch tt {
	case XOTemplate:
		s = "xo_db"
	case EnumTemplate:
		s = "enum"
	case ProcTemplate:
		s = "proc"
	case TypeTemplate:
		s = "type"
	case ForeignKeyTemplate:
		s = "foreignkey"
	case IndexTemplate:
		s = "index"
	case MapTemplate:
		s = "map"
	case QueryTypeTemplate:
		s = "querytype"
	case QueryTemplate:
		s = "query"
	case OptionalTemplate:
		s = "optional"
	case TypeProtoTemplate:
		s = "type"
	default:
		panic("unknown TemplateType")
	}
	return s
}

// RelType represents the different types of relational storage (table/view).
type RelType uint

const (
	// Table reltype
	Table RelType = iota

	// View reltype
	View
)

// EscType represents the different escape types.
type EscType uint

const (
	SchemaEsc = iota
	TableEsc
	ColumnEsc
)

type LoadType uint

const (
	LoadQueryFunc = iota
	LoadMapFunc
)

// String provides the string representation of RelType.
func (rt RelType) String() string {
	var s string
	switch rt {
	case Table:
		s = "TABLE"
	case View:
		s = "VIEW"
	default:
		panic("unknown RelType")
	}
	return s
}

type MethodsConfig struct {
	ListFields []string                  `yaml:"list_fields"`
	ModelToPB  map[string][]*TableConfig `yaml:"model_to_pb"`
}

type TableConfig struct {
	Name  string   `yaml:"name"`
	Skips []string `yaml:"skips"`
}

// EnumValue holds data for a single enum value.
type EnumValue struct {
	Name    string
	Val     *models.EnumValue
	Comment string
}

// Enum is a template item for a enum.
type Enum struct {
	Name              string
	Schema            string
	Values            []*EnumValue
	Enum              *models.Enum
	Comment           string
	ReverseConstNames bool
}

// Proc is a template item for a stored procedure.
type Proc struct {
	Name       string
	Schema     string
	ProcParams string
	Params     []*Field
	Return     *Field
	Proc       *models.Proc
	Comment    string
}

// Field contains field information.
type Field struct {
	Name    string
	Type    string
	NilType string
	Len     int
	Col     *models.Column
	Comment string
}

// Type is a template item for a type (ie, table/view/custom query).
type Type struct {
	Name             string
	Schema           string
	RelType          RelType
	PrimaryKey       *Field
	PrimaryKeyFields []*Field
	Fields           []*Field
	Indexes          map[string]*Index
	Table            *models.Table
	Comment          string
	HasDeletedField  bool
}

// ForeignKey is a template item for a foreign relationship on a table.
type ForeignKey struct {
	Name       string
	Schema     string
	Type       *Type
	Field      *Field
	RefType    *Type
	RefField   *Field
	ForeignKey *models.ForeignKey
	Comment    string
}

// Index is a template item for a index into a table.
type Index struct {
	FuncName    string
	MapFuncName string
	MapField    *Field
	Schema      string
	Type        *Type
	Fields      []*Field
	Index       *models.Index
	Comment     string
}

type MethodsOption struct {
	Type            *Type
	Sub             string
	ListFields      bool
	ModelToPB       bool
	ModelToPBConfig *ModelToPBConfig
}

type ModelToPBConfig struct {
	ImportService string
	SkipFields    map[string]struct{}
}

type ProtoConfig []*MethodsOption

func (t ProtoConfig) Len() int {
	return len(t)
}

func (t ProtoConfig) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t ProtoConfig) Less(i, j int) bool {
	return strings.Compare(t[i].Sub, t[j].Sub) < 0
}

// QueryParam is a query parameter for a custom query.
type QueryParam struct {
	Name        string
	Type        string
	Interpolate bool
}

// Query is a template item for a custom query.
type Query struct {
	Schema        string
	Name          string
	Query         []string
	QueryComments []string
	QueryParams   []*QueryParam
	OnlyOne       bool
	Interpolate   bool
	Type          *Type
	Comment       string
}

type Imports struct {
	Package string
	Imports []string
	Schema  string
}
