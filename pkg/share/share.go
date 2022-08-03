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

// Package share defines some variables shared by different packages
package share

type ClusterShareConfig struct {
	// GroupNum is the number of broker group
	GroupNum int

	// NameServersStr is the name server list
	NameServersStr string

	// IsNameServersStrUpdated is whether the name server list is updated
	IsNameServersStrUpdated bool

	// IsNameServersStrInitialized is whether the name server list is initialized
	IsNameServersStrInitialized bool

	// BrokerClusterName is the broker cluster name
	BrokerClusterName string
}

var shareConfigs map[string]*ClusterShareConfig

func GetShareConfig(clusterName string) *ClusterShareConfig {
	if shareConfigs == nil {
		shareConfigs = make(map[string]*ClusterShareConfig)
	}
	if config, ok := shareConfigs[clusterName]; ok {
		return config
	}

	config := ClusterShareConfig{
		GroupNum:                    0,
		NameServersStr:              "",
		IsNameServersStrUpdated:     false,
		IsNameServersStrInitialized: false,
		BrokerClusterName:           "",
	}
	shareConfigs[clusterName] = &config
	return &config
}
