// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package scheme

import (
	chaosoperatorv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	astsv1alpha1 "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"

	tidboperatorv1alpha "github.com/pingcap/tipocket/pkg/tidb-operator/client/clientset/versioned/scheme"
)

// Scheme gathers the schemes of native resources and custom resources used by tipocket
// in favor of the generic controller-runtime/client
var Scheme = runtime.NewScheme()

func init() {
	v1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(kubescheme.AddToScheme(Scheme))
	utilruntime.Must(tidboperatorv1alpha.AddToScheme(Scheme))
	utilruntime.Must(astsv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(chaosoperatorv1alpha1.AddToScheme(Scheme))
}
