package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/viruslobster/secret-santa/migrations"
	"github.com/viruslobster/secret-santa/server"
)

const usage = `usage:
  hoho migrate
  hoho create-server [comma separated names]
`

func main() {
	var arg_len = len(os.Args)

	if arg_len == 2 && os.Args[1] == "migrate" {
		if err := migrate(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations complete.")
		return
	}
	if arg_len == 3 && os.Args[1] == "create-server" {
		if err := createServer(os.Args[2]); err != nil {
			log.Fatal(err)
		}
		return
	}
	fmt.Fprint(os.Stderr, usage)
	os.Exit(1)
}

func migrate() error {
	store, err := server.NewSQLiteStore()
	if err != nil {
		return err
	}
	defer store.Close()
	return migrations.Run(store.Db)
}

func createServer(names_string string) error {
	store, err := server.NewSQLiteStore()
	if err != nil {
		return err
	}
	defer store.Close()

	names := parseNames(names_string)
	pairs, err := pairNames(names)
	if err != nil {
		return err
	}
	_, err = store.CreateServer(names, pairs)
	if err != nil {
		return err
	}
	return nil
}

func parseNames(names_string string) []string {
	names := strings.Split(names_string, ",")
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		trimmed = append(trimmed, strings.Trim(name, " "))
	}
	return names
}

func pairNames(names []string) ([]int, error) {
	if len(names) <= 1 {
		return nil, errors.New("need at least two names")
	}

	pairing := make([]int, len(names))
	for i := range pairing {
		pairing[i] = i
	}
	rand.Shuffle(len(pairing), func(i, j int) {
		var swap = pairing[i]
		pairing[i] = pairing[j]
		pairing[j] = swap
	})
	return pairing, nil
}
