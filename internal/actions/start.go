package actions

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/lima-vm/lima/pkg/hostagent"
	"github.com/lima-vm/lima/pkg/hostagent/api/server"
	"github.com/lima-vm/lima/pkg/instance"
	"github.com/lima-vm/lima/pkg/store"
	"github.com/lima-vm/lima/pkg/store/filenames"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func Start(c *cli.Context) error {
	log.Printf("Starting sandbox")
	yBytes, err := os.ReadFile("sample.yaml")
	if err != nil {
		return err
	}

	name := "sandcastle"

	inst, err := store.Inspect(name)
	if err != nil {
		inst, err = instance.Create(c.Context, "sandcastle", yBytes, false)
		if err != nil {
			return err
		}
	}

	inst.CPUs = 1
	inst.Memory = 1024

	haPIDPath := filepath.Join(inst.Dir, filenames.HostAgentPID)
	if _, err := os.Stat(haPIDPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("instance %q seems running (hint: remove %q if the instance is not actually running)", inst.Name, haPIDPath)
	}
	logrus.Infof("Starting the instance %q with VM driver %q", inst.Name, inst.VMType)

	haSockPath := filepath.Join(inst.Dir, filenames.HostAgentSock)

	_, err = instance.Prepare(c.Context, inst)
	if err != nil {
		return err
	}

	haStdoutPath := filepath.Join(inst.Dir, filenames.HostAgentStdoutLog)
	haStderrPath := filepath.Join(inst.Dir, filenames.HostAgentStderrLog)
	if err := os.RemoveAll(haStdoutPath); err != nil {
		return err
	}
	if err := os.RemoveAll(haStderrPath); err != nil {
		return err
	}
	haStdoutW, err := os.Create(haStdoutPath)
	if err != nil {
		return err
	}
	// no defer haStdoutW.Close()
	// haStderrW, err := os.Create(haStderrPath)
	// if err != nil {
	// 	return err
	// }

	if _, err := os.Stat(haPIDPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("pidfile %q already exists", haPIDPath)
	}
	if err := os.WriteFile(haPIDPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		return err
	}
	defer os.RemoveAll(haPIDPath)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	stdout := &syncWriter{w: haStdoutW}
	// stderr := &syncWriter{w: haStderrW}

	ha, err := hostagent.New(name, stdout, signalCh)
	if err != nil {
		return err
	}

	backend := &server.Backend{
		Agent: ha,
	}
	r := http.NewServeMux()
	server.AddRoutes(r, backend)
	srv := &http.Server{Handler: r}
	err = os.RemoveAll(haSockPath)
	if err != nil {
		return err
	}
	l, err := net.Listen("unix", haSockPath)
	logrus.Infof("hostagent socket created at %s", haSockPath)
	if err != nil {
		return err
	}

	go func() {
		defer os.RemoveAll(haSockPath)
		defer srv.Close()
		if serveErr := srv.Serve(l); serveErr != http.ErrServerClosed {
			logrus.WithError(serveErr).Warn("hostagent API server exited with an error")
		}
	}()

	err = ha.Run(c.Context)
	if err != nil {
		return err
	}

	return nil
}

type syncer interface {
	Sync() error
}

type syncWriter struct {
	w io.Writer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	written, err := w.w.Write(p)
	if err == nil {
		if s, ok := w.w.(syncer); ok {
			_ = s.Sync()
		}
	}
	return written, err
}
