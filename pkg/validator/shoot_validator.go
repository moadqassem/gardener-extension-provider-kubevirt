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

package validator

import (
	"context"
	"fmt"

	"github.com/gardener/gardener-extension-provider-kubevirt/pkg/apis/kubevirt"
	"github.com/gardener/gardener-extension-provider-kubevirt/pkg/apis/kubevirt/helper"
	kubevirtvalidation "github.com/gardener/gardener-extension-provider-kubevirt/pkg/apis/kubevirt/validation"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type validationContext struct {
	shoot              *core.Shoot
	infraConfig        *kubevirt.InfrastructureConfig
	cpConfig           *kubevirt.ControlPlaneConfig
	cloudProfile       *gardencorev1beta1.CloudProfile
	cloudProfileConfig *kubevirt.CloudProfileConfig
}

var (
	specPath           = field.NewPath("spec")
	providerConfigPath = specPath.Child("providerConfig")
	nwPath             = specPath.Child("networking")
	providerPath       = specPath.Child("provider")
	infraConfigPath    = providerPath.Child("infrastructureConfig")
	cpConfigPath       = providerPath.Child("controlPlaneConfig")
	workersPath        = providerPath.Child("workers")
)

func (v *Shoot) validateShootCreation(ctx context.Context, shoot *core.Shoot) error {
	valContext, err := v.newValidationContext(ctx, v.client, shoot)
	if err != nil {
		return err
	}

	allErrs := field.ErrorList{}

	allErrs = append(allErrs, kubevirtvalidation.ValidateInfrastructureConfigAgainstCloudProfile(valContext.infraConfig, shoot.Spec.Region, valContext.cloudProfileConfig, infraConfigPath)...)
	allErrs = append(allErrs, kubevirtvalidation.ValidateControlPlaneConfigAgainstCloudProfile(valContext.cpConfig, shoot.Spec.Region, valContext.cloudProfile, valContext.cloudProfileConfig, cpConfigPath)...)
	allErrs = append(allErrs, v.validateShoot(valContext)...)
	return allErrs.ToAggregate()
}

func (v *Shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	oldValContext, err := v.newValidationContext(ctx, v.client, oldShoot)
	if err != nil {
		return err
	}

	valContext, err := v.newValidationContext(ctx, v.client, shoot)
	if err != nil {
		return err
	}

	allErrs := field.ErrorList{}

	allErrs = append(allErrs, kubevirtvalidation.ValidateInfrastructureConfigUpdate(oldValContext.infraConfig, valContext.infraConfig, infraConfigPath)...)
	// Only validate against cloud profile when related configuration is updated.
	// This ensures that already running shoots won't break after constraints were removed from the cloud profile.
	if kubevirtvalidation.HasRelevantInfrastructureConfigUpdates(oldValContext.infraConfig, valContext.infraConfig) {
		allErrs = append(allErrs, kubevirtvalidation.ValidateInfrastructureConfigAgainstCloudProfile(valContext.infraConfig, shoot.Spec.Region, valContext.cloudProfileConfig, infraConfigPath)...)
	}

	allErrs = append(allErrs, kubevirtvalidation.ValidateControlPlaneConfigUpdate(oldValContext.cpConfig, valContext.cpConfig, cpConfigPath)...)
	// Only validate against cloud profile when related configuration is updated.
	// This ensures that already running shoots won't break after constraints were removed from the cloud profile.
	if kubevirtvalidation.HasRelevantControlPlaneConfigUpdates(oldValContext.cpConfig, valContext.cpConfig) {
		allErrs = append(allErrs, kubevirtvalidation.ValidateControlPlaneConfigAgainstCloudProfile(valContext.cpConfig, shoot.Spec.Region, valContext.cloudProfile, valContext.cloudProfileConfig, cpConfigPath)...)
	}

	allErrs = append(allErrs, kubevirtvalidation.ValidateWorkersUpdate(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers, workersPath)...)
	allErrs = append(allErrs, v.validateShoot(valContext)...)
	return allErrs.ToAggregate()
}

func (v *Shoot) validateShoot(context *validationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, kubevirtvalidation.ValidateNetworking(context.shoot.Spec.Networking, nwPath)...)
	allErrs = append(allErrs, kubevirtvalidation.ValidateInfrastructureConfig(context.infraConfig, infraConfigPath)...)
	allErrs = append(allErrs, kubevirtvalidation.ValidateControlPlaneConfig(context.cpConfig, cpConfigPath)...)
	allErrs = append(allErrs, kubevirtvalidation.ValidateWorkers(context.shoot.Spec.Provider.Workers, workersPath)...)
	return allErrs
}

func (v *Shoot) newValidationContext(ctx context.Context, c client.Client, shoot *core.Shoot) (*validationContext, error) {
	infraConfig := &kubevirt.InfrastructureConfig{}
	if shoot.Spec.Provider.InfrastructureConfig != nil {
		var err error
		infraConfig, err = helper.DecodeInfrastructureConfig(shoot.Spec.Provider.InfrastructureConfig, infraConfigPath)
		if err != nil {
			return nil, err
		}
	}

	if shoot.Spec.Provider.ControlPlaneConfig == nil {
		return nil, field.Required(cpConfigPath, "controlPlaneConfig must be set for kubevirt shoots")
	}
	cpConfig, err := helper.DecodeControlPlaneConfig(shoot.Spec.Provider.ControlPlaneConfig, cpConfigPath)
	if err != nil {
		return nil, err
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := c.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), cloudProfile); err != nil {
		return nil, err
	}

	if cloudProfile.Spec.ProviderConfig == nil {
		return nil, fmt.Errorf("providerConfig is not given for cloud profile %q", cloudProfile.Name)
	}
	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile.Spec.ProviderConfig, providerConfigPath)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while reading the cloud profile %q: %v", cloudProfile.Name, err)
	}

	for _, worker := range shoot.Spec.Provider.Workers {
		_, err := helper.DecodeWorkerConfig(v.decoder, worker.ProviderConfig)
		if err != nil {
			return nil, fmt.Errorf("an error occoured while decoding workerConfig: %v", err)
		}
	}

	return &validationContext{
		shoot:              shoot,
		infraConfig:        infraConfig,
		cpConfig:           cpConfig,
		cloudProfile:       cloudProfile,
		cloudProfileConfig: cloudProfileConfig,
	}, nil
}
