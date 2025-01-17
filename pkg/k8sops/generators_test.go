// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package k8sops

import (
	"testing"

	m3dboperator "github.com/m3db/m3db-operator/pkg/apis/m3dboperator"
	myspec "github.com/m3db/m3db-operator/pkg/apis/m3dboperator/v1alpha1"
	"github.com/m3db/m3db-operator/pkg/k8sops/annotations"
	"github.com/m3db/m3db-operator/pkg/k8sops/labels"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/kubernetes/utils/pointer"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCRD(t *testing.T) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: m3dboperator.Name,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group: m3dboperator.GroupName,
			Versions: []apiextensionsv1beta1.CustomResourceDefinitionVersion{
				{
					Name:    m3dboperator.Version,
					Served:  true,
					Storage: true,
				},
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: m3dboperator.ResourcePlural,
				Kind:   m3dboperator.ResourceKind,
			},
			Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}

	newCRD := GenerateCRD(false)
	assert.Equal(t, crd, newCRD)
	assert.Nil(t, newCRD.Spec.Validation)

	newCRD = GenerateCRD(true)
	expValidation := crdutils.GetCustomResourceValidation(_openAPISpecName, myspec.GetOpenAPIDefinitions)
	assert.Equal(t, expValidation, newCRD.Spec.Validation)
}

func TestGenerateStatefulSet(t *testing.T) {
	fixture := getFixture("testM3DBCluster.yaml", t)
	clusterSpec := fixture.Spec
	isolationGroup := fixture.Spec.IsolationGroups[0].Name
	instanceAmount := &fixture.Spec.IsolationGroups[0].NumInstances
	clusterName := fixture.GetName()

	ssName := StatefulSetName(clusterName, 0)

	health := &v1.Probe{
		TimeoutSeconds:      _probeTimeoutSeconds,
		InitialDelaySeconds: _probeInitialDelaySeconds,
		FailureThreshold:    _probeFailureThreshold,
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Port:   intstr.FromInt(PortM3DBHTTPNode),
				Path:   _probePathHealth,
				Scheme: v1.URISchemeHTTP,
			},
		},
	}

	readiness := &v1.Probe{
		TimeoutSeconds:      _probeTimeoutSeconds,
		InitialDelaySeconds: _probeInitialDelaySeconds,
		FailureThreshold:    _probeFailureThreshold,
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Port:   intstr.FromInt(PortM3DBHTTPNode),
				Path:   _probePathReady,
				Scheme: v1.URISchemeHTTP,
			},
		},
	}

	labels := map[string]string{
		"operator.m3db.io/cluster":         clusterName,
		"operator.m3db.io/app":             "m3db",
		"operator.m3db.io/component":       "m3dbnode",
		"operator.m3db.io/stateful-set":    ssName,
		"operator.m3db.io/isolation-group": "us-fake1-a",
	}

	baseSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ssName,
			Labels:      labels,
			Annotations: annotations.BaseAnnotations(fixture),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(fixture, schema.GroupVersionKind{
					Group:   myspec.SchemeGroupVersion.Group,
					Version: myspec.SchemeGroupVersion.Version,
					Kind:    "m3dbcluster",
				}),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "m3dbnode-m3db-cluster",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: instanceAmount,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					PriorityClassName: "m3db-priority",
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: pointer.Int64Ptr(10),
					},
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "zone",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"zone-a"},
											},
										},
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:           ssName,
							LivenessProbe:  health,
							ReadinessProbe: readiness,
							SecurityContext: &v1.SecurityContext{
								RunAsUser: pointer.Int64Ptr(20),
							},
							Command: []string{
								"m3dbnode",
							},
							Args: []string{
								"-f",
								_configurationFileLocation,
							},
							Image:           clusterSpec.Image,
							ImagePullPolicy: "Always",
							Env: []v1.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Ports: generateContainerPorts(),
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      _dataVolumeName,
									MountPath: _dataDirectory,
								},
								{
									Name:      "cache",
									MountPath: "/var/lib/m3kv/",
								},
								{
									Name:      "pod-identity",
									MountPath: "/etc/m3db/pod-identity",
								},
								{
									Name:      _configurationName,
									MountPath: _configurationDirectory,
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu":    resource.MustParse("2"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("1Gi"),
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "cache",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "pod-identity",
							VolumeSource: v1.VolumeSource{
								DownwardAPI: &v1.DownwardAPIVolumeSource{
									Items: []v1.DownwardAPIVolumeFile{
										{
											Path: "identity",
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.annotations['operator.m3db.io/pod-identity']",
											},
										},
									},
								},
							},
						},
						{
							Name: _configurationName,
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "m3db-config-map-m3db-cluster",
									},
								},
							},
						},
					},
					Tolerations: []v1.Toleration{
						{
							Key:      "m3db-dedicated",
							Effect:   "NoSchedule",
							Operator: "Exists",
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: _dataVolumeName,
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
						StorageClassName: pointer.StringPtr("fake-sc"),
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								"storage": resource.MustParse("1Gi"),
							},
							Requests: v1.ResourceList{
								"storage": resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}

	// Base config stateful set
	ss := baseSS.DeepCopy()
	newSS, err := GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)

	// Reset spec and fixture, test custom config map
	ss = baseSS.DeepCopy()
	fixture = getFixture("testM3DBCluster.yaml", t)
	fixture.Spec.ConfigMapName = pointer.StringPtr("mymap")
	ss.Spec.Template.Spec.Volumes[2].VolumeSource.ConfigMap.Name = "mymap"
	newSS, err = GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)

	// Reset spec and fixture, test custom volume claims
	ss = baseSS.DeepCopy()
	fixture = getFixture("testM3DBCluster.yaml", t)
	fixture.Spec.DataDirVolumeClaimTemplate = nil
	ss.Spec.VolumeClaimTemplates = nil
	ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, v1.Volume{
		Name: _dataVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	newSS, err = GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)

	// Reset spec and fixture, test per-isogroup storageclasses
	ss = baseSS.DeepCopy()
	fixture = getFixture("testM3DBCluster.yaml", t)
	fixture.Spec.IsolationGroups[0].StorageClassName = "foo"
	ss.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = pointer.StringPtr("foo")

	newSS, err = GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)

	// Ensure changing another isogroup doesn't affect this statefulset
	ss = baseSS.DeepCopy()
	fixture = getFixture("testM3DBCluster.yaml", t)
	fixture.Spec.IsolationGroups[1].StorageClassName = "foo"

	newSS, err = GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)

	// Test empty tolerations
	ss = baseSS.DeepCopy()
	ss.Spec.Template.Spec.Tolerations = nil
	fixture = getFixture("testM3DBCluster.yaml", t)
	fixture.Spec.Tolerations = nil

	newSS, err = GenerateStatefulSet(fixture, isolationGroup, *instanceAmount)
	assert.NoError(t, err)
	assert.NotNil(t, newSS)
	assert.Equal(t, ss, newSS)
}

