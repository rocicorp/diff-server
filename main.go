package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type command struct {
	cmd     *kingpin.CmdClause
	handler func()
}

func main() {
	app := kingpin.New("replicant", "Conflict-Free Replicated Database")

	commands := []command{
		put(app),
		get(app),
	}

	selected := kingpin.MustParse(app.Parse(os.Args[1:]))
	for _, c := range commands {
		if selected == c.cmd.FullCommand() {
			c.handler()
			break
		}
	}
}

func put(app *kingpin.Application) (c command) {
	c.cmd = app.Command("put", "Insert or replace an object")
	val := c.cmd.Arg("val", "JSON-encoded object to write. Must have an _id field.").Required().String()

	c.handler = func() {
		fmt.Printf("%s: %+v\n", val)
	}

	return c
}

func get(app *kingpin.Application) (c command) {
	c.cmd = app.Command("get", "Reads an object")
	id := c.cmd.Arg("id", "ID of object to get").Required().String()

	c.handler = func() {
		fmt.Printf("%s: %+s\n", id)
	}

	return c
}
