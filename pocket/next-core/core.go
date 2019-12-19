package core

import (
	"context"
	"github.com/pingcap/tipocket/pocket/config"
)

// Core is for random test scheduler
type Core struct {
	cfg *config.Config
}

// New creates a Core struct
func New(cfg *config.Config) *Core {
	return &Core{
		cfg,
	}
}

// Start test
func (c *Core) Start(ctx context.Context) {
	
}
