// The following directive is necessary to make the package coherent:

// +build ignore

// This program generates events.go. It can be invoked by running
// go generate
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/iancoleman/strcase"
)

// GoSQLTypeMappings contains mapping between types provided by the type mapping file
// and sql column types
var GoSQLTypeMappings = map[string]string{
	"int":       "int",
	"timestamp": "int64",
	"bigint":    "int64",
}

// GoTypeMappings contains mapping between types provided by the type mapping file
// and Go types
var GoTypeMappings = map[string]string{
	"int":       "int",
	"timestamp": "int64",
	"bigint":    "int64",
}

// GoNullableTypeMappings contains mapping between types provided by the type mapping file
// and Go types that can handle null values that are comming from db
var GoNullableTypeMappings = map[string]string{
	"int":       "sql.NullInt32",
	"timestamp": "sql.NullInt64",
	"bigint":    "sql.NullInt64",
}

// UnknownTypeErr occurs when the type used for field in the type mapping
// file is unknown.
var UnknownTypeErr = fmt.Errorf("unknown type")

// NoTimeFieldErr occurs when an event has not field "time".
var NoTimeFieldErr = fmt.Errorf("no time field error")

// typeDefinition features for a type, many of the feature can be generated based on
// each other, to simplify the code template, they are stored to be used then.
type typeDefinition struct {
	Name             string
	OriginalName     string
	Type             string
	TimeField        *field
	Fields           []*field
	InsertTemplate   string // $1, $2, ...
	ProjectedColumns string // column1, column2, ...
}

// types specifies a group of typeDefinitions
type types map[string]typeDefinition

// field features for a field (column), many of the feature can be generated based on
// each other, to simplify the code template, they are stored to be used then.
type field struct {
	Name               string
	Type               string
	CamelCaseType      string
	OriginalType       string
	OriginalName       string
	RandomFunctionName string
	Comment            string
	NullableAssignment string
	NullableDefinition string
	NullableType       string
	ShouldBeInserted   bool
	Tags               tagSet
}

// tagSet represents the tags associated with a field.
// example: to store `gorm:"id, index", json:"omitempty"`
type tagSet map[string]string

func (tags tagSet) addTag(tagName string) {
	if _, exists := tags[tagName]; !exists {
		tags[tagName] = ""
	}
}

// addTagField adds tagField to the tagName
// example: adding "omitempty" (tagField) to "json" (tagName)
func (tags tagSet) addTagField(tagName, tagField string) {
	if _, exists := tags[tagName]; !exists {
		panic(fmt.Errorf("there is no tag %s", tagName))
	} else {
		tags[tagName] += tagField + ","
	}
}

// String returns the stringified version of the tagSet
// `gorm:"id, index", json:"omitempty"`
func (tags tagSet) String() string {
	r := "`"
	for tag, values := range tags {
		r += fmt.Sprintf("%s:\"%s\" ", tag, values[:len(values)-1])
	}
	if len(r) > 1 {
		r = r[:len(r)-1]
		r += "`"
		return r
	} else {
		return ""
	}
}

func main() {
	const TYPE_MAPPING_FILE = "TYPE_MAPPING_PATH"
	typeMappingFilePath := os.Getenv(TYPE_MAPPING_FILE)

	if len(typeMappingFilePath) < 1 {
		die(fmt.Sprintf("the path to type mapping file must be specified %s environment variable", TYPE_MAPPING_FILE))
	}

	f, err := os.Open(typeMappingFilePath)
	if err != nil {
		die(err.Error())
	}
	defer f.Close()

	ts, err := readAndParseTypeMapping(f)
	if err != nil {
		die(fmt.Sprint("can't parse type mappings:", err.Error()))
	}

	var buf bytes.Buffer

	out, err := os.Create("events.go")
	if err != nil {
		die(err.Error())
	}
	defer out.Close()

	err = eventsTemplate.Execute(&buf, struct {
		Timestamp         time.Time
		Types             types
		TypeMappingSource string
	}{
		Timestamp:         time.Now(),
		Types:             *ts,
		TypeMappingSource: typeMappingFilePath,
	})

	if err != nil {
		log.Fatal(err)
	}

	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	out.Write(formattedCode)
	out.Close()
}

func die(message string) {
	log.Fatal(message)
}

// typeMapping is format of the type_mapping's individual events,
// used for parsing the file
type typeMapping struct {
	Type    string            `json:"type"`
	Mapping map[string]string `json:"type_mapping"`
}

