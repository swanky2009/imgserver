package g

import (
	"fmt"
	"runtime"
)

const Binary = "1.0.0-compat"

func Version() string {
	return fmt.Sprintf("v%s (built w/%s)", Binary, runtime.Version())
}
