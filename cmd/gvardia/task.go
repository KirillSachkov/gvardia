package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/tasks"
)

func runTask(args []string, configPath string) error {
	if len(args) == 0 {
		return errors.New("task subcommand is required: create or update")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	switch args[0] {
	case "create":
		return createTask(args[1:], cfg.DataDir)
	case "update":
		return updateTask(args[1:], cfg.DataDir)
	default:
		return fmt.Errorf("unknown task command %q", args[0])
	}
}

func createTask(args []string, dataDir string) error {
	fs := flag.NewFlagSet("gvardia task create", flag.ContinueOnError)
	id := fs.String("id", "", "task id; defaults to a title slug")
	title := fs.String("title", "", "task title")
	status := fs.String("status", "inbox", "task status")
	project := fs.String("project", "", "project name")
	body := fs.String("body", "", "task Markdown body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_, err := tasks.CreateGvardia(dataDir, model.Task{
		ID: *id, Title: *title, Status: *status, Project: *project, Body: *body,
	})
	return err
}

func updateTask(args []string, dataDir string) error {
	fs := flag.NewFlagSet("gvardia task update", flag.ContinueOnError)
	id := fs.String("id", "", "task id")
	title := fs.String("title", "", "new task title")
	status := fs.String("status", "", "new task status")
	project := fs.String("project", "", "new project name")
	body := fs.String("body", "", "new task Markdown body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *id == "" {
		return errors.New("task id is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var task *model.Task
	for _, candidate := range tasks.LoadGvardia(ctx, dataDir) {
		if candidate.ID == *id {
			copy := candidate
			task = &copy
			break
		}
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", *id)
	}
	if *title != "" {
		task.Title = *title
	}
	if *status != "" {
		task.Status = *status
	}
	if *project != "" {
		task.Project = *project
	}
	if *body != "" {
		task.Body = *body
	}
	_, err := tasks.UpdateGvardia(dataDir, *task)
	return err
}
