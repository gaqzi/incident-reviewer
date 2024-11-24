package test_test

import (
	"context"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/app"
)

func TestReviewing(t *testing.T) {
	t.Run("Create a new incident review", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cfg := app.NewConfig()
		cfg.Addr = ":0"
		server, err := app.Start(ctx, cfg)
		require.NoError(t, err, "failed to start the server")
		defer (func() { _ = server.Stop(context.Background()) })()

		pw, err := playwright.Run()
		require.NoError(t, err, "could not start playwright")
		browser, err := pw.Chromium.Launch()
		require.NoError(t, err, "failed to launch the browser")
		page, err := browser.NewPage()
		require.NoError(t, err, "could not create page")

		t.Log(server.Config.Addr)
		_, err = page.Goto("http://" + server.Config.Addr + "/reviews")
		require.NoError(t, err, "failed to open page")

		txt, err := page.Locator("h1").InnerText()
		require.NoError(t, err, "failed to find a h1 on the page")
		require.Equal(t, "New review", txt)

		require.NoError(t, pw.Stop(), "failed to stop playwright")
	})
}
