package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type actionSpec struct {
	method       string
	path         string
	bodyBuilder  func(args map[string]string, body string) ([]byte, error)
	queryBuilder func(args map[string]string) url.Values
}

var actionSpecs = map[string]actionSpec{
	"start": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery(""),
	},
	"stop": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("OFF"),
		queryBuilder: powerQuery("POWEROFF"),
	},
	"reset": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery("RESET"),
	},
	"powercycle": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery("POWERCYCLE"),
	},
	"suspend": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("SUSPENDED"),
		queryBuilder: powerQuery(""),
	},
	"rescue_activate": {
		method:      "POST",
		path:        "/api/v1/servers/%d/rescuesystem",
		bodyBuilder: emptyBody,
	},
	"rescue_deactivate": {
		method:      methodDelete,
		path:        "/api/v1/servers/%d/rescuesystem",
		bodyBuilder: emptyBody,
	},
	"iso_attach": {
		method:      "POST",
		path:        "/api/v1/servers/%d/iso",
		bodyBuilder: jsonBody,
	},
	"iso_detach": {
		method:      methodDelete,
		path:        "/api/v1/servers/%d/iso",
		bodyBuilder: emptyBody,
	},
	"snapshot_create": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots",
		bodyBuilder: jsonBody,
	},
	"snapshot_revert": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots/%s/revert",
		bodyBuilder: emptyBody,
	},
	"snapshot_export": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots/%s/export",
		bodyBuilder: emptyBody,
	},
	"snapshot_delete": {
		method:      methodDelete,
		path:        "/api/v1/servers/%d/snapshots/%s",
		bodyBuilder: emptyBody,
	},
	"snapshot_dryrun": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots:dryrun",
		bodyBuilder: jsonBody,
	},
	"disk_format": {
		method:      "POST",
		path:        "/api/v1/servers/%d/disks/%s:format",
		bodyBuilder: emptyBody,
	},
	"disk_driver_update": {
		method:      "PATCH",
		path:        "/api/v1/servers/%d/disks",
		bodyBuilder: driverBody,
	},
	"image_setup": {
		method:      "POST",
		path:        "/api/v1/servers/%d/image",
		bodyBuilder: jsonBody,
	},
	"user_image_setup": {
		method:      "POST",
		path:        "/api/v1/servers/%d/user-image",
		bodyBuilder: jsonBody,
	},
	"storage_optimize": {
		method:      "POST",
		path:        "/api/v1/servers/%d/storageoptimization",
		bodyBuilder: emptyOrJSONBody,
	},
	"firewall_reapply": {
		method:      "POST",
		path:        "/api/v1/servers/%d/interfaces/%s/firewall:reapply",
		bodyBuilder: emptyBody,
	},
	"firewall_restore": {
		method:      "POST",
		path:        "/api/v1/servers/%d/interfaces/%s/firewall:restore-copied-policies",
		bodyBuilder: emptyBody,
	},
}

func powerBody(state string) func(map[string]string, string) ([]byte, error) {
	return func(args map[string]string, body string) ([]byte, error) {
		m := map[string]any{"state": state}
		return json.Marshal(m)
	}
}

func powerQuery(defaultOption string) func(map[string]string) url.Values {
	return func(args map[string]string) url.Values {
		option := defaultOption
		if v, ok := args["state_option"]; ok && v != "" {
			option = v
		}
		if option == "" {
			return nil
		}
		return url.Values{"stateOption": []string{option}}
	}
}

func driverBody(args map[string]string, body string) ([]byte, error) {
	driver := body
	if driver == "" {
		driver = args["driver"]
	}
	if driver == "" {
		return nil, errors.New("disk_driver_update requires a driver argument or body")
	}
	return json.Marshal(map[string]any{"driver": driver})
}

func emptyBody(args map[string]string, body string) ([]byte, error) {
	return nil, nil
}

func emptyOrJSONBody(args map[string]string, body string) ([]byte, error) {
	if body == "" {
		return json.Marshal(map[string]any{})
	}
	return jsonBody(args, body)
}

func jsonBody(args map[string]string, body string) ([]byte, error) {
	if body == "" {
		return nil, errors.New("action requires a JSON request body")
	}
	return jsonBodyOrEmpty(body)
}

func jsonBodyOrEmpty(body string) ([]byte, error) {
	if body == "" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func requiredArgForAction(action string) (string, bool) {
	switch action {
	case "snapshot_revert", "snapshot_export", "snapshot_delete":
		return "snapshot_name", true
	case "disk_format":
		return "disk_name", true
	case "firewall_reapply", "firewall_restore":
		return "mac", true
	}
	return "", false
}

func actionPath(spec actionSpec, serverID int64, action string, args map[string]string) (string, error) {
	path := fmt.Sprintf(spec.path, serverID)
	if argName, required := requiredArgForAction(action); required {
		if _, ok := args[argName]; !ok {
			return "", fmt.Errorf("action %q requires argument %q", action, argName)
		}
		path = fmt.Sprintf(path, args[argName])
	}

	query := url.Values{}
	if spec.queryBuilder != nil {
		query = spec.queryBuilder(args)
	}
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	return path, nil
}
