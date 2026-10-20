import { useEffect, useMemo, useState } from 'react'
import { evaluateFuzzyTrigger } from '../api/telemetryApi'
import type { FuzzyTrigger, EvaluationResult, MembershipFunction } from '../api/telemetryApi'
import { translate, useI18n } from '../i18n'
import { validateFuzzyConfig } from '../utils/fuzzyValidation'
import {
  evalMembership,
  membershipExtent,
  MEMBERSHIP_SPECS,
  MEMBERSHIP_TYPES,
  resizeParameters,
  type MembershipType,
} from '../utils/membership'

type Props = {
  deviceId: string
  sensorId: string
  availableSensors?: string[]
  initialFuzzyConfig?: FuzzyTrigger | string | null
  onChange?: (cfg: FuzzyTrigger | null) => void
  onValidChange?: (valid: boolean) => void
}

const DEFAULT_TRIGGER_JSON = `{
  "name": "Example",
  "membershipFunctions": [
    { "name": "low", "type": "triangle", "parameters": [0, 10, 20] },
    { "name": "high", "type": "triangle", "parameters": [15, 25, 35] }
  ],
  "rules": [
    { "name": "rule1", "conditions": [{"sensor": "","membership": "low"}], "operator": "AND", "action": {"type": "activate","value": 1} }
  ]
}`

function parseInitialConfig(config: FuzzyTrigger | string | null | undefined): FuzzyTrigger | null {
  if (config === null || config === undefined) {
    return null
  }

  try {
    return typeof config === 'string' ? JSON.parse(config) as FuzzyTrigger : config
  } catch {
    return null
  }
}

