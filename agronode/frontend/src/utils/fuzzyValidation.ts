// validation util for fuzzy triggers

export type ValidationResult = {
  errors: string[]
  invalidMfIndexes: number[]
  invalidMfParamIndexesMap: Record<number, number[]>
  invalidRuleConditionIndexes: Record<number, number[]>
  invalidRuleActionIndexes: number[]
}

export function validateFuzzyConfig(cfg: any): ValidationResult {
  const errors: string[] = []
  const mfInvalid: number[] = []
  const mfParamInvalidMap: Record<number, number[]> = {}
  const ruleCondInvalid: Record<number, number[]> = {}
  const ruleActionInvalid: number[] = []

  if (!cfg) {
    return { errors, invalidMfIndexes: mfInvalid, invalidMfParamIndexesMap: mfParamInvalidMap, invalidRuleConditionIndexes: ruleCondInvalid, invalidRuleActionIndexes: ruleActionInvalid }
  }

  if (!Array.isArray(cfg.membershipFunctions)) {
    errors.push('membershipFunctions must be an array')
  } else if (cfg.membershipFunctions.length === 0) {
    errors.push('At least one membership function is required')
  } else {
    cfg.membershipFunctions.forEach((mf: any, idx: number) => {
      if (!mf.name || typeof mf.name !== 'string') {
        errors.push('Each membership function must have a name')
        mfInvalid.push(idx)
        return
      }
      if (mf.type !== 'triangle' && mf.type !== 'trapezoid') {
        errors.push(`Membership function ${mf.name} has invalid type`)
        mfInvalid.push(idx)
        return
      }
      if (!Array.isArray(mf.parameters) || mf.parameters.some((p: any) => typeof p !== 'number')) {
        errors.push(`Membership function ${mf.name} parameters must be numbers`)
        mfParamInvalidMap[idx] = mf.parameters.map((p: any, pi: number) => (typeof p !== 'number' ? pi : -1)).filter((v: number) => v >= 0)
        return
      }
      if (mf.type === 'triangle' && mf.parameters.length !== 3) {
        errors.push(`Triangle ${mf.name} requires 3 parameters`)
        mfParamInvalidMap[idx] = mf.parameters.map((_: any, pi: number) => pi)
        return
      }
      if (mf.type === 'trapezoid' && mf.parameters.length !== 4) {
        errors.push(`Trapezoid ${mf.name} requires 4 parameters`)
        mfParamInvalidMap[idx] = mf.parameters.map((_: any, pi: number) => pi)
        return
      }

      // ordering checks
      if (mf.type === 'triangle') {
        const [a, b, c] = mf.parameters
        const bad: number[] = []
        if (!(a < b)) { bad.push(0); bad.push(1) }
        if (!(b < c)) { bad.push(1); bad.push(2) }
        if (bad.length > 0) mfParamInvalidMap[idx] = Array.from(new Set(bad))
      } else {
        const [a, b, c, d] = mf.parameters
        const bad: number[] = []
        if (!(a <= b)) { bad.push(0); bad.push(1) }
        if (!(b <= c)) { bad.push(1); bad.push(2) }
        if (!(c <= d)) { bad.push(2); bad.push(3) }
        if (bad.length > 0) mfParamInvalidMap[idx] = Array.from(new Set(bad))
      }
    })
  }

  if (!Array.isArray(cfg.rules)) {
    errors.push('rules must be an array')
  } else {
    const mfNames = new Set((cfg.membershipFunctions || []).map((m: any) => m.name))
    cfg.rules.forEach((r: any, rIdx: number) => {
      if (!r.name) {
        errors.push('Each rule must have a name')
        return
      }
      if (!Array.isArray(r.conditions) || r.conditions.length === 0) {
        errors.push(`Rule ${r.name} must have at least one condition`)
        ruleCondInvalid[rIdx] = []
        return
      }
      r.conditions.forEach((c: any, cIdx: number) => {
        if (!c.membership) {
          errors.push(`Rule ${r.name} has a condition without membership`)
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
        if (!mfNames.has(c.membership)) {
          errors.push(`Rule ${r.name} references unknown membership ${c.membership}`)
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
      })
      if (!r.action || typeof r.action.value !== 'number') {
        errors.push(`Rule ${r.name} must have an action with numeric value`)
        ruleActionInvalid.push(rIdx)
        return
      }
    })
  }

  return { errors, invalidMfIndexes: mfInvalid, invalidMfParamIndexesMap: mfParamInvalidMap, invalidRuleConditionIndexes: ruleCondInvalid, invalidRuleActionIndexes: ruleActionInvalid }
}
