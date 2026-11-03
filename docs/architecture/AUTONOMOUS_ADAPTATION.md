# ISOTOPE Autonomous Adaptation

## Status: architecture

Autonomous adaptation is a mechanism by which the network automatically adjusts its parameters based on metrics.

In the current version (v2.0.6), parameters are set manually. Autonomous adaptation is planned (v2.5+).

---

## Why This Is Needed

The network grows, changes, and operates in different regions under different loads. Static parameters lead to:

- Choking on garbage (archive threshold too high)
- Loss of useful data (threshold too low)
- Network overload (too frequent synchronization)
- Slower learning (low speed under high load)

---

## How It Works

Metrics collection → Trend analysis → Rule application → Effect verification.

### 1. Metrics Collection

Each node locally collects:

| Group | Metric |
|-------|--------|
| **Messages** | Average weight, count < 0.15, count > 0.8 |
| **Nodes** | Average weight, count < 0.3, count > 0.7 |
| **Memory** | Message count, archived count |
| **Network** | Synchronization frequency |

### 2. Aggregation

Metrics are published to DHT anonymously (without node binding). Gossip message of type `"type":"metrics"` every 5 minutes.

### 3. Auto-correction Rules

| Condition | Action |
|-----------|--------|
| Average message weight < 0.3 | Lower archive threshold: 0.15 → 0.10 |
| Average message weight > 0.7 | Raise archive threshold: 0.15 → 0.20 |
| Nodes with weight < 0.3 > 30% | Raise activity threshold: 0.5 → 0.6 |
| Sync frequency < 1/10min | Increase sync frequency |
| Messages > 80% of limit | Speed up archiving |

### 4. Verification

After changing a parameter, the node collects metrics again after 5 minutes. If the situation improved — the parameter stays. If it worsened — rollback.

---

## What Can Be Changed

| Parameter | Range | Default |
|-----------|-------|---------|
| `archive_threshold` | 0.05-0.30 | 0.15 |
| `node_active_threshold` | 0.3-0.8 | 0.5 |
| `sync_interval` | 5-300 sec | 30 sec |
| `learning_rate` | 0.001-0.05 | 0.01 |

## What Cannot Be Changed

- **Ethical hash** — the DNA of the network
- **Vector dimension** (VectorDim = 100)
- **Activation function**

---

## Implementation

### v2.0.6 (current)

Metrics collection in logs, without auto-correction. Manual parameter management.

### v2.5 (planned)

Basic auto-correction rules. Operational parameters only.

### v3.0 (planned)

Full cycle: collection → analysis → correction → verification.

---

## Configuration

```json
{
  "auto_adapt": true,
  "archive_threshold": 0.15,
  "node_active_threshold": 0.5,
  "sync_interval_sec": 30,
  "learning_rate": 0.01,
  "metrics_publish_interval_sec": 300
}
Flag auto_adapt: false disables auto-correction (manual mode).

Logging
All parameter changes are logged with the [ADAPT] tag:

text
[ADAPT] archive_threshold changed: 0.15 → 0.10 (reason: avg_weight < 0.3)
[ADAPT] archive_threshold restored: 0.10 → 0.15 (reason: no improvement)
Safety of Auto-correction
For the Community
All parameter changes are logged

A node can disable auto-correction and use manual settings

Critical parameters (ethical filter) are not subject to auto-correction

Auto-correction applies only to operational parameters

For Enterprise
Auto-correction is predictable: rules are deterministic

All changes are reproducible and verifiable

In a corporate environment, parameters can be fixed and auto-correction fully disabled

Audit trail records every parameter change

Autonomous adaptation is not just optimization. It is a property of a living system.