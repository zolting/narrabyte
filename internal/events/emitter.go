package events

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var Emit = func(ctx context.Context, name string, evt ToolEvent) {}

func EnableRuntimeEmitter() {
	Emit = func(ctx context.Context, name string, evt ToolEvent) {
		if evt.SessionKey == "" {
			if session := SessionFromContext(ctx); session != "" {
				evt.SessionKey = session
			}
		}

		if evt.Type == EventSuccess || evt.Type == EventError {
			runtime.EventsEmit(ctx, name, evt)
		}

		logRuntimeEvent(ctx, name, evt)
	}

}

func SetCustomEmitter(f func(ctx context.Context, name string, evt ToolEvent)) {
	if f == nil {
		Emit = func(context.Context, string, ToolEvent) {}
		return
	}
	Emit = func(ctx context.Context, name string, evt ToolEvent) {
		if evt.SessionKey == "" {
			if session := SessionFromContext(ctx); session != "" {
				evt.SessionKey = session
			}
		}
		f(ctx, name, evt)
	}
}
