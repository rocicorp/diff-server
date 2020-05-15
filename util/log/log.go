package log

import (
	"fmt"

	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"roci.dev/diff-server/util/gid"
)

func Default() zl.Logger {
	return zlog.Hook(gid.ZLogHook{}).With().Timestamp().Logger()
}

func SetGlobalLevelFromString(s string) error {
	switch s {
	case "debug":
		zl.SetGlobalLevel(zl.DebugLevel)
	case "info":
		zl.SetGlobalLevel(zl.InfoLevel)
	case "error":
		zl.SetGlobalLevel(zl.ErrorLevel)
	default:
		return fmt.Errorf("Unknown log level: %s", s)
	}
	return nil
}
