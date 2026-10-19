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

type UnknownRecord = Record<string, unknown>

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isNumberArray(value: unknown): value is number[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'number')
}

export function validateFuzzyConfig(cfg: unknown): ValidationResult {
  const errors: string[] = []
  const mfInvalid: number[] = []
  const mfParamInvalidMap: Record<number, number[]> = {}
  const ruleCondInvalid: Record<number, number[]> = {}
  const ruleActionInvalid: number[] = []

  if (!cfg) {
    return { errors, invalidMfIndexes: mfInvalid, invalidMfParamIndexesMap: mfParamInvalidMap, invalidRuleConditionIndexes: ruleCondInvalid, invalidRuleActionIndexes: ruleActionInvalid }
  }

  const config = isRecord(cfg) ? cfg : {}

  if (!Array.isArray(config.membershipFunctions)) {
    errors.push(translate('fuzzy.err.mfArray'))
  } else if (config.membershipFunctions.length === 0) {
    errors.push(translate('fuzzy.err.noMembership'))
  } else {
    config.membershipFunctions.forEach((value, idx) => {
      const mf = isRecord(value) ? value : {}
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
      if (!isNumberArray(mf.parameters)) {
        errors.push(translate('fuzzy.err.mfParams', { name: mf.name }))
        mfParamInvalidMap[idx] = Array.isArray(mf.parameters)
          ? mf.parameters.flatMap((parameter, parameterIndex) =>
              typeof parameter === 'number' ? [] : [parameterIndex],
            )
          : []
        return
      }
      const check = checkMembershipParameters(
        mf.type,
        mf.name,
        mf.parameters,
      )
      if (check) {
        errors.push(check.error)
        mfParamInvalidMap[idx] = check.badIndexes
      }
    })
  }

  if (!Array.isArray(config.rules)) {
    errors.push(translate('fuzzy.err.rulesArray'))
  } else {
    const mfNames = new Set(
      (Array.isArray(config.membershipFunctions) ? config.membershipFunctions : [])
        .flatMap((value) => isRecord(value) && typeof value.name === 'string' ? [value.name] : []),
    )
    config.rules.forEach((value, rIdx) => {
      const rule = isRecord(value) ? value : {}
      if (typeof rule.name !== 'string' || !rule.name) {
        errors.push(translate('fuzzy.err.ruleName'))
        return
      }
      const ruleName = rule.name
      if (!Array.isArray(rule.conditions) || rule.conditions.length === 0) {
        errors.push(translate('fuzzy.err.ruleNoCondition', { rule: ruleName }))
        ruleCondInvalid[rIdx] = []
        return
      }
      rule.conditions.forEach((value, cIdx) => {
        const condition = isRecord(value) ? value : {}
        const membership = condition.membership
        if (typeof membership !== 'string' || !membership) {
          errors.push(translate('fuzzy.err.ruleNoMembership', { rule: ruleName }))
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
        if (!mfNames.has(membership)) {
          errors.push(translate('fuzzy.err.ruleUnknownMf', { rule: ruleName, membership }))
          ruleCondInvalid[rIdx] = ruleCondInvalid[rIdx] || []
          ruleCondInvalid[rIdx].push(cIdx)
          return
        }
      })
      const action = isRecord(rule.action) ? rule.action : null
      if (!action || typeof action.value !== 'number') {
        errors.push(translate('fuzzy.err.ruleAction', { rule: ruleName }))
        ruleActionInvalid.push(rIdx)
        return
      }

      if (action.type === 'http') {
        const url = typeof action.url === 'string' ? action.url.trim() : ''
        if (!url) {
          errors.push(translate('fuzzy.err.httpUrl', { rule: ruleName }))
          ruleActionInvalid.push(rIdx)
          return
        }

        if (!/^https?:\/\//i.test(url)) {
          errors.push(translate('fuzzy.err.httpScheme', { rule: ruleName }))
          ruleActionInvalid.push(rIdx)
          return
        }
      }
    })
  }

  return { errors, invalidMfIndexes: mfInvalid, invalidMfParamIndexesMap: mfParamInvalidMap, invalidRuleConditionIndexes: ruleCondInvalid, invalidRuleActionIndexes: ruleActionInvalid }
}
