package scpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type taskInfo struct {
	UUID    string `json:"uuid"`
	State   string `json:"state"`
	Message string `json:"message"`
}

func (c *Client) waitForTask(ctx context.Context, body []byte) error {
	var info taskInfo
	if err := json.Unmarshal(body, &info); err != nil {
		// Not a task body; treat the request as complete.
		return nil
	}
	if info.UUID == "" {
		return nil
	}

	tflog.Debug(ctx, "polling SCP task", map[string]any{
		"task_uuid": info.UUID,
	})

	ctx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()

	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		taskBody, err := c.Get(ctx, "/api/v1/tasks/"+info.UUID, nil)
		if err != nil {
			return fmt.Errorf("poll task %s: %w", info.UUID, err)
		}

		var task taskInfo
		if err := json.Unmarshal(taskBody, &task); err != nil {
			return fmt.Errorf("decode task %s: %w", info.UUID, err)
		}

		tflog.Debug(ctx, "SCP task state", map[string]any{
			"task_uuid": info.UUID,
			"state":     task.State,
		})

		switch task.State {
		case "FINISHED":
			return nil
		case "ERROR", "ROLLBACK":
			return fmt.Errorf("task %s failed: %s", info.UUID, task.Message)
		case "CANCELED", "WAITING_FOR_CANCEL":
			return fmt.Errorf("task %s was canceled", info.UUID)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for task %s", info.UUID)
		case <-ticker.C:
		}
	}
}
