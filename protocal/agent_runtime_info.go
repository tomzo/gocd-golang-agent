/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package protocal

type AgentIdentifier struct {
	HostName  string `json:"hostName"`
	IpAddress string `json:"ipAddress"`
	Uuid      string `json:"uuid"`
}

type AgentBuildingInfo struct {
	BuildingInfo string `json:"buildingInfo"`
	BuildLocator string `json:"buildingLocator"`
}

type AgentRuntimeInfo struct {
	Identifier                   *AgentIdentifier   `json:"identifier"`
	BuildingInfo                 *AgentBuildingInfo `json:"buildingInfo"`
	RuntimeStatus                string             `json:"runtimeStatus"`
	Location                     string             `json:"location"`
	UsableSpace                  int64              `json:"usableSpace"`
	OperatingSystemName          string             `json:"operatingSystemName"`
	Cookie                       string             `json:"cookie"`
	AgentLauncherVersion         string             `json:"agentLauncherVersion"`
	ElasticPluginId              string             `json:"elasticPluginId"`
	ElasticAgentId               string             `json:"elasticAgentId"`
	SupportsBuildCommandProtocol bool               `json:"supportsBuildCommandProtocol"`
}
