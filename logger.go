package libring

import (
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("libring")
var logFormat = logging.MustStringFormatter("[db] %{level} %{color}%{message}%{color:reset} [%{shortfile}]")

func init() {
	logging.SetFormatter(logFormat)
	logging.SetLevel(logging.DEBUG, "libring")
}
