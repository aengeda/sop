package options

import (
	"context"
	"flag"
)

// Injectable defines a set of flag based options to be parsed and injected
// into the operator contexts
type Injectable interface {
	// AddFlags adds the injectable's flags to karpenter's flag set
	AddFlags(*flag.FlagSet)
	// Parse parses the flag set and handles any required post-processing on
	// the flags
	Parse(*flag.FlagSet, ...string) error
	// ToContext injects the callee into the given context
	ToContext(context.Context) context.Context
}
