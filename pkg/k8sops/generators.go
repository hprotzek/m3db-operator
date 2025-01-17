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
	"errors"
	"fmt"

	m3dboperator "github.com/m3db/m3db-operator/pkg/apis/m3dboperator"
	myspec "github.com/m3db/m3db-operator/pkg/apis/m3dboperator/v1alpha1"
	"github.com/m3db/m3db-operator/pkg/k8sops/annotations"
	"github.com/m3db/m3db-operator/pkg/k8sops/labels"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/kubernetes/utils/pointer"
	pkgerrors "github.com/pkg/errors"
)

const (
	_probeTimeoutSeconds      = 30
	_probeInitialDelaySeconds = 10
	_probeFailureThreshold    = 15

	_probePathHealth = "/health"
	_probePathReady  = "/bootstrappedinplacementornoplacement"

	_dataDirectory             = "/var/lib/m3db/"
	_dataVolumeName            = "m3db-data"
	_configurationDirectory    = "/etc/m3db/"
	_configurationName         = "m3-configuration"
	_configurationFileLocation = _configurationDirectory + _configurationFileName
	_configurationFileName     = "m3.yml"
	_healthFileName            = "/bin/m3dbnode_bootstrapped.sh"
	_openAPISpecName           = "github.com/m3db/m3db-operator/pkg/apis/m3dboperator/v1alpha1.M3DBCluster"
)

var (
	errEmptyClusterName = errors.New("cluster name cannot be empty")
)

type m3dbPort struct {
	name     string
	port     Port
	protocol v1.Protocol
}

var baseM3DBPorts = [...]m3dbPort{
	{"client", PortM3DBNodeClient, v1.ProtocolTCP},
	{"cluster", PortM3DBNodeCluster, v1.ProtocolTCP},
	{"http-node", PortM3DBHTTPNode, v1.ProtocolTCP},
	{"http-cluster", PortM3DBHTTPCluster, v1.ProtocolTCP},
	{"debug", PortM3DBDebug, v1.ProtocolTCP},
	{"coordinator", PortM3Coordinator, v1.ProtocolTCP},
	{"coord-metrics", PortM3CoordinatorMetrics, v1.ProtocolTCP},
}

var baseCoordinatorPorts = [...]m3dbPort{
	{"coordinator", PortM3Coordinator, v1.ProtocolTCP},
	{"coord-metrics", PortM3CoordinatorMetrics, v1.ProtocolTCP},
}

// GenerateCRD generates the crd object needed for the M3DBCluster
func GenerateCRD(enableValidation bool) *apiextensionsv1beta1.CustomResourceDefinition {
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

	if enableValidation {
		crd.Spec.Validation = crdutils.GetCustomResourceValidation(_openAPISpecName, myspec.GetOpenAPIDefinitions)
	}

	return crd
}

