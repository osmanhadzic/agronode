import { useEffect, useState } from 'react'
import { evaluateFuzzyTrigger } from '../api/telemetryApi'
import type { FuzzyTrigger, EvaluationResult, MembershipFunction } from '../api/telemetryApi'

type Props = {
  deviceId: string
  sensorId: string
}

export function FuzzyTriggerEditor({ deviceId, sensorId }: Props) {
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
      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ flex: 1 }}>
          <label>Membership Functions</label>
          {triggerObj?.membershipFunctions?.map((mf, idx) => (
            <div key={idx} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'center' }}>
              <input value={mf.name} onChange={(e) => {
                const next = { ...(triggerObj as FuzzyTrigger) }
                next.membershipFunctions[idx].name = e.target.value
                setTriggerObj(next)
              }} />
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
                  }} style={{ width: 80 }} />
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
            <label>Fuzzy Trigger (JSON)</label>
            <textarea value={triggerJson} onChange={(e) => setTriggerJson(e.target.value)} rows={8} style={{ width: '100%' }} />
          </div>
        </div>

        <div style={{ width: 360 }}>
          <label>Inputs (JSON)</label>
          <textarea value={inputsJson} onChange={(e) => setInputsJson(e.target.value)} rows={6} style={{ width: '100%' }} />
          <div style={{ marginTop: 8 }}>
            <button type="button" onClick={handleEvaluate} disabled={isLoading || !triggerObj}>
              {isLoading ? 'Evaluating...' : 'Evaluate'}
            </button>
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
