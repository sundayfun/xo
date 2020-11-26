package internal

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/gedex/inflector"
	"github.com/kenshaw/snaker"
	"github.com/xo/xo/models"
)

// NewTemplateFuncs returns a set of template funcs bound to the supplied args.
func (a *ArgType) NewTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"colcount":           a.colcount,
		"colnames":           a.colnames,
		"colnamesgeo":        a.colnamesgeo,
		"colnamesmulti":      a.colnamesmulti,
		"colnamesgeomulti":   a.colnamesgeomulti,
		"colnamesquery":      a.colnamesquery,
		"colnamesquerymulti": a.colnamesquerymulti,
		"colprefixnames":     a.colprefixnames,
		"colvals":            a.colvals,
		"colvalsmulti":       a.colvalsmulti,
		"fieldnames":         a.fieldnames,
		"fieldnamesmulti":    a.fieldnamesmulti,
		"goparamlist":        a.goparamlist,
		"reniltype":          a.reniltype,
		"retype":             a.retype,
		"shortname":          a.shortname,
		"convext":            a.convext,
		"schema":             a.schemafn,
		"colname":            a.colname,
		"hascolumn":          a.hascolumn,
		"hasfield":           a.hasfield,
		"getstartcount":      a.getstartcount,
		"pluralize":          a.pluralize,
		"snaketocamel":       a.snaketocamel,
		"modelToPB":          a.modelToPB,
		"PBToModel":          a.PBToModel,
		"proto":              a.proto,
		"GoPackageName":      goPackageName,
	}
}

// retype checks typ against known types, and prefixing
// ArgType.CustomTypePackage (if applicable).
func (a *ArgType) retype(typ string) string {
	if strings.Contains(typ, ".") {
		return typ
	}

	prefix := ""
	for strings.HasPrefix(typ, "[]") {
		typ = typ[2:]
		prefix = prefix + "[]"
	}

	if _, ok := a.KnownTypeMap[typ]; !ok {
		pkg := a.CustomTypePackage
		if pkg != "" {
			pkg = pkg + "."
		}

		return prefix + pkg + typ
	}

	return prefix + typ
}

// reniltype checks typ against known nil types (similar to retype), prefixing
// ArgType.CustomTypePackage (if applicable).
func (a *ArgType) reniltype(typ string) string {
	if strings.Contains(typ, ".") {
		return typ
	}

	if strings.HasSuffix(typ, "{}") {
		if _, ok := a.KnownTypeMap[typ[:len(typ)-2]]; ok {
			return typ
		}

		pkg := a.CustomTypePackage
		if pkg != "" {
			pkg = pkg + "."
		}

		return pkg + typ
	}

	return typ
}

// shortname generates a safe Go identifier for typ. typ is first checked
// against ArgType.ShortNameTypeMap, and if not found, then the value is
// calculated and stored in the ShortNameTypeMap for future use.
//
// A shortname is the concatentation of the lowercase of the first character in
// the words comprising the name. For example, "MyCustomName" will have have
// the shortname of "mcn".
//
// If a generated shortname conflicts with a Go reserved name, then the
// corresponding value in goReservedNames map will be used.
//
// Generated shortnames that have conflicts with any scopeConflicts member will
// have ArgType.NameConflictSuffix appended.
//
// Note: recognized types for scopeConflicts are string, []*Field,
// []*QueryParam.
func (a *ArgType) shortname(typ string, scopeConflicts ...interface{}) string {
	var v string
	var ok bool

	// check short name map
	if v, ok = a.ShortNameTypeMap[typ]; !ok {
		// calc the short name
		u := []string{}
		for _, s := range strings.Split(strings.ToLower(snaker.CamelToSnake(typ)), "_") {
			if len(s) > 0 && s != "id" {
				u = append(u, s[:1])
			}
		}
		v = strings.Join(u, "")

		// check go reserved names
		if n, ok := goReservedNames[v]; ok {
			v = n
		}

		// store back to short name map
		a.ShortNameTypeMap[typ] = v
	}

	// initial conflicts are the default imported packages from
	// xo_package.go.tpl
	conflicts := map[string]bool{
		"sql":     true,
		"driver":  true,
		"csv":     true,
		"errors":  true,
		"fmt":     true,
		"regexp":  true,
		"strings": true,
		"time":    true,
	}

	// add scopeConflicts to conflicts
	for _, c := range scopeConflicts {
		switch k := c.(type) {
		case string:
			conflicts[k] = true

		case []*Field:
			for _, f := range k {
				conflicts[f.Name] = true
			}
		case []*QueryParam:
			for _, f := range k {
				conflicts[f.Name] = true
			}

		default:
			panic("not implemented")
		}
	}

	// append suffix if conflict exists
	if _, ok := conflicts[v]; ok {
		v = v + a.NameConflictSuffix
	}

	return v
}