// GenerateStatefulSet provides a statefulset object for a m3db cluster
func GenerateStatefulSet(
	cluster *myspec.M3DBCluster,
	isolationGroupName string,
	instanceAmount int32,
) (*appsv1.StatefulSet, error) {

	// TODO(schallert): always sort zones alphabetically.
	stsID := -1
	var isolationGroup myspec.IsolationGroup
	for i, g := range cluster.Spec.IsolationGroups {
		if g.Name == isolationGroupName {
			isolationGroup = g
			stsID = i
			break
		}
	}

	if stsID == -1 {
		return nil, fmt.Errorf("could not find isogroup '%s' in spec", isolationGroupName)
	}

	clusterSpec := cluster.Spec
	clusterName := cluster.GetName()
	ssName := StatefulSetName(clusterName, stsID)

	affinity, err := GenerateStatefulSetAffinity(isolationGroup)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error generating statefulset affinity")
	}

	statefulSet := NewBaseStatefulSet(ssName, isolationGroupName, cluster, instanceAmount)
	m3dbContainer := &statefulSet.Spec.Template.Spec.Containers[0]
	m3dbContainer.Resources = clusterSpec.ContainerResources
	m3dbContainer.Ports = generateContainerPorts()
	statefulSet.Spec.Template.Spec.Affinity = affinity
	statefulSet.Spec.Template.Spec.Tolerations = cluster.Spec.Tolerations

	// Set owner ref so sts will be GC'd when the cluster is deleted
	clusterRef := GenerateOwnerRef(cluster)
	statefulSet.OwnerReferences = []metav1.OwnerReference{*clusterRef}

	configVol, configVolMount, err := buildConfigMapComponents(cluster)
	if err != nil {
		return nil, err
	}

	m3dbContainer.VolumeMounts = append(m3dbContainer.VolumeMounts, configVolMount)
	vols := &statefulSet.Spec.Template.Spec.Volumes
	*vols = append(*vols, configVol)

	if cluster.Spec.DataDirVolumeClaimTemplate == nil {
		// No persistent volume claims, add an empty dir for m3db data.
		vols := &statefulSet.Spec.Template.Spec.Volumes
		*vols = append(*vols, v1.Volume{
			Name: _dataVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	} else {
		template := cluster.Spec.DataDirVolumeClaimTemplate.DeepCopy()
		template.ObjectMeta.Name = _dataVolumeName
		if sc := isolationGroup.StorageClassName; sc != "" {
			template.Spec.StorageClassName = pointer.StringPtr(sc)
		}
		statefulSet.Spec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*template}
	}

	return statefulSet, nil
}

// GenerateM3DBService will generate the headless service required for an M3DB
// StatefulSet.
func GenerateM3DBService(cluster *myspec.M3DBCluster) (*v1.Service, error) {
	if cluster.Name == "" {
		return nil, errEmptyClusterName
	}

	svcLabels := labels.BaseLabels(cluster)
	svcLabels[labels.Component] = labels.ComponentM3DBNode
	svcAnnotations := annotations.BaseAnnotations(cluster)
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        HeadlessServiceName(cluster.Name),
			Labels:      svcLabels,
			Annotations: svcAnnotations,
		},
		Spec: v1.ServiceSpec{
			Selector:  svcLabels,
			Ports:     generateM3DBServicePorts(),
			ClusterIP: v1.ClusterIPNone,
			Type:      v1.ServiceTypeClusterIP,
		},
	}, nil
}

// GenerateCoordinatorService creates a coordinator service given a cluster
// name.
func GenerateCoordinatorService(cluster *myspec.M3DBCluster) (*v1.Service, error) {
	if cluster.Name == "" {
		return nil, errEmptyClusterName
	}

	selectorLabels := labels.BaseLabels(cluster)
	selectorLabels[labels.Component] = labels.ComponentM3DBNode

	serviceLabels := labels.BaseLabels(cluster)
	serviceLabels[labels.Component] = labels.ComponentCoordinator

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   CoordinatorServiceName(cluster.Name),
			Labels: serviceLabels,
		},
		Spec: v1.ServiceSpec{
			Selector: selectorLabels,
			Ports:    generateCoordinatorServicePorts(),
			Type:     v1.ServiceTypeClusterIP,
		},
	}, nil
}

func buildServicePorts(ports []m3dbPort) []v1.ServicePort {
	svcPorts := []v1.ServicePort{}
	for _, p := range ports {
		newPortMapping := v1.ServicePort{
			Name:     p.name,
			Port:     int32(p.port),
			Protocol: p.protocol,
		}
		svcPorts = append(svcPorts, newPortMapping)
	}
	return svcPorts
}

func generateM3DBServicePorts() []v1.ServicePort {
	return buildServicePorts(baseM3DBPorts[:])
}

func generateCoordinatorServicePorts() []v1.ServicePort {
	return buildServicePorts(baseCoordinatorPorts[:])
}

// generateContainerPorts will produce default container ports.
func generateContainerPorts() []v1.ContainerPort {
	cntPorts := []v1.ContainerPort{}
	for _, v := range baseM3DBPorts {
		newPortMapping := v1.ContainerPort{
			Name:          v.name,
			ContainerPort: int32(v.port),
			Protocol:      v.protocol,
		}
		cntPorts = append(cntPorts, newPortMapping)
	}
	return cntPorts
}
