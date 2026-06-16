import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react'

import { fetchAllTelemetry, fetchTelemetryByDeviceId, fetchLatestTelemetryByDeviceId, type DateFilterPeriod } from '../api/telemetryApi'
import { createTelemetrySocket } from '../api/telemetrySocket'
import { DataModeSelector } from '../components/DataModeSelector'
import { DateFilter } from '../components/DateFilter'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import type { TelemetryReading } from '../types/telemetry'

// Lazy load the chart component for better initial load performance
const TelemetryLineChart = lazy(() =>
  import('../charts/TelemetryLineChart').then((module) => ({
    default: module.TelemetryLineChart,
  }))
)

// Downsampling function to reduce memory usage for large datasets
function downsampleData(data: TelemetryReading[], period: DateFilterPeriod): TelemetryReading[] {
  if (data.length <= 100) {
    return data
  }

  // Calculate target sample size based on period
  let targetSize: number
  switch (period) {
    case 'hour':
      targetSize = 60 // One per minute
      break
    case 'day':
      targetSize = 96 // One per 15 minutes
      break
    case 'week':
      targetSize = 168 // One per hour
      break
    case 'month':
      targetSize = 120 // ~4 per day
      break
    case 'year':
      targetSize = 365 // One per day
      break
    case 'custom':
      targetSize = 200
      break
    default:
      targetSize = 100
  }

  // If data is already smaller than target, return as-is
  if (data.length <= targetSize) {
    return data
  }

  // Calculate sampling interval
  const interval = Math.floor(data.length / targetSize)

  // Sample data at regular intervals
  const sampled: TelemetryReading[] = []
  for (let i = 0; i < data.length; i += interval) {
    sampled.push(data[i])
  }

  // Always include the last data point
  if (sampled[sampled.length - 1] !== data[data.length - 1]) {
    sampled.push(data[data.length - 1])
  }

  return sampled
}

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [liveData, setLiveData] = useState<TelemetryReading[]>([])
  const [latestDeviceReading, setLatestDeviceReading] = useState<TelemetryReading | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [dataMode, setDataMode] = useState<'live' | 'history'>('live')
  const [dateFilterPeriod, setDateFilterPeriod] = useState<DateFilterPeriod>('hour')
  const [customStartDate, setCustomStartDate] = useState<string>()
  const [customEndDate, setCustomEndDate] = useState<string>()

  // Throttle real-time updates to avoid overwhelming the UI
  const [updateQueue, setUpdateQueue] = useState<TelemetryReading[]>([])
  
  useEffect(() => {
    if (updateQueue.length === 0) return
    
    const timeoutId = setTimeout(() => {
      setLiveData((previous) => {
        const newReadings = updateQueue.filter((reading) => {
          return !previous.some(
            (item) =>
              item.deviceId === reading.deviceId &&
              item.createdAt === reading.createdAt
          )
        })
        
        if (newReadings.length === 0) return previous
        
        // Keep only last hour of data for live mode
        const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000).toISOString()
        const filtered = [...newReadings, ...previous].filter(
          (item) => item.createdAt >= oneHourAgo
        )
        return filtered
      })
      
      setUpdateQueue([])
    }, 500) // Batch updates every 500ms
    
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
          // For live mode, fetch last hour of data
          if (selectedDeviceId) {
            readings = await fetchTelemetryByDeviceId(selectedDeviceId, {
              period: 'hour',
            })
          } else {
            readings = await fetchAllTelemetry()
          }
        } else {
          // For history mode, use date filters
          if (selectedDeviceId && dateFilterPeriod) {
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
        }

        if (!isMounted) {
          return
        }

        if (dataMode === 'live') {
          setLiveData(readings)
        } else {
          setTelemetry(readings)
        }

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

        // Queue update for throttled batch processing
        setUpdateQueue((prev) => [...prev, reading])
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
  }, [selectedDeviceId, dataMode, dateFilterPeriod, customStartDate, customEndDate])

  const devices = useMemo(() => {
    const allReadings = dataMode === 'live' ? liveData : telemetry
    return [...new Set(allReadings.map((reading) => reading.deviceId))]
  }, [telemetry, liveData, dataMode])

  const deviceTelemetry = useMemo(() => {
    if (!selectedDeviceId) {
      return []
    }

    const data = dataMode === 'live' ? liveData : telemetry
    const filtered = data.filter((reading) => reading.deviceId === selectedDeviceId)
    
    // Apply downsampling for history mode with large datasets
    if (dataMode === 'history' && filtered.length > 100) {
      return downsampleData(filtered, dateFilterPeriod)
    }
    
    return filtered
  }, [selectedDeviceId, telemetry, liveData, dataMode, dateFilterPeriod])

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

  const handleToggleSensor = useCallback((sensorKey: string) => {
    setSelectedSensors((previous) => {
      const currentSelection =
        previous.length === 0 ? [...availableSensors] : [...previous]

      if (currentSelection.includes(sensorKey)) {
        const remaining = currentSelection.filter((key) => key !== sensorKey)
        return remaining.length > 0 ? remaining : previous
      }

      return [...currentSelection, sensorKey]
    })
  }, [availableSensors])

  const handleDateFilterChange = useCallback((period: DateFilterPeriod, startDate?: string, endDate?: string) => {
    setDateFilterPeriod(period)
    setCustomStartDate(startDate)
    setCustomEndDate(endDate)
  }, [])

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

      <DataModeSelector mode={dataMode} onChange={setDataMode} />

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

      {dataMode === 'history' && selectedDeviceId && (
        <DateFilter 
          onFilterChange={handleDateFilterChange}
          selectedPeriod={dateFilterPeriod}
        />
      )}

      {dataMode === 'live' && (
        <div style={{
          textAlign: 'center',
          padding: '0.75rem',
          backgroundColor: '#e7f3ff',
          borderRadius: '6px',
          margin: '1rem 0',
          color: '#0066cc',
          fontWeight: '500'
        }}>
          📡 Prikazano: Poslednji sat podataka u realnom vremenu
        </div>
      )}

      {dataMode === 'history' && deviceTelemetry.length > 0 && (() => {
        const originalData = telemetry.filter(
          (reading) => reading.deviceId === selectedDeviceId
        )
        const isDownsampled = originalData.length > 100 && deviceTelemetry.length < originalData.length
        
        return isDownsampled ? (
          <div style={{
            textAlign: 'center',
            padding: '0.75rem',
            backgroundColor: '#fff3cd',
            borderRadius: '6px',
            margin: '1rem 0',
            color: '#856404',
            fontWeight: '500'
          }}>
            📊 Prikazano: {deviceTelemetry.length} od {originalData.length} podataka (optimizovano za performanse)
          </div>
        ) : null
      })()}

      <Suspense fallback={
        <div style={{
          textAlign: 'center',
          padding: '2rem',
          color: '#6c757d'
        }}>
          Loading chart...
        </div>
      }>
        <TelemetryLineChart data={deviceTelemetry} selectedSensors={visibleSensors} />
      </Suspense>
    </main>
  )
}
