// setup:feature:demo

package demo

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SensorType is the kind of reading produced by a sensor; ranges and units are looked up in sensorSpecs.
type SensorType string

// Sensor type constants.
const (
	SensorTemp     SensorType = "temp"
	SensorHumidity SensorType = "humidity"
	SensorPressure SensorType = "pressure"
	SensorLight    SensorType = "light"
)

// AllSensorTypes is the canonical sensor-type list in grid order.
var AllSensorTypes = []SensorType{SensorTemp, SensorHumidity, SensorPressure, SensorLight}

// SensorReading is a single measurement; Topic is "sensors/floor{N}/{Type}" matching AllTopics order.
type SensorReading struct {
	Timestamp time.Time
	Topic     string
	Type      SensorType
	Unit      string
	Floor     int
	Value     float64
}

// sensorSpec defines the range and unit for a sensor type.
type sensorSpec struct {
	unit string
	min  float64
	max  float64
}

var sensorSpecs = map[SensorType]sensorSpec{
	SensorTemp:     {min: 18, max: 32, unit: "°C"},
	SensorHumidity: {min: 30, max: 80, unit: "%"},
	SensorPressure: {min: 990, max: 1030, unit: "hPa"},
	SensorLight:    {min: 100, max: 1000, unit: "lux"},
}

// SensorGrid simulates a 4-floor x 4-sensor IoT grid.
type SensorGrid struct {
	sensors  map[string]*sensorState
	mu       sync.RWMutex
	flooding atomic.Bool
}

type sensorState struct {
	history []float64
	reading SensorReading
}

// NewSensorGrid pre-seeds 16 sensors (4 floors x 4 types) with random values inside each spec range.
func NewSensorGrid() *SensorGrid {
	g := &SensorGrid{
		sensors: make(map[string]*sensorState, 16),
	}
	for floor := 1; floor <= 4; floor++ {
		for _, st := range AllSensorTypes {
			spec := sensorSpecs[st]
			topic := fmt.Sprintf("sensors/floor%d/%s", floor, st)
			val := spec.min + rand.Float64()*(spec.max-spec.min)
			val = math.Round(val*10) / 10
			s := &sensorState{
				reading: SensorReading{
					Topic:     topic,
					Floor:     floor,
					Type:      st,
					Value:     val,
					Unit:      spec.unit,
					Timestamp: time.Now(),
				},
				history: []float64{val},
			}
			g.sensors[topic] = s
		}
	}
	return g
}

// AllTopics is the 16 sensor topics emitted in floor-then-type order (deterministic).
func (g *SensorGrid) AllTopics() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	topics := make([]string, 0, len(g.sensors))
	for floor := 1; floor <= 4; floor++ {
		for _, st := range AllSensorTypes {
			topics = append(topics, fmt.Sprintf("sensors/floor%d/%s", floor, st))
		}
	}
	return topics
}

// Tick advances all sensors by one random-walk step and returns readings that
// actually changed value (after rounding).
func (g *SensorGrid) Tick() []SensorReading {
	g.mu.Lock()
	defer g.mu.Unlock()
	var changed []SensorReading
	for _, s := range g.sensors {
		spec := sensorSpecs[s.reading.Type]
		delta := (rand.Float64() - 0.5) * (spec.max - spec.min) * 0.05
		newVal := s.reading.Value + delta
		newVal = math.Max(spec.min, math.Min(spec.max, newVal))
		newVal = math.Round(newVal*10) / 10
		if newVal != s.reading.Value {
			s.reading.Value = newVal
			s.reading.Timestamp = time.Now()
			s.history = append(s.history, newVal)
			if len(s.history) > 10 {
				s.history = s.history[len(s.history)-10:]
			}
			changed = append(changed, s.reading)
		}
	}
	return changed
}

// SetFloodMode enables or disables high-frequency publish mode.
func (g *SensorGrid) SetFloodMode(on bool) {
	g.flooding.Store(on)
}

// IsFlooding reports whether high-frequency publish mode is currently enabled.
func (g *SensorGrid) IsFlooding() bool {
	return g.flooding.Load()
}

// Snapshot picks topics matching a glob: * is one segment, ** is any path tail.
func (g *SensorGrid) Snapshot(pattern string) map[string]SensorReading {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make(map[string]SensorReading)
	for topic, s := range g.sensors {
		if matchGlob(pattern, topic) {
			result[topic] = s.reading
		}
	}
	return result
}

// Reading looks up topic; ok is false when it isn't one of the 16 known sensors.
func (g *SensorGrid) Reading(topic string) (SensorReading, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	s, ok := g.sensors[topic]
	if !ok {
		return SensorReading{}, false
	}
	return s.reading, true
}

// History is up to the last 10 values for a topic's sparkline; nil if topic is unknown.
func (g *SensorGrid) History(topic string) []float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	s, ok := g.sensors[topic]
	if !ok {
		return nil
	}
	out := make([]float64, len(s.history))
	copy(out, s.history)
	return out
}

// AllReadings emits every sensor reading in floor-then-type order.
func (g *SensorGrid) AllReadings() []SensorReading {
	g.mu.RLock()
	defer g.mu.RUnlock()
	readings := make([]SensorReading, 0, len(g.sensors))
	for floor := 1; floor <= 4; floor++ {
		for _, st := range AllSensorTypes {
			topic := fmt.Sprintf("sensors/floor%d/%s", floor, st)
			if s, ok := g.sensors[topic]; ok {
				readings = append(readings, s.reading)
			}
		}
	}
	return readings
}

// matchGlob performs simple glob matching: ** matches any path segments,
// * matches a single segment.
func matchGlob(pattern, topic string) bool {
	if pattern == "**" || pattern == "sensors/**" {
		return strings.HasPrefix(topic, "sensors/")
	}
	// Replace ** with a regex-like all-match, * with single-segment match
	// Simple approach: split both and compare
	patParts := strings.Split(pattern, "/")
	topParts := strings.Split(topic, "/")
	return matchParts(patParts, topParts)
}

func matchParts(pat, top []string) bool {
	pi, ti := 0, 0
	for pi < len(pat) && ti < len(top) {
		if pat[pi] == "**" {
			// ** matches zero or more segments
			if pi == len(pat)-1 {
				return true
			}
			// Try matching rest from every position
			for k := ti; k <= len(top); k++ {
				if matchParts(pat[pi+1:], top[k:]) {
					return true
				}
			}
			return false
		}
		if pat[pi] == "*" || pat[pi] == top[ti] {
			pi++
			ti++
			continue
		}
		return false
	}
	// Consume trailing **
	for pi < len(pat) && pat[pi] == "**" {
		pi++
	}
	return pi == len(pat) && ti == len(top)
}
