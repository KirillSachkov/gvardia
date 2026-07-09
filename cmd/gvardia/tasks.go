package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/tasks"
)

// runTasks implements `gvardia tasks`: it dumps configured task sources.
func runTasks(_ []string, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var list []model.Task
	for _, source := range cfg.TaskSources {
		switch source {
		case "gvardia":
			list = append(list, tasks.LoadGvardia(ctx, cfg.DataDir)...)
		case "brain":
			list = append(list, tasks.Load(ctx, cfg.Brain)...)
		}
	}
	for _, t := range list {
		proj := ""
		if t.Project != "" {
			proj = " [" + t.Project + "]"
		}
		fmt.Printf("%-7s %s%s\n", t.Status, t.Title, proj)
	}
	return nil
}
