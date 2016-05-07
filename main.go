package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
	_ "github.com/lib/pq"
)

type pgAccess struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DbName   string `yaml:"database"`
	UseSSL   bool   `yaml:"use_ssl,omitempty"`
}

type AuthError struct {
	Username string
}

func (e AuthError) Error() string {
	return fmt.Sprintf("Auth failed: %s", e.Username)
}

var (
	connStrTpl = "user=%s password=%s host=%s port=%d dbname=%s connect_timeout=%d sslmode=%s"
	checkCreds = "SELECT id FROM users " +
		"WHERE email=$1 AND access_token=$2 AND is_active='t'"
)

func checkErr(err error) {
	switch err.(type) {
		case nil:
			return
		case AuthError:
			log.Fatal(err)
			os.Exit(1)
		default:
			panic(err)
	}
}

func getDBConfig(configFile *string) (dbConfig *pgAccess, err error) {
	configStream, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(configStream, &dbConfig)

	return
}

func readCredentials(credentialsFile *string) (creds []string, err error) {
	credFp, err := os.Open(*credentialsFile)
	defer credFp.Close()

	if err != nil {
		return
	}

	credScanner := bufio.NewScanner(credFp)

	for credScanner.Scan() {
		creds = append(creds, credScanner.Text())
	}
	return
}

func checkCredentials(dbConfig *pgAccess, creds []string) (error) {
	connStr := fmt.Sprintf(connStrTpl, dbConfig.Username, dbConfig.Password,
		dbConfig.Host, dbConfig.Port, dbConfig.DbName, 3, "disable")

	db, err := sql.Open("postgres", connStr)
	defer db.Close()
	if err != nil { return err }

	rows, err := db.Query(checkCreds, creds[0], creds[1])
	defer rows.Close()
	if err != nil { return err }

	for rows.Next() {
		var Id int
		err := rows.Scan(&Id)
		if err != nil { return err }

		if Id > 0 {
			return nil
		}
	}

	return AuthError{creds[0]}
}

func main() {
	configFile := flag.String("config", "./config.yml", "Specify configuration to use")
	credentialsFile := flag.String("credentials", "", "Specify file to read credentials from")

	if len(os.Args) < 2 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if flag.Parsed() {

		dbConfig, err := getDBConfig(configFile)
		checkErr(err)

		creds, err := readCredentials(credentialsFile)
		checkErr(err)

		err = checkCredentials(dbConfig, creds)
		checkErr(err)

		if err == nil {
			log.Printf("Authenticated: %s", creds[0])
			os.Exit(0)
		}
	}
	os.Exit(1)
}
