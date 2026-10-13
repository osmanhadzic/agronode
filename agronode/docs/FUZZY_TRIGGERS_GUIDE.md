# Fuzzy triggers – user guide

Instead of a single threshold (e.g. "temperature > 30"), a fuzzy trigger uses a **degree of membership**: a temperature of 28 °C can be 0.6 "warm" and 0.2 "hot". Rules then produce a gradual reaction (e.g. fan at 60% instead of just on/off).

## How it works

1. A sensor reading arrives.
2. For each **membership function**, a degree between 0 and 1 is computed (how much the value belongs to, say, "hot").
3. Each **rule** combines the degrees of its conditions into a **strength** (0–1):
   - `AND` (default) = the smallest degree,
   - `OR` = the largest degree.
4. If a rule's strength is greater than 0, its **action** is executed. The action value is `value × strength`.

Example: `value: 100`, strength 0.6 → 60 is sent.

## Configuration structure

```json
{
  "name": "cooling",
  "membershipFunctions": [
    { "name": "warm", "sensor": "temperature", "type": "triangle",  "parameters": [20, 28, 36] },
    { "name": "hot",  "sensor": "temperature", "type": "right_shoulder", "parameters": [30, 40] },
    { "name": "dry",  "sensor": "humidity",    "type": "left_shoulder",  "parameters": [30, 50] }
  ],
  "rules": [
    {
      "name": "fan-medium",
      "operator": "AND",
      "conditions": [
        { "sensor": "temperature", "membership": "warm" },
        { "sensor": "humidity", "membership": "dry" }
      ],
      "action": { "type": "set", "value": 50 }
    },
    {
      "name": "fan-full",
      "conditions": [{ "sensor": "temperature", "membership": "hot" }],
      "action": { "type": "set", "value": 100 }
    }
  ]
}
```

- `sensor` in a membership function may be omitted – the trigger's own sensor is used then.
- Every rule needs at least one condition, and each `membership` must exist in the list of membership functions.

### The `sensor` field in a rule condition

A condition is a `sensor` + `membership` pair meaning "sensor X is in state Y", e.g. `temperature` is `hot`. A membership function is computed for a single sensor, but a rule can combine several of them:

```json
"conditions": [
  { "sensor": "temperature", "membership": "hot" },
  { "sensor": "humidity",    "membership": "dry" }
]
```

This reads as "temperature is high **and** humidity is low" (or `OR`, depending on the rule's operator).

- The name must exactly match the sensor key in telemetry (e.g. `temperature`, `humidity`) and the `sensor` field of the membership function.
- If `sensor` is empty it is treated as `"default"`, which will not match membership functions bound to a specific sensor, so the degree will be 0. Always fill it in.
- If a sensor has no reading, its degree is 0, so an `AND` rule does not fire (an `OR` rule can still fire through its other conditions).
- In the editor this is a dropdown listing the device's sensors.

## Membership functions

| Type | Parameters | Shape / rules |
|------|------------|---------------|
| `triangle` | `[a, b, c]` | Triangle peaking at `b`; `a ≤ b ≤ c` |
| `trapezoid` | `[a, b, c, d]` | Flat top between `b` and `c`; `a ≤ b ≤ c ≤ d` |
| `left_shoulder` | `[a, b]` | 1 up to `a`, falls to 0 at `b`; `a < b` (e.g. "cold") |
| `right_shoulder` | `[a, b]` | 0 up to `a`, rises to 1 at `b`; `a < b` (e.g. "hot") |
| `gaussian` | `[mean, sigma]` | Bell-shaped curve; `sigma > 0` |
| `bell` | `[width, slope, center]` | Generalized bell; `width`, `slope` > 0 |
| `sigmoid` | `[center, slope]` | S-curve; a negative `slope` gives a falling curve; `slope ≠ 0` |

Tip: use `left_shoulder` / `right_shoulder` for edge categories ("very cold", "very hot") and `triangle` or `trapezoid` for the middle ones. Let neighbouring functions overlap so the transition is gradual.

## Actions

| `type` | What happens |
|--------|--------------|
| `activate`, `set` (and anything other than `http`) | The backend sends an MQTT command to the target device. `type` is passed along as a label and the value is `value × strength`. What the device does with it is up to the firmware. |
| `http` | The backend sends an HTTP request to `url`. |

Recommendation: use `activate` for "turn on" and `set` for "set a level" (e.g. 0–100).

### HTTP action

```json
{
  "type": "http",
  "value": 1,
  "url": "https://example.com/webhook",
  "method": "POST",
  "headers": { "Authorization": "Bearer <token>" },
  "body": ""
}
```

- `url` is required and must start with `http://` or `https://`.
- `method` defaults to `POST`.
- If `body` is empty, a JSON payload is sent with the rule, strength, value, inputs and membership degrees.

### Target device

The command goes to the trigger's **target device**. If none is set, it goes to the same device that sent the reading.

## Using it in the dashboard

1. Select a device and open the **Triggers** tab.
2. Select a sensor and click **Open Fuzzy Editor**.
3. Add membership functions (**Add Membership**), pick a type and enter parameters. The chart preview updates live.
4. Add rules: conditions, operator (`AND`/`OR`) and action.
5. Red fields mark invalid parameters; while there are errors, saving the trigger is disabled.
6. **Evaluate** runs a trial calculation with your inputs, **without** sending any commands.

Evaluate is also an API: `POST /api/triggers/:deviceId/:sensorId/evaluate` with `{ "inputs": {...}, "trigger": {...} }`. It returns the membership degrees and the strength of each rule, and activates nothing.

## Tips

- Test with Evaluate first, using a few different values.
- A rule with strength 0 does nothing, while a rule with a small strength (e.g. 0.05) still sends a command with a very small value.
- Several rules can be active at once; each sends its own action.
- If a sensor has no reading, the degree of its membership functions is 0.
