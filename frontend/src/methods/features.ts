// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import type { Ref } from 'vue'
import { computed, ref } from 'vue'
import type { Resource } from '@/api/grpc'
import { ResourceService } from '@/api/grpc'
import type { FeaturesConfigSpec } from '@/api/omni/specs/omni.pb'
import Watch from '@/api/watch'
import { Runtime } from '@/api/common/omni.pb'
import { FeaturesConfigType, DefaultNamespace, FeaturesConfigID } from '@/api/resources'
import { withRuntime } from '@/api/options'

export const setupWorkloadProxyingEnabledFeatureWatch = (): Ref<boolean> => {
  const featuresConfig: Ref<Resource<FeaturesConfigSpec> | undefined> = ref()
  const featuresConfigWatch = new Watch(featuresConfig)
  featuresConfigWatch.setup({
    resource: {
      type: FeaturesConfigType,
      namespace: DefaultNamespace,
      id: FeaturesConfigID,
    },
    runtime: Runtime.Omni,
  })

  return computed(() => {
    return featuresConfig?.value?.spec?.enable_workload_proxying ?? false
  })
}

let cachedFeaturesConfig: Resource<FeaturesConfigSpec> | undefined

export const embeddedDiscoveryServiceFeatureAvailable = async (): Promise<boolean> => {
  const featuresConfig = await getFeaturesConfig()

  return featuresConfig.spec?.embedded_discovery_service ?? false
}

export const auditLogEnabled = async (): Promise<boolean> => {
  const featuresConfig = await getFeaturesConfig()

  return featuresConfig.spec?.audit_log_enabled ?? false
}

export const getImageFactoryBaseURL = async (): Promise<string> => {
  const featuresConfig = await getFeaturesConfig()

  if (!featuresConfig.spec?.image_factory_base_url) {
    throw new Error('image_factory_base_url is not set in features config')
  }

  return featuresConfig.spec?.image_factory_base_url
}

const getFeaturesConfig = async (): Promise<Resource<FeaturesConfigSpec>> => {
  if (!cachedFeaturesConfig) {
    cachedFeaturesConfig = await ResourceService.Get(
      {
        type: FeaturesConfigType,
        namespace: DefaultNamespace,
        id: FeaturesConfigID,
      },
      withRuntime(Runtime.Omni),
    )
  }

  return cachedFeaturesConfig
}
