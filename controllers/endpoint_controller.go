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
	"yunion.io/x/onecloud-service-operator/pkg/resources"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onecloudv1 "yunion.io/x/onecloud-service-operator/api/v1"
)

// EndpointReconciler reconciles a Endpoint object
type EndpointReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=onecloud.yunion.io,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=onecloud.yunion.io,resources=endpoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=onecloud.yunion.io,resources=virtualmachines,verbs=get;list;watch

func (r *EndpointReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("endpoint", req.NamespacedName)

	var endpoint onecloudv1.Endpoint
	if err := r.Get(ctx, req.NamespacedName, &endpoint); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	remoteEP := resources.NewEndpoint(&endpoint)

	dealErr := func(err error) (ctrl.Result, error) {
		return dealErr(ctx, log, r, &endpoint, resources.ResourceEndpoint, err)
	}


	UseFinallizer(ctx, r, &endpoint, func(ctx context.Context) (ctrl.Result, error) {
		ret, err := r.realDelete(ctx, remoteEP)
		if err != nil {
			return dealErr(err)
		}
		return ret, err
	})



	return ctrl.Result{}, nil
}

func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&onecloudv1.Endpoint{}).
		Complete(r)
}
