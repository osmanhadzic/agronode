import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react'

import {
  fetchAllTelemetry,
  fetchDevices,
  fetchTelemetryByDeviceId,
  fetchLatestTelemetryByDeviceId,
  type DateFilterPeriod,
} from '../api/telemetryApi'
import { createDeviceStatusSocket, createTelemetrySocket } from '../api/telemetrySocket'
import { DateFilter } from '../components/DateFilter'
import { DataModeSelector } from '../components/DataModeSelector'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import type { TelemetryReading } from '../types/telemetry'

const TelemetryLineChart = lazy(() =>
  import('../charts/TelemetryLineChart').then((module) => ({
    default: module.TelemetryLineChart,
  })),
)

function downsampleData(
  data: TelemetryReading[],
  period: DateFilterPeriod,
): TelemetryReading[] {
  if (data.length <= 100) {
    return data
  }

  let targetSize: number

  switch (period) {
    case 'hour':
      targetSize = 60
      break
    case 'day':
      targetSize = 96
      break
    case 'week':
      targetSize = 168
      break
    case 'month':
      targetSize = 120
      break
    case 'year':
      targetSize = 365
      break
    case 'custom':
      targetSize = 200
      break
    default:
      targetSize = 100
  }

  if (data.length <= targetSize) {
    return data
  }

  const interval = Math.max(1, Math.floor(data.length / targetSize))
  const sampled: TelemetryReading[] = []

  for (let i = 0; i < data.length; i += interval) {
    sampled.push(data[i])
  }

  if (sampled[sampled.length - 1] !== data[data.length - 1]) {
    sampled.push(data[data.length - 1])
  }

  return sampled
}

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [liveData, setLiveData] = useState<TelemetryReading[]>([])
  const [deviceStatuses, setDeviceStatuses] = useState<Record<string, string>>({})
  const [latestDeviceReading, setLatestDeviceReading] =
    useState<TelemetryReading | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [dataMode, setDataMode] = useState<'live' | 'history'>('live')
  const [dateFilterPeriod, setDateFilterPeriod] =
    useState<DateFilterPeriod>('hour')
  const [customStartDate, setCustomStartDate] = useState<string>()
  const [customEndDate, setCustomEndDate] = useState<string>()
  const [updateQueue, setUpdateQueue] = useState<TelemetryReading[]>([])

  // Batch websocket messages to avoid excessive UI updates.
  useEffect(() => {
    if (updateQueue.length === 0) {
      return
    }

    const timeoutId = setTimeout(() => {
      setLiveData((previous) => {
        const newReadings = updateQueue.filter(
          (reading) =>
            !previous.some(
              (item) =>
                item.deviceId === reading.deviceId &&
                item.createdAt === reading.createdAt,
            ),
        )

        if (newReadings.length === 0) {
          return previous
        }

        const oneHourAgo = new Date(
          Date.now() - 60 * 60 * 1000,
        ).toISOString()

        return [...newReadings, ...previous]
          .filter((item) => item.createdAt >= oneHourAgo)
          .sort(
            (a, b) =>
              new Date(b.createdAt).getTime() -
              new Date(a.createdAt).getTime(),
          )
      })

      setUpdateQueue([])
    }, 500)

    return () => clearTimeout(timeoutId)
  }, [updateQueue])

  useEffect(() => {
    let isMounted = true

    async function loadTelemetry() {
      setIsLoading(true)
      setError('')

      try {
        let readings: TelemetryReading[]

        if (dataMode === 'live') {
          if (selectedDeviceId) {
            readings = await fetchTelemetryByDeviceId(selectedDeviceId, {
              period: 'hour',
            })
          } else {
            readings = await fetchAllTelemetry()
          }
        } else if (selectedDeviceId && dateFilterPeriod) {
          readings = await fetchTelemetryByDeviceId(selectedDeviceId, {
            period: dateFilterPeriod,
            startDate: customStartDate,
            endDate: customEndDate,
          })
        } else if (selectedDeviceId) {
          readings = await fetchTelemetryByDeviceId(selectedDeviceId)
        } else {
          readings = await fetchAllTelemetry()
        }

        const devices = await fetchDevices()

        if (!isMounted) {
          return
        }

        if (dataMode === 'live') {
          setLiveData(readings)
        } else {
          setTelemetry(readings)
        }

        setDeviceStatuses(
          devices.reduce<Record<string, string>>((result, device) => {
            result[device.deviceId] = device.status
            return result
          }, {}),
        )

        const availableDeviceIds = [
          ...new Set([
            ...readings.map((reading) => reading.deviceId),
            ...devices.map((device) => device.deviceId),
          ]),
        ]

        if (!selectedDeviceId) {
          setSelectedDeviceId(availableDeviceIds[0] ?? '')
        }

        if (selectedDeviceId) {
          const latest = await fetchLatestTelemetryByDeviceId(selectedDeviceId)

          if (isMounted) {
            setLatestDeviceReading(latest)
          }
        } else {
          setLatestDeviceReading(readings[0] ?? null)
        }
      } catch {
        if (isMounted) {
          setError('Failed to load telemetry data')
        }
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    void loadTelemetry()

    const cleanupSocket = createTelemetrySocket(
      (reading) => {
        if (!isMounted) {
          return
        }

        setError('')
        setLatestDeviceReading((previous) =>
          reading.deviceId === selectedDeviceId ? reading : previous,
        )

        setDeviceStatuses((previous) => ({
          ...previous,
          [reading.deviceId]: 'online',
        }))

        setSelectedDeviceId((previous) => previous || reading.deviceId)

        if (dataMode === 'live') {
          setUpdateQueue((previous) => [...previous, reading])
        }
      },
      () => {
        if (isMounted) {
          setError('Realtime connection lost. Reconnecting...')
        }
      },
    )

    const cleanupDeviceStatusSocket = createDeviceStatusSocket(
      (event) => {
        if (!isMounted) {
          return
        }

        setDeviceStatuses((previous) => ({
          ...previous,
          [event.deviceId]: event.newStatus,
        }))

        setSelectedDeviceId((previous) => previous || event.deviceId)
      },
      () => {
        if (isMounted) {
          setError('Realtime connection lost. Reconnecting...')
        }
      },
    )

    return () => {
      isMounted = false
      cleanupSocket()
      cleanupDeviceStatusSocket()
    }
  }, [
    selectedDeviceId,
    dataMode,
    dateFilterPeriod,
    customStartDate,
    customEndDate,
  ])

  const devices = useMemo(() => {
    const currentData = dataMode === 'live' ? liveData : telemetry

    return [
      ...new Set([
        ...currentData.map((reading) => reading.deviceId),
        ...Object.keys(deviceStatuses),
        ...(selectedDeviceId ? [selectedDeviceId] : []),
      ]),
    ]
  }, [telemetry, liveData, dataMode, deviceStatuses, selectedDeviceId])

  const deviceTelemetry = useMemo(() => {
    if (!selectedDeviceId) {
      return []
    }

    const data = dataMode === 'live' ? liveData : telemetry
    const filtered = data.filter(
      (reading) => reading.deviceId === selectedDeviceId,
    )

    if (dataMode === 'history' && filtered.length > 100) {
      return downsampleData(filtered, dateFilterPeriod)
    }

    return filtered
  }, [
    selectedDeviceId,
    telemetry,
    liveData,
    dataMode,
    dateFilterPeriod,
  ])

  const latestReading = deviceTelemetry[0] ?? null

  const selectedDeviceStatus = selectedDeviceId
    ? deviceStatuses[selectedDeviceId] ?? 'unknown'
    : 'unknown'

  const availableSensors = useMemo(() => {
    const sensorSet = new Set<string>(['temperature', 'humidity'])

    for (const reading of deviceTelemetry) {
      for (const sensorKey of Object.keys(reading.sensors ?? {})) {
        sensorSet.add(sensorKey)
      }
    }

    return [...sensorSet]
  }, [deviceTelemetry])

  const latestSensors = useMemo(() => {
    const reading = latestDeviceReading ?? latestReading

    if (!reading) {
      return null
    }

    return {
      temperature: reading.temperature,
      humidity: reading.humidity,
      ...(reading.sensors ?? {}),
    } as Record<string, number>
  }, [latestDeviceReading, latestReading])

  const visibleSensors = useMemo(() => {
    if (availableSensors.length === 0) {
      return []
    }

    if (selectedSensors.length === 0) {
      return availableSensors
    }

    const filtered = selectedSensors.filter((sensorKey) =>
      availableSensors.includes(sensorKey),
    )

    return filtered.length > 0 ? filtered : availableSensors
  }, [availableSensors, selectedSensors])

  const formatSensorLabel = (sensorKey: string) => {
    if (sensorKey === 'co2') {
      return 'CO₂'
    }

    return sensorKey
      .split('_')
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ')
  }

  const getSensorUnit = (sensorKey: string) => {
    if (sensorKey === 'temperature') {
      return '°C'
    }

    if (sensorKey === 'humidity') {
      return '%'
    }

    if (sensorKey === 'co2') {
      return 'ppm'
    }

    return ''
  }

  const handleToggleSensor = useCallback(
    (sensorKey: string) => {
      setSelectedSensors((previous) => {
        const currentSelection =
          previous.length === 0 ? [...availableSensors] : [...previous]

        if (currentSelection.includes(sensorKey)) {
          const remaining = currentSelection.filter((key) => key !== sensorKey)
          return remaining.length > 0 ? remaining : previous
        }

        return [...currentSelection, sensorKey]
      })
    },
    [availableSensors],
  )

  const handleDateFilterChange = useCallback(
    (period: DateFilterPeriod, startDate?: string, endDate?: string) => {
      setDateFilterPeriod(period)
      setCustomStartDate(startDate)
      setCustomEndDate(endDate)
    },
    [],
  )

  return (
    <main className="dashboard">
      <header className="dashboard-header">
        <div>
          <h1 className="dashboard-title">AgroNode Dashboard</h1>

          {selectedDeviceId && (
            <p className="device-status">
              Status:{' '}
              <span
                className={`device-status-value device-status-${selectedDeviceStatus}`}
              >
                {selectedDeviceStatus}
              </span>
            </p>
          )}
        </div>

        {devices.length > 0 && (
          <DeviceSelector
            devices={devices}
            selectedDeviceId={selectedDeviceId}
            onChange={setSelectedDeviceId}
          />
        )}
      </header>

      <DeviceMetaPanel meta={latestDeviceReading?.meta} />

      <DataModeSelector mode={dataMode} onChange={setDataMode} />

      {isLoading && (
        <p className="dashboard-message">Loading telemetry...</p>
      )}

      {error && <p className="dashboard-message">{error}</p>}

      {!isLoading && !error && devices.length === 0 && (
        <p className="dashboard-message">No telemetry data available</p>
      )}

      <SensorVisibilitySelector
        sensors={availableSensors}
        selectedSensors={visibleSensors}
        onToggleSensor={handleToggleSensor}
      />

      <section className="sensor-grid">
        {visibleSensors.map((sensorKey) => (
          <SensorCard
            key={sensorKey}
            label={formatSensorLabel(sensorKey)}
            value={latestSensors?.[sensorKey] ?? null}
            unit={getSensorUnit(sensorKey)}
          />
        ))}
      </section>

      {dataMode === 'history' && selectedDeviceId && (
        <DateFilter
          onFilterChange={handleDateFilterChange}
          selectedPeriod={dateFilterPeriod}
        />
      )}

      {dataMode === 'live' && (
        <div
          style={{
            textAlign: 'center',
            padding: '0.75rem',
            backgroundColor: '#e7f3ff',
            borderRadius: '6px',
            margin: '1rem 0',
            color: '#0066cc',
            fontWeight: '500',
          }}
        >
          📡 Prikazano: Poslednji sat podataka u realnom vremenu
        </div>
      )}

      {dataMode === 'history' && deviceTelemetry.length > 0 && (() => {
        const originalData = telemetry.filter(
          (reading) => reading.deviceId === selectedDeviceId,
        )
        const isDownsampled =
          originalData.length > 100 &&
          deviceTelemetry.length < originalData.length

        return isDownsampled ? (
          <div
            style={{
              textAlign: 'center',
              padding: '0.75rem',
              backgroundColor: '#fff3cd',
              borderRadius: '6px',
              margin: '1rem 0',
              color: '#856404',
              fontWeight: '500',
            }}
          >
            📊 Prikazano: {deviceTelemetry.length} od {originalData.length}{' '}
            podataka (optimizovano za performanse)
          </div>
        ) : null
      })()}

      <Suspense
        fallback={
          <div
            style={{
              textAlign: 'center',
              padding: '2rem',
              color: '#6c757d',
            }}
          >
            Loading chart...
          </div>
        }
      >
        <TelemetryLineChart
          data={deviceTelemetry}
          selectedSensors={visibleSensors}
        />
      </Suspense>
    </main>
  )
}