// colnames creates a list of the column names found in fields, excluding any
// Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of column names, that can be used in
// a SELECT, or UPDATE, or other SQL clause requiring an list of identifiers
// (ie, "field_1, field_2, field_3, ...").
func (a *ArgType) colnames(fields []*Field, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		str = str + a.colname(f.Col)
		i++
	}

	return str
}

// colnamesgeo creates a list of the column names found in fields, excluding any
// Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of column names, that can be used in
// a SELECT, or UPDATE, or other SQL clause requiring an list of identifiers
// (ie, "field_1, field_2, field_3, ...").
// Used in select query ST_AsBinary for geo info fields
func (a *ArgType) colnamesgeo(fields []*Field, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + fmt.Sprintf("ST_AsBinary(%s)", a.colname(f.Col))
		} else {
			str = str + a.colname(f.Col)
		}
		i++
	}

	return str
}

// colnamesmulti creates a list of the column names found in fields, excluding any
// Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of column names, that can be used in
// a SELECT, or UPDATE, or other SQL clause requiring an list of identifiers
// (ie, "field_1, field_2, field_3, ...").
func (a *ArgType) colnamesmulti(fields []*Field, ignoreNames []*Field) string {
	ignore := map[string]bool{}
	for _, f := range ignoreNames {
		ignore[f.Name] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		str = str + a.colname(f.Col)
		i++
	}

	return str
}

// colnamesgeomulti creates a list of the column names found in fields, excluding any
// Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of column names, that can be used in
// a SELECT, or UPDATE, or other SQL clause requiring an list of identifiers
// (ie, "field_1, field_2, field_3, ...").
// Used in select query ST_AsBinary for geo info fields
func (a *ArgType) colnamesgeomulti(fields []*Field, ignoreNames []*Field) string {
	ignore := map[string]bool{}
	for _, f := range ignoreNames {
		ignore[f.Name] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + fmt.Sprintf("ST_AsBinary(%s)", a.colname(f.Col))
		} else {
			str = str + a.colname(f.Col)
		}
		i++
	}

	return str
}

// colnamesquery creates a list of the column names in fields as a query and
// joined by sep, excluding any Field with Name contained in ignoreNames.
//
// Used to create a list of column names in a WHERE clause (ie, "field_1 = $1
// AND field_2 = $2 AND ...") or in an UPDATE clause (ie, "field = $1, field =
// $2, ...").
func (a *ArgType) colnamesquery(fields []*Field, hasDeletedField bool, sep string, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	var skipDeleted bool
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if f.Name == "is_deleted" {
			skipDeleted = true
		}

		if i != 0 {
			str = str + sep
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + a.colname(f.Col) + " = " + "ST_GeomFromWKB(?)"
		} else {
			str = str + a.colname(f.Col) + " = " + a.Loader.NthParam(i)
		}
		i++
	}

	if hasDeletedField && !skipDeleted {
		str += sep
		str += "is_deleted = false"
	}

	return str
}

// colnamesquerymulti creates a list of the column names in fields as a query and
// joined by sep, excluding any Field with Name contained in the slice of fields in ignoreNames.
//
// Used to create a list of column names in a WHERE clause (ie, "field_1 = $1
// AND field_2 = $2 AND ...") or in an UPDATE clause (ie, "field = $1, field =
// $2, ...").
func (a *ArgType) colnamesquerymulti(fields []*Field, hasDeletedField bool, sep string, startCount int, ignoreNames []*Field) string {
	ignore := map[string]bool{}
	for _, f := range ignoreNames {
		ignore[f.Name] = true
	}

	str := ""
	i := startCount
	var skipDeleted bool
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if f.Name == "is_deleted" {
			skipDeleted = true
		}

		if i > startCount {
			str = str + sep
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + a.colname(f.Col) + " = " + "ST_GeomFromWKB(?)"
		} else {
			str = str + a.colname(f.Col) + " = " + a.Loader.NthParam(i)
		}
		i++
	}

	if hasDeletedField && !skipDeleted {
		str += sep
		str += "is_deleted = false"
	}

	return str
}

