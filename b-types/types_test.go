package b_types

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type VolumeMeta map[string]string

type VolumeType string

const (
	SSDVolume = VolumeType("ssd")
	HDDVolume = VolumeType("hdd")
)

type Volume struct {
	ID         string
	Name       string
	Size       int64
	Type       VolumeType
	VolumeMeta VolumeMeta
}

type Network struct {
	ID        string
	Name      string
	IPAddress []string
}

type FlavorType string

const (
	Core4Memory16 = FlavorType("core-4-memory-16")
)

type Machine struct {
	ID       string
	Name     string
	Flavor   FlavorType
	Volumes  []Volume
	Networks []Network
}

type FileShareMachine struct {
	Machine
	Shares []Volume
}

type BootMachineFunc func(m *Machine) error

type VolumeService interface {
	Create(ctx context.Context, volume *Volume) (*Volume, error)
}

type Int interface {
	~int
}

func add[T Int](x, y T) T {
	return x + y
}

type VolumeSize int

func TestVolumeSize(t *testing.T) {
	volumeSize := VolumeSize(10)
	newSize := add(volumeSize, 10)
	assert.Equal(t, VolumeSize(20), newSize)
}
