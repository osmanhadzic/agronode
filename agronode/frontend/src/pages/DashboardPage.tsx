import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react'

import {
  deleteSensorTriggerByDeviceId,
  fetchAllTelemetry,
  fetchDevices,
  fetchTelemetryByDeviceIdAndSensorId,
  fetchSensorsByDeviceId,
  fetchSensorTriggerByDeviceId,
  fetchTelemetryByDeviceId,
  fetchTriggersByDeviceId,
  fetchLatestTelemetryByDeviceId,
  saveSensorTriggerByDeviceId,
  sendDeviceStreamControl,
  type DateFilterPeriod,
} from '../api/telemetryApi'
import { createDeviceStatusSocket, createTelemetrySocket } from '../api/telemetrySocket'
import { DateFilter } from '../components/DateFilter'
import { DataModeSelector } from '../components/DataModeSelector'
import { DeviceMetaPanel } from '../components/DeviceMetaPanel'
import { DeviceSelector } from '../components/DeviceSelector'
import { SensorCard } from '../components/SensorCard'
import { SensorVisibilitySelector } from '../components/SensorVisibilitySelector'
import { clearSession, loadSession } from '../api/session'
import type { TelemetryReading, TriggerListItem, DeviceSensor } from '../types/telemetry'

type TriggerEvent = {
  id: string
  deviceId: string
  targetDeviceId: string
  sensor: string
  type: 'activation' | 'recovery'
  limitType: 'min' | 'max'
  value: number
  threshold: number
  timestamp: string
}

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
  const [liveSensorSnapshots, setLiveSensorSnapshots] = useState<Record<string, Record<string, number>>>({})
  const [deviceStatuses, setDeviceStatuses] = useState<Record<string, string>>({})
  const [deviceTypes, setDeviceTypes] = useState<Record<string, string>>({})
  const [latestDeviceReading, setLatestDeviceReading] =
    useState<TelemetryReading | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [selectedSensors, setSelectedSensors] = useState<string[]>([])
  const [deviceSensors, setDeviceSensors] = useState<DeviceSensor[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [dataMode, setDataMode] = useState<'live' | 'history'>('live')
  const [dateFilterPeriod, setDateFilterPeriod] =
    useState<DateFilterPeriod>('hour')
  const [customStartDate, setCustomStartDate] = useState<string>()
  const [customEndDate, setCustomEndDate] = useState<string>()
  const [selectedTelemetrySensor, setSelectedTelemetrySensor] = useState('all')
  const [updateQueue, setUpdateQueue] = useState<TelemetryReading[]>([])
  const [streamingByDevice, setStreamingByDevice] = useState<Record<string, boolean>>({})
  const [streamingBySensor, setStreamingBySensor] = useState<Record<string, boolean>>({})
  const [isSendingStreamCommand, setIsSendingStreamCommand] = useState(false)
  const [sendingSensorId, setSendingSensorId] = useState('')
  const [streamControlError, setStreamControlError] = useState('')

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

  const [selectedTriggerSensor, setSelectedTriggerSensor] = useState('temperature')
  const [targetTriggerDeviceId, setTargetTriggerDeviceId] = useState('')
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
  const [activeTab, setActiveTab] = useState<'overview' | 'telemetry' | 'sensors' | 'triggers'>('overview')
  const triggerActivationState = useRef<Record<string, { min: boolean; max: boolean }>>({})
  const selectedDeviceRef = useRef('')
  const deviceSensorIdsRef = useRef<Set<string>>(new Set())
  const triggerMapRef = useRef<Record<string, TriggerListItem>>({})
  const toastTimerRef = useRef<number | null>(null)
  const session = loadSession()
  const dashboardTabs = [
    { id: 'overview', label: 'Pregled' },
    { id: 'telemetry', label: 'Telemetrija' },
    { id: 'sensors', label: 'Senzori' },
    { id: 'triggers', label: 'Triggeri' },
  ] as const

  const getReadingSensorValues = useCallback((reading: TelemetryReading): Record<string, number> => {
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
  }, [])

  const mergeTelemetryReadings = useCallback((readings: TelemetryReading[]) => {
    const deduplicatedByKey = new Map<string, TelemetryReading>()

    for (const reading of readings) {
      const key = `${reading.deviceId}:${reading.sensorId ?? 'na'}:${reading.createdAt}`
      deduplicatedByKey.set(key, reading)
    }

    return [...deduplicatedByKey.values()].sort(
      (left, right) =>
        new Date(right.createdAt).getTime() -
        new Date(left.createdAt).getTime(),
    )
  }, [])

  const mergeLiveSensorSnapshot = useCallback((readings: TelemetryReading[]) => {
    if (readings.length === 0) {
      return
    }

    setLiveSensorSnapshots((previous) => {
      const next = { ...previous }

      for (const reading of readings) {
        const sensorValues = getReadingSensorValues(reading)
        if (Object.keys(sensorValues).length === 0) {
          continue
        }

        const deviceSnapshot = { ...(next[reading.deviceId] ?? {}) }
        for (const [sensorKey, value] of Object.entries(sensorValues)) {
          deviceSnapshot[sensorKey] = value
        }

        next[reading.deviceId] = deviceSnapshot
      }

      return next
    })
  }, [getReadingSensorValues])

  useEffect(() => {
    let isMounted = true

    const processTriggerTransitions = (reading: TelemetryReading) => {
      if (reading.deviceId !== selectedDeviceRef.current) {
        return
      }

      const sensorValues = getReadingSensorValues(reading)

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
              targetDeviceId: trigger.targetDeviceId ?? reading.deviceId,
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
              targetDeviceId: trigger.targetDeviceId ?? reading.deviceId,
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
              targetDeviceId: trigger.targetDeviceId ?? reading.deviceId,
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
              targetDeviceId: trigger.targetDeviceId ?? reading.deviceId,
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
        const devices = await fetchDevices()

        let sensors: DeviceSensor[] = []
        let sensorIdList: string[] = []

        if (selectedDeviceId) {
          try {
            sensors = await fetchSensorsByDeviceId(selectedDeviceId)
            sensorIdList = [
              ...new Set(
                sensors
                  .map((sensor) => sensor.sensorId.trim())
                  .filter((sensorId) => sensorId.length > 0),
              ),
            ]
          } catch {
            sensors = []
            sensorIdList = []
          }
        }

        const fetchBySensorList = async () => {
          if (!selectedDeviceId || sensorIdList.length === 0) {
            return [] as TelemetryReading[]
          }

          const dataBySensor = await Promise.all(
            sensorIdList.map((sensorId) =>
              fetchTelemetryByDeviceIdAndSensorId(selectedDeviceId, sensorId),
            ),
          )

          return mergeTelemetryReadings(dataBySensor.flat())
        }

        let readings: TelemetryReading[]

        if (dataMode === 'live') {
          if (selectedDeviceId) {
            if (selectedTelemetrySensor !== 'all') {
              readings = await fetchTelemetryByDeviceIdAndSensorId(
                selectedDeviceId,
                selectedTelemetrySensor,
              )
            } else if (sensorIdList.length > 0) {
              readings = await fetchBySensorList()
            } else {
              readings = await fetchTelemetryByDeviceId(selectedDeviceId, {
                period: 'hour',
              })
            }
          } else {
            readings = await fetchAllTelemetry()
          }
        } else if (selectedDeviceId && dateFilterPeriod) {
          if (selectedTelemetrySensor !== 'all') {
            readings = await fetchTelemetryByDeviceIdAndSensorId(
              selectedDeviceId,
              selectedTelemetrySensor,
            )
          } else {
            const filtered = await fetchTelemetryByDeviceId(selectedDeviceId, {
              period: dateFilterPeriod,
              startDate: customStartDate,
              endDate: customEndDate,
            })

            if (sensorIdList.length > 0) {
              const sensorIdSet = new Set(sensorIdList)
              readings = filtered.filter((reading) =>
                reading.sensorId ? sensorIdSet.has(reading.sensorId) : false,
              )
            } else {
              readings = filtered
            }
          }
        } else if (selectedDeviceId) {
          if (selectedTelemetrySensor !== 'all') {
            readings = await fetchTelemetryByDeviceIdAndSensorId(
              selectedDeviceId,
              selectedTelemetrySensor,
            )
          } else if (sensorIdList.length > 0) {
            readings = await fetchBySensorList()
          } else {
            readings = await fetchTelemetryByDeviceId(selectedDeviceId)
          }
        } else {
          readings = await fetchAllTelemetry()
        }

        if (!isMounted) {
          return
        }

        if (dataMode === 'live') {
          setLiveData(readings)
          mergeLiveSensorSnapshot(readings)
        } else {
          setTelemetry(readings)
        }

        setDeviceStatuses(
          devices.reduce<Record<string, string>>((result, device) => {
            result[device.deviceId] = device.status
            return result
          }, {}),
        )

        setDeviceTypes(
          devices.reduce<Record<string, string>>((result, device) => {
            result[device.deviceId] = device.deviceType ?? 'unknown'
            return result
          }, {}),
        )

        setDeviceSensors(sensors)

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

        mergeLiveSensorSnapshot([reading])

        if (dataMode === 'live') {
          const selectedDevice = selectedDeviceRef.current
          const knownSensorIds = deviceSensorIdsRef.current
          const isSelectedDeviceReading =
            selectedDevice.length > 0 && reading.deviceId === selectedDevice
          const isKnownSensorReading =
            knownSensorIds.size === 0 ||
            (reading.sensorId ? knownSensorIds.has(reading.sensorId) : false)
          const isSelectedSensorReading =
            selectedTelemetrySensor === 'all' ||
            selectedTelemetrySensor === reading.sensorId

          if (isSelectedDeviceReading && !isKnownSensorReading) {
            return
          }

          if (isSelectedDeviceReading && !isSelectedSensorReading) {
            return
          }

          setUpdateQueue((previous) => [...previous, reading])
        }

        processTriggerTransitions(reading)
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

        setSelectedDeviceId((previous) => previous || event.deviceId)      },
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
    selectedTelemetrySensor,
    dataMode,
    dateFilterPeriod,
    customStartDate,
    customEndDate,
    mergeLiveSensorSnapshot,
    mergeTelemetryReadings,
  ])

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

  const selectedDeviceStatus = selectedDeviceId
    ? deviceStatuses[selectedDeviceId] ?? 'unknown'
    : 'unknown'

  const isSelectedDeviceStreaming = selectedDeviceId
    ? (streamingByDevice[selectedDeviceId] ?? true)
    : true

  const availableSensors = useMemo(() => {
    const sensorSet = new Set<string>()

    for (const sensor of deviceSensors) {
      sensorSet.add(sensor.sensorId)
    }

    for (const reading of deviceTelemetry) {
      if (reading.sensorId) {
        sensorSet.add(reading.sensorId)
      }

      for (const sensorKey of Object.keys(getReadingSensorValues(reading))) {
        sensorSet.add(sensorKey)
      }
    }

    if (sensorSet.size === 0) {
      sensorSet.add('temperature')
    }

    return [...sensorSet]
  }, [deviceTelemetry, deviceSensors, getReadingSensorValues])

  const activeTriggerSensor = useMemo(() => {
    if (availableSensors.length === 0) {
      return 'temperature'
    }

    if (availableSensors.includes(selectedTriggerSensor)) {
      return selectedTriggerSensor
    }

    return availableSensors[0]
  }, [availableSensors, selectedTriggerSensor])

  const sensorCatalog = useMemo(() => {
    const sensorSet = new Set<string>()

    for (const sensor of deviceSensors) {
      sensorSet.add(sensor.sensorId)
    }

    for (const sensorKey of availableSensors) {
      sensorSet.add(sensorKey)
    }

    return [...sensorSet].sort((left, right) => left.localeCompare(right))
  }, [deviceSensors, availableSensors])

  const latestSensors = useMemo(() => {
    const latestBySensor: Record<string, number> = {}

    if (dataMode === 'live' && selectedDeviceId) {
      const snapshot = liveSensorSnapshots[selectedDeviceId]
      if (snapshot) {
        for (const [sensorKey, value] of Object.entries(snapshot)) {
          latestBySensor[sensorKey] = value
        }
      }
    }

    const readings = latestDeviceReading
      ? [latestDeviceReading, ...deviceTelemetry]
      : deviceTelemetry

    for (const reading of readings) {
      const sensorValues = getReadingSensorValues(reading)

      for (const [sensorKey, value] of Object.entries(sensorValues)) {
        if (!(sensorKey in latestBySensor)) {
          latestBySensor[sensorKey] = value
        }
      }
    }

    return Object.keys(latestBySensor).length > 0 ? latestBySensor : null
  }, [dataMode, deviceTelemetry, latestDeviceReading, liveSensorSnapshots, selectedDeviceId, getReadingSensorValues])

  const triggerMapBySensor = useMemo(() => {
    const map: Record<string, TriggerListItem> = {}
    for (const trigger of deviceTriggers) {
      map[trigger.sensorId] = trigger
    }

    return map
  }, [deviceTriggers])

  useEffect(() => {
    selectedDeviceRef.current = selectedDeviceId
  }, [selectedDeviceId])

  useEffect(() => {
    deviceSensorIdsRef.current = new Set(
      deviceSensors
        .map((sensor) => sensor.sensorId.trim())
        .filter((sensorId) => sensorId.length > 0),
    )
  }, [deviceSensors])

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

  const getSensorUnit = (sensorKey: string) => {
    if (sensorKey === 'temperature' || sensorKey === 'dht11-temp') {
      return '°C'
    }

    if (sensorKey === 'humidity' || sensorKey === 'humidity_dht11' || sensorKey === 'dht11-humidity') {
      return '%'
    }

    if (sensorKey === 'signal_strength') {
      return 'dBm'
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

  useEffect(() => {
    let isMounted = true

    async function loadTrigger() {
      if (!selectedDeviceId || !activeTriggerSensor) {
        setMinThresholdInput('')
        setMaxThresholdInput('')
        setTargetTriggerDeviceId(selectedDeviceId)
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
        setTargetTriggerDeviceId(trigger?.targetDeviceId ?? selectedDeviceId)
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
          left.sensorId.localeCompare(right.sensorId),
        )

        setDeviceTriggers(sortedTriggers)
        setTargetTriggerDeviceId((previous) => previous || selectedDeviceId)
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
      left.sensorId.localeCompare(right.sensorId),
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
          targetDeviceId: targetTriggerDeviceId.trim() || selectedDeviceId,
        },
      )

      setMinThresholdInput(savedTrigger.min !== undefined ? String(savedTrigger.min) : '')
      setMaxThresholdInput(savedTrigger.max !== undefined ? String(savedTrigger.max) : '')
      setTargetTriggerDeviceId(savedTrigger.targetDeviceId ?? selectedDeviceId)
      setTriggerMessage('Trigger saved successfully')
      await refreshDeviceTriggers(selectedDeviceId)
    } catch {
      setTriggerError('Failed to save trigger values')
    } finally {
      setIsSavingTrigger(false)
    }
  }

  const handleEditTrigger = (trigger: TriggerListItem) => {
    setSelectedTriggerSensor(trigger.sensorId)
    setMinThresholdInput(trigger.min !== undefined ? String(trigger.min) : '')
    setMaxThresholdInput(trigger.max !== undefined ? String(trigger.max) : '')
    setTargetTriggerDeviceId(trigger.targetDeviceId ?? selectedDeviceId)
    setTriggerError('')
    setTriggerMessage('Trigger loaded into form')
  }

  const handleDeleteTrigger = async (sensorId: string) => {
    if (!selectedDeviceId) {
      return
    }

    setDeletingSensor(sensorId)
    setTriggerError('')
    setTriggerMessage('')

    try {
      await deleteSensorTriggerByDeviceId(selectedDeviceId, sensorId)
      await refreshDeviceTriggers(selectedDeviceId)

      if (activeTriggerSensor === sensorId) {
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
    setStreamControlError('')
    setSelectedTelemetrySensor('all')
    setActiveTab('sensors')
    triggerActivationState.current = {}
    setActiveTriggerEvent(null)
    setTriggerEvents([])
    setToastTriggerEvent(null)
  }

  const handleToggleDeviceStream = async () => {
    if (!selectedDeviceId) {
      return
    }

    const action = isSelectedDeviceStreaming ? 'pause' : 'resume'
    setIsSendingStreamCommand(true)
    setStreamControlError('')

    try {
      const response = await sendDeviceStreamControl(selectedDeviceId, action, 'all')
      setStreamingByDevice((previous) => ({
        ...previous,
        [selectedDeviceId]: response.streaming,
      }))
      setStreamingBySensor((previous) => {
        const next = { ...previous }
        for (const sensorId of sensorCatalog) {
          next[`${selectedDeviceId}:${sensorId}`] = response.streaming
        }

        return next
      })
    } catch {
      setStreamControlError('Failed to send stream control command')
    } finally {
      setIsSendingStreamCommand(false)
    }
  }

  const isSensorStreaming = (sensorId: string): boolean => {
    if (!selectedDeviceId) {
      return true
    }

    const key = `${selectedDeviceId}:${sensorId}`
    if (key in streamingBySensor) {
      return streamingBySensor[key]
    }

    return isSelectedDeviceStreaming
  }

  const handleToggleSensorStream = async (sensorId: string) => {
    if (!selectedDeviceId) {
      return
    }

    const currentlyStreaming = isSensorStreaming(sensorId)
    const action = currentlyStreaming ? 'pause' : 'resume'
    setSendingSensorId(sensorId)
    setStreamControlError('')

    try {
      const response = await sendDeviceStreamControl(selectedDeviceId, action, sensorId)
      const key = `${selectedDeviceId}:${response.sensorId}`
      setStreamingBySensor((previous) => ({
        ...previous,
        [key]: response.streaming,
      }))
    } catch {
      setStreamControlError(`Failed to send stream control for sensor ${sensorId}`)
    } finally {
      setSendingSensorId('')
    }
  }

  const handleOpenSensorTelemetry = (sensorId: string) => {
    setSelectedTelemetrySensor(sensorId)
    setActiveTab('telemetry')
  }

  const handleConfigureSensorTrigger = (sensorId: string) => {
    setSelectedTriggerSensor(sensorId)
    setActiveTab('triggers')
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
            Izvor: {toastTriggerEvent.deviceId} · Target: {toastTriggerEvent.targetDeviceId} · Vreme: {formatEventTime(toastTriggerEvent.timestamp)}
          </p>
        </div>
      )}

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

          {session && (
            <p className="dashboard-session">
              Signed in as <strong>{session.email}</strong> · Org {session.organizationId}
            </p>
          )}
        </div>

        <div className="dashboard-header-actions">
          {devices.length > 0 && (
            <DeviceSelector
              devices={devices}
              selectedDeviceId={selectedDeviceId}
              onChange={handleSelectDevice}
              deviceTypes={deviceTypes}
            />
          )}

          <button
            type="button"
            className={`dashboard-logout-button dashboard-stream-button ${isSelectedDeviceStreaming ? 'dashboard-stream-button-stop' : 'dashboard-stream-button-start'}`}
            onClick={() => {
              void handleToggleDeviceStream()
            }}
            disabled={!selectedDeviceId || isSendingStreamCommand}
          >
            {isSendingStreamCommand
              ? 'Sending...'
              : isSelectedDeviceStreaming
                ? 'Stop MQTT'
                : 'Start MQTT'}
          </button>

          <button
            type="button"
            className="dashboard-logout-button"
            onClick={() => clearSession()}
          >
            Logout
          </button>
        </div>
      </header>

      {streamControlError && <p className="dashboard-message">{streamControlError}</p>}

      <nav className="dashboard-tabs" role="tablist" aria-label="Dashboard sections">
        {dashboardTabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id}
            className={`dashboard-tab ${activeTab === tab.id ? 'dashboard-tab-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      {isLoading && (
        <p className="dashboard-message">Loading telemetry...</p>
      )}

      {error && <p className="dashboard-message">{error}</p>}

      {!isLoading && !error && devices.length === 0 && (
        <p className="dashboard-message">No telemetry data available</p>
      )}


      {activeTab === 'overview' && (
        <section className="dashboard-tab-panel dashboard-overview-panel">
          <div className="dashboard-split-grid">
            <DeviceMetaPanel meta={latestDeviceReading?.meta} />

            <section className="dashboard-card dashboard-card-soft">
              <p className="dashboard-card-eyebrow">Brzi pregled</p>
              <h2 className="dashboard-card-title">Trenutni uređaj</h2>
              <p className="dashboard-card-copy">
                Pregled statusa uređaja, metapodataka i glavnih senzora na jednom mjestu.
              </p>
              <div className="dashboard-stat-row">
                <div>
                  <span className="dashboard-stat-label">Uređaj</span>
                  <strong className="dashboard-stat-value">{selectedDeviceId || 'Nije izabran'}</strong>
                </div>
                <div>
                  <span className="dashboard-stat-label">Senzori</span>
                  <strong className="dashboard-stat-value">{availableSensors.length}</strong>
                </div>
                <div>
                  <span className="dashboard-stat-label">Status</span>
                  <strong className={`dashboard-stat-value device-status-value device-status-${selectedDeviceStatus}`}>
                    {selectedDeviceStatus}
                  </strong>
                </div>
                <div>
                  <span className="dashboard-stat-label">Type</span>
                  <strong className="dashboard-stat-value">
                    {deviceTypes[selectedDeviceId] ?? 'unknown'}
                  </strong>
                </div>
              </div>
            </section>
          </div>
        </section>
      )}

      {activeTab === 'telemetry' && (
        <section className="dashboard-tab-panel">
          <DataModeSelector mode={dataMode} onChange={setDataMode} />

          {selectedDeviceId && (
            <section className="dashboard-card dashboard-card-soft telemetry-sensor-filter">
              <label className="trigger-field">
                <span>Filtriraj po sensorId</span>
                <select
                  value={selectedTelemetrySensor}
                  onChange={(event) => setSelectedTelemetrySensor(event.target.value)}
                >
                  <option value="all">Svi senzori</option>
                  {availableSensors.map((sensorKey) => (
                    <option key={sensorKey} value={sensorKey}>
                      {formatSensorLabel(sensorKey)}
                    </option>
                  ))}
                </select>
              </label>
            </section>
          )}

          {selectedDeviceId && dataMode === 'history' && (
            <DateFilter
              onFilterChange={handleDateFilterChange}
              selectedPeriod={dateFilterPeriod}
            />
          )}

          {dataMode === 'live' && (
            <div className="dashboard-banner dashboard-banner-live">
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
              <div className="dashboard-banner dashboard-banner-warning">
                📊 Prikazano: {deviceTelemetry.length} od {originalData.length} podataka (optimizovano za performanse)
              </div>
            ) : null
          })()}

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

          <div className="dashboard-chart-shell">
            <Suspense
              fallback={
                <div className="dashboard-loading-chart">
                  Loading chart...
                </div>
              }
            >
              <TelemetryLineChart
                data={deviceTelemetry}
                selectedSensors={visibleSensors}
              />
            </Suspense>
          </div>
        </section>
      )}

      {activeTab === 'sensors' && (
        <section className="dashboard-tab-panel">
          <section className="dashboard-card dashboard-card-soft">
            <p className="dashboard-card-eyebrow">Senzori</p>
            <h2 className="dashboard-card-title">Senzori po uređaju</h2>
            <p className="dashboard-card-copy">
              Izaberi senzor i otvori telemetriju filtriranu po sensorId ili odmah podesi trigger.
            </p>
          </section>

          {!selectedDeviceId && (
            <p className="dashboard-message">Prvo odaberi uređaj.</p>
          )}

          {selectedDeviceId && sensorCatalog.length === 0 && (
            <p className="dashboard-message">Nema registrovanih senzora za ovaj uređaj.</p>
          )}

          {selectedDeviceId && sensorCatalog.length > 0 && (
            <section className="dashboard-card">
              <table className="trigger-table">
                <thead>
                  <tr>
                    <th>Sensor ID</th>
                    <th>Naziv</th>
                    <th>Akcije</th>
                  </tr>
                </thead>
                <tbody>
                  {sensorCatalog.map((sensorId) => (
                    <tr key={sensorId}>
                      <td>{sensorId}</td>
                      <td>{formatSensorLabel(sensorId)}</td>
                      <td className="trigger-actions-cell">
                        <button
                          type="button"
                          onClick={() => handleOpenSensorTelemetry(sensorId)}
                        >
                          Telemetrija
                        </button>
                        <button
                          type="button"
                          onClick={() => handleConfigureSensorTrigger(sensorId)}
                        >
                          Trigger
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            void handleToggleSensorStream(sensorId)
                          }}
                          disabled={sendingSensorId === sensorId}
                        >
                          {sendingSensorId === sensorId
                            ? 'Sending...'
                            : isSensorStreaming(sensorId)
                              ? 'Stop data'
                              : 'Start data'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          )}
        </section>
      )}

      {activeTab === 'triggers' && (
        <section className="dashboard-tab-panel">
          <section className="trigger-panel dashboard-card">
            <div className="trigger-panel-header">
              <div>
                <p className="dashboard-card-eyebrow">Automatika</p>
                <h2 className="trigger-title">Sensor Trigger</h2>
              </div>
              <p className="dashboard-card-copy">
                Dodaj pragove po senzoru i ciljnom uređaju.
              </p>
            </div>

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
              <label className="trigger-field">
                <span>Target Device</span>
                <select
                  value={targetTriggerDeviceId || selectedDeviceId}
                  onChange={(event) => setTargetTriggerDeviceId(event.target.value)}
                  disabled={devices.length === 0}
                >
                  {devices.map((deviceId) => (
                    <option key={deviceId} value={deviceId}>
                      {deviceId}
                    </option>
                  ))}
                </select>
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
                      <th>Target Device</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deviceTriggers.map((trigger) => (
                      <tr key={trigger.sensorId}>
                        <td>{formatSensorLabel(trigger.sensorId)}</td>
                        <td>{trigger.min !== undefined ? trigger.min : '-'}</td>
                        <td>{trigger.max !== undefined ? trigger.max : '-'}</td>
                        <td>{trigger.targetDeviceId ?? selectedDeviceId}</td>
                        <td className="trigger-actions-cell">
                          <button
                            type="button"
                            onClick={() => handleEditTrigger(trigger)}
                            disabled={deletingSensor === trigger.sensorId}
                          >
                            Edit
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDeleteTrigger(trigger.sensorId)}
                            disabled={deletingSensor === trigger.sensorId}
                          >
                            {deletingSensor === trigger.sensorId ? 'Deleting...' : 'Delete'}
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

          <section className="trigger-events-panel dashboard-card">
            <div className="trigger-panel-header">
              <div>
                <p className="dashboard-card-eyebrow">Događaji</p>
                <h2 className="trigger-title">Trigger Događaji</h2>
              </div>
            </div>

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
        </section>
      )}
    </main>
  )
}
