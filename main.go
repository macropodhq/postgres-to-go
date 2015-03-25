package main

import (
	"database/sql"
	"fmt"
	"go/format"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/serenize/snaker"
)

var typeMap = map[string]string{
	"boolean|NO":                      "bool",
	"integer|NO":                      "int64",
	"character varying|NO":            "string",
	"text|NO":                         "string",
	"json|NO":                         "map[string]interface{}",
	"timestamp without time zone|NO":  "time.Time",
	"boolean|YES":                     "sql.NullBool",
	"integer|YES":                     "sql.NullInt64",
	"character varying|YES":           "sql.NullString",
	"text|YES":                        "sql.NullString",
	"json|YES":                        "map[string]interface{}",
	"timestamp without time zone|YES": "sql.NullTime",
}

var helpTemplate = `
usage:
	%s <connection_string>

example:
	%s 'dbname=example sslmode=disable'

`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, helpTemplate, os.Args[0], os.Args[0])
		os.Exit(1)
	}

	db, err := sql.Open("postgres", os.Args[1])
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tq, err := db.Prepare("select table_name from information_schema.tables where table_schema = 'public' order by table_name asc")
	if err != nil {
		panic(err)
	}

	trows, err := tq.Query()
	if err != nil {
		panic(err)
	}

	var tableNames []string

	for trows.Next() {
		var tableName string
		if err := trows.Scan(&tableName); err != nil {
			panic(err)
		}

		tableNames = append(tableNames, tableName)
	}

	var s string

	s += fmt.Sprintf("package models\n\nimport (\n\"database/sql\"\n\"time\"\n)\n\n")

	for _, tableName := range tableNames {
		s += fmt.Sprintf("type %s struct {\n", snaker.SnakeToCamel(tableName))

		cq, err := db.Prepare("select column_name, data_type, is_nullable from information_schema.columns where table_name = $1")
		if err != nil {
			panic(err)
		}

		crows, err := cq.Query(tableName)
		if err != nil {
			panic(err)
		}

		for crows.Next() {
			var columnName, dataType, isNullable string

			if err := crows.Scan(&columnName, &dataType, &isNullable); err != nil {
				panic(err)
			}

			camelColumnName := snaker.SnakeToCamel(columnName)

			s += fmt.Sprintf("  %s %s `json:\"%s\"`\n", camelColumnName, typeMap[dataType+"|"+isNullable], strings.ToLower(camelColumnName[0:1])+camelColumnName[1:])
		}

		s += fmt.Sprintf("}\n\n")
	}

	d, err := format.Source([]byte(s))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", d)
}
