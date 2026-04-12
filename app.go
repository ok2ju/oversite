package main

import "context"

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx context.Context
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so it can be used by other methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(_ context.Context) {
}

// Greet returns a greeting for the given name.
// This is a placeholder binding to verify frontend-to-Go communication.
func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to Oversite!"
}
