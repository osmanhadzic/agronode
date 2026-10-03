import { useEffect, useState } from 'react'
import { evaluateFuzzyTrigger } from '../api/telemetryApi'
import type { FuzzyTrigger, EvaluationResult, MembershipFunction } from '../api/telemetryApi'
import { validateFuzzyConfig } from '../utils/fuzzyValidation'

type Props = {
  deviceId: string
  sensorId: string
  initialFuzzyConfig?: any
  onChange?: (cfg: any) => void
  onValidChange?: (valid: boolean) => void
}

export function FuzzyTriggerEditor({ deviceId, sensorId, initialFuzzyConfig, onChange, onValidChange }: Props) {
  const [triggerJson, setTriggerJson] = useState<string>(`{
  "name": "Example",
  "membershipFunctions": [
    { "name": "low", "type": "triangle", "parameters": [0, 10, 20] },
    { "name": "high", "type": "triangle", "parameters": [15, 25, 35] }
  ],
  "rules": [
    { "name": "rule1", "conditions": [{"sensor": "","membership": "low"}], "operator": "AND", "action": {"type": "activate","value": 1} }
  ]
}`)
  const [inputsJson, setInputsJson] = useState<string>(`{ "${sensorId}": 12 }`)
  const [result, setResult] = useState<EvaluationResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [triggerObj, setTriggerObj] = useState<FuzzyTrigger | null>(null)
  const [validationErrors, setValidationErrors] = useState<string[]>([])
  const [invalidMfIndexes, setInvalidMfIndexes] = useState<number[]>([])
  const [invalidMfParamIndexesMap, setInvalidMfParamIndexesMap] = useState<Record<number, number[]>>({})
  // rule name invalid indexes not used for per-field highlighting currently
  const [invalidRuleConditionIndexes, setInvalidRuleConditionIndexes] = useState<Record<number, number[]>>({})
  const [invalidRuleActionIndexes, setInvalidRuleActionIndexes] = useState<number[]>([])

  // keep triggerObj and triggerJson in sync
  useEffect(() => {
    try {
      const parsed = JSON.parse(triggerJson) as FuzzyTrigger
      setTriggerObj(parsed)
      setError(null)
    } catch {
      // ignore parse errors here
    }
  }, [triggerJson])

  useEffect(() => {
    if (triggerObj) {
      setTriggerJson(JSON.stringify(triggerObj, null, 2))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [triggerObj])

  // notify parent when trigger object changes
  useEffect(() => {
    if (typeof onChange === 'function') {
      onChange(triggerObj ?? null)
    }
  }, [triggerObj, onChange])

  // validate triggerObj and notify parent about validity
  useEffect(() => {
    const errors: string[] = []
    const mfInvalid: number[] = []
    const mfParamInvalidMap: Record<number, number[]> = {}
    const ruleNameInvalid: number[] = []
    const ruleCondInvalid: Record<number, number[]> = {}
    const ruleActionInvalid: number[] = []
    const cfg = triggerObj
    if (!cfg) {
      setValidationErrors([])
      if (typeof onValidChange === 'function') onValidChange(true)
      return
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
        // parameter count checks
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
        // ordering checks: mark specific parameter indexes that violate ordering
        if (mf.type === 'triangle') {
          const [a, b, c] = mf.parameters
          const bad: number[] = []
          if (!(a < b)) {
            bad.push(0)
            bad.push(1)
          }
          if (!(b < c)) {
            bad.push(1)
            bad.push(2)
          }
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
          ruleNameInvalid.push(rIdx)
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

    setValidationErrors(errors)
    setInvalidMfIndexes(mfInvalid)
    setInvalidMfParamIndexesMap(mfParamInvalidMap)
    setInvalidRuleConditionIndexes(ruleCondInvalid)
    setInvalidRuleActionIndexes(ruleActionInvalid)
    if (typeof onValidChange === 'function') onValidChange(errors.length === 0)
  }, [triggerObj, onValidChange])
  // validate using shared util and notify parent
  useEffect(() => {
    const res = validateFuzzyConfig(triggerObj as any)
    setValidationErrors(res.errors)
    setInvalidMfIndexes(res.invalidMfIndexes)
    setInvalidMfParamIndexesMap(res.invalidMfParamIndexesMap)
    setInvalidRuleConditionIndexes(res.invalidRuleConditionIndexes)
    setInvalidRuleActionIndexes(res.invalidRuleActionIndexes)
    if (typeof onValidChange === 'function') onValidChange(res.errors.length === 0)
  }, [triggerObj, onValidChange])

  // initialize from saved config when provided
  useEffect(() => {
    // @ts-ignore
    if (typeof initialFuzzyConfig !== 'undefined' && initialFuzzyConfig !== null) {
      try {
        const cfg = typeof initialFuzzyConfig === 'string' ? JSON.parse(initialFuzzyConfig) : initialFuzzyConfig
        setTriggerObj(cfg)
        setTriggerJson(JSON.stringify(cfg, null, 2))
      } catch (e) {
        // ignore parse errors
      }
    }
  }, [initialFuzzyConfig])

  const handleEvaluate = async () => {
    setError(null)
    setResult(null)
    let trigger: FuzzyTrigger
    let inputs: Record<string, number> | undefined
    try {
      trigger = JSON.parse(triggerJson)
    } catch (e) {
      setError('Invalid trigger JSON')
      return
    }

    try {
      inputs = JSON.parse(inputsJson)
    } catch (e) {
      setError('Invalid inputs JSON')
      return
    }

    setIsLoading(true)
    try {
      const res = await evaluateFuzzyTrigger(deviceId, sensorId, { inputs, trigger })
      setResult(res)
    } catch (e: any) {
      setError(e?.message ?? 'Evaluation failed')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fuzzy-editor">
      {validationErrors.length > 0 && (
        <div style={{ border: '1px solid #e66', padding: 8, marginBottom: 12, background: '#fff7f7', color: '#900' }}>
          <strong>Validation errors:</strong>
          <ul style={{ marginTop: 6, marginBottom: 0 }}>
            {validationErrors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        </div>
      )}
      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ flex: 1 }}>
          <label>Membership Functions</label>
          {triggerObj?.membershipFunctions?.map((mf, idx) => (
            <div key={idx} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'center' }}>
              <input value={mf.name} onChange={(e) => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions[idx].name = e.target.value
                setTriggerObj(next)
              }} style={ invalidMfIndexes.includes(idx) ? { borderColor: '#e66', borderWidth: 1 } : undefined } />
              <select value={mf.type} onChange={(e) => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions[idx].type = e.target.value as any
                // adjust parameters length
                if (e.target.value === 'triangle') {
                  next.membershipFunctions[idx].parameters = next.membershipFunctions[idx].parameters.slice(0, 3)
                  while (next.membershipFunctions[idx].parameters.length < 3) next.membershipFunctions[idx].parameters.push(0)
                } else {
                  for (let i = 0; i < 4; i++) if (next.membershipFunctions[idx].parameters[i] === undefined) next.membershipFunctions[idx].parameters[i] = 0
                }
                setTriggerObj(next)
              }}>
                <option value="triangle">Triangle</option>
                <option value="trapezoid">Trapezoid</option>
              </select>
              <div style={{ display: 'flex', gap: 4 }}>
                {mf.parameters.map((p, pi) => (
                  <input key={pi} type="number" step="0.1" value={String(p)} onChange={(e) => {
                    const val = Number(e.target.value)
                    const next = { ...(triggerObj as FuzzyTrigger) }
                    next.membershipFunctions[idx].parameters[pi] = val
                    setTriggerObj(next)
                  }} style={ (invalidMfParamIndexesMap[idx] || []).includes(pi) ? { width: 80, borderColor: '#e66', borderWidth: 1 } : { width: 80 } } />
                ))}
              </div>
              <button type="button" onClick={() => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions.splice(idx, 1)
                setTriggerObj(next)
              }}>Remove</button>
            </div>
          ))}
          <div style={{ marginTop: 8 }}>
            <button type="button" onClick={() => {
              const next: FuzzyTrigger = triggerObj ? { ...triggerObj } : { name: 'trigger', membershipFunctions: [], rules: [] }
              const name = `mf${(next.membershipFunctions?.length ?? 0) + 1}`
              const mf: MembershipFunction = { name, sensor: '', type: 'triangle', parameters: [0, 10, 20] }
              next.membershipFunctions = [...(next.membershipFunctions || []), mf]
              setTriggerObj(next)
            }}>Add Membership</button>
            <button type="button" onClick={() => setTriggerJson(JSON.stringify(triggerObj, null, 2))} style={{ marginLeft: 8 }}>Sync JSON</button>
          </div>

          <div style={{ marginTop: 12 }}>
            <h4>Rules</h4>
            {triggerObj?.rules?.map((rule, rIdx) => (
              <div key={rIdx} style={{ border: '1px solid #eee', padding: 8, marginBottom: 8 }}>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <input value={rule.name} onChange={(e) => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules[rIdx].name = e.target.value
                    setTriggerObj(next)
                  }} style={{ flex: 1 }} />
                  <select value={rule.operator ?? 'AND'} onChange={(e) => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules[rIdx].operator = e.target.value as any
                    setTriggerObj(next)
                  }}>
                    <option value="AND">AND</option>
                    <option value="OR">OR</option>
                  </select>
                  <button type="button" onClick={() => {
                    const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                    next.rules.splice(rIdx, 1)
                    setTriggerObj(next)
                  }}>Remove Rule</button>
                </div>

                <div style={{ marginTop: 8 }}>
                    <strong>Conditions</strong>
                    {(rule.conditions || []).map((cond, cIdx) => (
                      <div key={cIdx} style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 6 }}>
                        <input placeholder="sensor" value={cond.sensor} onChange={(e) => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions[cIdx].sensor = e.target.value
                          setTriggerObj(next)
                        }} style={{ flex: 1 }} />
                        <select value={cond.membership} onChange={(e) => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions[cIdx].membership = e.target.value
                          setTriggerObj(next)
                        }} style={ (invalidRuleConditionIndexes[rIdx] || []).includes(cIdx) ? { borderColor: '#e66', borderWidth: 1 } : undefined }>
                          <option value="">Select MF</option>
                          {triggerObj?.membershipFunctions?.map((mf) => (
                            <option key={mf.name} value={mf.name}>{mf.name}</option>
                          ))}
                        </select>
                        <button type="button" onClick={() => {
                          const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                          next.rules[rIdx].conditions.splice(cIdx, 1)
                          setTriggerObj(next)
                        }}>Remove</button>
                      </div>
                    ))}
                  <div style={{ marginTop: 8 }}>
                    <button type="button" onClick={() => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].conditions = next.rules[rIdx].conditions || []
                      next.rules[rIdx].conditions.push({ sensor: sensorId || '', membership: '' })
                      setTriggerObj(next)
                    }}>Add Condition</button>
                  </div>
                </div>

                <div style={{ marginTop: 8 }}>
                  <strong>Action</strong>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 6 }}>
                    <select value={rule.action?.type ?? 'activate'} onChange={(e) => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].action = next.rules[rIdx].action || { type: e.target.value, value: 1 }
                      next.rules[rIdx].action.type = e.target.value
                      setTriggerObj(next)
                    }}>
                      <option value="activate">activate</option>
                      <option value="set">set</option>
                    </select>
                    <input type="number" step="0.1" value={String(rule.action?.value ?? 1)} onChange={(e) => {
                      const next = JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger
                      next.rules[rIdx].action = next.rules[rIdx].action || { type: 'activate', value: 1 }
                      next.rules[rIdx].action.value = Number(e.target.value)
                      setTriggerObj(next)
                    }} style={ invalidRuleActionIndexes.includes(rIdx) ? { width: 120, borderColor: '#e66', borderWidth: 1 } : { width: 120 } } />
                  </div>
                </div>
              </div>
            ))}
            <div>
              <button type="button" onClick={() => {
                const next = triggerObj ? JSON.parse(JSON.stringify(triggerObj)) as FuzzyTrigger : { name: 'trigger', membershipFunctions: [], rules: [] }
                const rName = `rule${(next.rules?.length ?? 0) + 1}`
                next.rules = next.rules || []
                next.rules.push({ name: rName, conditions: [{ sensor: sensorId || '', membership: '' }], operator: 'AND', action: { type: 'activate', value: 1 } })
                setTriggerObj(next)
              }}>Add Rule</button>
            </div>
          </div>

          <div style={{ marginTop: 12 }}>
            <label>Fuzzy Trigger (JSON)</label>
            <textarea value={triggerJson} onChange={(e) => setTriggerJson(e.target.value)} rows={8} style={{ width: '100%' }} />
          </div>
        </div>

        <div style={{ width: 360 }}>
          <label>Inputs (JSON)</label>
          <textarea value={inputsJson} onChange={(e) => setInputsJson(e.target.value)} rows={6} style={{ width: '100%' }} />
            <div style={{ marginTop: 8 }}>
            <button type="button" onClick={handleEvaluate} disabled={isLoading || !triggerObj || validationErrors.length > 0}>
              {isLoading ? 'Evaluating...' : 'Evaluate'}
            </button>
            {validationErrors.length > 0 && (
              <div style={{ color: '#900', marginTop: 8 }}>Fix validation errors to evaluate or save.</div>
            )}
          </div>
          {error && <div style={{ color: 'var(--color-danger)', marginTop: 8 }}>{error}</div>}

          {triggerObj && (
            <div style={{ marginTop: 12 }}>
              <label>Preview</label>
              <div style={{ border: '1px solid #ddd', padding: 8 }}>
                <MembershipsPreview membershipFunctions={triggerObj.membershipFunctions} />
              </div>
            </div>
          )}
        </div>
      </div>

      <div style={{ marginTop: 12 }}>
        <label>Result</label>
        {result ? (
          <div style={{ display: 'flex', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <h4>Memberships</h4>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr><th>Sensor</th><th>Membership</th><th>Degree</th></tr>
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
              <h4>Rules</h4>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr><th>Name</th><th>Strength</th><th>Value</th></tr>
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
          <pre style={{ maxHeight: 300, overflow: 'auto', background: '#f7f7f7', padding: 8 }}>No result yet</pre>
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
    mf.parameters.forEach((p) => {
      if (p < min) min = p
      if (p > max) max = p
    })
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
  const p = mf.parameters
  if (mf.type === 'triangle') {
    if (p.length < 3) return 0
    const a = p[0], b = p[1], c = p[2]
    if (x <= a || x >= c) return 0
    if (x === b) return 1
    if (x > a && x < b) return (x-a)/(b-a)
    return (c-x)/(c-b)
  }
  // trapezoid
  if (p.length < 4) return 0
  const a = p[0], b = p[1], c = p[2], d = p[3]
  if (x <= a || x >= d) return 0
  if (x >= b && x <= c) return 1
  if (x > a && x < b) return (x-a)/(b-a)
  return (d-x)/(d-c)
}
