package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/dpkg"
	"github.com/loilo-inc/bistro/pkg/ldd"
	"github.com/urfave/cli/v3"
)

type App interface {
	Run(ctx context.Context, args []string) error
}

type app struct {
	NewInput
}

var _ App = (*app)(nil)

type NewInput struct {
	Version string
	Commit  string
	Date    string
	Cwd     string
	Stdout  io.Writer
	Stderr  io.Writer
}

func New(input NewInput) App {
	return &app{NewInput: input}
}

func load(di bistro.DI) error {
	if err := di.Query.Load(); err != nil {
		return err
	}
	if err := di.Ldd.Load(); err != nil {
		return err
	}
	return nil
}

func (c *app) command() *cli.Command {
	return &cli.Command{
		Name:    "bistro",
		Usage:   "Build a distroless root filesystem",
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", c.Version, c.Commit, c.Date),
		Commands: []*cli.Command{
			c.scanCommand(),
			c.bundleCommand(),
		},
	}
}

func (c *app) Run(ctx context.Context, args []string) error {
	return c.command().Run(ctx, args)
}

func (c *app) depsFromFs(fss fs.FS) bistro.DI {
	ldd := ldd.NewResolver(ldd.ResolverOptions{
		LdLibraryPath: os.Getenv("LD_LIBRARY_PATH"),
		Cwd:           c.Cwd,
		FS:            fss,
	})
	query := dpkg.NewQuery(fss)
	logger := bistro.NewLogger(c.stdout(), c.stderr())
	return bistro.NewDI(logger, ldd, query, fss)
}

func (c *app) stdout() io.Writer {
	if c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
}

func (c *app) stderr() io.Writer {
	if c.Stderr != nil {
		return c.Stderr
	}
	return os.Stderr
}

func (c *app) printf(format string, args ...any) {
	fmt.Fprintf(c.stdout(), format, args...)
}

func (c *app) errorf(format string, args ...any) {
	fmt.Fprintf(c.stderr(), format, args...)
}

func (c *app) absPath(p string) string {
	if path.IsAbs(p) {
		return p
	}
	return path.Join(c.Cwd, p)
}
