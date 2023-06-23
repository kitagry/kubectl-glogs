# glogs kubectl

A `kubectl` plugin to logging from [GCP Cloud Logging](https://cloud.google.com/logging).

## Quick Start

```bash
git clone https://github.com/kitagry/kubectl-glogs.git
cd kubectl-glogs
make bin
cp ./bin/glogs ~/go/bin/kubectl-glogs
```

## Usage

```bash
# Change context to your GKE cluster
kubectl ctx

# Change namespace in which you want to see logs.
kubectl ns

# All logs in the namespace
kubectl glogs

# Look at specified logs
kubectl glogs cronjob CRONJOB_NAME
kubectl glogs deploy DEPLOYMENT_NAME
kubectl glogs jobs JOB_NAME

# Open browser
kubectl glogs --web cronjob CRONJOB_NAME
```

## TODO

- [ ] krew setting
