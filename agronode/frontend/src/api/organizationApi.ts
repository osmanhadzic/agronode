import { httpClient } from './httpClient'

import type { OrganizationSummary } from '../types/organization'

export async function fetchOrganizations(): Promise<OrganizationSummary[]> {
  const { data } = await httpClient.get<OrganizationSummary[]>('/api/organizations')
  return data
}
