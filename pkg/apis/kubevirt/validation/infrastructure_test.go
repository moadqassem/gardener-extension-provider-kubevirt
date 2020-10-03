// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation_test

import (
	api "github.com/gardener/gardener-extension-provider-kubevirt/pkg/apis/kubevirt"
	. "github.com/gardener/gardener-extension-provider-kubevirt/pkg/apis/kubevirt/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		nilPath *field.Path

		infrastructureConfig *api.InfrastructureConfig
	)

	BeforeEach(func() {
		infrastructureConfig = &api.InfrastructureConfig{}
	})

	Describe("#ValidateInfrastructureConfig", func() {
		It("should return no errors for a valid configuration", func() {
			Expect(ValidateInfrastructureConfig(infrastructureConfig, nilPath)).To(BeEmpty())
		})

		// TODO Test invalid configuration
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig, nilPath)
			Expect(errorList).To(BeEmpty())
		})

		// TODO Test changed networks
	})
})
