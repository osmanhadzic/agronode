import { useEffect, useMemo, useState } from 'react'

import { fetchAllTelemetry, fetchTelemetryByDeviceId, fetchLatestTelemetryByDeviceId, type DateFilterPeriod } from '../api/telemetryApi'
import { createTelemetrySocket } from '../api/telemetrySocket'
import { TelemetryLineChart } from '../charts/TelemetryLineChart'
import { DateFilter } from '../components/DateFilter'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import type { TelemetryReading } from '../types/telemetry'

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [latestDeviceReading, setLatestDeviceReading] = useState<TelemetryReading | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [dateFilterPeriod, setDateFilterPeriod] = useState<DateFilterPeriod>('')
  const [customStartDate, setCustomStartDate] = useState<string>()
  const [customEndDate, setCustomEndDate] = useState<string>()

  useEffect(() => {
    let isMounted = true

    async function loadTelemetry() {
      setIsLoading(true)
      setError('')

      try {
        let readings: TelemetryReading[]

        // If device is selected and date filter is active, fetch filtered data
        if (selectedDeviceId && dateFilterPeriod) {
          readings = await fetchTelemetryByDeviceId(selectedDeviceId, {
            period: dateFilterPeriod,
            startDate: customStartDate,
            endDate: customEndDate,
          })
        } else if (selectedDeviceId) {
          // Fetch all data for selected device
          readings = await fetchTelemetryByDeviceId(selectedDeviceId)
        } else {
          // Fetch all data
          readings = await fetchAllTelemetry()
        }

        if (!isMounted) {
          return
        }

        setTelemetry(readings)

        // Set initial device selection if none selected
        if (!selectedDeviceId) {
          const availableDeviceIds = [...new Set(readings.map((reading) => reading.deviceId))]
          const firstDeviceId = availableDeviceIds[0] ?? ''
          setSelectedDeviceId(firstDeviceId)
        }

        // Fetch latest reading for device meta information
        if (selectedDeviceId) {
          const latest = await fetchLatestTelemetryByDeviceId(selectedDeviceId)
          if (isMounted) {
            setLatestDeviceReading(latest)
          }
        }

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

        // Update latest reading if it's for the selected device
        if (reading.deviceId === selectedDeviceId) {
          setLatestDeviceReading(reading)
        }

        // Only add real-time updates if no date filter is applied
        if (!dateFilterPeriod) {
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
        }
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
    }
  }, [selectedDeviceId, dateFilterPeriod, customStartDate, customEndDate])

  const devices = useMemo(() => {
    return [...new Set(telemetry.map((reading) => reading.deviceId))]
  }, [telemetry])

  const deviceTelemetry = useMemo(() => {
    if (!selectedDeviceId) {
      return []
    }

    return telemetry.filter((reading) => reading.deviceId === selectedDeviceId)
  }, [selectedDeviceId, telemetry])

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
    // Use latestDeviceReading for current values, fallback to latest from filtered data
    const reading = latestDeviceReading ?? deviceTelemetry[0] ?? null
    
    if (!reading) {
      return null
    }

    const normalized: Record<string, number> = {
      temperature: reading.temperature,
      humidity: reading.humidity,
      ...(reading.sensors ?? {}),
    }

    return normalized
  }, [latestDeviceReading, deviceTelemetry])

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

  const handleDateFilterChange = (period: DateFilterPeriod, startDate?: string, endDate?: string) => {
    setDateFilterPeriod(period)
    setCustomStartDate(startDate)
    setCustomEndDate(endDate)
  }

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

      <DeviceMetaPanel meta={latestDeviceReading?.meta} />

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

      {selectedDeviceId && (
        <DateFilter 
          onFilterChange={handleDateFilterChange}
          selectedPeriod={dateFilterPeriod}
        />
      )}

      <TelemetryLineChart data={deviceTelemetry} selectedSensors={visibleSensors} />
    </main>
  )
}
