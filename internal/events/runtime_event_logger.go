package events

import (
	"context"
	"encoding/json"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func logRuntimeEvent(ctx context.Context, name string, event ToolEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		runtime.LogError(ctx, "loggers: failed to marshal tool event: "+err.Error())
		return
	}

	payload := string(data)

	switch event.Type {
	case EventSuccess:
		runtime.LogInfo(ctx, payload)
	case EventError:
		runtime.LogError(ctx, payload)
	case EventWarn:
		runtime.LogWarning(ctx, payload)
	case EventInfo:
		runtime.LogInfo(ctx, payload)
	default:
		runtime.LogInfo(ctx, payload)
	}
}