// validateTypeMappings validates:
// - all events need to have time field
// - all field for all events need to have valid type
func validateTypeMappings(in map[string]typeMapping) error {
	// validating
	for eventName, mappingDetails := range in {
		timeExists := false

		for fieldName, fieldType := range mappingDetails.Mapping {
			if fieldName == "time" {
				timeExists = true
			}

			if _, exists := GoTypeMappings[fieldType]; !exists {
				return fmt.Errorf("error validating %s.%s: %w", eventName, fieldName, UnknownTypeErr)
			}
		}

		if !timeExists {
			return fmt.Errorf("error validating %s: %w", eventName, NoTimeFieldErr)
		}
	}
	return nil
}

// readAndParseTypeMapping reads the type mapping and generates the Types
func readAndParseTypeMapping(r io.Reader) (*types, error) {
	// reading
	mappings := make(map[string]typeMapping)
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&mappings)
	if err != nil {
		return nil, err
	}

	// validate
	err = validateTypeMappings(mappings)
	if err != nil {
		return nil, err
	}

	// parsing
	ts := make(map[string]typeDefinition)
	for eventName, mappingDetails := range mappings {
		camelCaseName := strcase.ToCamel(eventName)
		td := typeDefinition{}
		td.Name = camelCaseName
		td.OriginalName = eventName
		td.Type = mappingDetails.Type
		td.Fields = make([]*field, 0)
		for fld, fieldType := range mappingDetails.Mapping {
			f := &field{}
			f.Name = strcase.ToCamel(fld)
			f.OriginalName = fld
			f.Type = GoTypeMappings[fieldType]
			f.CamelCaseType = strcase.ToCamel(f.Type)
			f.NullableType = GoNullableTypeMappings[fieldType]
			f.OriginalType = fieldType
			f.Tags = make(tagSet)
			f.Comment = fmt.Sprintf("generated based on %s with type %s", f.OriginalName, f.OriginalType)
			f.ShouldBeInserted = true

			if f.Name == "Time" {
				f.RandomFunctionName = "RandomTimeValue"
			} else {
				f.RandomFunctionName = "Random" + f.CamelCaseType + "Value"
			}

			// add tags
			f.Tags = make(tagSet)
			f.Tags.addTag("gorm")
			f.Tags.addTagField("gorm", GoSQLTypeMappings[fieldType])
			f.Tags.addTagField("gorm", "column:"+f.OriginalName)
			f.Tags.addTag("json")
			f.Tags.addTagField("json", f.OriginalName)
			if f.Name == "Time" {
				td.TimeField = f
				f.Tags.addTagField("gorm", "index")
			}
			td.Fields = append(td.Fields, f)
		}

		createdAtField := &field{
			Name:               "CreatedAt",
			Type:               "int64",
			CamelCaseType:      "Int",
			OriginalType:       "timestamp",
			OriginalName:       "created_at",
			NullableType:       GoNullableTypeMappings["timestamp"],
			RandomFunctionName: "",
			Comment:            "added by the generator so all types have created_at",
			ShouldBeInserted:   true,
			Tags:               make(tagSet),
		}

		createdAtField.Tags.addTag("gorm")
		createdAtField.Tags.addTagField("gorm", "int")
		createdAtField.Tags.addTagField("gorm", "column:created_at")
		createdAtField.Tags.addTag("json")
		createdAtField.Tags.addTagField("json", "created_at")

		td.Fields = append(td.Fields, createdAtField)

		// create templates for SQL
		td.InsertTemplate = ""
		td.ProjectedColumns = ""
		i := 1
		for _, f := range td.Fields {
			if !f.ShouldBeInserted {
				continue
			}
			td.InsertTemplate += "$" + strconv.Itoa(i)
			td.ProjectedColumns += f.OriginalName
			if i != len(td.Fields) {
				td.InsertTemplate += ", "
				td.ProjectedColumns += ", "
			}
			i++
		}

		ts[td.Name] = td
	}

	es := (types)(ts)
	return &es, nil
}

// eventsTemplate the templated used for generating events.go
var eventsTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// {{ .Timestamp }}
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"gorm.io/driver/postgres"
	"time"
	"gorm.io/gorm"
	"math/rand"
)

// TypeMappingSource stores the value from the environment 
// variable that refer to type_mapping file other tools,
// such as monitoring system can exploit this.
const TypeMappingSource = "{{$.TypeMappingSource}}"

// RandomGeneratorConstructors a map from event name to the 
// function that can generate a new random instance of that 
// event type
var RandomGeneratorConstructors = map[string]func(*rand.Rand) Event{{"{"}} {{ range $_, $t := $.Types }}
"{{$t.OriginalName}}":NewRandom{{$t.Name}}Event, {{end}} {{"}"}}

