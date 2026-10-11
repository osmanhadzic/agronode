import {
  Area,
  Brush,
  ReferenceArea,
  ReferenceLine,
  CartesianGrid,
  ComposedChart,
  Line,
    ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { useI18n } from '../i18n'
import type { TelemetryAggregate, TelemetryReading, TriggerExecutionEvent, TriggerListItem } from '../types/telemetry'

type TelemetryLineChartProps = {
  data: TelemetryReading[]
  selectedSensors: string[]
  statsData?: TelemetryReading[]
  mode?: 'live' | 'history'
  aggregate?: TelemetryAggregate | null
  triggers?: TriggerListItem[]
  triggerEvents?: TriggerExecutionEvent[]
  deviceId?: string
}

type ChartPoint = {
  time: number
  value: number
}

type GapRange = { from: number; to: number }

type GapResult<T extends { time: number }> = {
  points: (T | { time: number; value: null; range: null })[]
  gaps: GapRange[]
}

// A gap is a pause much longer than the typical sampling interval (device offline).
function detectGaps<T extends { time: number }>(points: T[]): GapResult<T> {
  if (points.length < 3) {
    return { points, gaps: [] }
  }

  const deltas = points.slice(1).map((point, i) => point.time - points[i].time).sort((a, b) => a - b)
  const median = deltas[Math.floor(deltas.length / 2)]
  const threshold = Math.max(median * 2.5, 2 * 60 * 1000)

  const result: GapResult<T> = { points: [], gaps: [] }
  points.forEach((point, i) => {
    if (i > 0 && point.time - points[i - 1].time > threshold) {
      result.gaps.push({ from: points[i - 1].time, to: point.time })
      result.points.push({ time: points[i - 1].time + 1, value: null, range: null })
    }
    result.points.push(point)
  })

  return result
}

function formatDuration(ms: number): string {
  const minutes = Math.round(ms / 60000)
  if (minutes < 60) return minutes + ' min'
  const hours = Math.floor(minutes / 60)
  if (hours < 48) return hours + ' h ' + (minutes % 60) + ' min'
  return Math.round(hours / 24) + ' d'
}

type SensorStats = {
  min: number
  max: number
  avg: number
}

const DAY_MS = 24 * 60 * 60 * 1000

function formatAxisTime(ts: number, spanMs: number, locale: string): string {
  const date = new Date(ts)
  if (spanMs <= DAY_MS) {
    return date.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })
  }
  if (spanMs <= 7 * DAY_MS) {
    return date.toLocaleString(locale, {
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }
  if (spanMs <= 400 * DAY_MS) {
    return date.toLocaleDateString(locale, { day: '2-digit', month: '2-digit' })
  }
  return date.toLocaleDateString(locale, { month: '2-digit', year: 'numeric' })
}

function formatTooltipTime(ts: number, locale: string): string {
  return new Date(ts).toLocaleString(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function csvCell(value: string | number): string {
  const text = String(value)
  return /[",\n]/.test(text) ? '"' + text.replace(/"/g, '""') + '"' : text
}

function downloadCsv(filename: string, rows: (string | number)[][]) {
  const content = rows.map((row) => row.map(csvCell).join(',')).join('\n')
  const url = URL.createObjectURL(new Blob([content], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function computeStats(values: number[]): SensorStats | null {
  if (values.length === 0) {
    return null
  }
  let min = values[0]
  let max = values[0]
  let sum = 0
  for (const value of values) {
    if (value < min) min = value
    if (value > max) max = value
    sum += value
  }
  return { min, max, avg: sum / values.length }
}

const lineColors = ['#2563eb', '#16a34a', '#a855f7', '#f59e0b', '#ef4444', '#0ea5e9']

function getReadingSensorValues(reading: TelemetryReading): Record<string, number> {
  const readingSensors = reading.sensors ?? {}
  if (Object.keys(readingSensors).length > 0) {
    return readingSensors
  }

  const fallback: Record<string, number> = {}

  if (!reading.sensorId) {
    if (typeof reading.temperature === 'number') {
      fallback.temperature = reading.temperature
    }
    if (typeof reading.humidity === 'number') {
      fallback.humidity = reading.humidity
    }

    return fallback
  }

  if (
    (reading.sensorId === 'temperature' || reading.sensorId === 'dht11-temp') &&
    typeof reading.temperature === 'number'
  ) {
    fallback[reading.sensorId] = reading.temperature
  }

  if (
    (reading.sensorId === 'humidity' ||
      reading.sensorId === 'humidity_dht11' ||
      reading.sensorId === 'dht11-humidity') &&
    typeof reading.humidity === 'number'
  ) {
    fallback[reading.sensorId] = reading.humidity
  }

  return fallback
}

function toSensorLabel(sensorKey: string): string {
  if (sensorKey === 'co2') {
    return 'CO₂'
  }

  if (sensorKey === 'dht11-temp') {
    return 'DHT11 Temp'
  }

  if (sensorKey === 'dht11-humidity' || sensorKey === 'humidity_dht11') {
    return 'DHT11 Humidity'
  }

  if (sensorKey === 'signal_strength') {
    return 'Signal Strength'
  }

  return sensorKey
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function TelemetryLineChart({
  data,
  selectedSensors,
  statsData,
  mode = 'live',
  aggregate = null,
  triggers = [],
  triggerEvents = [],
  deviceId = 'device',
}: TelemetryLineChartProps) {
  const { t, locale } = useI18n()
  const sourceForStats = statsData ?? data
  const times = data
    .map((reading) => new Date(reading.createdAt).getTime())
    .filter((time) => !Number.isNaN(time))
  const minTime = times.length ? Math.min(...times) : 0
  const maxTime = times.length ? Math.max(...times) : 0
  const aggTimes = (aggregate?.buckets ?? []).map((bucket) => new Date(bucket.time).getTime())
  const spanMs =
    aggTimes.length > 1 ? Math.max(...aggTimes) - Math.min(...aggTimes) : maxTime - minTime

  const pointsBySensor = selectedSensors.reduce<Record<string, ChartPoint[]>>(
    (result, sensorKey) => {
      result[sensorKey] = [...data]
        .reverse()
        .filter((reading) => !Number.isNaN(new Date(reading.createdAt).getTime()))
        .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
        .flatMap((reading) => {
          const sensorValues = getReadingSensorValues(reading)
          const value = sensorValues[sensorKey]

          if (value === undefined) {
            return []
          }

          return [
            {
              time: new Date(reading.createdAt).getTime(),
              value,
            },
          ]
        })

      return result
    },
    {},
  )

  const exportCsv = () => {
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
    if (mode === 'history' && aggregate && aggregate.buckets.length > 0) {
      const rows: (string | number)[][] = [['time', 'sensor', 'min', 'max', 'avg', 'count']]
      for (const bucket of aggregate.buckets) {
        for (const sensorKey of selectedSensors) {
          const stat = bucket.sensors[sensorKey]
          if (stat) {
            rows.push([bucket.time, sensorKey, stat.min, stat.max, stat.avg, stat.count])
          }
        }
      }
      downloadCsv(deviceId + '-aggregate-' + stamp + '.csv', rows)
      return
    }

    const rows: (string | number)[][] = [['time', 'sensor', 'value']]
    for (const sensorKey of selectedSensors) {
      for (const point of pointsBySensor[sensorKey] ?? []) {
        rows.push([new Date(point.time).toISOString(), sensorKey, point.value])
      }
    }
    downloadCsv(deviceId + '-telemetry-' + stamp + '.csv', rows)
  }

  return (
    <div className="chart-panel">
      <h2 className="chart-title">
        {mode === 'history' ? t('chart.history') : t('chart.live')}
      </h2>
      {selectedSensors.length > 0 && (
        <button type="button" className="chart-export-button" onClick={exportCsv}>
          {t('chart.export')}
        </button>
      )}
      {selectedSensors.length === 0 ? (
        <p className="dashboard-message">{t('chart.noSensors')}</p>
      ) : (
        <div className="chart-grid">
          {selectedSensors.map((sensorKey, index) => {
            const bandPoints = (aggregate?.buckets ?? []).flatMap((bucket) => {
              const stat = bucket.sensors[sensorKey]
              return stat
                ? [{ time: new Date(bucket.time).getTime(), value: stat.avg, range: [stat.min, stat.max] as [number, number] }]
                : []
            })
            const useBand = mode === 'history' && bandPoints.length > 0
            const trigger = triggers.find((item) => item.sensorId === sensorKey)
            const { points: chartData, gaps } = detectGaps(
              useBand ? bandPoints : (pointsBySensor[sensorKey] ?? []),
            )
            const sensorEvents = triggerEvents
              .map((event) => ({ event, time: new Date(event.triggeredAt).getTime() }))
              .filter(
                ({ event, time }) =>
                  event.sensorId === sensorKey &&
                  time >= (chartData[0]?.time ?? Infinity) &&
                  time <= (chartData[chartData.length - 1]?.time ?? -Infinity),
              )
            const summaryStats = aggregate?.summary[sensorKey]
            const stats = useBand && summaryStats ? summaryStats : computeStats(
              sourceForStats.flatMap((reading) => {
                const value = getReadingSensorValues(reading)[sensorKey]
                return value === undefined ? [] : [value]
              }),
            )

            return (
            <section key={sensorKey} className="chart-card">
              <h3 className="chart-card-title">{toSensorLabel(sensorKey)}</h3>
              {stats && (
                <div className="chart-stats">
                  <span>{t('chart.min')}: <strong>{stats.min.toFixed(1)}</strong></span>
                  <span>{t('chart.max')}: <strong>{stats.max.toFixed(1)}</strong></span>
                  <span>{t('chart.avg')}: <strong>{stats.avg.toFixed(1)}</strong></span>
                </div>
              )}
              <ResponsiveContainer width="100%" height={220}>
                <ComposedChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="time"
                    type="number"
                    scale="time"
                    domain={['dataMin', 'dataMax']}
                    tickFormatter={(ts: number) => formatAxisTime(ts, spanMs, locale)}
                    minTickGap={40}
                  />
                  <YAxis domain={['auto', 'auto']} />
                  <Tooltip
                    labelFormatter={(ts) => formatTooltipTime(Number(ts), locale)}
                    formatter={(value, name) => [
                      Array.isArray(value)
                        ? `${Number(value[0]).toFixed(2)} – ${Number(value[1]).toFixed(2)}`
                        : typeof value === 'number'
                          ? value.toFixed(2)
                          : value,
                      name === t('chart.range') ? t('chart.range') : useBand ? t('chart.avgSeries', { sensor: toSensorLabel(sensorKey) }) : toSensorLabel(sensorKey),
                    ]}
                  />
                  {useBand && (
                    <Area
                      type="monotone"
                      dataKey="range"
                      stroke="none"
                      fill={lineColors[index % lineColors.length]}
                      fillOpacity={0.15}
                      name={t('chart.range')}
                      isAnimationActive={false}
                    />
                  )}
                  {mode === 'history' &&
                    gaps.map((gap) => (
                      <ReferenceArea
                        key={'gap-' + gap.from}
                        x1={gap.from}
                        x2={gap.to}
                        fill="#9ca3af"
                        fillOpacity={0.2}
                        stroke="none"
                        label={{ value: t('chart.offline', { duration: formatDuration(gap.to - gap.from) }), fontSize: 10, fill: '#6b7280', position: 'insideTop' }}
                      />
                    ))}
                  {sensorEvents.map(({ event, time }) => (
                    <ReferenceLine
                      key={'trigger-' + event.ruleId + '-' + time}
                      x={time}
                      stroke={event.status === 'succeeded' ? '#f59e0b' : '#dc2626'}
                      strokeWidth={2}
                      label={{
                        value: '⚡ ' + new Date(time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }),
                        fontSize: 10,
                        fill: '#b45309',
                        position: 'top',
                      }}
                    />
                  ))}
                  {typeof trigger?.min === 'number' && (
                    <ReferenceLine
                      y={trigger.min}
                      stroke="#ef4444"
                      strokeDasharray="6 4"
                      label={{ value: t('chart.minLimit', { value: trigger.min }), fontSize: 11, fill: '#ef4444', position: 'insideTopLeft' }}
                    />
                  )}
                  {typeof trigger?.max === 'number' && (
                    <ReferenceLine
                      y={trigger.max}
                      stroke="#ef4444"
                      strokeDasharray="6 4"
                      label={{ value: t('chart.maxLimit', { value: trigger.max }), fontSize: 11, fill: '#ef4444', position: 'insideBottomLeft' }}
                    />
                  )}
                  <Line
                    type="monotone"
                    dataKey="value"
                    stroke={lineColors[index % lineColors.length]}
                    strokeWidth={2}
                    dot={false}
                    name={toSensorLabel(sensorKey)}
                  />
                  {mode === 'history' && chartData.length > 20 && (
                    <Brush
                      dataKey="time"
                      height={24}
                      stroke="#94a3b8"
                      tickFormatter={(ts: number) => formatAxisTime(ts, spanMs, locale)}
                    />
                  )}
                </ComposedChart>
              </ResponsiveContainer>
            </section>
            )
          })}
        </div>
      )}
    </div>
  )
}
