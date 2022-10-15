package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type LogConfig struct {
	Resources []resource
	ProjectID string
	Location  string
	Cluster   string
	Namespace string
}

type GoogleCloudLogger struct {
	clientset *kubernetes.Clientset
	config    *LogConfig

	duration time.Duration
}

func NewGoogleCloudLogger(configFlags *genericclioptions.ConfigFlags, duration time.Duration, args []string) (*GoogleCloudLogger, error) {
	config, err := buildLogConfig(configFlags)
	if err != nil {
		return nil, err
	}

	config.Resources, err = getResources(args)
	if err != nil {
		return nil, err
	}

	restConfig, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &GoogleCloudLogger{
		clientset: clientset,
		config:    config,
		duration:  duration,
	}, nil
}

func buildLogConfig(configFlags *genericclioptions.ConfigFlags) (*LogConfig, error) {
	config, err := configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return nil, err
	}

	projectID, location, clusterName, ok := extractGKEInfo(config)
	if !ok {
		return nil, fmt.Errorf("Failed to extract gke info")
	}

	namespace := extractNamespace(configFlags, config)

	return &LogConfig{
		ProjectID: projectID,
		Location:  location,
		Cluster:   clusterName,
		Namespace: namespace,
	}, nil
}

func extractGKEInfo(config clientcmdapi.Config) (projectID, location, cluster string, ok bool) {
	clusterName := config.Contexts[config.CurrentContext].Cluster

	// clusterName might be `gke_${projectID}_${location}_${clusterName}`
	splitted := strings.Split(clusterName, "_")
	if len(splitted) != 4 {
		return "", "", "", false
	}

	return splitted[1], splitted[2], splitted[3], true
}

func extractNamespace(configFlags *genericclioptions.ConfigFlags, config clientcmdapi.Config) string {
	if configFlags.Namespace != nil && *configFlags.Namespace != "" {
		return *configFlags.Namespace
	}
	return config.Contexts[config.CurrentContext].Namespace
}

func (g *GoogleCloudLogger) Gather(ctx context.Context, entryChan chan<- *logging.Entry) error {
	client, err := logadmin.NewClient(ctx, fmt.Sprintf("projects/%s", g.config.ProjectID))
	if err != nil {
		return err
	}
	defer client.Close()

	// TODO: editable from args
	defaultTimestamp := time.Now().Add(-g.duration)

	filter := fmt.Sprintf(`resource.type = "k8s_container"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
resource.labels.namespace_name="%s"
timestamp >= "%s"`, g.config.ProjectID, g.config.Location, g.config.Cluster, g.config.Namespace, defaultTimestamp.Format(time.RFC3339))

	resourceFilter, err := g.filterResources()
	if err != nil {
		return err
	}
	if resourceFilter != "" {
		filter += "\n" + resourceFilter
	}

	iter := client.Entries(ctx, logadmin.Filter(filter))
	for {
		entry, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		entryChan <- entry
	}

	close(entryChan)
	return nil
}

func (g *GoogleCloudLogger) filterResources() (string, error) {
	if len(g.config.Resources) == 0 {
		return "", nil
	}

	results := make([]string, 0, len(g.config.Resources))
	for _, r := range g.config.Resources {
		switch r.Type {
		case Deployment:
			filter, err := g.filterDeployments(r)
			if err != nil {
				return "", err
			}
			results = append(results, filter)
		case CronJob:
			results = append(results, g.filterCronJobs(r))
		case Job:
			results = append(results, g.filterJobs(r))
		case Pod:
			results = append(results, g.filterPods(r))
		}
	}
	return fmt.Sprintf("(%s)", strings.Join(results, " OR ")), nil
}

func (g *GoogleCloudLogger) filterDeployments(r resource) (string, error) {
	deployment, err := g.clientset.AppsV1().Deployments(g.config.Namespace).Get(r.Name, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get deployment: %w", err)
	}

	if deployment.Spec.Selector == nil {
		return "", fmt.Errorf("deployment doesn't have selector")
	}

	for key, value := range deployment.Spec.Selector.MatchLabels {
		return fmt.Sprintf(`labels.k8s-pod/%s="%s"`, key, value), nil
	}

	return "", fmt.Errorf("deployment doesn't have selector")
}

func (g *GoogleCloudLogger) filterCronJobs(r resource) string {
	return fmt.Sprintf(`labels.k8s-pod/job-name:"%s-"`, r.Name)
}

func (g *GoogleCloudLogger) filterJobs(r resource) string {
	return fmt.Sprintf(`resource.labels.pod_name:"%s-"`, r.Name)
}

func (g *GoogleCloudLogger) filterPods(r resource) string {
	return fmt.Sprintf(`resource.labels.pod_name="%s"`, r.Name)
}
