package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/tasks"
)

// runTasks implements `gvardia tasks`: it dumps the personal kanban (from the
// configured brain) grouped by column.
func runTasks(_ []string, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, t := range tasks.Load(ctx, cfg.Brain) {
		proj := ""
		if t.Project != "" {
			proj = " [" + t.Project + "]"
		}
		fmt.Printf("%-7s %s%s\n", t.Status, t.Title, proj)
	}
	return nil
}
