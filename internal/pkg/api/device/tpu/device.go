/*
Copyright 2026 The HAMi Authors.

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

package tpu

import (
	"fmt"

	"github.com/HAMi/mock-device-plugin/internal/pkg/api/device"
	"github.com/HAMi/mock-device-plugin/internal/pkg/mock"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const (
	RegisterAnnos = "hami.io/node-tpu-register"
	TpuDevice     = "TPU"
	Vendor        = "google.com"
)

type TpuConfig struct {
	ResourceCountName            string   `yaml:"resourceCountName"`
	ResourceMemoryName           string   `yaml:"resourceMemoryName"`
	ResourceCoreName             string   `yaml:"resourceCoreName"`
	ResourceMemoryPercentageName string   `yaml:"resourceMemoryPercentageName"`
	ResourcePriorityName         string   `yaml:"resourcePriorityName"`
	OverwriteEnv                 bool     `yaml:"overwriteEnv"`
	DefaultMemory                int32    `yaml:"defaultMemory"`
	DefaultCores                 int32    `yaml:"defaultCores"`
	DefaultTPUNum                int32    `yaml:"defaultTPUNum"`
	MemoryFactor                 int32    `yaml:"memoryFactor"`
	DeviceSplitCount             *uint    `yaml:"deviceSplitCount"`
	DeviceMemoryScaling          *float64 `yaml:"deviceMemoryScaling"`
	DeviceCoreScaling            *float64 `yaml:"deviceCoreScaling"`
}

type TpuDevices struct {
	config TpuConfig
}

func InitTpuDevice(config TpuConfig) *TpuDevices {
	klog.InfoS("initializing tpu device", "resourceName", config.ResourceCountName, "resourceMem", config.ResourceMemoryName)
	return &TpuDevices{
		config: config,
	}
}

func (dev *TpuDevices) CommonWord() string {
	return TpuDevice
}

func (dev *TpuDevices) GetNodeDevices(n *corev1.Node) ([]*device.DeviceInfo, error) {
	devEncoded, ok := n.Annotations[RegisterAnnos]
	if !ok {
		klog.Infof("annos not found %s, use default TPUs", RegisterAnnos)
		nodedevices := []*device.DeviceInfo{}
		for i := 0; i < int(dev.config.DefaultTPUNum); i++ {
			nodedevices = append(nodedevices, &device.DeviceInfo{
				ID:           fmt.Sprintf("mock-tpu-id-%d", i),
				Devcore:      dev.config.DefaultCores,
				Devmem:       dev.config.DefaultMemory,
				Type:         "mock",
				Health:       true,
				DeviceVendor: dev.CommonWord(),
			})
		}
		return nodedevices, nil
	}
	nodedevices, err := device.UnMarshalNodeDevices(devEncoded)
	if err != nil {
		return []*device.DeviceInfo{}, err
	}
	for idx := range nodedevices {
		nodedevices[idx].DeviceVendor = dev.CommonWord()
	}
	return nodedevices, nil
}

func (dev *TpuDevices) GetResource(n *corev1.Node) map[string]int {
	memoryResourceName := device.GetResourceName(dev.config.ResourceMemoryName)
	coreResourceName := device.GetResourceName(dev.config.ResourceCoreName)
	memoryPercentageName := device.GetResourceName(dev.config.ResourceMemoryPercentageName)
	resourceMap := map[string]int{
		memoryResourceName:   0,
		coreResourceName:     0,
		memoryPercentageName: 0,
	}
	if !device.CheckHealthy(n, dev.config.ResourceCountName) {
		klog.Infof("device %s is unhealthy on this node", dev.CommonWord())
		return resourceMap
	}
	devs, err := dev.GetNodeDevices(n)
	if err != nil {
		klog.Infof("no device %s on this node", TpuDevice)
		return resourceMap
	}
	for _, val := range devs {
		resourceMap[memoryResourceName] += int(val.Devmem)
		resourceMap[coreResourceName] += int(val.Devcore)
		resourceMap[memoryPercentageName] += 100
	}
	if dev.config.MemoryFactor > 1 {
		rawMemory := resourceMap[memoryResourceName]
		resourceMap[memoryResourceName] /= int(dev.config.MemoryFactor)
		klog.InfoS("Update memory", "raw", rawMemory, "after", resourceMap[memoryResourceName], "factor", dev.config.MemoryFactor)
	}
	klog.InfoS("Add resources",
		memoryResourceName,
		resourceMap[memoryResourceName],
		coreResourceName,
		resourceMap[coreResourceName],
		memoryPercentageName,
		resourceMap[memoryPercentageName],
	)
	return resourceMap
}

func (dev *TpuDevices) RunManager() {
	lmock := mock.NewMockLister(Vendor)
	go device.Register(lmock, dev)
	mockmanager := dpm.NewManager(lmock)
	klog.Infof("Running mocking dp: %s", dev.CommonWord())
	mockmanager.Run()
}
