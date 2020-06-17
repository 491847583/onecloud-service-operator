package resources

import (
	"context"
	onecloudv1 "yunion.io/x/onecloud-service-operator/api/v1"
)

type OCResource interface {
	GetResourceName() Resource
	GetIResource() onecloudv1.IResource
	Create(ctx context.Context, params interface{}) (onecloudv1.ExternalInfoBase, error)
	Delete(ctx context.Context) (onecloudv1.ExternalInfoBase, error)
	GetStatus(ctx context.Context) (onecloudv1.IResourceStatus, error)
}