// colprefixnames creates a list of the column names found in fields with the
// supplied prefix, excluding any Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of column names with a prefix. Used in
// a SELECT, or UPDATE (ie, "t.field_1, t.field_2, t.field_3, ...").
func (a *ArgType) colprefixnames(fields []*Field, prefix string, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		str = str + prefix + "." + a.colname(f.Col)
		i++
	}

	return str
}

// colvals creates a list of value place holders for fields excluding any Field
// with Name contained in ignoreNames.
//
// Used to present a comma separated list of column place holders, used in a
// SELECT or UPDATE statement (ie, "$1, $2, $3 ...").
func (a *ArgType) colvals(fields []*Field, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + "ST_GeomFromWKB(?)"
		} else {
			str = str + a.Loader.NthParam(i)
		}
		i++
	}

	return str
}

// colvalsmulti creates a list of value place holders for fields excluding any Field
// with Name contained in ignoreNames.
//
// Used to present a comma separated list of column place holders, used in a
// SELECT or UPDATE statement (ie, "$1, $2, $3 ...").
func (a *ArgType) colvalsmulti(fields []*Field, ignoreNames []*Field) string {
	ignore := map[string]bool{}
	for _, f := range ignoreNames {
		ignore[f.Name] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		if _, ok := a.GeoInfoTypeMap[f.Type]; ok {
			str = str + "ST_GeomFromWKB(?)"
		} else {
			str = str + a.Loader.NthParam(i)
		}
		i++
	}

	return str
}

// fieldnames creates a list of field names from fields of the adding the
// provided prefix, and excluding any Field with Name contained in ignoreNames.
//
// Used to present a comma separated list of field names, ie in a Go statement
// (ie, "t.Field1, t.Field2, t.Field3 ...")
func (a *ArgType) fieldnames(fields []*Field, prefix string, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		str = str + prefix + "." + f.Name
		i++
	}

	return str
}

// fieldnamesmulti creates a list of field names from fields of the adding the
// provided prefix, and excluding any Field with the slice contained in ignoreNames.
//
// Used to present a comma separated list of field names, ie in a Go statement
// (ie, "t.Field1, t.Field2, t.Field3 ...")
func (a *ArgType) fieldnamesmulti(fields []*Field, prefix string, ignoreNames []*Field) string {
	ignore := map[string]bool{}
	for _, f := range ignoreNames {
		ignore[f.Name] = true
	}

	str := ""
	i := 0
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		if i != 0 {
			str = str + ", "
		}
		str = str + prefix + "." + f.Name
		i++
	}

	return str
}

// colcount returns the 1-based count of fields, excluding any Field with Name
// contained in ignoreNames.
//
// Used to get the count of fields, and useful for specifying the last SQL
// parameter.
func (a *ArgType) colcount(fields []*Field, ignoreNames ...string) int {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	i := 1
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		i++
	}
	return i
}

// goReservedNames is a map of of go reserved names to "safe" names.
var goReservedNames = map[string]string{
	"break":       "brk",
	"case":        "cs",
	"chan":        "chn",
	"const":       "cnst",
	"continue":    "cnt",
	"default":     "def",
	"defer":       "dfr",
	"else":        "els",
	"fallthrough": "flthrough",
	"for":         "fr",
	"func":        "fn",
	"go":          "goVal",
	"goto":        "gt",
	"if":          "ifVal",
	"import":      "imp",
	"interface":   "iface",
	"map":         "mp",
	"package":     "pkg",
	"range":       "rnge",
	"return":      "ret",
	"select":      "slct",
	"struct":      "strct",
	"switch":      "swtch",
	"type":        "typ",
	"var":         "vr",

	// go types
	"error":      "e",
	"bool":       "b",
	"string":     "str",
	"byte":       "byt",
	"rune":       "r",
	"uintptr":    "uptr",
	"int":        "i",
	"int8":       "i8",
	"int16":      "i16",
	"int32":      "i32",
	"int64":      "i64",
	"uint":       "u",
	"uint8":      "u8",
	"uint16":     "u16",
	"uint32":     "u32",
	"uint64":     "u64",
	"float32":    "z",
	"float64":    "f",
	"complex64":  "c",
	"complex128": "c128",
}

