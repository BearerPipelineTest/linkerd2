package public

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/linkerd/linkerd2/controller/api/public"
	publicPb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/linkerd/linkerd2/pkg/k8s"
	pb "github.com/linkerd/linkerd2/viz/metrics-api/gen/viz"
)

// RawPublicAPIClient creates a raw public API client with no validation.
func RawPublicAPIClient(ctx context.Context, kubeAPI *k8s.KubernetesAPI, controlPlaneNamespace string, apiAddr string) (publicPb.ApiClient, error) {
	if apiAddr != "" {
		return public.NewInternalPublicClient(controlPlaneNamespace, apiAddr)
	}

	return public.NewExternalPublicClient(ctx, controlPlaneNamespace, kubeAPI)
}

// RawVizAPIClient creates a raw viz API client with no validation.
func RawVizAPIClient(ctx context.Context, kubeAPI *k8s.KubernetesAPI, controlPlaneNamespace string, apiAddr string) (pb.ApiClient, error) {
	if apiAddr != "" {
		return public.NewInternalClient(controlPlaneNamespace, apiAddr)
	}

	return public.NewExternalClient(ctx, controlPlaneNamespace, kubeAPI)
}

// CheckPublicAPIClientOrExit builds a new Public API client and executes default status
// checks to determine if the client can successfully perform cli commands. If the
// checks fail, then CLI will print an error and exit.
func CheckPublicAPIClientOrExit(hcOptions healthcheck.Options) public.PublicAPIClient {
	hcOptions.RetryDeadline = time.Time{}
	return CheckPublicAPIClientOrRetryOrExit(hcOptions, false)
}

// CheckVizAPIClientOrExit builds a new Viz API client and executes default status
// checks to determine if the client can successfully perform cli commands. If the
// checks fail, then CLI will print an error and exit.
func CheckVizAPIClientOrExit(hcOptions healthcheck.Options) public.VizAPIClient {
	hcOptions.RetryDeadline = time.Time{}
	return CheckVizAPIClientOrRetryOrExit(hcOptions, false)
}

// CheckPublicAPIClientOrRetryOrExit builds a new Public API client and executes status
// checks to determine if the client can successfully connect to the API. If the
// checks fail, then CLI will print an error and exit. If the hcOptions.retryDeadline
// param is specified, then the CLI will print a message to stderr and retry.
func CheckPublicAPIClientOrRetryOrExit(hcOptions healthcheck.Options, apiChecks bool) public.PublicAPIClient {
	checks := []healthcheck.CategoryID{
		healthcheck.KubernetesAPIChecks,
		healthcheck.LinkerdControlPlaneExistenceChecks,
	}

	if apiChecks {
		checks = append(checks, healthcheck.LinkerdAPIChecks)
	}

	hc := healthcheck.NewHealthChecker(checks, &hcOptions)

	hc.RunChecks(exitOnError)
	return hc.PublicAPIClient()
}

// CheckVizAPIClientOrRetryOrExit builds a new Viz API client and executes status
// checks to determine if the client can successfully connect to the API. If the
// checks fail, then CLI will print an error and exit. If the hcOptions.retryDeadline
// param is specified, then the CLI will print a message to stderr and retry.
func CheckVizAPIClientOrRetryOrExit(hcOptions healthcheck.Options, apiChecks bool) public.VizAPIClient {
	checks := []healthcheck.CategoryID{
		healthcheck.KubernetesAPIChecks,
		healthcheck.LinkerdControlPlaneExistenceChecks,
	}

	if apiChecks {
		checks = append(checks, healthcheck.LinkerdAPIChecks)
	}

	hc := healthcheck.NewHealthChecker(checks, &hcOptions)

	hc.RunChecks(exitOnError)
	return hc.VizAPIClient()
}

func exitOnError(result *healthcheck.CheckResult) {
	if result.Retry {
		fmt.Fprintln(os.Stderr, "Waiting for control plane to become available")
		return
	}

	if result.Err != nil && !result.Warning {
		var msg string
		switch result.Category {
		case healthcheck.KubernetesAPIChecks:
			msg = "Cannot connect to Kubernetes"
		case healthcheck.LinkerdControlPlaneExistenceChecks:
			msg = "Cannot find Linkerd"
		case healthcheck.LinkerdAPIChecks:
			msg = "Cannot connect to Linkerd"
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, result.Err)

		checkCmd := "linkerd check"
		fmt.Fprintf(os.Stderr, "Validate the install with: %s\n", checkCmd)

		os.Exit(1)
	}
}