export function FuzzyTriggerEditor({ deviceId, sensorId, availableSensors, initialFuzzyConfig, onChange, onValidChange }: Props) {
  const { t } = useI18n()
  const initialTrigger = parseInitialConfig(initialFuzzyConfig)
  const [triggerJson, setTriggerJson] = useState<string>(
    () => initialTrigger ? JSON.stringify(initialTrigger, null, 2) : DEFAULT_TRIGGER_JSON,
  )
  const [inputsJson, setInputsJson] = useState<string>(`{ "${sensorId}": 12 }`)
  const [result, setResult] = useState<EvaluationResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [triggerObj, setTriggerObj] = useState<FuzzyTrigger | null>(
    () => initialTrigger ?? JSON.parse(DEFAULT_TRIGGER_JSON) as FuzzyTrigger,
  )
  const validation = useMemo(() => validateFuzzyConfig(triggerObj), [triggerObj])
  const {
    errors: validationErrors,
    invalidMfIndexes,
    invalidMfParamIndexesMap,
    invalidRuleConditionIndexes,
    invalidRuleActionIndexes,
  } = validation
  // rule name invalid indexes not used for per-field highlighting currently

  const commitTrigger = (next: FuzzyTrigger) => {
    setTriggerObj(next)
    setTriggerJson(JSON.stringify(next, null, 2))
  }

  const handleTriggerJsonChange = (value: string) => {
    setTriggerJson(value)
    try {
      const parsed = JSON.parse(value) as FuzzyTrigger
      setTriggerObj(parsed)
      setError(null)
    } catch {
      // Keep the last valid trigger while the JSON draft is incomplete.
    }
  }

  // notify parent when trigger object changes
  useEffect(() => {
    if (typeof onChange === 'function') {
      onChange(triggerObj ?? null)
    }
  }, [triggerObj, onChange])

  // Notify the parent when validation changes.
  useEffect(() => {
    if (typeof onValidChange === 'function') onValidChange(validationErrors.length === 0)
  }, [validationErrors.length, onValidChange])

  const handleEvaluate = async () => {
    setError(null)
    setResult(null)
    let trigger: FuzzyTrigger
    let inputs: Record<string, number> | undefined
    try {
      trigger = JSON.parse(triggerJson)
    } catch {
      setError(translate('fuzzy.err.triggerJson'))
      return
    }

    try {
      inputs = JSON.parse(inputsJson)
    } catch {
      setError(translate('fuzzy.err.inputsJson'))
      return
    }

    setIsLoading(true)
    try {
      const res = await evaluateFuzzyTrigger(deviceId, sensorId, { inputs, trigger })
      setResult(res)
    } catch (evaluationError) {
      setError(
        evaluationError instanceof Error
          ? evaluationError.message
          : translate('fuzzy.err.evalFailed'),
      )
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fuzzy-editor">
      {validationErrors.length > 0 && (
        <div style={{ border: '1px solid #e66', padding: 8, marginBottom: 12, background: '#fff7f7', color: '#900' }}>
          <strong>{t('fuzzy.validationErrors')}</strong>
          <ul style={{ marginTop: 6, marginBottom: 0 }}>
            {validationErrors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        </div>
      )}
      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ flex: 1 }}>
          <label>{t('fuzzy.membershipFunctions')}</label>
          {triggerObj?.membershipFunctions?.map((mf, idx) => (
            <div key={idx} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'center' }}>
              <input value={mf.name} onChange={(e) => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions[idx].name = e.target.value
                commitTrigger(next)
              }} style={ invalidMfIndexes.includes(idx) ? { borderColor: '#e66', borderWidth: 1 } : undefined } />
              <select value={mf.type} onChange={(e) => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions[idx].type = e.target.value as MembershipType
                next.membershipFunctions[idx].parameters = resizeParameters(e.target.value as MembershipType, next.membershipFunctions[idx].parameters)
                commitTrigger(next)
              }}>
                {MEMBERSHIP_TYPES.map((type) => (
                  <option key={type} value={type}>{MEMBERSHIP_SPECS[type].label}</option>
                ))}
              </select>
              <div style={{ display: 'flex', gap: 4 }}>
                {mf.parameters.map((p, pi) => (
                  <input key={pi} type="number" step="0.1" value={String(p)} title={MEMBERSHIP_SPECS[mf.type]?.paramNames[pi]} placeholder={MEMBERSHIP_SPECS[mf.type]?.paramNames[pi]} onChange={(e) => {
                    const val = Number(e.target.value)
                    const next = { ...(triggerObj as FuzzyTrigger) }
                    next.membershipFunctions[idx].parameters[pi] = val
                    commitTrigger(next)
                  }} style={ (invalidMfParamIndexesMap[idx] || []).includes(pi) ? { width: 80, borderColor: '#e66', borderWidth: 1 } : { width: 80 } } />
                ))}
              </div>
              <button type="button" onClick={() => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions.splice(idx, 1)
                commitTrigger(next)
              }}>{t('common.remove')}</button>
            </div>
          ))}
          <div style={{ marginTop: 8 }}>
            <button type="button" onClick={() => {
              const next: FuzzyTrigger = triggerObj ? { ...triggerObj } : { name: 'trigger', membershipFunctions: [], rules: [] }
              const name = `mf${(next.membershipFunctions?.length ?? 0) + 1}`
              const mf: MembershipFunction = { name, sensor: '', type: 'triangle', parameters: [0, 10, 20] }
              next.membershipFunctions = [...(next.membershipFunctions || []), mf]
              commitTrigger(next)
            }}>{t('fuzzy.addMembership')}</button>
            <button type="button" onClick={() => setTriggerJson(JSON.stringify(triggerObj, null, 2))} style={{ marginLeft: 8 }}>{t('fuzzy.syncJson')}</button>
          </div>

          <div style={{ marginTop: 12 }}>
            <h4>{t('fuzzy.rules')}</h4>
            {triggerObj?.rules?.map((rule, rIdx) => (
              <div key={rIdx} style={{ border: '1px solid #eee', padding: 8, marginBottom: 8 }}>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <input value={rule.name} onChange={(e) => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules[rIdx].name = e.target.value
                    commitTrigger(next)
                  }} style={{ flex: 1 }} />
                  <select value={rule.operator ?? 'AND'} onChange={(e) => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules[rIdx].operator = e.target.value as 'AND' | 'OR'
                    commitTrigger(next)
                  }}>
                    <option value="AND">AND</option>
                    <option value="OR">OR</option>
                  </select>
                  <button type="button" onClick={() => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules.splice(rIdx, 1)
                    commitTrigger(next)
                  }}>{t('fuzzy.removeRule')}</button>
                </div>

                <div style={{ marginTop: 8 }}>
                    <strong>{t('fuzzy.conditions')}</strong>
                    {(rule.conditions || []).map((cond, cIdx) => (
                      <div key={cIdx} style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 6 }}>
                        <select value={cond.sensor} onChange={(e) => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions[cIdx].sensor = e.target.value
                          commitTrigger(next)
                        }} style={{ flex: 1 }}>
                          {Array.from(new Set(['', sensorId, ...(availableSensors ?? []), cond.sensor])).map((key) => (
                            <option key={key} value={key}>{key || 'sensor'}</option>
                          ))}
                        </select>
                        <select value={cond.membership} onChange={(e) => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions[cIdx].membership = e.target.value
                          commitTrigger(next)
                        }} style={ (invalidRuleConditionIndexes[rIdx] || []).includes(cIdx) ? { borderColor: '#e66', borderWidth: 1 } : undefined }>
                          <option value="">{t('fuzzy.selectMf')}</option>
                          {triggerObj?.membershipFunctions?.map((mf) => (
                            <option key={mf.name} value={mf.name}>{mf.name}</option>
                          ))}
                        </select>
                        <button type="button" onClick={() => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions.splice(cIdx, 1)
                          commitTrigger(next)
                        }}>{t('common.remove')}</button>
                      </div>
                    ))}
                  <div style={{ marginTop: 8 }}>
                    <button type="button" onClick={() => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].conditions = next.rules[rIdx].conditions || []
                      next.rules[rIdx].conditions.push({ sensor: sensorId || '', membership: '' })
                      commitTrigger(next)
                    }}>{t('fuzzy.addCondition')}</button>
                  </div>
                </div>

                <div style={{ marginTop: 8 }}>
                  <strong>{t('fuzzy.action')}</strong>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 6 }}>
                    <select value={rule.action?.type ?? 'activate'} onChange={(e) => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].action = next.rules[rIdx].action || { type: e.target.value, value: 1 }
                      next.rules[rIdx].action.type = e.target.value
                      if (e.target.value === 'http') {
                        if (!next.rules[rIdx].action.method) next.rules[rIdx].action.method = 'POST'
                        if (!next.rules[rIdx].action.url) next.rules[rIdx].action.url = ''
                      }
                      commitTrigger(next)
                    }}>
                      <option value="activate">activate</option>
                      <option value="set">set</option>
                      <option value="http">http</option>
                    </select>
                    <input type="number" step="0.1" value={String(rule.action?.value ?? 1)} onChange={(e) => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].action = next.rules[rIdx].action || { type: 'activate', value: 1 }
                      next.rules[rIdx].action.value = Number(e.target.value)
                      commitTrigger(next)
                    }} style={ invalidRuleActionIndexes.includes(rIdx) ? { width: 120, borderColor: '#e66', borderWidth: 1 } : { width: 120 } } />
                  </div>
                  {rule.action?.type === 'http' && (
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 8 }}>
                      <select value={rule.action?.method ?? 'POST'} onChange={(e) => {
                        const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                        next.rules[rIdx].action = next.rules[rIdx].action || { type: 'http', value: 1 }
                        next.rules[rIdx].action.method = e.target.value
                        commitTrigger(next)
                      }}>
                        <option value="POST">POST</option>
                        <option value="GET">GET</option>
                        <option value="PUT">PUT</option>
                        <option value="PATCH">PATCH</option>
                      </select>
                      <input placeholder="https://example.com/webhook" value={rule.action?.url ?? ''} onChange={(e) => {
                        const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                        next.rules[rIdx].action = next.rules[rIdx].action || { type: 'http', value: 1 }
                        next.rules[rIdx].action.url = e.target.value
                        commitTrigger(next)
                      }} style={ invalidRuleActionIndexes.includes(rIdx) ? { flex: 1, borderColor: '#e66', borderWidth: 1 } : { flex: 1 } } />
                    </div>
                  )}
                  {rule.action?.type === 'http' && (
                    <div style={{ marginTop: 8 }}>
                      <textarea placeholder={t('fuzzy.webhookBody')} value={rule.action?.body ?? ''} onChange={(e) => {
                        const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                        next.rules[rIdx].action = next.rules[rIdx].action || { type: 'http', value: 1 }
                        next.rules[rIdx].action.body = e.target.value
                        commitTrigger(next)
                      }} rows={3} style={{ width: '100%' }} />
                    </div>
                  )}
                </div>
              </div>
            ))}
            <div>
              <button type="button" onClick={() => {
                const next = triggerObj ? JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger : { name: 'trigger', membershipFunctions: [], rules: [] }
                const rName = `rule${(next.rules?.length ?? 0) + 1}`
                next.rules = next.rules || []
                next.rules.push({ name: rName, conditions: [{ sensor: sensorId || '', membership: '' }], operator: 'AND', action: { type: 'activate', value: 1 } })
                commitTrigger(next)
              }}>{t('fuzzy.addRule')}</button>
            </div>
          </div>

          <div style={{ marginTop: 12 }}>
            <label>{t('fuzzy.triggerJson')}</label>
            <textarea value={triggerJson} onChange={(e) => handleTriggerJsonChange(e.target.value)} rows={8} style={{ width: '100%' }} />
          </div>
        </div>

        <div style={{ width: 360 }}>
          <label>{t('fuzzy.inputsJson')}</label>
          <textarea value={inputsJson} onChange={(e) => setInputsJson(e.target.value)} rows={6} style={{ width: '100%' }} />
            <div style={{ marginTop: 8 }}>
            <button type="button" onClick={handleEvaluate} disabled={isLoading || !triggerObj || validationErrors.length > 0}>
              {isLoading ? t('fuzzy.evaluating') : t('fuzzy.evaluate')}
            </button>
            {validationErrors.length > 0 && (
              <div style={{ color: '#900', marginTop: 8 }}>{t('fuzzy.fixErrors')}</div>
            )}
          </div>
          {error && <div style={{ color: 'var(--color-danger)', marginTop: 8 }}>{error}</div>}

          {triggerObj && (
            <div style={{ marginTop: 12 }}>
              <label>{t('fuzzy.preview')}</label>
              <div style={{ border: '1px solid #ddd', padding: 8 }}>
                <MembershipsPreview membershipFunctions={triggerObj.membershipFunctions} />
              </div>
            </div>
          )}
        </div>
      </div>

      <div style={{ marginTop: 12 }}>
        <label>{t('fuzzy.result')}</label>
        {result ? (
          <div style={{ display: 'flex', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <h4>{t('fuzzy.memberships')}</h4>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr><th>{t('fuzzy.colSensor')}</th><th>{t('fuzzy.colMembership')}</th><th>{t('fuzzy.colDegree')}</th></tr>
                </thead>
                <tbody>
                  {Object.entries(result.memberships).flatMap(([sensor, map]) => (
                    Object.entries(map).map(([m, deg]) => (
                      <tr key={`${sensor}-${m}`}><td>{sensor}</td><td>{m}</td><td>{deg.toFixed(3)}</td></tr>
                    ))
                  ))}
                </tbody>
              </table>
            </div>
            <div style={{ width: 320 }}>
              <h4>{t('fuzzy.rules')}</h4>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr><th>{t('fuzzy.colName')}</th><th>{t('fuzzy.colStrength')}</th><th>{t('fuzzy.colValue')}</th></tr>
                </thead>
                <tbody>
                  {result.rules.map((r) => (
                    <tr key={r.name}><td>{r.name}</td><td>{r.strength.toFixed(3)}</td><td>{r.value.toFixed(3)}</td></tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <pre style={{ maxHeight: 300, overflow: 'auto', background: '#f7f7f7', padding: 8 }}>{t('fuzzy.noResult')}</pre>
        )}
      </div>
    </div>
  )
}

export default FuzzyTriggerEditor

function MembershipsPreview({ membershipFunctions }: { membershipFunctions: MembershipFunction[] }) {
  // compute x range
  let min = Infinity
  let max = -Infinity
  membershipFunctions.forEach((mf) => {
    const [lo, hi] = membershipExtent(mf)
    if (lo < min) min = lo
    if (hi > max) max = hi
  })
  if (!isFinite(min) || !isFinite(max)) { min = 0; max = 1 }
  if (min === max) { max = min + 1 }

  const width = 320, height = 100, padding = 8
  const scaleX = (x: number) => padding + ((x - min) / (max - min)) * (width - padding*2)
  const scaleY = (y: number) => height - padding - y * (height - padding*2)

  const colors = ['#2b8cbe','#7bccc4','#f03b20','#f1a340','#6a51a3']

  return (
    <svg width={width} height={height} style={{ display: 'block' }}>
      <rect x={0} y={0} width={width} height={height} fill="#fff" stroke="#eee" />
      {membershipFunctions.map((mf, i) => {
        const pts: [number, number][] = []
        const steps = 40
        for (let s = 0; s <= steps; s++) {
          const x = min + (s/steps)*(max-min)
          const y = evalMembershipAt(mf, x)
          pts.push([scaleX(x), scaleY(y)])
        }
        const path = pts.map(([x,y]) => `${x},${y}`).join(' ')
        return (
          <polyline key={i} points={path} fill="none" stroke={colors[i % colors.length]} strokeWidth={2} />
        )
      })}
    </svg>
  )
}

function evalMembershipAt(mf: MembershipFunction, x: number) {
  return evalMembership(mf, x)
}
