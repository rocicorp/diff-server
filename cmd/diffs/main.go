package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"runtime/trace"
	"syscall"

	"github.com/attic-labs/noms/go/spec"
	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	servepkg "roci.dev/diff-server/serve"
	"roci.dev/diff-server/serve/accounts"
	"roci.dev/diff-server/util/log"
	"roci.dev/diff-server/util/loghttp"
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
	zlog.Logger = zlog.Output(zl.ConsoleWriter{Out: os.Stderr})
	l := log.Default()

	app := kingpin.New("diffs", "")
	app.ErrorWriter(errs)
	app.UsageWriter(errs)
	app.Terminate(exit)

	v := app.Flag("version", "Prints the version of diffs - same as the 'version' command.").Short('v').Bool()
	sps := app.Flag("db", "The prefix to use for databases managed. Both local and remote databases are supported. For local databases, specify a directory path to store the database in. For remote databases, specify the http(s) URL to the database (usually https://serve.replicate.to/<mydb>).").PlaceHolder("/path/to/db").Required().String()
	tf := app.Flag("trace", "Name of a file to write a trace to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)
	cpu := app.Flag("cpu", "Name of file to write CPU profile to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)
	lv := app.Flag("log-level", "Verbosity of logging to print").Default("info").Enum("error", "info", "debug")

	app.PreAction(func(pc *kingpin.ParseContext) error {
		if *v {
			fmt.Println(version.Version())
			exit(0)
		}

		switch *lv {
		case "debug":
			zl.SetGlobalLevel(zl.DebugLevel)
		case "info":
			zl.SetGlobalLevel(zl.InfoLevel)
		case "error":
			zl.SetGlobalLevel(zl.ErrorLevel)
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

	serve(app, sps, errs, l)

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

func serve(parent *kingpin.Application, sps *string, errs io.Writer, l zl.Logger) {
	kc := parent.Command("serve", "Starts a local diff-server.")
	port := kc.Flag("port", "The port to run on").Default("7001").Int()
	enableInject := kc.Flag("enable-inject", "Enable /inject endpoint which writes directly to the database for testing").Default("false").Bool()
	overrideClientViewURL := parent.Flag("client-view", "URL to use for all accounts' Client View").PlaceHolder("http://localhost:8000/replicache-client-view").Default("").String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		l.Info().Msgf("Listening on %d...", *port)
		s := servepkg.NewService(*sps, accounts.Accounts(), *overrideClientViewURL, servepkg.ClientViewGetter{}, *enableInject)
		return http.ListenAndServe(fmt.Sprintf(":%d", *port), loghttp.Wrap(s, l))
	})
}
