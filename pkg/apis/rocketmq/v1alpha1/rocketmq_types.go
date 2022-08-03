/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ConsoleConfig struct {
	v1.LocalObjectReference `json:",inline"`
	Spec                    ConsoleSpec `json:"spec,omitempty"`
}

type NameServiceConfig struct {
	v1.LocalObjectReference `json:",inline"`
	Spec                    NameServiceSpec `json:"spec,omitempty"`
}

type BrokerClusterConfig struct {
	v1.LocalObjectReference `json:",inline"`
	Spec                    BrokerSpec `json:"spec,omitempty"`
}

type BrokerClustersConfig struct {
	Replica  *int                  `json:"replica,omitempty"`
	Template *BrokerSpec           `json:"template,omitempty"`
	Items    []BrokerClusterConfig `json:"items,omitempty"`
}

// RocketMQSpec defines the desired state of RocketMQ
// +k8s:openapi-gen=true
type RocketMQSpec struct {
	Console        ConsoleConfig        `json:"console"`
	NameService    NameServiceConfig    `json:"nameService"`
	BrokerClusters BrokerClustersConfig `json:"brokerClusters"`
}

// RocketMQStatus defines the observed state of RocketMQ
// +k8s:openapi-gen=true
type RocketMQStatus struct {
	NameServers        []string `json:"nameServers,omitempty"`
	ConsoleReady       int32    `json:"consoleReady"`
	NameServiceReady   int32    `json:"nameServiceReady"`
	BrokerClusterReady int32    `json:"brokerClusterReady"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RocketMQ is the Schema for the rocketmqs API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="NameService",type="integer",JSONPath=".status.nameServiceReady"
// +kubebuilder:printcolumn:name="Broker-Clusters",type="integer",JSONPath=".status.brokerClusterReady"
// +kubebuilder:printcolumn:name="Console",type="integer",JSONPath=".status.consoleReady"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
type RocketMQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RocketMQSpec   `json:"spec,omitempty"`
	Status RocketMQStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RocketMQList contains a list of RocketMQ
type RocketMQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RocketMQ `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RocketMQ{}, &RocketMQList{})
}
