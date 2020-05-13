package log

import (
	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"roci.dev/diff-server/util/gid"
)

func Default() zl.Logger {
	return zlog.Hook(gid.ZLogHook{}).With().Timestamp().Logger()
}
