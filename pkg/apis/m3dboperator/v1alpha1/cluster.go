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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterConditionType represents the various type of cluster conditions.
type ClusterConditionType string

// IsolationGroups is a slice of IsolationGroup. IsolationGroups satisfies the
// sort.Sort interface, sorting by name.
type IsolationGroups []IsolationGroup

func (g IsolationGroups) Len() int           { return len(g) }
func (g IsolationGroups) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g IsolationGroups) Less(i, j int) bool { return g[i].Name < g[j].Name }

const (
	// ClusterConditionPlacementInitialized indicates an initial placement has
	// been created for the cluster.
	ClusterConditionPlacementInitialized ClusterConditionType = "PlacementInitialized"

	// ClusterConditionPodBootstrapping indicates there is a pod bootstrapping.
	ClusterConditionPodBootstrapping ClusterConditionType = "PodBootstrapping"
)

// M3DBCluster defines the cluster
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type M3DBCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Type              string      `json:"type"`
	Spec              ClusterSpec `json:"spec"`
	Status            M3DBStatus  `json:"status,omitempty"`
}

// M3DBClusterList represents a list of M3DB Clusters
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type M3DBClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []M3DBCluster `json:"items"`
}

// M3DBStatus contains the current state the M3DB cluster along with a human
// readable message
// +k8s:openapi-gen=true
type M3DBStatus struct {
	// State is a enum of green, yellow, and red denoting the health of the
	// cluster
	State M3DBState `json:"state,omitempty"`

	// Various conditions about the cluster.
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// Message is a human readable message indicating why the cluster is in it's
	// current state
	Message string `json:"message,omitempty"`

	// ObservedGeneration is the last generation of the cluster the controller
	// observed. Kubernetes will automatically increment metadata.Generation every
	// time the cluster spec is changed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

func (s *M3DBStatus) hasConditionTrue(cond ClusterConditionType) bool {
	for _, c := range s.Conditions {
		if c.Type == cond && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// HasInitializedPlacement returns true if the conditions indicate an initial
// placement has been created.
func (s *M3DBStatus) HasInitializedPlacement() bool {
	return s.hasConditionTrue(ClusterConditionPlacementInitialized)
}

// HasPodBootstrapping returns true if conditions indicate a pod is currently
// bootstrapping.
func (s *M3DBStatus) HasPodBootstrapping() bool {
	return s.hasConditionTrue(ClusterConditionPodBootstrapping)
}

// GetCondition returns the specified cluster condition if it exists with a bool
// indicating whether it was found.
func (s *M3DBStatus) GetCondition(checkCond ClusterConditionType) (ClusterCondition, bool) {
	for _, cond := range s.Conditions {
		if cond.Type == checkCond {
			return cond, true
		}
	}
	return ClusterCondition{}, false
}

// UpdateCondition updates one of the status's conditions, replacing the state
// of cond.Type if it exists or adding the condition if it doesn't exist.
func (s *M3DBStatus) UpdateCondition(newCond ClusterCondition) {
	for i, cond := range s.Conditions {
		if cond.Type == newCond.Type {
			s.Conditions[i] = newCond
			return
		}
	}

	s.Conditions = append(s.Conditions, newCond)
}

// ClusterCondition represents various conditions the cluster can be in.
// +k8s:openapi-gen=true
type ClusterCondition struct {
	// Type of cluster condition.
	Type ClusterConditionType `json:"type,omitempty"`

	// Status of the condition (True, False, Unknown).
	Status corev1.ConditionStatus `json:"status,omitempty"`

	// Last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	// Last time this condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	// Reason this condition last changed.
	Reason string `json:"reason,omitempty"`

	// Human-friendly message about this condition.
	Message string `json:"message,omitempty"`
}

// M3DBState contains the state of the M3DB cluster
type M3DBState string

const (
	// GreenState indicates a healthy state of the M3DB cluster
	GreenState M3DBState = "green"

	// YellowState indicates a caution state of the M3DB cluster
	YellowState M3DBState = "yellow"

	// RedState indicates a critical state of the M3DB cluster
	RedState M3DBState = "red"
)

// ClusterSpec defines the desired state for a M3 cluster to be converge to.
// +k8s:openapi-gen=true
type ClusterSpec struct {
	// Image specifies which docker image to use with the cluster
	Image string `json:"image,omitempty"`

	// ReplicationFactor defines how many replicas
	ReplicationFactor int32 `json:"replicationFactor,omitempty"`

	// NumberOfShards defines how many shards in total
	NumberOfShards int32 `json:"numberOfShards,omitempty"`

	// IsolationGroups specifies a map of key-value pairs. Defines which isolation groups
	// to deploy persistent volumes for data nodes
	IsolationGroups []IsolationGroup `json:"isolationGroups,omitempty"`

	// Namespaces specifies the namespaces this cluster will hold.
	Namespaces []Namespace `json:"namespaces,omitempty"`

	// EtcdEndpoints defines the etcd endpoints to use for service discovery. Must
	// be set if no custom configmap is defined. If set, etcd endpoints will be
	// templated in to the default configmap template.
	// +optional
	EtcdEndpoints []string `json:"etcdEndpoints,omitempty"`

	// KeepEtcdDataOnDelete determines whether the operator will remove cluster
	// metadata (placement + namespaces) in etcd when the cluster is deleted.
	// Unless true, etcd data will be cleared when the cluster is deleted.
	// +optional
	KeepEtcdDataOnDelete bool `json:"keepEtcdDataOnDelete,omitempty"`

	// ConfigMapName specifies the ConfigMap to use for this cluster. If unset a
	// default configmap with template variables for etcd endpoints will be used.
	// See "Configuring M3DB" in the docs for more.
	// +optional
	ConfigMapName *string `json:"configMapName,omitempty"`

	// PodIdentityConfig sets the configuration for pod identity. If unset only
	// pod name and UID will be used.
	// +optional
	PodIdentityConfig *PodIdentityConfig `json:"podIdentityConfig,omitempty"`

	// Resources defines memory / cpu constraints for each container in the
	// cluster.
	// +optional
	ContainerResources corev1.ResourceRequirements `json:"containerResources,omitempty"`

	// DataDirVolumeClaimTemplate is the volume claim template for an M3DB
	// instance's data. It claims PersistentVolumes for cluster storage, volumes
	// are dynamically provisioned by when the StorageClass is defined.
	// +optional
	DataDirVolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"dataDirVolumeClaimTemplate,omitempty"`

	// PodSecurityContext allows the user to specify an optional security context
	// for pods.
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// SecurityContext allows the user to specify a container-level security
	// context.
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// Labels sets the base labels that will be applied to resources created by
	// the cluster. // TODO(schallert): design doc on labeling scheme.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations sets the base annotations that will be applied to resources created by
	// the cluster.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`

	// Tolerations sets the tolerations that will be applied to all M3DB pods.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PriorityClassName sets the priority class for all M3DB pods.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// NodeAffinityTerm represents a node label and a set of label values, any of
// which can be matched to assign a pod to a node.
// +k8s:openapi-gen=true
type NodeAffinityTerm struct {
	// Key is the label of the node.
	Key string `json:"key"`

	// Values is an array of values, any of which a node can have for a pod to be
	// assigned to it.
	Values []string `json:"values"`
}

// IsolationGroup defines the name of zone as well attributes for the zone configuration
// +k8s:openapi-gen=true
type IsolationGroup struct {
	// Name is the value that will be used in StatefulSet labels, pod labels, and
	// M3DB placement "isolationGroup" fields.
	Name string `json:"name"`

	// NodeAffinityTerms is an array of NodeAffinityTerm requirements, which are
	// ANDed together to indicate what nodes an isolation group can be assigned
	// to.
	NodeAffinityTerms []NodeAffinityTerm `json:"nodeAffinityTerms,omitempty"`

	// NumInstances defines the number of instances.
	NumInstances int32 `json:"numInstances"`

	// StorageClassName is the name of the StorageClass to use for this isolation
	// group. This allows ensuring that PVs will be created in the same zone as
	// the pinned statefulset on Kubernetes < 1.12 (when topology aware volume
	// scheduling was introduced). Only has effect if the clusters
	// `dataDirVolumeClaimTemplate` is non-nil. If set, the volume claim template
	// will have its storageClassName field overridden per-isolationgroup. If
	// unset the storageClassName of the volumeClaimTemplate will be used.
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// GetByName fetches an IsolationGroup by name.
func (g IsolationGroups) GetByName(name string) (IsolationGroup, bool) {
	for _, group := range g {
		if group.Name == name {
			return group, true
		}
	}
	return IsolationGroup{}, false
}