func TestGenerateM3DBService(t *testing.T) {
	cluster := &myspec.M3DBCluster{}
	svc, err := GenerateM3DBService(cluster)
	assert.Error(t, err)
	assert.Nil(t, svc)

	cluster.Name = "cluster-a"
	svc, err = GenerateM3DBService(cluster)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	baseLabels := map[string]string{
		labels.Cluster:   cluster.Name,
		labels.App:       labels.AppM3DB,
		labels.Component: labels.ComponentM3DBNode,
	}

	expSvc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "m3dbnode-cluster-a",
			Labels: baseLabels,
			Annotations: map[string]string{
				annotations.App:     annotations.AppM3DB,
				annotations.Cluster: cluster.Name,
			},
		},
		Spec: v1.ServiceSpec{
			Selector:  baseLabels,
			Ports:     generateM3DBServicePorts(),
			ClusterIP: v1.ClusterIPNone,
			Type:      v1.ServiceTypeClusterIP,
		},
	}

	assert.Equal(t, expSvc, svc)
}

func TestGenerateCoordinatorService(t *testing.T) {
	cluster := &myspec.M3DBCluster{}
	svc, err := GenerateCoordinatorService(cluster)
	assert.Error(t, err)
	assert.Nil(t, svc)

	cluster.Name = "cluster-a"
	svc, err = GenerateCoordinatorService(cluster)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	selectLabels := map[string]string{
		labels.Cluster:   cluster.Name,
		labels.App:       labels.AppM3DB,
		labels.Component: labels.ComponentM3DBNode,
	}

	svcLabels := map[string]string{
		labels.Cluster:   cluster.Name,
		labels.App:       labels.AppM3DB,
		labels.Component: labels.ComponentCoordinator,
	}

	expSvc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "m3coordinator-cluster-a",
			Labels: svcLabels,
		},
		Spec: v1.ServiceSpec{
			Selector: selectLabels,
			Ports:    generateCoordinatorServicePorts(),
			Type:     v1.ServiceTypeClusterIP,
		},
	}

	assert.Equal(t, expSvc, svc)
}
