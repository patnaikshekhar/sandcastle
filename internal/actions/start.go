package actions

import (
	"log"
	"os"

	"github.com/lima-vm/lima/pkg/instance"
	"github.com/urfave/cli/v2"
)

func Start(c *cli.Context) error {
	log.Printf("Starting sandbox")
	yBytes, err := os.ReadFile("sample.yaml")
	if err != nil {
		return err
	}

	_, err = instance.Create(c.Context, "default", yBytes, false)
	if err != nil {
		return err
	}
	return nil
}
