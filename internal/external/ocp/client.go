package ocp

import (
	"context"

	projectv1 "github.com/openshift/api/project/v1"
	projectsv1 "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	applyprojectv1 "github.com/openshift/client-go/project/applyconfigurations/project/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// ProjectV1ClientSet is a openshift client interface for project resources used internally.
// It copies functions from
// github.com/openshift/client-go/project/clientset/versioned/typed/project/v1
//
//counterfeiter:generate .	ProjectV1Client
type ProjectV1Client interface {
	Projects() projectsv1.ProjectInterface
}

// Projects is a openshift client interface for project resources used internally.
// It copies functions from
// github.com/openshift/client-go/project/clientset/versioned/typed/project/v1
//
//counterfeiter:generate . ProjectInterface
type ProjectInterface interface {
	Create(ctx context.Context, project *projectv1.Project, opts metav1.CreateOptions) (*projectv1.Project, error)
	Update(ctx context.Context, project *projectv1.Project, opts metav1.UpdateOptions) (*projectv1.Project, error)
	UpdateStatus(ctx context.Context, project *projectv1.Project, opts metav1.UpdateOptions) (*projectv1.Project, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*projectv1.Project, error)
	List(ctx context.Context, opts metav1.ListOptions) (*projectv1.ProjectList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *projectv1.Project, err error) //nolint:lll
	Apply(ctx context.Context, project *applyprojectv1.ProjectApplyConfiguration, opts metav1.ApplyOptions) (result *projectv1.Project, err error)
	ApplyStatus(ctx context.Context, project *applyprojectv1.ProjectApplyConfiguration, opts metav1.ApplyOptions) (result *projectv1.Project, err error)
}
