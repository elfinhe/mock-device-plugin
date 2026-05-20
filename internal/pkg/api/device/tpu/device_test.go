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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetResource(t *testing.T) {
	// Create TPU config
	config := TpuConfig{
		ResourceCountName:            "google.com/tpu",
		ResourceMemoryName:           "google.com/tpumem",
		ResourceCoreName:             "google.com/tpucores",
		ResourceMemoryPercentageName: "google.com/tpumem-percentage",
		ResourcePriorityName:         "google.com/priority",
		DefaultMemory:                192000000,
		DefaultCores:                 8,
		DefaultTPUNum:                4,
		MemoryFactor:                 1,
	}

	dev := InitTpuDevice(config)

	// Test default configuration (no node annotations)
	t.Run("Test TPU default config", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-node-tpu-default",
				Annotations: map[string]string{},
			},
		}
		result := dev.GetResource(node)

		expectedCount := 4
		if actualCount, ok := result["tpu"]; !ok || actualCount != expectedCount {
			t.Errorf("expected tpu count %d, got %v (exists: %v)", expectedCount, actualCount, ok)
		}

		expectedMem := 768000000
		if actualMem, ok := result["tpumem"]; !ok || actualMem != expectedMem {
			t.Errorf("expected tpumem %d, got %v (exists: %v)", expectedMem, actualMem, ok)
		}

		expectedCores := 32
		if actualCores, ok := result["tpucores"]; !ok || actualCores != expectedCores {
			t.Errorf("expected tpucores %d, got %v (exists: %v)", expectedCores, actualCores, ok)
		}

		expectedPercentage := 400
		if actualPercentage, ok := result["tpumem-percentage"]; !ok || actualPercentage != expectedPercentage {
			t.Errorf("expected tpumem-percentage %d, got %v (exists: %v)", expectedPercentage, actualPercentage, ok)
		}
	})

	// Test configuration with some resource names not configured (empty)
	t.Run("Test TPU partial config", func(t *testing.T) {
		configPartial := TpuConfig{
			ResourceCountName:  "google.com/tpu",
			ResourceMemoryName: "google.com/tpumem",
			DefaultMemory:      192000000,
			DefaultCores:       8,
			DefaultTPUNum:      4,
			MemoryFactor:       1,
		}
		devPartial := InitTpuDevice(configPartial)
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-node-tpu-partial",
				Annotations: map[string]string{},
			},
		}
		result := devPartial.GetResource(node)

		if _, ok := result["tpucores"]; ok {
			t.Errorf("expected tpucores to not be present, but it was found")
		}
		if _, ok := result["tpumem-percentage"]; ok {
			t.Errorf("expected tpumem-percentage to not be present, but it was found")
		}
		if val, ok := result["tpu"]; !ok || val != 4 {
			t.Errorf("expected tpu count 4, got %v (exists: %v)", val, ok)
		}
		if val, ok := result["tpumem"]; !ok || val != 768000000 {
			t.Errorf("expected tpumem 768000000, got %v (exists: %v)", val, ok)
		}
	})

	// Test with node annotations
	t.Run("Test TPU node annotations", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node-tpu-annos",
				Annotations: map[string]string{
					RegisterAnnos: `[
						{"id":"mock-tpu-id-0","devcore":16,"devmem":256000000,"type":"mock","health":true,"devicevendor":"TPU"},
						{"id":"mock-tpu-id-1","devcore":16,"devmem":256000000,"type":"mock","health":true,"devicevendor":"TPU"}
					]`,
				},
			},
		}
		result := dev.GetResource(node)

		if val, ok := result["tpu"]; !ok || val != 2 {
			t.Errorf("expected tpu count 2, got %v (exists: %v)", val, ok)
		}
		if val, ok := result["tpumem"]; !ok || val != 512000000 {
			t.Errorf("expected tpumem 512000000, got %v (exists: %v)", val, ok)
		}
		if val, ok := result["tpucores"]; !ok || val != 32 {
			t.Errorf("expected tpucores 32, got %v (exists: %v)", val, ok)
		}
		if val, ok := result["tpumem-percentage"]; !ok || val != 200 {
			t.Errorf("expected tpumem-percentage 200, got %v (exists: %v)", val, ok)
		}
	})

	// Test default configuration with MemoryFactor > 1
	t.Run("Test TPU default config with MemoryFactor", func(t *testing.T) {
		configFactor := TpuConfig{
			ResourceCountName:  "google.com/tpu",
			ResourceMemoryName: "google.com/tpumem",
			DefaultMemory:      192000000,
			DefaultTPUNum:      4,
			MemoryFactor:       2,
		}
		devFactor := InitTpuDevice(configFactor)
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-node-tpu-factor",
				Annotations: map[string]string{},
			},
		}
		result := devFactor.GetResource(node)

		expectedMem := 384000000 // (192000000 * 4) / 2
		if val, ok := result["tpumem"]; !ok || val != expectedMem {
			t.Errorf("expected tpumem %d, got %v (exists: %v)", expectedMem, val, ok)
		}
	})

	// Test with invalid node annotations
	t.Run("Test TPU node annotations invalid JSON", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node-tpu-invalid-annos",
				Annotations: map[string]string{
					RegisterAnnos: `invalid-json`,
				},
			},
		}
		result := dev.GetResource(node)

		// Should return empty map or at least not contain tpu resources because GetNodeDevices fails
		if val, ok := result["tpu"]; ok {
			t.Errorf("expected tpu to not be present due to invalid JSON, but got %v", val)
		}
		if val, ok := result["tpumem"]; ok {
			t.Errorf("expected tpumem to not be present due to invalid JSON, but got %v", val)
		}
	})
}
