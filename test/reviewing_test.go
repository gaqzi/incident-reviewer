package test_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/app"
)

var (
	Headful = os.Getenv("HEADFUL") == "" // runs test in headless if the variable is set to something
)

func getBrowserName() string {
	browserName, hasEnv := os.LookupEnv("BROWSER")
	if hasEnv {
		return browserName
	}
	return "chromium"
}

func getBrowser(pw *playwright.Playwright) playwright.BrowserType {
	browserName := getBrowserName()
	switch browserName {
	case "chromium", "":
		return pw.Chromium
	case "firefox":
		return pw.Firefox
	case "webkit":
		return pw.WebKit
	default:
		panic("unknown browser name: " + browserName)
	}
}

func TestReviewing(t *testing.T) {
	t.Run("Create a new incident review", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cfg := app.NewConfig()
		cfg.Addr = "localhost:0" // bind to localhost to avoid firewall warnings
		server, err := app.Start(ctx, cfg)
		require.NoError(t, err, "failed to start the server")
		defer (func() { _ = server.Stop(context.Background()) })()

		pw, err := playwright.Run()
		require.NoError(t, err, "could not start playwright")
		browser, err := getBrowser(pw).Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(Headful),
		})
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
		require.NoError(t, form.Locator(`[name="title"]`).Fill("Higher than normal latency due undersea cable breaking"))
		require.NoError(t, form.Locator(`[name="description"]`).Fill("Customers in Latvia started seeing higher latency due to an undersea cable breaking and getting routed through other cables"))
		require.NoError(t, form.Locator(`[name="impact"]`).Fill("5% of all customers in Latvia seeing higher than average latency, no impact on orders"))
		require.NoError(t, form.Locator(`[name="where"]`).Fill("Latvia"))
		require.NoError(t, form.Locator(`[name="reportProximalCause"]`).Fill("A broken undersea cabel"))
		require.NoError(t, form.Locator(`[name="reportTrigger"]`).Fill("A Chinese vessel dragged it's anchor across the seabed for 100km"))
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

		// Time to click into it and verify the fields
		require.NoError(t, page.Locator(".new .notice a").Click())
		createdAt, err := page.Locator(".details .createdAt").GetAttribute("datetime")
		require.NoError(t, err, "failed to get createdAt")
		updatedAt, err := page.Locator(".details .updatedAt").GetAttribute("datetime")
		require.NoError(t, err, "failed to get updatedAt")
		require.NoError(t, assert.Locator(page.Locator(".details .where")).ToContainText("Latvia"))
		require.NoError(t, assert.Locator(page.Locator(".details .reportProximalCause")).ToContainText("A broken undersea cabel"))
		require.NoError(t, assert.Locator(page.Locator(".details .reportTrigger")).ToContainText("A Chinese vessel dragged it's anchor across the seabed for 100km"))

		// Time to click in and edit a field
		require.NoError(t, page.Locator(`.details form button[type="submit"]`).Click())

		require.NoError(t, page.Locator(`.details form [name="title"]`).Fill("Broken cable undersea"))
		require.NoError(t, page.Locator(`.details form button[type="submit"]`).Click())

		require.NoError(t, assert.Locator(page.Locator(`.details .title`)).ToHaveText("Broken cable undersea"))
		newCreatedAt, err := page.Locator(".details .createdAt").GetAttribute("datetime")
		require.NoError(t, err, "failed to get the new createdAt")
		newUpdatedAt, err := page.Locator(".details .updatedAt").GetAttribute("datetime")
		require.NoError(t, err, "failed to get the new updatedAt")

		require.Equal(t, createdAt, newCreatedAt, "expected to not have changed the created at when saving")
		require.NotEqual(t, updatedAt, newUpdatedAt, "expected to have changed the updatedAt when saving")

		// Time to bind a normalized contributing cause
		// TODO: break this test up, it shouldn't be this big, keeping it for now because it means I'd have to refactor into
		//  nicer and reusable components, which is good, but I just want to keep going right now.

		causesForm := page.Locator(`contributing-causes form.new`)
		// wow, this is a nasty way to get at the options, but it kinda works, so probably need to find some better way of selecting it when extracting into something reusable.
		options, err := causesForm.Locator(`[name="contributingCauseID"] option`).All()
		require.NoError(t, err)
		var chosenOption string
		for _, opt := range options {
			innerTexts, err := opt.AllInnerTexts()
			require.NoError(t, err)
			for _, text := range innerTexts {
				text = strings.TrimSpace(text)
				if strings.HasPrefix(text, "Third party outage") {
					// Turns out, selecting by the text (at least when it's this long) breaks in firefox, so select it by the value instead of label.
					chosenOption, err = opt.GetAttribute("value")
					require.NoError(t, err, "failed to get value for option")
					break
				}
			}
		}
		require.NotEmpty(t, chosenOption, "expected to have found a chosen option")
		_, err = causesForm.Locator(`[name="contributingCauseID"]`).SelectOption(playwright.SelectOptionValues{Values: &[]string{chosenOption}})
		require.NoError(t, err, "failed to select the contribution cause")
		require.NoError(t, causesForm.Locator(`[name="why"]`).
			Fill("There's literally nothing we could've done since we, like everyone else, rely on core internet infrastructure."))
		require.NoError(t, causesForm.Locator(`[name="isProximalCause"]`).Click())
		require.NoError(t, causesForm.Locator(`button.bind[type="submit"]`).Click())

		causesListing := page.Locator(`contributing-causes ul.listing`)
		firstCause := causesListing.Locator(`li`).First()
		require.NoError(
			t,
			assert.Locator(firstCause).ToHaveCount(1),
			"expected the new listing to be showing up",
		)
		require.NoError(t, assert.Locator(firstCause.Locator(".contributingCause")).ToContainText("Third party outage"))
		require.NoError(t, assert.Locator(firstCause.Locator(".why")).
			ToContainText("There's literally nothing we could've done since we, like everyone else, rely on core internet infrastructure."))
		require.NoError(t, assert.Locator(firstCause).ToHaveClass("proximalCause"), "expected to have set as the proximal cause")

		// Time to suggest another contributing cause, that doesn't exist, and then suggest why it should be added.
		// but first, let's fill in the why for the new cause first, and make sure it stays around while we add the new cause,
		// so that we don't lose important information while saving stuff.
		require.NoError(t, causesForm.Locator(`[name="why"]`).Fill("look, it just fits!"))
		require.NoError(t, causesForm.Locator(`#causes button.propose`).Click())

		newCauseForm := causesForm.Locator("#causes form")
		require.NoError(t, newCauseForm.Locator(`[name="name"]`).Fill("__Inconceivable__"))
		require.NoError(t, newCauseForm.Locator(`[name="description"]`).Fill("The mind boggles to understand the reason for picking this cause"))
		_, err = newCauseForm.Locator(`[name="category"]`).SelectOption(playwright.SelectOptionValues{Values: &[]string{"Implementation"}})
		require.NoError(t, err)
		require.NoError(t, causesForm.Locator(`[name="isProximalCause"]`).Click(), "expected to have checked the proximal cause so it would mark the second as the only proximal cause")
		require.NoError(t, newCauseForm.Locator(`button[type="submit"]`).Click())

		// Time to add the contributing cause with why that we added before, and see that we have two causes in our list
		require.NoError(t, causesForm.Locator(`button.bind[type="submit"]`).Click())
		require.NoError(
			t,
			assert.Locator(causesListing.Locator(`li`)).ToHaveCount(2),
			"expected both of our two bound contributing causes to be shown in the listing",
		)
		require.NoError(t, assert.Locator(causesListing.Locator(`li.proximalCause`)).ToHaveCount(1))
		require.NoError(
			t,
			assert.Locator(causesListing.Locator(`li.proximalCause .contributingCause`)).ToHaveText("__Inconceivable__"),
			"expected the most recently added cause to be the only one set as proximal",
		)

		// And then it's time to edit a contributing cause and see that works
		require.NoError(t, firstCause.Locator(`button.edit`).Click())
		require.NoError(t, firstCause.Locator(`[name="why"]`).Fill("I want to say something else now"))
		require.NoError(t, firstCause.Locator(`button.bind[type="submit"]`).Click())
		require.NoError(t, assert.Locator(firstCause.Locator(".why")).ToContainText("I want to say something else now"))

		// Add a trigger
		triggerForm := page.Locator(`#triggers form.new`)
		options, err = triggerForm.Locator(`[name="triggerID"] option`).All() // TODO: extract selecting an option into a helper
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(options), 1)
		for _, opt := range options {
			innerTexts, err := opt.AllInnerTexts()
			require.NoError(t, err)
			for _, text := range innerTexts {
				text = strings.TrimSpace(text)
				if strings.HasPrefix(text, "Traffic increase") {
					// Turns out, selecting by the value (at least when it's this long) breaks in firefox, so select it by the value instead of label.
					chosenOption, err = opt.GetAttribute("value")
					require.NoError(t, err, "failed to get value for option")
					break
				}
			}
		}
		require.NotEmpty(t, chosenOption, "expected to have found a chosen option")
		_, err = triggerForm.Locator(`[name="triggerID"]`).SelectOption(playwright.SelectOptionValues{Values: &[]string{chosenOption}})
		require.NoError(t, err, "expected to have set the option")

		require.NoError(t, triggerForm.Locator(`[name="why"]`).Fill("Marketing campaign started"))
		require.NoError(t, triggerForm.Locator(`[type="submit"]`).Click())

		// After adding the trigger let's see it in the list
		triggerListing := page.Locator(`#triggers ul.listing`)
		require.NoError(
			t,
			assert.Locator(triggerListing.Locator(`li`)).ToHaveCount(1),
			"expected to have one item listed after creating it",
		)

		firstTrigger := triggerListing.Locator("li").First()
		require.NoError(t, assert.Locator(firstTrigger.Locator(".name")).ToContainText("Traffic increase"))
		require.NoError(t, assert.Locator(firstTrigger.Locator(".why")).
			ToContainText("Marketing campaign started"))

		// Edit the trigger and change the description
		require.NoError(t, firstTrigger.Locator(`form button.edit`).Click())
		require.NoError(t, firstTrigger.Locator(`[name="why"]`).Fill("Marketing campaign started again"))
		require.NoError(t, firstTrigger.Locator(`[type="submit"]`).Click())
		require.NoError(t, assert.Locator(firstTrigger.Locator(".why")).ToContainText("Marketing campaign started again"))

		// Time to suggest another trigger, that doesn't exist, and then suggest why it should be added.
		// but first, let's fill in the why for the new trigger first, and make sure it stays around while we add the new trigger,
		// so that we don't lose important information while saving stuff.
		require.NoError(t, triggerForm.Locator(`[name="why"]`).Fill("this is a critical trigger!"))
		require.NoError(t, triggerForm.Locator(`button.propose`).Click())

		newTriggerForm := triggerForm.Locator("form")
		require.NoError(t, newTriggerForm.Locator(`[name="name"]`).Fill("__New Trigger__"))
		require.NoError(t, newTriggerForm.Locator(`[name="description"]`).Fill("This is a new trigger that should be added to the system"))
		require.NoError(t, newTriggerForm.Locator(`button[type="submit"]`).Click())

		// Time to add the trigger with why that we added before, and see that we have two triggers in our list
		require.NoError(t, triggerForm.Locator(`button.bind[type="submit"]`).Click())
		require.NoError(
			t,
			assert.Locator(triggerListing.Locator(`li`)).ToHaveCount(2),
			"expected both of our two bound triggers to be shown in the listing",
		)
		require.NoError(
			t,
			assert.Locator(triggerListing.Locator(`li:nth-child(2) .name`)).ToHaveText("__New Trigger__"),
			"expected the most recently added trigger to be shown in the listing",
		)

		// Time to verify that the option has been added to the select
		// XXX: urgh, this is not pretty, need to find some kind of pattern for this
		options, err = triggerForm.Locator(`select[name="triggerID"] option`).All()
		require.NoError(t, err)
		require.Equal(t, 3, len(options), "expected to have three options after adding a second selectable trigger")
		var hasNewTrigger bool
		var foundTriggers []string
		for _, opt := range options {
			text, err := opt.AllInnerTexts()
			require.NoError(t, err)

			for _, tt := range text {
				if strings.Contains(tt, "__New Trigger__") {
					hasNewTrigger = true
					break
				}

				foundTriggers = append(foundTriggers, strings.TrimSpace(tt))
			}
		}
		require.True(t, hasNewTrigger, "expected to have found the new trigger in the list of options, found triggers: %s", foundTriggers)

		require.NoError(t, pw.Stop(), "failed to stop playwright")
	})
}
