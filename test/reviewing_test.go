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
		browserOpts := playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
		}
		browser, err := pw.Chromium.Launch(browserOpts)
		require.NoError(t, err, "failed to launch the browser")
		page, err := browser.NewPage()
		require.NoError(t, err, "could not create page")
		assert := playwright.NewPlaywrightAssertions()

		t.Log(server.Config.Addr)
		_, err = page.Goto("http://" + server.Config.Addr + "/reviews")
		require.NoError(t, err, "failed to open page")

		require.NoError(t, assert.Locator(page.Locator(".listing ul li")).ToHaveCount(0), "expected to not have any reviews before creating one")

		form := page.Locator(".new form")
		require.NoError(t, form.Locator(`[name="url"]`).Fill("https://example.com/incident/1"))
		require.NoError(t, form.Locator(`[name="title"]`).Fill("Undersea cable broken"))
		require.NoError(t, form.Locator(`[name="description"]`).Fill("A Chinese vessel dragged it's anchor across the seabed for 100km"))
		require.NoError(t, form.Locator(`[name="impact"]`).Fill("5% of all customers in Latvia seeing higher than average latency, no impact on orders"))
		require.NoError(t, form.Locator(`[type="submit"]`).Click())

		require.NoError(
			t,
			assert.Locator(page.Locator(".new .notice")).ToContainText("created"),
			"expected to have some variant of created to indicate that we successfully started the review",
		)
		require.NoError(
			t,
			assert.Locator(page.Locator(".listing ul li")).ToHaveCount(1),
			"expected to have the newly created review shown in the listing",
		)

		require.NoError(t, pw.Stop(), "failed to stop playwright")
	})
}