// goparamlist converts a list of fields into their named Go parameters,
// skipping any Field with Name contained in ignoreNames. addType will cause
// the go Type to be added after each variable name. addPrefix will cause the
// returned string to be prefixed with ", " if the generated string is not
// empty.
//
// Any field name encountered will be checked against goReservedNames, and will
// have its name substituted by its corresponding looked up value.
//
// Used to present a comma separated list of Go variable names for use with as
// either a Go func parameter list, or in a call to another Go func.
// (ie, ", a, b, c, ..." or ", a T1, b T2, c T3, ...").
func (a *ArgType) goparamlist(fields []*Field, addPrefix bool, addType bool, ignoreNames ...string) string {
	ignore := map[string]bool{}
	for _, n := range ignoreNames {
		ignore[n] = true
	}

	i := 0
	vals := []string{}
	for _, f := range fields {
		if ignore[f.Name] {
			continue
		}

		s := "v" + strconv.Itoa(i)
		if len(f.Name) > 0 {
			n := strings.Split(snaker.CamelToSnake(f.Name), "_")
			s = strings.ToLower(n[0]) + f.Name[len(n[0]):]
		}

		// check go reserved names
		if r, ok := goReservedNames[strings.ToLower(s)]; ok {
			s = r
		}

		// add the go type
		if addType {
			s += " " + a.retype(f.Type)
		}

		// add to vals
		vals = append(vals, s)

		i++
	}

	// concat generated values
	str := strings.Join(vals, ", ")
	if addPrefix && str != "" {
		return ", " + str
	}

	return str
}

// convext generates the Go conversion for f in order for it to be assignable
// to t.
//
// FIXME: this should be a better name, like "goconversion" or some such.
func (a *ArgType) convext(prefix string, f *Field, t *Field) string {
	expr := prefix + "." + f.Name
	if f.Type == t.Type {
		return expr
	}

	ft := f.Type
	if strings.HasPrefix(ft, "sql.Null") {
		expr = expr + "." + f.Type[8:]
		ft = strings.ToLower(f.Type[8:])
	}

	if t.Type != ft {
		expr = t.Type + "(" + expr + ")"
	}

	return expr
}

// schemafn takes a series of names and joins them with the schema name.
func (a *ArgType) schemafn(s string, names ...string) string {
	// escape table names
	if a.EscapeTableNames {
		for i, t := range names {
			names[i] = a.Loader.Escape(TableEsc, t)
		}
	}

	n := strings.Join(names, ".")

	if s == "" && n == "" {
		return ""
	}

	if s != "" && n != "" {
		if a.EscapeSchemaName {
			s = a.Loader.Escape(SchemaEsc, s)
		}
		s = s + "."
	}

	return s + n
}

// colname returns the ColumnName of col, optionally escaping it if
// ArgType.EscapeColumnNames is toggled.
func (a *ArgType) colname(col *models.Column) string {
	if a.EscapeColumnNames {
		return a.Loader.Escape(ColumnEsc, col.ColumnName)
	}

	return col.ColumnName
}

// hascolumn takes a list of fields and determines if field with the specified
// column name is in the list.
func (a *ArgType) hascolumn(fields []*Field, name string) bool {
	for _, f := range fields {
		if f.Col.ColumnName == name {
			return true
		}
	}

	return false
}

// hasfield takes a list of fields and determines if field with the specified
// field name is in the list.
func (a *ArgType) hasfield(fields []*Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}

	return false
}

// getstartcount returns a starting count for numbering columsn in queries
func (a *ArgType) getstartcount(fields []*Field, pkFields []*Field) int {
	return len(fields) - len(pkFields)
}

func (a *ArgType) pluralize(name string) string {
	return inflector.Pluralize(name)
}

func (a *ArgType) snaketocamel(name string) string {
	return snaker.ForceLowerCamelIdentifier(name)
}

