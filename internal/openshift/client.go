package openshift

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path"

	projectv1 "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/observatorium/opa-openshift/internal/external/k8s"
	"github.com/observatorium/opa-openshift/internal/external/ocp"
)

// Client is the standard openshift client to
// check authentication and authorization for
// subjects.
type Client interface {
	SubjectAccessReview(user string, groups []string, verb, resource, resourceName, apiGroup, namespace string) (bool, error)
	ListNamespaces() ([]string, error)
}

type client struct {
	k8sClient     k8s.ClientSet
	projectClient ocp.ProjectV1Client
}

// NewClient returns a new OpenShift client holding a pointer to a k8s clientset
// and a pointer to the OpenShift project clientset. Both clientset require a
// kube config file on a prescribed path or under $HOME/.kube. The loaded kube
// configuration is sanitized and augmented with the subject's forwarded bearer
// token.
func NewClient(wt transport.WrapperFunc, kubeconfigPath, token string) (Client, error) {
	cfg, err := getConfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	cfg.WrapTransport = wt

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	// Set user token to acces the project clienset
	// to request only user-accessible projects.
	cfg = rest.AnonymousClientConfig(cfg)
	cfg.BearerToken = token
	cfg.WrapTransport = wt

	projectClient, err := projectv1.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ocp project clientset: %w", err)
	}

	return &client{
		k8sClient:     clientset,
		projectClient: projectClient,
	}, nil
}

// SubjectAccessReview requests a self subject access review from the k8s api server
// for an authenticated user.
func (c *client) SubjectAccessReview(user string, groups []string, verb, resource, resourceName, apiGroup, namespace string) (bool, error) {
	ssar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   user,
			Groups: groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     apiGroup,
				Resource:  resource,
				Name:      resourceName,
			},
		},
	}

	res, err := c.k8sClient.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), ssar, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to create subject access review: %w", err)
	}

	return res.Status.Allowed, nil
}

// ListNamespaces provides a list of all namespaces an authenticated user
// has access to or an error on failure.
func (c *client) ListNamespaces() ([]string, error) {
	projects, err := c.projectClient.Projects().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	namespaces := make([]string, 0, len(projects.Items))
	for _, ns := range projects.Items {
		namespaces = append(namespaces, ns.Name)
	}

	return namespaces, nil
}

func getConfig(kubeconfig string) (*rest.Config, error) {
	if len(kubeconfig) > 0 {
		loader := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}

		return loadConfig(loader)
	}

	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) == 0 {
		return rest.InClusterConfig() //nolint:wrapcheck
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %w", err)
		}

		p := path.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
		loadingRules.Precedence = append(loadingRules.Precedence, p)
	}

	return loadConfig(loadingRules)
}

func loadConfig(loader clientcmd.ClientConfigLoader) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil).ClientConfig() //nolint:wrapcheck
}
