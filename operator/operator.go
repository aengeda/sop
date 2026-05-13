package operator

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/aengeda/sop/controller"
	"github.com/aengeda/sop/injection"
	"github.com/aengeda/sop/options"
)

var AppName = "coreweave_default_app"

// Version is the karpenter app version injected during compilation
// when using the Makefile
var Version = "unspecified"

type Operator struct {
	manager.Manager
}

func NewOperator(crds ...schema.GroupVersionKind) (context.Context, Operator) {
	// Root Context
	ctx := context.Background()

	// Logging
	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	// Client Config
	config := ctrl.GetConfigOrDie()

	config.UserAgent = "dev-cluster-operator"
	config.UserAgent = fmt.Sprintf("%s/%s", AppName, Version)

	// Manager
	mgrOpts := ctrl.Options{
		Logger:                        logger,
		LeaderElection:                !options.FromContext(ctx).DisableLeaderElection,
		LeaderElectionNamespace:       options.FromContext(ctx).LeaderElectionNamespace,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		Metrics: server.Options{
			BindAddress: fmt.Sprintf(":%d", options.FromContext(ctx).MetricsPort),
		},
		HealthProbeBindAddress: fmt.Sprintf(":%d", options.FromContext(ctx).HealthProbePort),
		BaseContext: func() context.Context {
			ctx := log.IntoContext(context.Background(), logger)
			ctx = injection.WithOptionsOrDie(ctx, options.Injectables...)
			return ctx
		},
		Controller: ctrlconfig.Controller{
			// EnableWarmup allows controllers to start their sources (watches/informers) before leader election
			// is won. This pre-populates caches and improves leader failover time. Only effective when leader
			// election is enabled, so we only set it when both conditions are true.
			EnableWarmup: lo.ToPtr(!options.FromContext(ctx).DisableLeaderElection),
		},
	}
	if options.FromContext(ctx).EnableProfiling {
		mgrOpts.Metrics.ExtraHandlers = lo.Assign(mgrOpts.Metrics.ExtraHandlers, map[string]http.Handler{
			"/debug/pprof/":             http.HandlerFunc(pprof.Index),
			"/debug/pprof/cmdline":      http.HandlerFunc(pprof.Cmdline),
			"/debug/pprof/profile":      http.HandlerFunc(pprof.Profile),
			"/debug/pprof/symbol":       http.HandlerFunc(pprof.Symbol),
			"/debug/pprof/trace":        http.HandlerFunc(pprof.Trace),
			"/debug/pprof/allocs":       pprof.Handler("allocs"),
			"/debug/pprof/heap":         pprof.Handler("heap"),
			"/debug/pprof/block":        pprof.Handler("block"),
			"/debug/pprof/goroutine":    pprof.Handler("goroutine"),
			"/debug/pprof/threadcreate": pprof.Handler("threadcreate"),
		})
	}
	mgr, err := ctrl.NewManager(config, mgrOpts)
	mgr = lo.Must(mgr, err, "failed to setup manager")

	lo.Must0(mgr.AddReadyzCheck("manager", func(req *http.Request) error {
		return lo.Ternary(mgr.GetCache().WaitForCacheSync(req.Context()), nil, fmt.Errorf("failed to sync caches"))
	}))
	lo.Must0(mgr.AddReadyzCheck("crd", func(_ *http.Request) error {
		for _, obj := range crds {
			if _, err := mgr.GetRESTMapper().RESTMapping(obj.GroupKind(), obj.Version); err != nil {
				return err
			}
		}
		return nil
	}))
	lo.Must0(mgr.AddHealthzCheck("healthz", healthz.Ping))
	lo.Must0(mgr.AddReadyzCheck("readyz", healthz.Ping))

	return ctx, Operator{
		Manager: mgr,
	}
}

func (o *Operator) WithControllers(ctx context.Context, controllers ...controller.Controller) *Operator {
	for _, c := range controllers {
		lo.Must0(c.Register(ctx, o.Manager))
	}
	return o
}

func (o *Operator) Start(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		lo.Must0(o.Manager.Start(ctx))
	}()
	wg.Wait()
}
