package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"runtime/trace"
	"syscall"
	"time"

	"github.com/attic-labs/noms/go/spec"
	"github.com/gorilla/mux"
	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"roci.dev/diff-server/account"
	servepkg "roci.dev/diff-server/serve"
	"roci.dev/diff-server/serve/signup"
	"roci.dev/diff-server/util/log"
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
	zlog.Logger = zlog.Output(zl.ConsoleWriter{Out: os.Stderr, TimeFormat: "02 Jan 06 15:04:05.000 -0700"})
	l := log.Default()

	app := kingpin.New("diffs", "")
	app.ErrorWriter(errs)
	app.UsageWriter(errs)
	app.Terminate(exit)

	v := app.Flag("version", "Prints the version of diffs - same as the 'version' command.").Short('v').Bool()
	sps := app.Flag("db", "The prefix to use for databases managed. Both local and remote databases are supported. For local databases, specify a directory path to store the database in. For remote databases, specify the http(s) URL to the database (usually https://serve.replicate.to/<mydb>).").PlaceHolder("/path/to/db").Required().String()
	ads := app.Flag("account-db", "Prefix for the account database. Both local and remote databases are supported. For local databases, this is a directory path.").PlaceHolder("/path/to/db").Required().String()
	tf := app.Flag("trace", "Name of a file to write a trace to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)
	cpu := app.Flag("cpu", "Name of file to write CPU profile to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)
	lv := app.Flag("log-level", "Verbosity of logging to print").Default("info").Enum("error", "info", "debug")

	app.PreAction(func(pc *kingpin.ParseContext) error {
		if *v {
			fmt.Println(version.Version())
			exit(0)
		}
		return log.SetGlobalLevelFromString(*lv)
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

	serve(app, sps, ads, errs, l)

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

func serve(parent *kingpin.Application, sps *string, ads *string, errs io.Writer, l zl.Logger) {
	kc := parent.Command("serve", "Starts a local diff-server.")
	port := kc.Flag("port", "The port to run on").Default("7001").Int()
	enableInject := kc.Flag("enable-inject", "Enable /inject endpoint which writes directly to the database for testing").Default("false").Bool()
	disableAuth := parent.Flag("disable-auth", "Disable auth check in pull").Default("false").Bool()
	kc.Action(func(_ *kingpin.ParseContext) error {
		l.Info().Msgf("Listening on %d...", *port)

		// Set up diffserver service (pull, inject, etc).
		if *disableAuth {
			l.Info().Msg("Pull auth check disabled")
		}

		accountDB, err := account.NewDB(*ads)
		if err != nil {
			panic(err)
		}

		svc := servepkg.NewService(*sps, account.MaxASClientViewHosts, accountDB, *disableAuth, servepkg.ClientViewGetter{}, *enableInject)
		mux := mux.NewRouter()
		servepkg.RegisterHandlers(svc, mux)

		// Set up signup service.
		tmpl := template.Must(signup.ParseTemplates(signup.Templates()))
		service := signup.NewService(l, tmpl, *ads)
		signup.RegisterHandlers(service, mux)

		server := &http.Server{
			Addr:         fmt.Sprintf(":%d", *port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		return server.ListenAndServe()
	})
}
