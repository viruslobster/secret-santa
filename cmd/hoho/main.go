package main

import (
	"fmt"
	"log"
	"os"

	"github.com/viruslobster/secret-santa/migrations"
	"github.com/viruslobster/secret-santa/server"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "migrate" {
		fmt.Fprintln(os.Stderr, "usage: hoho migrate")
		os.Exit(1)
	}
	if err := migrate(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Migrations complete.")
}

func migrate() error {
	store, err := server.NewSQLiteStore()
	if err != nil {
		return err
	}
	defer store.Close()
	return migrations.Run(store.Db)
}
