// Copyright 2019 The Kubernetes Authors.
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
package watcher

import (
	stdlog "log"
	"os"
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cfg *rest.Config

var deployment = v1.Deployment{
	TypeMeta: metav1.TypeMeta{
		Kind: "deployment",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "deployment",
		Namespace: "default",
	},
}
var daemonset = v1.DaemonSet{
	TypeMeta: metav1.TypeMeta{
		Kind: "daemonset",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "daemonset",
		Namespace: "default",
	}}
var statefulset = v1.StatefulSet{
	TypeMeta: metav1.TypeMeta{
		Kind: "statefulset",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "statefulset",
		Namespace: "default",
	}}

func TestMain(m *testing.M) {
	t := &envtest.Environment{}

	var err error
	if cfg, err = t.Start(); err != nil {
		stdlog.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}
