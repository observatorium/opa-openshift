package openshift

import (
	"context"
	"testing"

	"github.com/observatorium/opa-openshift/internal/external/k8s/k8sfakes"
	"github.com/observatorium/opa-openshift/internal/external/ocp/ocpfakes"
	projectv1 "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListNamespaces_ReturnsOnlyNames(t *testing.T) {
	k8sClient := &k8sfakes.FakeClientSet{}
	projectsClient := &ocpfakes.FakeProjectV1Client{}
	project := &ocpfakes.FakeProjectInterface{}
	projectsClient.ProjectsReturns(project)

	fakeProjects := &projectv1.ProjectList{
		Items: []projectv1.Project{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns2",
				},
			},
		},
	}

	project.ListReturns(fakeProjects, nil)

	c := client{
		k8sClient:     k8sClient,
		projectClient: projectsClient,
	}

	got, err := c.ListNamespaces()
	require.NoError(t, err)

	want := []string{"ns1", "ns2"}
	require.ElementsMatch(t, got, want)
}

func TestSubjectAccessReview_HandleResourceAttributesOnly(t *testing.T) {
	authzv1 := &k8sfakes.FakeAuthorizationV1Interface{}
	sar := &k8sfakes.FakeSubjectAccessReviewInterface{}
	k8sClient := &k8sfakes.FakeClientSet{}

	authzv1.SubjectAccessReviewsReturns(sar)
	k8sClient.AuthorizationV1Returns(authzv1)

	c := client{k8sClient: k8sClient}

	type sarInput struct {
		user         string
		groups       []string
		verb         string
		apiGroup     string
		resource     string
		resourceName string
	}

	input := sarInput{
		user:         "robocop",
		groups:       []string{"detroit", "police"},
		verb:         "get",
		apiGroup:     "group.me.io",
		resource:     "tenantID",
		resourceName: "resource",
	}

	sar.CreateCalls(func(_ context.Context, sar *authorizationv1.SubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) { //nolint:lll
		require.Equal(t, input.user, sar.Spec.User)
		require.Equal(t, input.groups, sar.Spec.Groups)
		require.NotNil(t, sar.Spec.ResourceAttributes)
		require.Equal(t, input.verb, sar.Spec.ResourceAttributes.Verb)
		require.Equal(t, input.resource, sar.Spec.ResourceAttributes.Resource)
		require.Equal(t, input.resourceName, sar.Spec.ResourceAttributes.Name)
		require.Equal(t, input.apiGroup, sar.Spec.ResourceAttributes.Group)
		sar.Status = authorizationv1.SubjectAccessReviewStatus{Allowed: true}

		return sar, nil
	})

	_, err := c.SubjectAccessReview(input.user, input.groups, input.verb, input.resource, input.resourceName, input.apiGroup, "")
	require.NoError(t, err)
}
