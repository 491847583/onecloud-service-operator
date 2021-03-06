// Copyright 2020 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onecloudv1 "yunion.io/x/onecloud-service-operator/api/v1"
	"yunion.io/x/onecloud-service-operator/pkg/resources"
	"yunion.io/x/pkg/utils"
)

type ReconcilerBase struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ReconcilerBase) GetLog(resource onecloudv1.IResource) logr.Logger {
	kind := resource.GetObjectKind().GroupVersionKind().Kind
	namespace := resource.GetNamespace()
	name := resource.GetName()
	return r.Log.WithValues(kind, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	})
}

func (r *ReconcilerBase) getExternalId(resource onecloudv1.IResource) string {
	return resource.GetResourceStatus().GetBaseExternalInfo().Id
}

func (r *ReconcilerBase) setExternalId(resource onecloudv1.IResource, id string) {
	status := resource.GetResourceStatus()
	info := status.GetBaseExternalInfo()
	info.Id = id
	status.SetBaseExternalInfo(info)
}

func (r *ReconcilerBase) dealErr(ctx context.Context, ocResouce resources.OCResource, err error) (ctrl.Result, error) {

	re := ocResouce.GetIResource()
	log := r.GetLog(re)
	reErr, ok := err.(*resources.SRequestErr)
	if !ok {
		log.Error(err, "")
		return ctrl.Result{}, err
	}

	if reErr.IsNotFound(ocResouce.GetResourceName()) {
		r.setExternalId(re, "")
	}
	status := re.GetResourceStatus()
	if reErr.IsClientErr() {
		status.SetPhase(onecloudv1.ResourcePending, reErr.Error())
	}
	if reErr.IsServerErr() {
		status.SetPhase(onecloudv1.ResourceUnkown, reErr.Error())
	}
	return ctrl.Result{}, r.Status().Update(ctx, re)
}

func (r *ReconcilerBase) UseFinallizer(ctx context.Context, ocResource resources.OCResource) (has bool, ret ctrl.Result, err error) {
	myFinalizerName := "common.finalizers.onecloud.yunion.io"
	resource := ocResource.GetIResource()
	finalizers := resource.GetFinalizers()
	if resource.GetDeletionTimestamp().IsZero() {
		if !utils.IsInStringArray(myFinalizerName, finalizers) {
			finalizers = append(finalizers, myFinalizerName)
			resource.SetFinalizers(finalizers)
			err = r.Update(ctx, resource)
			return
		}
	} else {
		if utils.IsInStringArray(myFinalizerName, finalizers) {
			if len(r.getExternalId(resource)) == 0 {
				finalizers = removeString(finalizers, myFinalizerName)
				resource.SetFinalizers(finalizers)
				err = r.Update(ctx, resource)
				return
			}
			ret, err = r.RealDelete(ctx, ocResource)
			if err != nil {
				ret, err = r.dealErr(ctx, ocResource, err)
				return false, ret, err
			}
			return
		}
		return
	}
	has = true
	return
}

func (r *ReconcilerBase) Create(ctx context.Context, ocResource resources.OCResource, params interface{}) error {
	resource := ocResource.GetIResource()
	maxRetryTimes := resource.GetResourceSpec().GetMaxRetryTimes()
	rs := resource.GetResourceStatus()
	retryTimes := rs.GetTryTimes()
	if retryTimes-1 == maxRetryTimes {
		rs.SetPhase(onecloudv1.ResourceInvalid, fmt.Sprintf("The number of consecutive retry creation failures exceeds the maximum %d", maxRetryTimes))
	}
	extInfo, err := ocResource.Create(ctx, params)
	if err != nil {
		return err
	}
	rs.SetBaseExternalInfo(extInfo)
	rs.SetPhase(onecloudv1.ResourcePending, "")
	rs.SetTryTimes(retryTimes + 1)
	return r.Status().Update(ctx, resource)
}

func (r *ReconcilerBase) RealDelete(ctx context.Context, ocResource resources.OCResource) (ctrl.Result, error) {
	reStatus, err := ocResource.GetStatus(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	resource := ocResource.GetIResource()
	if !reflect.DeepEqual(reStatus, resource.GetResourceStatus()) {
		resource.SetResourceStatus(reStatus)
		return ctrl.Result{}, r.Status().Update(ctx, resource)
	}
	if reStatus.GetPhase() == onecloudv1.ResourcePending {
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}
	if reStatus.GetPhase() == onecloudv1.ResourceUnkown {
		return ctrl.Result{Requeue: true, RequeueAfter: 2 * time.Second}, nil
	}
	// delete this
	if err := r.Delete(ctx, ocResource); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ReconcilerBase) Delete(ctx context.Context, ocResource resources.OCResource) error {
	extInfo, err := ocResource.Delete(ctx)
	if err != nil {
		return err
	}
	status := ocResource.GetIResource().GetResourceStatus()
	status.SetPhase(onecloudv1.ResourcePending, "")
	status.SetBaseExternalInfo(extInfo)
	return r.Status().Update(ctx, ocResource.GetIResource())
}
