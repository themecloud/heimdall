package outputs

import (
	"github.com/themecloud/heimdall"
)

type Creator func() heimdall.Output

var Outputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Outputs[name] = creator
}
