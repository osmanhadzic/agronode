// validation util for fuzzy triggers
import { checkMembershipParameters, isMembershipType } from './membership'

import { translate } from '../i18n'

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
    errors.push(translate('fuzzy.err.mfArray'))
  } else if (cfg.membershipFunctions.length === 0) {
    errors.push(translate('fuzzy.err.noMembership'))
  } else {
    cfg.membershipFunctions.forEach((mf: any, idx: number) => {
      if (!mf.name || typeof mf.name !== 'string') {
        errors.push(translate('fuzzy.err.membershipName'))
        mfInvalid.push(idx)
        return
      }
      if (!isMembershipType(mf.type)) {
        errors.push(`Membership function ${mf.name} has invalid type`)
        mfInvalid.push(idx)
        return
      }
      if (!Array.isArray(mf.parameters) || mf.parameters.some((p: any) => typeof p !== 'number')) {
        errors.push(translate('fuzzy.err.mfParams', { name: mf.name }))
        mfParamInvalidMap[idx] = mf.parameters.map((p: any, pi: number) => (typeof p !== 'number' ? pi : -1)).filter((v: number) => v >= 0)
        return
      }
      const check = checkMembershipParameters(mf.type, mf.name, mf.parameters)
      if (check) {
        errors.push(check.error)
        mfParamInvalidMap[idx] = check.badIndexes
      }
    })
  }

  if (!Array.isArray(cfg.rules)) {
    errors.push(translate('fuzzy.err.rulesArray'))
  } else {
    const mfNames = new Set((cfg.membershipFunctions || []).map((m: any) => m.name))
    cfg.rules.forEach((r: any, rIdx: number) => {
      if (!r.name) {
        errors.push(translate('fuzzy.err.ruleName'))
        return
      }
      if (!Array.isArray(r.conditions) || r.conditions.length === 0) {
        errors.push(translate('fuzzy.err.ruleNoCondition', { rule: r.name }))
        ruleCondInvalid[rIdx] = []
        return
      }
      r.conditions.forEach((c: any, cIdx: number) => {
        if (!c.membership) {
          errors.push(translate('fuzzy.err.ruleNoMembership', { rule: r.name }))
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
        if (!mfNames.has(c.membership)) {
          errors.push(translate('fuzzy.err.ruleUnknownMf', { rule: r.name, membership: c.membership }))
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
      })
      if (!r.action || typeof r.action.value !== 'number') {
        errors.push(translate('fuzzy.err.ruleAction', { rule: r.name }))
        ruleActionInvalid.push(rIdx)
        return
      }

      if (r.action.type === 'http') {
        const url = typeof r.action.url === 'string' ? r.action.url.trim() : ''
        if (!url) {
          errors.push(translate('fuzzy.err.httpUrl', { rule: r.name }))
          ruleActionInvalid.push(rIdx)
          return
        }

        if (!/^https?:\/\//i.test(url)) {
          errors.push(translate('fuzzy.err.httpScheme', { rule: r.name }))
          ruleActionInvalid.push(rIdx)
          return
        }
      }
    })
  }

  return { errors, invalidMfIndexes: mfInvalid, invalidMfParamIndexesMap: mfParamInvalidMap, invalidRuleConditionIndexes: ruleCondInvalid, invalidRuleActionIndexes: ruleActionInvalid }
}
