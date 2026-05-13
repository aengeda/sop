package injection

import (
	"context"
	"flag"
	"os"

	"github.com/samber/lo"

	"github.com/aengeda/simple-operator-pattern/options"
)

type controllerNameKeyType struct{}

var controllerNameKey = controllerNameKeyType{}

func WithControllerName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, controllerNameKey, name)
}

func GetControllerName(ctx context.Context) string {
	name := ctx.Value(controllerNameKey)
	if name == nil {
		return ""
	}
	return name.(string)
}

func WithOptionsOrDie(ctx context.Context, opts ...options.Injectable) context.Context {
	fs := flag.NewFlagSet("operator", flag.ContinueOnError)
	for _, opt := range opts {
		opt.AddFlags(fs)
	}
	for _, opt := range opts {
		lo.Must0(opt.Parse(fs, os.Args[1:]...))
	}
	for _, opt := range opts {
		ctx = opt.ToContext(ctx)
	}
	return ctx
}
