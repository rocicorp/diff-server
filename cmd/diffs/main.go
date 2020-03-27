package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"runtime/trace"
	"syscall"

	"github.com/attic-labs/noms/go/spec"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	servepkg "roci.dev/diff-server/serve"
	"roci.dev/diff-server/serve/accounts"
	rlog "roci.dev/diff-server/util/log"
	"roci.dev/diff-server/util/version"
)

const (
	dropWarning = "This command deletes an entire database and its history. This operations is not recoverable. Proceed? y/n\n"
)

type opt struct {
	Args     []string
	OutField string
}

func main() {
	impl(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Exit)
}

func impl(args []string, in io.Reader, out, errs io.Writer, exit func(int)) {
	app := kingpin.New("diffs", "")
	app.ErrorWriter(errs)
	app.UsageWriter(errs)
	app.Terminate(exit)

	v := app.Flag("version", "Prints the version of diffs - same as the 'version' command.").Short('v').Bool()
	sps := app.Flag("db", "The prefix to use for databases managed. Both local and remote databases are supported. For local databases, specify a directory path to store the database in. For remote databases, specify the http(s) URL to the database (usually https://serve.replicate.to/<mydb>).").PlaceHolder("/path/to/db").Required().String()
	tf := app.Flag("trace", "Name of a file to write a trace to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)
	cpu := app.Flag("cpu", "Name of file to write CPU profile to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)

	app.PreAction(func(pc *kingpin.ParseContext) error {
		if *v {
			fmt.Println(version.Version())
			exit(0)
		}
		return nil
	})

	stopCPUProfile := func() {
		if *cpu != nil {
			pprof.StopCPUProfile()
		}
	}
	stopTrace := func() {
		if *tf != nil {
			trace.Stop()
		}
	}
	defer stopTrace()
	defer stopCPUProfile()

	app.Action(func(pc *kingpin.ParseContext) error {
		if pc.SelectedCommand == nil {
			return nil
		}

		// Init logging
		logOptions := rlog.Options{}
		if pc.SelectedCommand.Model().Name == "serve" {
			logOptions.Prefix = true
		}
		rlog.Init(errs, logOptions)

		if *tf != nil {
			err := trace.Start(*tf)
			if err != nil {
				return err
			}
		}
		if *cpu != nil {
			err := pprof.StartCPUProfile(*tf)
			if err != nil {
				return err
			}
		}
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			stopTrace()
			stopCPUProfile()
			os.Exit(1)
		}()

		return nil
	})

	serve(app, sps, errs)

	if len(args) == 0 {
		app.Usage(args)
		return
	}

	_, err := app.Parse(args)
	if err != nil {
		fmt.Fprintln(errs, err.Error())
		exit(1)
	}
}

type gsp func() (spec.Spec, error)

func serve(parent *kingpin.Application, sps *string, errs io.Writer) {
	kc := parent.Command("serve", "Starts a local diff-server.")
	port := kc.Flag("port", "The port to run on").Default("7001").Int()
	overrideClientViewURL := parent.Flag("clientview", "URL to always use for client view eg 'https://example.com/clientview'").Default("").String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		ps := fmt.Sprintf(":%d", *port)
		log.Printf("Listening on %s...", ps)
		s := servepkg.NewService(*sps, accounts.Accounts(), *overrideClientViewURL, servepkg.ClientViewGetter{})
		return http.ListenAndServe(fmt.Sprintf(":%d", *port), s)
	})
}
