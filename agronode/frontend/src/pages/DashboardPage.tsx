import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'

import {
  deleteSensorTriggerByDeviceId,
  fetchAllTelemetry,
  fetchSensorTriggerByDeviceId,
  fetchTriggersByDeviceId,
  saveSensorTriggerByDeviceId,
} from '../api/telemetryApi'
import { createTelemetrySocket } from '../api/telemetrySocket'
import { TelemetryLineChart } from '../charts/TelemetryLineChart'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import type { TelemetryReading, TriggerListItem } from '../types/telemetry'

type TriggerEvent = {
  id: string
  deviceId: string
  sensor: string
  type: 'activation' | 'recovery'
  limitType: 'min' | 'max'
  value: number
  threshold: number
  timestamp: string
}

export function DashboardPage() {
  const [telemetry, setTelemetry] = useState<TelemetryReading[]>([])
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedTriggerSensor, setSelectedTriggerSensor] = useState('temperature')
  const [minThresholdInput, setMinThresholdInput] = useState('')
  const [maxThresholdInput, setMaxThresholdInput] = useState('')
  const [triggerMessage, setTriggerMessage] = useState('')
  const [triggerError, setTriggerError] = useState('')
  const [isSavingTrigger, setIsSavingTrigger] = useState(false)
  const [deviceTriggers, setDeviceTriggers] = useState<TriggerListItem[]>([])
  const [isLoadingTriggers, setIsLoadingTriggers] = useState(false)
  const [deletingSensor, setDeletingSensor] = useState('')
  const [activeTriggerEvent, setActiveTriggerEvent] = useState<TriggerEvent | null>(null)
  const [triggerEvents, setTriggerEvents] = useState<TriggerEvent[]>([])
  const [toastTriggerEvent, setToastTriggerEvent] = useState<TriggerEvent | null>(null)
  const triggerActivationState = useRef<Record<string, { min: boolean; max: boolean }>>({})
  const selectedDeviceRef = useRef('')
  const triggerMapRef = useRef<Record<string, TriggerListItem>>({})
  const toastTimerRef = useRef<number | null>(null)

  useEffect(() => {
    let isMounted = true

    const processTriggerTransitions = (reading: TelemetryReading) => {
      if (reading.deviceId !== selectedDeviceRef.current) {
        return
      }

      const sensorValues: Record<string, number> = {
        temperature: reading.temperature,
        humidity: reading.humidity,
        ...(reading.sensors ?? {}),
      }

      const nextEvents: TriggerEvent[] = []

      for (const [sensor, trigger] of Object.entries(triggerMapRef.current)) {
        const value = sensorValues[sensor]
        if (value === undefined) {
          continue
        }

        const previousState = triggerActivationState.current[sensor] ?? { min: false, max: false }
        const nextState = { ...previousState }

        if (trigger.min !== undefined) {
          const isActive = value <= trigger.min
          if (isActive && !previousState.min) {
            nextEvents.push({
              id: `${reading.deviceId}-${sensor}-min-activation-${reading.createdAt}`,
              deviceId: reading.deviceId,
              sensor,
              type: 'activation',
              limitType: 'min',
              value,
              threshold: trigger.min,
              timestamp: reading.createdAt,
            })
          }

          if (!isActive && previousState.min) {
            nextEvents.push({
              id: `${reading.deviceId}-${sensor}-min-recovery-${reading.createdAt}`,
              deviceId: reading.deviceId,
              sensor,
              type: 'recovery',
              limitType: 'min',
              value,
              threshold: trigger.min,
              timestamp: reading.createdAt,
            })
          }

          nextState.min = isActive
        }

        if (trigger.max !== undefined) {
          const isActive = value >= trigger.max
          if (isActive && !previousState.max) {
            nextEvents.push({
              id: `${reading.deviceId}-${sensor}-max-activation-${reading.createdAt}`,
              deviceId: reading.deviceId,
              sensor,
              type: 'activation',
              limitType: 'max',
              value,
              threshold: trigger.max,
              timestamp: reading.createdAt,
            })
          }

          if (!isActive && previousState.max) {
            nextEvents.push({
              id: `${reading.deviceId}-${sensor}-max-recovery-${reading.createdAt}`,
              deviceId: reading.deviceId,
              sensor,
              type: 'recovery',
              limitType: 'max',
              value,
              threshold: trigger.max,
              timestamp: reading.createdAt,
            })
          }

          nextState.max = isActive
        }

        triggerActivationState.current[sensor] = nextState
      }

      if (nextEvents.length === 0) {
        return
      }

      setTriggerEvents((previous) => [...nextEvents, ...previous].slice(0, 12))

      const newestActivation = nextEvents.find((event) => event.type === 'activation')
      if (newestActivation) {
        setActiveTriggerEvent(newestActivation)
        setToastTriggerEvent(newestActivation)
        return
      }

      if (nextEvents.some((event) => event.type === 'recovery')) {
        setActiveTriggerEvent(null)
      }
    }

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

        processTriggerTransitions(reading)
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
  }, [])

  useEffect(() => {
    if (!toastTriggerEvent) {
      return
    }

    if (toastTimerRef.current !== null) {
      window.clearTimeout(toastTimerRef.current)
    }

    toastTimerRef.current = window.setTimeout(() => {
      setToastTriggerEvent(null)
      toastTimerRef.current = null
    }, 5000)

    return () => {
      if (toastTimerRef.current !== null) {
        window.clearTimeout(toastTimerRef.current)
      }
    }
  }, [toastTriggerEvent])

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

  const activeTriggerSensor = useMemo(() => {
    if (availableSensors.length === 0) {
      return 'temperature'
    }

    if (availableSensors.includes(selectedTriggerSensor)) {
      return selectedTriggerSensor
    }

    return availableSensors[0]
  }, [availableSensors, selectedTriggerSensor])

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

  const triggerMapBySensor = useMemo(() => {
    const map: Record<string, TriggerListItem> = {}
    for (const trigger of deviceTriggers) {
      map[trigger.sensor] = trigger
    }

    return map
  }, [deviceTriggers])

  useEffect(() => {
    selectedDeviceRef.current = selectedDeviceId
  }, [selectedDeviceId])

  useEffect(() => {
    triggerMapRef.current = triggerMapBySensor
  }, [triggerMapBySensor])

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

  useEffect(() => {
    let isMounted = true

    async function loadTrigger() {
      if (!selectedDeviceId || !activeTriggerSensor) {
        setMinThresholdInput('')
        setMaxThresholdInput('')
        setTriggerError('')
        setTriggerMessage('')
        return
      }

      setTriggerError('')
      setTriggerMessage('')

      try {
        const trigger = await fetchSensorTriggerByDeviceId(
          selectedDeviceId,
          activeTriggerSensor,
        )
        if (!isMounted) {
          return
        }

        setMinThresholdInput(trigger?.min !== undefined ? String(trigger.min) : '')
        setMaxThresholdInput(trigger?.max !== undefined ? String(trigger.max) : '')
      } catch {
        if (!isMounted) {
          return
        }

        setTriggerError('Failed to load trigger values')
      }
    }

    void loadTrigger()

    return () => {
      isMounted = false
    }
  }, [selectedDeviceId, activeTriggerSensor])

  useEffect(() => {
    let isMounted = true

    async function loadDeviceTriggers() {
      if (!selectedDeviceId) {
        setDeviceTriggers([])
        return
      }

      setIsLoadingTriggers(true)

      try {
        const response = await fetchTriggersByDeviceId(selectedDeviceId)
        if (!isMounted) {
          return
        }

        const sortedTriggers = [...response.triggers].sort((left, right) =>
          left.sensor.localeCompare(right.sensor),
        )

        setDeviceTriggers(sortedTriggers)
      } catch {
        if (!isMounted) {
          return
        }

        setTriggerError('Failed to load trigger list')
      } finally {
        if (isMounted) {
          setIsLoadingTriggers(false)
        }
      }
    }

    void loadDeviceTriggers()

    return () => {
      isMounted = false
    }
  }, [selectedDeviceId])

  const refreshDeviceTriggers = async (deviceId: string) => {
    const response = await fetchTriggersByDeviceId(deviceId)
    const sortedTriggers = [...response.triggers].sort((left, right) =>
      left.sensor.localeCompare(right.sensor),
    )
    setDeviceTriggers(sortedTriggers)
  }

  const handleSaveTrigger = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!selectedDeviceId) {
      setTriggerError('Select a device first')
      setTriggerMessage('')
      return
    }

    if (!activeTriggerSensor) {
      setTriggerError('Select a sensor first')
      setTriggerMessage('')
      return
    }

    const min = minThresholdInput.trim() === '' ? undefined : Number(minThresholdInput)
    const max = maxThresholdInput.trim() === '' ? undefined : Number(maxThresholdInput)

    if (min === undefined && max === undefined) {
      setTriggerError('Set at least one threshold value')
      setTriggerMessage('')
      return
    }

    if (
      (min !== undefined && Number.isNaN(min)) ||
      (max !== undefined && Number.isNaN(max))
    ) {
      setTriggerError('Thresholds must be valid numbers')
      setTriggerMessage('')
      return
    }

    setIsSavingTrigger(true)
    setTriggerError('')
    setTriggerMessage('')

    try {
      const savedTrigger = await saveSensorTriggerByDeviceId(
        selectedDeviceId,
        activeTriggerSensor,
        {
          min,
          max,
        },
      )

      setMinThresholdInput(savedTrigger.min !== undefined ? String(savedTrigger.min) : '')
      setMaxThresholdInput(savedTrigger.max !== undefined ? String(savedTrigger.max) : '')
      setTriggerMessage('Trigger saved successfully')
      await refreshDeviceTriggers(selectedDeviceId)
    } catch {
      setTriggerError('Failed to save trigger values')
    } finally {
      setIsSavingTrigger(false)
    }
  }

  const handleEditTrigger = (trigger: TriggerListItem) => {
    setSelectedTriggerSensor(trigger.sensor)
    setMinThresholdInput(trigger.min !== undefined ? String(trigger.min) : '')
    setMaxThresholdInput(trigger.max !== undefined ? String(trigger.max) : '')
    setTriggerError('')
    setTriggerMessage('Trigger loaded into form')
  }

  const handleDeleteTrigger = async (sensor: string) => {
    if (!selectedDeviceId) {
      return
    }

    setDeletingSensor(sensor)
    setTriggerError('')
    setTriggerMessage('')

    try {
      await deleteSensorTriggerByDeviceId(selectedDeviceId, sensor)
      await refreshDeviceTriggers(selectedDeviceId)

      if (activeTriggerSensor === sensor) {
        setMinThresholdInput('')
        setMaxThresholdInput('')
      }

      setTriggerMessage('Trigger deleted successfully')
    } catch {
      setTriggerError('Failed to delete trigger')
    } finally {
      setDeletingSensor('')
    }
  }

  const handleSelectDevice = (deviceId: string) => {
    setSelectedDeviceId(deviceId)
    triggerActivationState.current = {}
    setActiveTriggerEvent(null)
    setTriggerEvents([])
    setToastTriggerEvent(null)
  }

  const formatEventTime = (timestamp: string) => {
    const parsedDate = new Date(timestamp)
    if (Number.isNaN(parsedDate.getTime())) {
      return timestamp
    }

    return parsedDate.toLocaleTimeString()
  }

  const formatEventDescription = (event: TriggerEvent) => {
    const sensorLabel = formatSensorLabel(event.sensor)
    const unit = getSensorUnit(event.sensor)
    const limitText = event.limitType === 'max' ? 'Iznad max' : 'Ispod min'

    if (event.type === 'activation') {
      return `ALERT: ${sensorLabel} ${limitText} (vrednost ${event.value}${unit}, prag ${event.threshold}${unit})`
    }

    return `INFO: ${sensorLabel} normalizovan (vrednost ${event.value}${unit}, prag ${event.threshold}${unit})`
  }

  return (
    <main className="dashboard">
      {toastTriggerEvent && (
        <div className="trigger-toast" role="status" aria-live="polite">
          <p className="trigger-event-text">{formatEventDescription(toastTriggerEvent)}</p>
          <p className="trigger-event-meta">
            Uređaj: {toastTriggerEvent.deviceId} · Vreme: {formatEventTime(toastTriggerEvent.timestamp)} · Aktivacija poslata uređaju
          </p>
        </div>
      )}

      <header className="dashboard-header">
        <h1 className="dashboard-title">AgroNode Dashboard</h1>
        {devices.length > 0 && (
          <DeviceSelector
            devices={devices}
            selectedDeviceId={selectedDeviceId}
            onChange={handleSelectDevice}
          />
        )}
      </header>

      <DeviceMetaPanel meta={latestReading?.meta} />

      {isLoading && <p className="dashboard-message">Loading telemetry...</p>}
      {error && <p className="dashboard-message">{error}</p>}
      {!isLoading && !error && devices.length === 0 && (
        <p className="dashboard-message">No telemetry data available</p>
      )}

      <section className="trigger-panel">
        <h2 className="trigger-title">Sensor Trigger</h2>
        <form className="trigger-form" onSubmit={handleSaveTrigger}>
          <label className="trigger-field">
            <span>Sensor</span>
            <select
              value={activeTriggerSensor}
              onChange={(event) => setSelectedTriggerSensor(event.target.value)}
              disabled={availableSensors.length === 0}
            >
              {availableSensors.length === 0 && (
                <option value="temperature">Temperature</option>
              )}
              {availableSensors.map((sensorKey) => (
                <option key={sensorKey} value={sensorKey}>
                  {formatSensorLabel(sensorKey)}
                </option>
              ))}
            </select>
          </label>
          <label className="trigger-field">
            <span>Min</span>
            <input
              type="number"
              step="0.1"
              value={minThresholdInput}
              onChange={(event) => setMinThresholdInput(event.target.value)}
            />
          </label>
          <label className="trigger-field">
            <span>Max</span>
            <input
              type="number"
              step="0.1"
              value={maxThresholdInput}
              onChange={(event) => setMaxThresholdInput(event.target.value)}
            />
          </label>
          <button type="submit" disabled={isSavingTrigger || !selectedDeviceId}>
            {isSavingTrigger ? 'Saving...' : 'Save Trigger'}
          </button>
        </form>
        <div className="trigger-list">
          <h3 className="trigger-list-title">Configured Triggers</h3>
          {isLoadingTriggers && <p className="dashboard-message">Loading triggers...</p>}
          {!isLoadingTriggers && deviceTriggers.length === 0 && (
            <p className="dashboard-message">No triggers configured for this device</p>
          )}
          {!isLoadingTriggers && deviceTriggers.length > 0 && (
            <table className="trigger-table">
              <thead>
                <tr>
                  <th>Sensor</th>
                  <th>Min</th>
                  <th>Max</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {deviceTriggers.map((trigger) => (
                  <tr key={trigger.sensor}>
                    <td>{formatSensorLabel(trigger.sensor)}</td>
                    <td>{trigger.min !== undefined ? trigger.min : '-'}</td>
                    <td>{trigger.max !== undefined ? trigger.max : '-'}</td>
                    <td className="trigger-actions-cell">
                      <button
                        type="button"
                        onClick={() => handleEditTrigger(trigger)}
                        disabled={deletingSensor === trigger.sensor}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDeleteTrigger(trigger.sensor)}
                        disabled={deletingSensor === trigger.sensor}
                      >
                        {deletingSensor === trigger.sensor ? 'Deleting...' : 'Delete'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        {triggerMessage && <p className="dashboard-message">{triggerMessage}</p>}
        {triggerError && <p className="dashboard-message">{triggerError}</p>}
      </section>

      <section className="trigger-events-panel">
        <h2 className="trigger-title">Trigger Događaji</h2>

        {activeTriggerEvent ? (
          <div className="trigger-event-active">
            <p className="trigger-event-text">{formatEventDescription(activeTriggerEvent)}</p>
            <p className="trigger-event-meta">
              Uređaj: {activeTriggerEvent.deviceId} · Vreme: {formatEventTime(activeTriggerEvent.timestamp)} · Status: Aktivacija poslata uređaju
            </p>
          </div>
        ) : (
          <p className="dashboard-message">Nema aktivnih trigger alarma</p>
        )}

        <div className="trigger-event-history">
          <h3 className="trigger-list-title">Poslednji događaji</h3>
          {triggerEvents.length === 0 ? (
            <p className="dashboard-message">Još nema trigger događaja</p>
          ) : (
            <ul className="trigger-event-list">
              {triggerEvents.map((event) => (
                <li key={event.id} className="trigger-event-item">
                  <p className="trigger-event-text">{formatEventDescription(event)}</p>
                  <p className="trigger-event-meta">
                    Uređaj: {event.deviceId} · Vreme: {formatEventTime(event.timestamp)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

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
