import type { MembershipFunction } from '../api/telemetryApi'

export type MembershipType = MembershipFunction['type']

type MembershipSpec = {
  label: string
  paramNames: string[]
  defaults: number[]
}

export const MEMBERSHIP_SPECS: Record<MembershipType, MembershipSpec> = {
  triangle: { label: 'Triangle', paramNames: ['a', 'b', 'c'], defaults: [0, 10, 20] },
  trapezoid: { label: 'Trapezoid', paramNames: ['a', 'b', 'c', 'd'], defaults: [0, 10, 20, 30] },
  left_shoulder: { label: 'Left shoulder', paramNames: ['a (full until)', 'b (zero from)'], defaults: [10, 20] },
  right_shoulder: { label: 'Right shoulder', paramNames: ['a (zero until)', 'b (full from)'], defaults: [10, 20] },
  gaussian: { label: 'Gaussian', paramNames: ['mean', 'sigma'], defaults: [20, 5] },
  bell: { label: 'Bell', paramNames: ['width', 'slope', 'center'], defaults: [5, 2, 20] },
  sigmoid: { label: 'Sigmoid', paramNames: ['center', 'slope'], defaults: [20, 1] },
}

export const MEMBERSHIP_TYPES = Object.keys(MEMBERSHIP_SPECS) as MembershipType[]

export function isMembershipType(value: unknown): value is MembershipType {
  return typeof value === 'string' && Object.prototype.hasOwnProperty.call(MEMBERSHIP_SPECS, value)
}

export function resizeParameters(type: MembershipType, current: number[]): number[] {
  const { defaults } = MEMBERSHIP_SPECS[type]
  return defaults.map((fallback, i) => (typeof current[i] === 'number' ? current[i] : fallback))
}

/** Returns an error message (or null) and the indexes of offending parameters. */
export function checkMembershipParameters(
  type: MembershipType,
  name: string,
  p: number[],
): { error: string; badIndexes: number[] } | null {
  const spec = MEMBERSHIP_SPECS[type]
  if (p.length !== spec.paramNames.length) {
    return {
      error: `${spec.label} ${name} requires ${spec.paramNames.length} parameters`,
      badIndexes: p.map((_, i) => i),
    }
  }

  const bad = new Set<number>()
  switch (type) {
    case 'triangle':
      if (!(p[0] < p[1])) { bad.add(0); bad.add(1) }
      if (!(p[1] < p[2])) { bad.add(1); bad.add(2) }
      break
    case 'trapezoid':
      for (let i = 0; i < 3; i++) if (!(p[i] <= p[i + 1])) { bad.add(i); bad.add(i + 1) }
      break
    case 'left_shoulder':
    case 'right_shoulder':
      if (!(p[0] < p[1])) { bad.add(0); bad.add(1) }
      break
    case 'gaussian':
      if (!(p[1] > 0)) bad.add(1)
      break
    case 'bell':
      if (!(p[0] > 0)) bad.add(0)
      if (!(p[1] > 0)) bad.add(1)
      break
    case 'sigmoid':
      if (p[1] === 0) bad.add(1)
      break
  }

  if (bad.size === 0) return null
  return {
    error: `${spec.label} ${name} has invalid parameters`,
    badIndexes: Array.from(bad).sort((x, y) => x - y),
  }
}

export function evalMembership(mf: MembershipFunction, x: number): number {
  const p = mf.parameters
  if (!isMembershipType(mf.type) || p.length < MEMBERSHIP_SPECS[mf.type].paramNames.length) return 0

  switch (mf.type) {
    case 'triangle': {
      const [a, b, c] = p
      if (x <= a || x >= c) return 0
      if (x === b) return 1
      return x < b ? (x - a) / (b - a) : (c - x) / (c - b)
    }
    case 'trapezoid': {
      const [a, b, c, d] = p
      if (x <= a || x >= d) return 0
      if (x >= b && x <= c) return 1
      return x < b ? (x - a) / (b - a) : (d - x) / (d - c)
    }
    case 'left_shoulder': {
      const [a, b] = p
      if (x <= a) return 1
      if (x >= b) return 0
      return (b - x) / (b - a)
    }
    case 'right_shoulder': {
      const [a, b] = p
      if (x <= a) return 0
      if (x >= b) return 1
      return (x - a) / (b - a)
    }
    case 'gaussian': {
      const [mean, sigma] = p
      if (!(sigma > 0)) return 0
      const d = (x - mean) / sigma
      return Math.exp(-0.5 * d * d)
    }
    case 'bell': {
      const [width, slope, center] = p
      if (!(width > 0) || !(slope > 0)) return 0
      return 1 / (1 + Math.pow(Math.abs((x - center) / width), 2 * slope))
    }
    case 'sigmoid': {
      const [center, slope] = p
      return 1 / (1 + Math.exp(-slope * (x - center)))
    }
  }
}

/** Sensible x-range [lo, hi] to draw a membership function over. */
export function membershipExtent(mf: MembershipFunction): [number, number] {
  const p = mf.parameters.filter((v) => Number.isFinite(v))
  if (p.length === 0) return [0, 1]
  switch (mf.type) {
    case 'left_shoulder':
    case 'right_shoulder': {
      const span = Math.abs((p[1] ?? p[0]) - p[0]) || 1
      return [p[0] - span * 0.5, (p[1] ?? p[0]) + span * 0.5]
    }
    case 'gaussian': {
      const sigma = Math.abs(p[1] ?? 1) || 1
      return [p[0] - 3 * sigma, p[0] + 3 * sigma]
    }
    case 'bell': {
      const width = Math.abs(p[0]) || 1
      const center = p[2] ?? 0
      return [center - 2 * width, center + 2 * width]
    }
    case 'sigmoid': {
      const slope = Math.abs(p[1] ?? 1) || 1
      return [p[0] - 6 / slope, p[0] + 6 / slope]
    }
    default:
      return [Math.min(...p), Math.max(...p)]
  }
}
