package k8s

import (
	"context"

	authorizationapiv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Client is a kubernetes clientset interface used internally. It copies functions from
// k8s.io/client-go/kubernetes
//
//counterfeiter:generate . ClientSet
type ClientSet interface {
	AuthenticationV1() authenticationv1.AuthenticationV1Interface
	AuthorizationV1() authorizationv1.AuthorizationV1Interface
}

// Client is a kubernetes clientset interface used internally. It copies functions from
// k8s.io/client-go/kubernetes
//
//counterfeiter:generate . AuthorizationV1Interface
type AuthorizationV1Interface interface {
	SubjectAccessReviews() authorizationv1.SubjectAccessReviewInterface
	LocalSubjectAccessReviews(namespace string) authorizationv1.LocalSubjectAccessReviewInterface
	SelfSubjectAccessReviews() authorizationv1.SelfSubjectAccessReviewInterface
	SelfSubjectRulesReviews() authorizationv1.SelfSubjectRulesReviewInterface
	RESTClient() rest.Interface
}

// Client is a kubernetes clientset interface used internally. It copies functions from
// k8s.io/client-go/kubernetes
//
//counterfeiter:generate . SubjectAccessReviewInterface
type SubjectAccessReviewInterface interface {
	//nolint:lll
	Create(ctx context.Context, subjectAccessReview *authorizationapiv1.SubjectAccessReview, opts metav1.CreateOptions) (*authorizationapiv1.SubjectAccessReview, error)
}

// Client is a kubernetes clientset interface used internally. It copies functions from
// k8s.io/client-go/kubernetes
//
//counterfeiter:generate . SelfSubjectAccessReviewInterface
type SelfSubjectAccessReviewInterface interface {
	//nolint:lll
	Create(ctx context.Context, selfSubjectAccessReview *authorizationapiv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authorizationapiv1.SelfSubjectAccessReview, error)
}
