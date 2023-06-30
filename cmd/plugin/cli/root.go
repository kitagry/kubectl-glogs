package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kitagry/kubectl-glogs/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ConfigFlags *plugin.ConfigFlags

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "glogs",
		Short: "Print the logs for all namespace resource or specified resource.",
		Long:  `Print the logs for all namespace resource or specified resource. You should set kubectl context. https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl#view_the_current_context_for_kubectl`,
		Example: `  # Return logs in the namespaces
  kubectl glogs

  # Return logs of the specified CronJob
  kubectl glogs cronjob nginx
  kubectl glogs cj nginx

  # Return logs of the specified Deployment
  kubectl glogs deployment nginx
  kubectl glogs deploy nginx

  # Return logs of the multiple resources
  kubectl glogs deploy/nginx pods/item

  # Return logs in 2 hours. (default 30m)
  kubectl glogs --duration 2h

  # Return logs with filter
  kubectl glogs --filter 'severity = "ERROR"'

  # Open browser
  kubectl glogs --web`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := plugin.RunPlugin(ConfigFlags, args); err != nil {
				return errors.Unwrap(err)
			}

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	ConfigFlags = plugin.NewConfigFlags()
	ConfigFlags.AddFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
