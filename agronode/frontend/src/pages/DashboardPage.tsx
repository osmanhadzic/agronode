import { useEffect, useMemo, useState } from 'react'

import { fetchAllTelemetry } from '../api/telemetryApi'
import { TelemetryLineChart } from '../charts/TelemetryLineChart'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import type { TelemetryReading } from '../types/telemetry'

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
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

        const availableDeviceIds = [...new Set(readings.map((reading) => reading.deviceId))]
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

    const interval = window.setInterval(() => {
      void loadTelemetry()
    }, 5000)

    return () => {
      isMounted = false
      window.clearInterval(interval)
    }
  }, [])

  const devices = useMemo(() => {
    return [...new Set(telemetry.map((reading) => reading.deviceId))]
  }, [telemetry])

  const deviceTelemetry = useMemo(() => {
    if (!selectedDeviceId) {
      return []
    }

    return telemetry.filter((reading) => reading.deviceId === selectedDeviceId)
  }, [selectedDeviceId, telemetry])

  const latestReading = deviceTelemetry[0] ?? null

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
