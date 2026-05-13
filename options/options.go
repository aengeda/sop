package options

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/samber/lo"
)

type optionsKey struct{}

var (
	validLogLevels = []string{"", "debug", "info", "error"}

	Injectables = []Injectable{&Options{}}
)

type Options struct {
	DisableLeaderElection   bool
	LeaderElectionName      string
	LeaderElectionNamespace string

	EnableProfiling bool
	HealthProbePort int
	MetricsPort     int
	LogLevel        string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	// General Controller Settings
	fs.BoolVar(&o.EnableProfiling, "enable-profiling", false, "Enable the profiling on the profiling endpoint")
	fs.IntVar(&o.HealthProbePort, "health-probe-port", 8081, "The port the health probe endpoint binds to for reporting controller health")
	fs.IntVar(&o.MetricsPort, "metrics-port", 8080, "The port the metric endpoint binds to for operating metrics about the controller itself")
	fs.StringVar(&o.LogLevel, "log-level", "info", "Log verbosity level. Can be one of 'debug', 'info', or 'error'")

	// Leader election settings
	fs.StringVar(&o.LeaderElectionName, "leader-election-name", "operator-leader-election", "Leader election name to create and monitor the lease if running outside the cluster")
	fs.StringVar(&o.LeaderElectionNamespace, "leader-election-namespace", "", "Leader election namespace to create and monitor the lease if running outside the cluster")
	fs.BoolVar(&o.DisableLeaderElection, "disable-leader-election", false, "Disable the leader election client before executing the main loop. Disable when running replicated components for high availability is not desired.")
}

func (o *Options) Parse(fs *flag.FlagSet, args ...string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		return fmt.Errorf("parsing flags, %w", err)
	}
	if !lo.Contains(validLogLevels, o.LogLevel) {
		return fmt.Errorf("validating cli flags, invalid log-level %q", o.LogLevel)
	}
	return nil
}

func (o *Options) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, o)
}

func ToContext(ctx context.Context, opts *Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, opts)
}

func FromContext(ctx context.Context) *Options {
	retval := ctx.Value(optionsKey{})
	if retval == nil {
		// This is a developer error if this happens, so we should panic
		panic("options doesn't exist in context")
	}
	return retval.(*Options)
}
