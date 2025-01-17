/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"

	netapi "knative.dev/networking/pkg/apis/networking"
	netv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	netheader "knative.dev/networking/pkg/http/header"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
	routeresources "knative.dev/serving/pkg/reconciler/route/resources"
)

// MakeIngress creates an Ingress object for a DomainMapping.  The Ingress is
// always created in the same namespace as the DomainMapping, and the ingress
// backend is always in the same namespace also (as this is required by
// KIngress).  The created ingress will contain a RewriteHost rule to cause the
// given hostName to be used as the host.
func MakeIngress(dm *servingv1alpha1.DomainMapping, backendServiceName, hostName, ingressClass string, httpOption netv1alpha1.HTTPOption, tls []netv1alpha1.IngressTLS, acmeChallenges ...netv1alpha1.HTTP01Challenge) *netv1alpha1.Ingress {
	paths, hosts := routeresources.MakeACMEIngressPaths(acmeChallenges, sets.NewString(dm.GetName()))
	return &netv1alpha1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kmeta.ChildName(dm.GetName(), ""),
			Namespace: dm.Namespace,
			Annotations: kmeta.FilterMap(kmeta.UnionMaps(map[string]string{
				netapi.IngressClassAnnotationKey: ingressClass,
			}, dm.GetAnnotations()), routeresources.ExcludedAnnotations.Has),
			Labels: kmeta.UnionMaps(dm.Labels, map[string]string{
				serving.DomainMappingUIDLabelKey:       string(dm.UID),
				serving.DomainMappingNamespaceLabelKey: dm.Namespace,
			}),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(dm)},
		},
		Spec: netv1alpha1.IngressSpec{
			HTTPOption: httpOption,
			TLS:        tls,
			Rules: []netv1alpha1.IngressRule{{
				Hosts:      append(hosts, dm.Name),
				Visibility: netv1alpha1.IngressVisibilityExternalIP,
				HTTP: &netv1alpha1.HTTPIngressRuleValue{
					// The order of the paths is sensitive, always put tls challenge first
					Paths: append(paths,
						[]netv1alpha1.HTTPIngressPath{{
							RewriteHost: hostName,
							Splits: []netv1alpha1.IngressBackendSplit{{
								Percent: 100,
								AppendHeaders: map[string]string{
									netheader.OriginalHostKey: dm.Name,
								},
								IngressBackend: netv1alpha1.IngressBackend{
									ServiceNamespace: dm.Namespace,
									ServiceName:      backendServiceName,
									ServicePort:      intstr.FromInt(80),
								},
							}},
						}}...),
				},
			}},
		},
	}
}
