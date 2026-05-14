import { useEffect, useMemo, useState } from 'react'

import { fetchAllTelemetry, fetchTelemetryByDeviceId } from '../api/telemetryApi'
import { TelemetryLineChart } from '../charts/TelemetryLineChart'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import type { TelemetryReading } from '../types/telemetry'

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [deviceTelemetry, setDeviceTelemetry] = useState<TelemetryReading[]>([])
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [latestReading, setLatestReading] = useState<TelemetryReading | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let isMounted = true

    async function loadTelemetry() {
      setIsLoading(true)
      setError('')

      try {
        const readings = await fetchAllTelemetry()
        if (!isMounted) {
          return
        }

        setTelemetry(readings)

        const firstDeviceId = readings[0]?.deviceId ?? ''
        setSelectedDeviceId((previous) => previous || firstDeviceId)
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

    return () => {
      isMounted = false
    }
  }, [])

  useEffect(() => {
    let isMounted = true

    async function loadDeviceTelemetry() {
      if (!selectedDeviceId) {
        setDeviceTelemetry([])
        setLatestReading(null)
        return
      }

      try {
        const readings = await fetchTelemetryByDeviceId(selectedDeviceId)
        if (!isMounted) {
          return
        }

        setDeviceTelemetry(readings)
        setLatestReading(readings[0] ?? null)
      } catch {
        if (!isMounted) {
          return
        }

        setError('Failed to load latest device reading')
      }
    }

    void loadDeviceTelemetry()

    const interval = window.setInterval(() => {
      void loadDeviceTelemetry()
    }, 5000)

    return () => {
      isMounted = false
      window.clearInterval(interval)
    }
  }, [selectedDeviceId])

  const devices = useMemo(() => {
    return [...new Set(telemetry.map((reading) => reading.deviceId))]
  }, [telemetry])

  return (
    <main className="dashboard">
      <header className="dashboard-header">
        <h1 className="dashboard-title">AgroNode Dashboard</h1>
        {devices.length > 0 && (
          <DeviceSelector
            devices={devices}
            selectedDeviceId={selectedDeviceId}
            onChange={setSelectedDeviceId}
          />
        )}
      </header>

      {isLoading && <p className="dashboard-message">Loading telemetry...</p>}
      {error && <p className="dashboard-message">{error}</p>}
      {!isLoading && !error && devices.length === 0 && (
        <p className="dashboard-message">No telemetry data available</p>
      )}

      <section className="sensor-grid">
        <SensorCard
          label="Temperature"
          value={latestReading?.temperature ?? null}
          unit="°C"
        />
        <SensorCard
          label="Humidity"
          value={latestReading?.humidity ?? null}
          unit="%"
        />
      </section>

      <TelemetryLineChart data={deviceTelemetry} />
    </main>
  )
}
