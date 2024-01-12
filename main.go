package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
)

type DatabaseResourceInputs struct {
	Provider     string
	DatabaseName string
	User         string
	Password     string
}

func (inputs DatabaseResourceInputs) ParseConnString() (string, error) {
	if inputs.Provider == "mysql" {
		return "admin:admin@tcp(localhost:3306)/admin", nil
	} else {
		return pq.ParseURL("postgres://admin:admin@localhost:5432?sslmode=disable")
	}
}

func main() {

	inputs := DatabaseResourceInputs{
		Provider:     "mysql",
		DatabaseName: "test_db",
		User:         "test_usr",
		Password:     "test_pwd",
	}

	connString, err := inputs.ParseConnString()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open(inputs.Provider, connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = createDatabase(db, &inputs)
	if err != nil {
		log.Fatal(err)
		return
	}

	// err = createUser(db, &inputs)
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }

	log.Printf("Successfully created database '%s' and user '%s'.\n", inputs.DatabaseName, inputs.User)
}

func parseTemplate(contentTemplate string, inputs *DatabaseResourceInputs) (*bytes.Buffer, error) {
	runner, err := template.New("template").Parse(contentTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	var query bytes.Buffer
	err = runner.Execute(&query, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return &query, err
}

func createDatabase(db *sql.DB, inputs *DatabaseResourceInputs) error {
	query, err := parseDatabaseTemplate(inputs)
	result, err := execQuery(db, query)
	if err != nil {
		return err
	}
	log.Println(result)
	return err
}

func createUser(db *sql.DB, inputs *DatabaseResourceInputs) error {
	query, err := parseUserTemplate(inputs)
	if err != nil {
		return err
	}
	result, err := execQuery(db, query)
	if err != nil {
		return err
	} else {
		log.Println(result)
	}
	return err
}

func execQuery(db *sql.DB, query *bytes.Buffer) (sql.Result, error) {
	result, err := db.Exec(query.String())
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return result, nil
}

func parseDatabaseTemplate(inputs *DatabaseResourceInputs) (*bytes.Buffer, error) {
	var template string
	if inputs.Provider == "mysql" {
		template = `
		GRANT ALL PRIVILEGES ON {{ .DatabaseName }}.* TO '{{ .User }}'@'localhost'
		CREATE DATABASE {{ .Database }};`
	} else {
		template = "CREATE DATABASE {{ .DatabaseName }};"
	}
	return parseTemplate(template, inputs)
}

func parseUserTemplate(inputs *DatabaseResourceInputs) (*bytes.Buffer, error) {
	var template string
	if inputs.Provider == "mysql" {
		template = `
		CREATE USER '{{ .User }}'@'localhost' IDENTIFIED BY '{{ .Password }}';
		GRANT ALL PRIVILEGES ON {{ .DatabaseName }}.* TO '{{ .User }}'@'localhost';
		REVOKE ALL PRIVILEGES ON {{ .DatabaseName }}.* FROM PUBLIC;`
	} else {
		template = `
		CREATE USER {{ .User }} WITH PASSWORD '{{ .Password }}';
		GRANT ALL PRIVILEGES ON DATABASE {{ .DatabaseName }} TO {{ .User }};
		REVOKE ALL ON DATABASE {{ .DatabaseName }} FROM PUBLIC;`
	}
	return parseTemplate(template, inputs)
}
