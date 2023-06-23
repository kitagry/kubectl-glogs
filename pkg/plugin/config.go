package plugin

import (
	"time"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ConfigFlags struct {
	Kubernetes *genericclioptions.ConfigFlags

	// Log time duration
	Duration time.Duration

	// Add Cloud logging filter. https://cloud.google.com/logging/docs/view/building-queries
	Filter string

	// Open web browser
	Web bool
}

func NewConfigFlags() *ConfigFlags {
	return &ConfigFlags{
		Kubernetes: &genericclioptions.ConfigFlags{
			Namespace:  toPtr(""),
			Context:    toPtr(""),
			KubeConfig: toPtr(""),
		},
	}
}

func (c *ConfigFlags) AddFlags(flags *pflag.FlagSet) {
	c.Kubernetes.AddFlags(flags)

	flags.DurationVar(&c.Duration, "duration", 30*time.Minute, "Log time duration")
	flags.StringVar(&c.Filter, "filter", "", "Add Cloud logging filter. https://cloud.google.com/logging/docs/view/building-queries")
	flags.BoolVar(&c.Web, "web", false, "Open web browser")
}

func toPtr[T any](s T) *T {
	return &s
}