// Event is the contaract for storing and retrieving events form a database
// all event types implement using auto-generated codes.
type Event interface{

	// Store stores the single instance
	Store(ctx context.Context, db *sql.DB) error

	// Retrieve retrieves events from database filtered by start and the end time
	Retrieve(ctx context.Context, start, end int64, db *sql.DB) ([]Event, error)
}

{{ range $_, $t := $.Types }}
// {{ $t.Name }}Event coresponds to the type {{$t.OriginalName}} in the type_mappings
type {{ $t.Name }}Event struct {
	{{- range $_, $field := $t.Fields}}

	// {{$field.Comment}}
	{{ $field.Name }} {{$field.Type}} {{ $field.Tags -}}
	{{ end }}
}

// TableName returns the desired name for the table,
// used by the GORM for setting the table name.
func (o {{$t.Name}}Event ) TableName() string {
	return "{{$t.OriginalName}}"
}

// NewRandom{{$t.Name}}Event() genrates a new instance
// of {{$t.Name}}Event and returns is as an Event
func NewRandom{{$t.Name}}Event(r *rand.Rand) Event{
	e := &{{$t.Name}}Event{{"{}"}}
	{{ range $_, $field := $t.Fields}}
	{{- if $field.RandomFunctionName -}}
	e.{{ $field.Name }} = {{ $field.RandomFunctionName }}(r)
	{{end -}}
	{{ end }}
	e.CreatedAt = time.Now().UnixNano() / int64(time.Millisecond)
	return e
}

{{- end }}

// Migrate uses GORM to migrate the structs defined
// above. Calling this method multiple times is fine.
func Migrate(db *sql.DB, dsn string) error{
	var err error

	dialector := postgres.New(postgres.Config{
		DSN:                  dsn,
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})

	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	{{- range $_, $t := $.Types }}
	  // migrating {{ $t.Name }}Event
		err = gdb.AutoMigrate(&{{ $t.Name }}Event{})
		if err != nil{
			return err
		}
	{{ end }}
	return nil
}


{{- range $_, $t := $.Types }}
// Store {{ $t.Name }}Event in the database
func (e *{{ $t.Name }}Event) Store(ctx context.Context, db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO {{$t.OriginalName}} ({{$t.ProjectedColumns}}) VALUES ({{$t.InsertTemplate}})")
	if err != nil {{"{"}}
		return err
	{{"}"}}
	_, err = stmt.Exec({{$first := true}}{{range $_, $f := $t.Fields}}{{if $f.ShouldBeInserted}}{{if $first}}{{$first = false}}{{else}}, {{end}}e.{{$f.Name}}{{end}}{{end}})
	if err != nil {{"{"}}
		return err
	{{"}"}}
	return nil
}

// Retrieve {{ $t.Name }}Event queries the database and filters record that have
// time between start and end
func (e *{{ $t.Name }}Event) Retrieve(ctx context.Context, start, end int64, db *sql.DB) ([]Event, error) {
	rows, err := db.QueryContext(ctx, "SELECT {{$t.ProjectedColumns}} FROM {{$t.OriginalName}} WHERE {{ (index $t.TimeField ).OriginalName}} BETWEEN $1 AND $2", start, end)

	if err != nil {{"{"}}
		return nil, err
	{{"}"}}

	defer rows.Close()
	result := make([]Event,0)

	for rows.Next(){{"{"}}
		e := &{{ $t.Name }}Event{{"{}"}}
		{{range $_, $f := $t.Fields}}
		var {{$f.Name}} {{$f.NullableType}}
		{{end}}
		err := rows.Scan({{$first := true}}{{range $_, $f := $t.Fields}}{{if $f.ShouldBeInserted}}{{if $first}}{{$first = false}}{{else}}, {{end}}&{{$f.Name}}{{end}}{{end}})
		if err != nil{
			return nil, err
		}
		var temp driver.Value
		{{range $_, $f := $t.Fields}}
		temp, err = {{$f.Name}}.Value()
		if temp == nil {{"{"}}
		e.{{$f.Name}} = 0		
		{{"}else{"}}
		e.{{$f.Name}} = {{$f.Type}}(temp.(int64))
		{{"}"}}
		if err != nil{
			return nil, err
		}
		{{end}}
		result = append(result, e)
	{{"}"}}

	err = rows.Err()
	if err != nil {{"{"}}
		return nil, err
	{{"}"}}

	return result, nil
}
{{end}}

`))
