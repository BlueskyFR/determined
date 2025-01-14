package resourcemanagers

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

// agentState holds the scheduler state for an agent. The implementation of agent-related operations
// (e.g., socket I/O) is deferred to the actor.
type agentState struct {
	handler  *actor.Ref
	devices  map[device.Device]*cproto.ID
	label    string
	enabled  bool
	draining bool

	// Since we only model GPUs as devices/slots and assume each slot can be allocated with
	// one container, we add one additional field to keep track of zero-slot containers.
	// We need this field to know if the agent is idle.
	zeroSlotContainers    map[cproto.ID]bool
	maxZeroSlotContainers int
}

// newAgentState returns a new agent empty agent state backed by the handler.
func newAgentState(msg sproto.AddAgent, maxZeroSlotContainers int) *agentState {
	return &agentState{
		handler:               msg.Agent,
		label:                 msg.Label,
		devices:               make(map[device.Device]*cproto.ID),
		zeroSlotContainers:    make(map[cproto.ID]bool),
		maxZeroSlotContainers: maxZeroSlotContainers,
		enabled:               true,
	}
}

func (a *agentState) numSlots() int {
	switch {
	case a.draining:
		return a.numUsedSlots()
	case !a.enabled:
		return 0
	default:
		return len(a.devices)
	}
}

// numEmptySlots returns the number of slots that have not been allocated to containers.
func (a *agentState) numEmptySlots() (slots int) {
	switch {
	case a.draining || !a.enabled:
		return 0
	default:
		return a.numSlots() - a.numUsedSlots()
	}
}

// numUsedSlots returns the number of slots that have been allocated to containers.
func (a *agentState) numUsedSlots() (slots int) {
	for _, id := range a.devices {
		if id != nil {
			slots++
		}
	}
	return slots
}

func (a *agentState) numUsedZeroSlots() int {
	return len(a.zeroSlotContainers)
}

func (a *agentState) numZeroSlots() int {
	switch {
	case a.draining:
		return a.numUsedZeroSlots()
	case !a.enabled:
		return 0
	default:
		return a.maxZeroSlotContainers
	}
}

func (a *agentState) numEmptyZeroSlots() int {
	switch {
	case a.draining || !a.enabled:
		return 0
	default:
		return a.numZeroSlots() - a.numUsedZeroSlots()
	}
}

func (a *agentState) idle() bool {
	return a.numUsedZeroSlots() == 0 && a.numUsedSlots() == 0
}

func (a *agentState) allocateFreeDevices(slots int, id cproto.ID) []device.Device {
	if slots == 0 {
		a.zeroSlotContainers[id] = true
		return nil
	}
	cid := id
	devices := make([]device.Device, 0, slots)
	for d, dcid := range a.devices {
		if dcid == nil {
			a.devices[d] = &cid
			devices = append(devices, d)
		}
		if len(devices) == slots {
			break
		}
	}
	check.Panic(check.True(len(devices) == slots, "not enough devices"))
	return devices
}

func (a *agentState) deallocateContainer(id cproto.ID) {
	delete(a.zeroSlotContainers, id)
	for d, cid := range a.devices {
		if cid != nil && *cid == id {
			a.devices[d] = nil
		}
	}
}

func (a *agentState) deepCopy() *agentState {
	copiedAgent := &agentState{
		handler:               a.handler,
		label:                 a.label,
		devices:               make(map[device.Device]*cproto.ID),
		zeroSlotContainers:    make(map[cproto.ID]bool),
		maxZeroSlotContainers: a.maxZeroSlotContainers,
		enabled:               a.enabled,
		draining:              a.draining,
	}

	for originalDevice, id := range a.devices {
		copiedDevice := device.Device{
			ID:    originalDevice.ID,
			Brand: originalDevice.Brand,
			UUID:  originalDevice.UUID,
			Type:  originalDevice.Type,
		}
		copiedAgent.devices[copiedDevice] = id
	}

	for originalKey, originalValue := range a.zeroSlotContainers {
		copiedAgent.zeroSlotContainers[originalKey] = originalValue
	}

	return copiedAgent
}
