package scpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type taskInfo struct {
	UUID    string          `json:"uuid"`
	State   string          `json:"state"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func (c *Client) waitForTask(ctx context.Context, body []byte) ([]byte, error) {
	info, ok := parseTaskInfo(body)
	if !ok {
		return nil, nil
	}

	tflog.Debug(ctx, "polling SCP task", map[string]any{
		"task_uuid": info.UUID,
	})

	if err := checkTaskState(info); err != nil || info.State == "FINISHED" {
		return taskResultBytes(info.Result), err
	}

	ctx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()

	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task %s", info.UUID)
		case <-ticker.C:
		}

		task, err := c.pollTask(ctx, info.UUID)
		if err != nil {
			return nil, err
		}
		if err := checkTaskState(task); err != nil {
			return nil, err
		}
		if task.State == "FINISHED" {
			return taskResultBytes(task.Result), nil
		}
	}
}

func taskResultBytes(r json.RawMessage) []byte {
	if len(r) == 0 || bytes.Equal(r, []byte("null")) {
		return nil
	}
	return r
}

func parseTaskInfo(body []byte) (taskInfo, bool) {
	var info taskInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return taskInfo{}, false
	}
	return info, info.UUID != ""
}

func (c *Client) pollTask(ctx context.Context, uuid string) (taskInfo, error) {
	body, err := c.Get(ctx, "/api/v1/tasks/"+uuid, nil)
	if err != nil {
		return taskInfo{}, fmt.Errorf("poll task %s: %w", uuid, err)
	}

	var task taskInfo
	if err := json.Unmarshal(body, &task); err != nil {
		return taskInfo{}, fmt.Errorf("decode task %s: %w", uuid, err)
	}

	tflog.Debug(ctx, "SCP task state", map[string]any{
		"task_uuid": uuid,
		"state":     task.State,
	})

	return task, nil
}

func checkTaskState(task taskInfo) error {
	switch task.State {
	case "FINISHED", "PENDING", "RUNNING", "":
		return nil
	case "ERROR", "ROLLBACK":
		return fmt.Errorf("task %s failed: %s", task.UUID, task.Message)
	case "CANCELED", "WAITING_FOR_CANCEL":
		return fmt.Errorf("task %s was canceled", task.UUID)
	default:
		return nil
	}
}
