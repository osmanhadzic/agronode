import { useEffect, useMemo, useState } from 'react'

import { fetchAllTelemetry, fetchDevices } from '../api/telemetryApi'
import { createDeviceStatusSocket, createTelemetrySocket } from '../api/telemetrySocket'
import { TelemetryLineChart } from '../charts/TelemetryLineChart'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import type { TelemetryReading } from '../types/telemetry'

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [deviceStatuses, setDeviceStatuses] = useState<Record<string, string>>({})
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let isMounted = true

    async function loadTelemetry() {
      setIsLoading(true)
      setError('')

      try {
        const [readings, devices] = await Promise.all([
          fetchAllTelemetry(),
          fetchDevices(),
        ])

        if (!isMounted) {
          return
        }

        setTelemetry(readings)

        const initialStatuses = devices.reduce<Record<string, string>>((result, device) => {
          result[device.deviceId] = device.status
          return result
        }, {})
        setDeviceStatuses(initialStatuses)

        const availableDeviceIds = [
          ...new Set([
            ...readings.map((reading) => reading.deviceId),
            ...devices.map((device) => device.deviceId),
          ]),
        ]
        const firstDeviceId = availableDeviceIds[0] ?? ''

        setSelectedDeviceId((previous) => {
          if (!previous) {
            return firstDeviceId
          }

          if (!availableDeviceIds.includes(previous)) {
            return firstDeviceId
          }

          return previous
        })

        setError('')
      } catch {
        if (!isMounted) {
          return
        }

        setError('Failed to load telemetry data')
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

        setTelemetry((previous) => {
          const exists = previous.some(
            (item) =>
              item.deviceId === reading.deviceId &&
              item.createdAt === reading.createdAt &&
              item.temperature === reading.temperature &&
              item.humidity === reading.humidity &&
              JSON.stringify(item.sensors ?? {}) === JSON.stringify(reading.sensors ?? {}),
          )

          if (exists) {
            return previous
          }

          return [reading, ...previous]
        })

        setDeviceStatuses((previous) => ({
          ...previous,
          [reading.deviceId]: 'online',
        }))

        setSelectedDeviceId((previous) => previous || reading.deviceId)
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
  }, [])

  const devices = useMemo(() => {
    return [...new Set([...telemetry.map((reading) => reading.deviceId), ...Object.keys(deviceStatuses)])]
  }, [telemetry, deviceStatuses])

  const deviceTelemetry = useMemo(() => {
    if (!selectedDeviceId) {
      return []
    }

    return telemetry.filter((reading) => reading.deviceId === selectedDeviceId)
  }, [selectedDeviceId, telemetry])

  const latestReading = deviceTelemetry[0] ?? null

  const selectedDeviceStatus = selectedDeviceId
    ? deviceStatuses[selectedDeviceId] ?? 'unknown'
    : 'unknown'

  const availableSensors = useMemo(() => {
    const sensorSet = new Set<string>(['temperature', 'humidity'])

    for (const reading of deviceTelemetry) {
      const sensors = reading.sensors ?? {}
      for (const sensorKey of Object.keys(sensors)) {
        sensorSet.add(sensorKey)
      }
    }

    return [...sensorSet]
  }, [deviceTelemetry])

  const latestSensors = useMemo(() => {
    if (!latestReading) {
      return null
    }

    const normalized: Record<string, number> = {
      temperature: latestReading.temperature,
      humidity: latestReading.humidity,
      ...(latestReading.sensors ?? {}),
    }

    return normalized
  }, [latestReading])

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

  const handleToggleSensor = (sensorKey: string) => {
    setSelectedSensors((previous) => {
      const currentSelection =
        previous.length === 0 ? [...availableSensors] : [...previous]

      if (currentSelection.includes(sensorKey)) {
        const remaining = currentSelection.filter((key) => key !== sensorKey)
        return remaining.length > 0 ? remaining : previous
      }

      return [...currentSelection, sensorKey]
    })
  }

  return (
    <main className="dashboard">
      <header className="dashboard-header">
        <div>
          <h1 className="dashboard-title">AgroNode Dashboard</h1>
          {selectedDeviceId && (
            <p className="device-status">
              Status: <span className={`device-status-value device-status-${selectedDeviceStatus}`}>{selectedDeviceStatus}</span>
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

      <DeviceMetaPanel meta={latestReading?.meta} />

      {isLoading && <p className="dashboard-message">Loading telemetry...</p>}
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

      <TelemetryLineChart data={deviceTelemetry} selectedSensors={visibleSensors} />
    </main>
  )
}
