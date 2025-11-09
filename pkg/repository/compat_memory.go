package repository

import "github.com/secmon-lab/warren/pkg/repository/memory"

type Memory = memory.Memory

func NewMemory() *Memory {
	return memory.New()
}
