package events

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var Emit = func(ctx context.Context, name string, evt ToolEvent) {}

func EnableRuntimeEmitter() {
	Emit = func(ctx context.Context, name string, evt ToolEvent) {
		runtime.EventsEmit(ctx, name, evt)
	}
}

func SetCustomEmitter(f func(ctx context.Context, name string, evt ToolEvent)) {
	if f == nil {
		Emit = func(context.Context, string, ToolEvent) {}
		return
	}
	Emit = f
}
