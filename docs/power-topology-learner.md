# Power Topology Learner

The Power Topology Learner is an optional, read-only Home Assistant feature of Energy Security Monitor.

It watches natural ON/OFF transitions of physical metered switches and compares the resulting changes across physical power sensors. Repeated matching deltas can identify likely upstream/downstream relationships such as a device socket behind a UPS or another metered socket.

## Safety model

- The learner never switches a device for testing.
- It does not modify Home Assistant automations, helpers, dashboards, Energy Dashboard configuration, or existing power/energy totals.
- Template, Powercalc, utility-meter, statistics and other virtual/accounting entities are excluded from physical parent candidates.
- Near-simultaneous switch transitions are treated as contaminated evidence and ignored.
- V1 is diagnostic only. Learned relationships are not automatically applied to accounting totals.

## Evidence and confidence

Evidence is collected at device level so entity renames do not erase a relationship. The learner tracks matching and contradictory transitions, ON/OFF evidence, and the observed parent/child power ratio. The ratio allows normal conversion overhead, including UPS losses, to be learned rather than assuming an exact 1:1 watt change.

Relationships progress through `learning`, `suspected`, `strong`, and `confirmed`. A confirmed relationship requires at least eight retained matching observations, at least 90% support, and evidence from both ON and OFF transitions.

Confirmed transitive ancestors are marked as non-direct when an intermediate confirmed parent exists.

## Storage limits

Persistent evidence is intentionally compact. Raw high-frequency power samples exist only in memory for short transition analysis and are never written to disk.

Persisted evidence is aggregated by UTC calendar day and has a hard retention limit of **100 calendar days**. Older daily evidence is automatically purged. The persisted relationship set is also capped at 200 entries as a second storage guardrail.

The diagnostic entity `sensor.energy_security_power_topology` reports learner status, retained relationship counts, the 100-day retention setting, and the strongest current relationships. Its attributes explicitly identify the learner as read-only and as not affecting energy totals.