func (a *ArgType) modelToPB(option *MethodsOption) string {
	if !option.ModelToPB {
		return ""
	}
	var body, shortName string

	fieldsAssignment := make([]string, 0, len(option.Type.Fields))

	shortName = a.shortname(option.Type.Name)

	for _, field := range option.Type.Fields {
		if _, ok := option.ModelToPBConfig.SkipFields[field.Col.ColumnName]; ok {
			continue
		}
		if !field.Col.NotNull && field.Type != "[]byte" {
			continue
		}
		var f, fa string
		if field.Type == "time.Time" {
			f = snaker.ForceLowerCamelIdentifier(field.Col.ColumnName)
			body = body + fmt.Sprintf(
				`%s, err := ptypes.TimestampProto(%s.%s)
	if err != nil {
		return nil, err
	}
`, f, shortName, field.Name)
			fa = fmt.Sprintf(`%s:%s,`, SnakeToCamelWithoutInitialisms(field.Col.ColumnName), f)
		} else {
			if t, ok := a.ToPBTypeMap[field.Type]; ok && !a.IncompatilbePBType[t] {
				fa = fmt.Sprintf(`%s:%s(%s.%s),`, SnakeToCamelWithoutInitialisms(field.Col.ColumnName),
					t, shortName, field.Name)
			} else {
				fa = fmt.Sprintf(`%s:%s.%s,`, SnakeToCamelWithoutInitialisms(field.Col.ColumnName),
					shortName, field.Name)
			}
		}
		fieldsAssignment = append(fieldsAssignment, fa)
	}
	body = body + fmt.Sprintf(
		`proto%s := &%s.%s{
	%s
}
`, option.Type.Name, goPackageName(option.ModelToPBConfig.ImportService), option.Type.Name, strings.Join(fieldsAssignment, "\n"))
	for _, field := range option.Type.Fields {
		if field.Col.NotNull || field.Type == "[]byte" {
			continue
		}
		if _, ok := option.ModelToPBConfig.SkipFields[field.Col.ColumnName]; ok {
			continue
		}
		var s, fa string
		s = SnakeToCamelWithoutInitialisms(field.Col.ColumnName)
		if field.Type == "mysql.NullTime" {
			f := snaker.ForceLowerCamelIdentifier(field.Col.ColumnName)
			fa = fmt.Sprintf(
				`if %s.%s.Valid {
	%s, err := ptypes.TimestampProto(%s.%s.Time)
	if err != nil {
		return nil, err
	}
	proto%s.%s = %s
}
`, shortName, field.Name,
				f, shortName, field.Name,
				option.Type.Name, s, f)
		} else if typ, ok := a.WrapperTypeMap[field.Type]; ok {
			fa = fmt.Sprintf(
				`if %s.%s.Valid {
	proto%s.%s = &wrappers.%s{Value:%s.%s.%s}
}
`, shortName, field.Name, option.Type.Name, s, typ, shortName, field.Name, strings.Trim(typ, "Value"))
		} else {
			log.Printf("WARN: %s.%s could be null, skipping!", option.Type.Name, field.Name)
		}
		body = body + fa
	}

	body = body + fmt.Sprintf("\nreturn proto%s, nil", option.Type.Name)
	return body
}

func (a *ArgType) PBToModel(option *MethodsOption) string {
	if !option.ModelToPB {
		return ""
	}
	var body, shortName string

	fieldsAssignment := make([]string, 0, len(option.Type.Fields))

	shortName = a.shortname(option.Type.Name)

	for _, field := range option.Type.Fields {
		if _, ok := option.ModelToPBConfig.SkipFields[field.Col.ColumnName]; ok {
			continue
		}
		if !field.Col.NotNull && field.Type != "[]byte" {
			continue
		}
		var f, fa string
		if field.Type == "time.Time" {
			f = snaker.ForceLowerCamelIdentifier(field.Col.ColumnName)
			body = body + fmt.Sprintf(
				`%s, err := ptypes.Timestamp(proto%s.%s)
	if err != nil {
		return nil, err
	}
`, f, option.Type.Name, field.Name)
			fa = fmt.Sprintf(`%s:%s,`, field.Name, f)

		} else if t, ok := a.ToPBTypeMap[field.Type]; ok && !a.IncompatilbePBType[t] {
			fa = fmt.Sprintf(`%s:%s(proto%s.%s),`, field.Name, field.Type, option.Type.Name,
				SnakeToCamelWithoutInitialisms(field.Col.ColumnName))
		} else {
			fa = fmt.Sprintf(`%s:proto%s.%s,`, field.Name, option.Type.Name,
				SnakeToCamelWithoutInitialisms(field.Col.ColumnName))
		}
		fieldsAssignment = append(fieldsAssignment, fa)
	}

	body = body + fmt.Sprintf(
		`%s := &%s{
	%s
}
`, shortName, option.Type.Name, strings.Join(fieldsAssignment, "\n"))

	for _, field := range option.Type.Fields {
		if _, ok := option.ModelToPBConfig.SkipFields[field.Col.ColumnName]; ok {
			continue
		}
		if field.Col.NotNull || field.Type == "[]byte" {
			continue
		}
		var s, fa string
		s = SnakeToCamelWithoutInitialisms(field.Col.ColumnName)
		if field.Type == "mysql.NullTime" {
			f := snaker.ForceLowerCamelIdentifier(field.Col.ColumnName)
			fa = fmt.Sprintf(
				`if proto%s.%s != nil {
	%s, err := ptypes.Timestamp(proto%s.%s)
	if err != nil {
		return nil, err
	}
	%s.%s = mysql.NullTime{Time:%s, Valid:true}
}
`, option.Type.Name, s,
				f, option.Type.Name, s,
				shortName, field.Name, f)

		} else if typ, ok := a.WrapperTypeMap[field.Type]; ok {
			fa = fmt.Sprintf(
				`if proto%s.%s != nil {
	%s.%s = %s{%s:proto%s.%s.Value, Valid:true}
}
`, option.Type.Name, s, shortName, field.Name, field.Type, strings.TrimSuffix(typ, "Value"), option.Type.Name, s)
		} else {
			log.Printf("WARN: %s.%s could be null, skipping!", option.Type.Name, field.Name)
		}
		body = body + fa
	}

	body = body + fmt.Sprintf("\nreturn %s, nil", shortName)
	return body
}

