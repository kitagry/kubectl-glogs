package plugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/fatih/color"
)

type resource struct {
	Type ResourceType
	Name string
}

type ResourceType string

const (
	Deployment       ResourceType = "Deployment"
	CronJob          ResourceType = "CronJob"
	Job              ResourceType = "Job"
	Pod              ResourceType = "Pod"
	ArgoWorkflow     ResourceType = "Workflow"
	ArgoCronWorkflow ResourceType = "CronWorkflow"
)

func RunPlugin(configFlags *ConfigFlags, args []string) error {
	ctx := context.Background()

	logger, err := NewGoogleCloudLogger(configFlags, args)
	if err != nil {
		return err
	}

	if configFlags.Web {
		query, err := logger.BuildQuery(true)
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}
		cloudLoggingURL, err := url.JoinPath("https://console.cloud.google.com", "logs", fmt.Sprintf("query;query=%s;cursorTimestamp=%s", url.PathEscape(query), time.Now().Format(time.RFC3339Nano)))
		if err != nil {
			return fmt.Errorf("failed to joinpath: %w", err)
		}
		cloudLoggingURL += "?" + url.Values{"project": []string{logger.config.ProjectID}}.Encode()
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("xdg-open", cloudLoggingURL)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", cloudLoggingURL)
		case "darwin":
			cmd = exec.Command("open", cloudLoggingURL)
		default:
			return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
		}

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to command `%s`: %w", cmd.String(), err)
		}
		return nil
	}

	ch := make(chan *logging.Entry, 100)
	go logger.Gather(ctx, ch)

	w := bufio.NewWriterSize(os.Stdout, 1<<15)
	defer w.Flush()

	for e := range ch {
		var err error
		switch e.Severity {
		case logging.Error:
			_, err = color.New(color.FgRed).Fprintln(w, e.Payload)
		case logging.Warning:
			_, err = color.New(color.FgYellow).Fprintln(w, e.Payload)
		default:
			_, err = w.WriteString(fmt.Sprintf("%s\n", e.Payload))
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func getResources(args []string) ([]resource, error) {
	args = separateArgs(args)
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("args should be even")
	}

	result := make([]resource, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		resourceType, err := getResourceType(args[i])
		if err != nil {
			return nil, err
		}
		result = append(result, resource{
			Type: resourceType,
			Name: args[i+1],
		})
	}

	return result, nil
}

func separateArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, a := range args {
		result = append(result, strings.Split(a, "/")...)
	}
	return result
}

func getResourceType(s string) (ResourceType, error) {
	switch strings.ToLower(s) {
	case "deployments", "deployment", "deploy":
		return Deployment, nil
	case "cronjobs", "cronjob", "cj":
		return CronJob, nil
	case "jobs", "job":
		return Job, nil
	case "pods", "pod", "po":
		return Pod, nil
	case "workflows", "workflow", "wf":
		return ArgoWorkflow, nil
	case "cronworkflows", "cronworkflow", "cwf", "cronwf":
		return ArgoCronWorkflow, nil
	default:
		return "", fmt.Errorf(`resource type "%s" is not supported`, s)
	}
}

type nullWriter struct {
	io.Writer
}

func (n *nullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
