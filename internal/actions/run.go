package actions

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Run(c *cli.Context) error {
	fmt.Println("Running code")
	return nil
}