func (a *ArgType) proto(pc ProtoConfig) string {
	if len(pc) == 0 {
		return ""
	}
	sort.Stable(pc)

	svc := pc[0].ModelToPBConfig.ImportService
	var body string

	imports := make([]string, 0)
	m := make(map[string]struct{}, 0)
	for _, p := range pc {
		for _, f := range p.Type.Fields {
			if _, ok := p.ModelToPBConfig.SkipFields[f.Col.ColumnName]; ok {
				continue
			}
			if i, ok := a.ImportMap[f.Type]; ok {
				if _, ok = m[i]; ok {
					continue
				}
				m[i] = struct{}{}
				imports = append(imports, fmt.Sprintf(`import "%s";`, i))
			}
		}
	}

	body = body + fmt.Sprintf(
		`package proto.%s;

%s

option go_package = "%s/service/%s";
option java_multiple_files = true;
option objc_class_prefix = "RPC";

`, ProtoName(svc), strings.Join(imports, "\n"), a.ServerProtoPathPrefix, goPackageName(svc))

	for _, p := range pc {
		fieldsDef := make([]string, 0, len(p.Type.Fields))
		count := 1
		for _, f := range p.Type.Fields {
			if _, ok := p.ModelToPBConfig.SkipFields[f.Col.ColumnName]; ok {
				continue
			}
			def := fmt.Sprintf("\t%s %s = %d;", f.Type, f.Col.ColumnName, count)
			if v, ok := a.ToPBTypeMap[f.Type]; ok {
				def = fmt.Sprintf("\t%s %s = %d;", v, f.Col.ColumnName, count)
			}
			if v, ok := a.WrapperTypeMap[f.Type]; ok {
				def = fmt.Sprintf("\tgoogle.protobuf.%s %s = %d;", v, f.Col.ColumnName, count)
			}
			if f.Type == "mysql.NullTime" || f.Type == "time.Time" {
				def = fmt.Sprintf("\tgoogle.protobuf.Timestamp %s = %d;", f.Col.ColumnName, count)
			}
			if f.Comment != "" {
				def = fmt.Sprintf("%s\n%s", f.Comment, def)
			}
			fieldsDef = append(fieldsDef, def)
			count++
		}
		body = body + fmt.Sprintf(
			`message %s {
%s
}
`, p.Type.Name, strings.Join(fieldsDef, "\n"))
	}
	return body
}

// proto 对于 id -> Id
func SnakeToCamelWithoutInitialisms(str string) string {
	var r string

	for _, w := range strings.Split(str, "_") {
		if w == "" {
			continue
		}
		r += strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return r
}

// goPackageName public_story -> publicstory
func goPackageName(name string) string {
	name = strings.ReplaceAll(name, "-", "")
	return strings.ReplaceAll(name, "_", "")
}

func ProtoName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
