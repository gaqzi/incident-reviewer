package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/form/v4"
	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

type reviewingService interface {
	// Get finds the review or returns NotFoundError.
	Get(ctx context.Context, id uuid.UUID) (reviewing.Review, error)

	// Save validates and stores a review.
	Save(ctx context.Context, review reviewing.Review) (reviewing.Review, error)

	// All returns all the stored reviews with the most recent first.
	All(ctx context.Context) ([]reviewing.Review, error)

	// AddContributingCause validates that the cause can be added to the review.
	AddContributingCause(ctx context.Context, reviewID uuid.UUID, causeID uuid.UUID, why string) error
}

type causeAller interface {
	All(ctx context.Context) ([]contributing.Cause, error)
}

type reviewsHandler struct {
	htmx       *htmx.HTMX
	decoder    *form.Decoder
	causeStore causeAller
	service    reviewingService
}

func ReviewsHandler(service reviewingService, causeStore causeAller) func(chi.Router) {
	app := reviewsHandler{
		htmx:       htmx.New(),
		decoder:    form.NewDecoder(),
		causeStore: causeStore,
		service:    service,
	}

	// Handle UUIDs transparently in the forms.
	app.decoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		if len(vals) == 0 {
			return uuid.Nil, nil
		}

		return uuid.Parse(vals[0])
	}, uuid.UUID{})

	return func(r chi.Router) {
		r.Get("/", app.Index)
		r.Post("/", app.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", app.Show)
			r.Get("/edit", app.Edit)
			r.Post("/edit", app.Update)

			r.Post("/contributing-causes", app.CreateContributingCause)
		})
	}
}

func (a *reviewsHandler) Index(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	a.renderIndex(h, r, map[string]any{})
}

type ReviewBasic struct {
	ID                  uuid.UUID `form:"id"`
	URL                 string    `form:"url"`
	Title               string    `form:"title"`
	Description         string    `form:"description"`
	Impact              string    `form:"impact"`
	Where               string    `form:"where"`
	ReportProximalCause string    `form:"reportProximalCause"`
	ReportTrigger       string    `form:"reportTrigger"`

	// Related items that are not changed from the forms but by other calls
	ContributingCauses []ReviewCauseBasic

	UpdatedAt time.Time
	CreatedAt time.Time
}

type ReviewCauseForm struct {
	ReviewID            uuid.UUID `form:"reviewID"`
	ContributingCauseID uuid.UUID `form:"contributingCauseID"`
	Why                 string    `form:"why"`

	UpdatedAt time.Time
	CreatedAt time.Time
}

type ReviewCauseBasic struct {
	Name     string
	Why      string
	Category string
}

type ContributingCauseBasic struct {
	ID          uuid.UUID
	Name        string
	Description string
	Category    string
}

func (a *reviewsHandler) Create(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inc ReviewBasic
	if err := a.decoder.Decode(&inc, r.Form); err != nil {
		slog.Error("failed to decode basic incident form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	rev := fromHttpObject(inc)
	rev, err := a.service.Save(r.Context(), rev)
	if err != nil {
		slog.Error("failed to save incident", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		h.JustWriteString(err.Error())
		return
	}

	a.renderIndex(h, r, map[string]any{
		"New": map[string]any{
			"Created": map[string]any{
				"ID":    rev.ID,
				"Title": rev.Title,
			},
		},
	})
}

func (a *reviewsHandler) renderIndex(h *htmx.Handler, r *http.Request, data map[string]any) {
	if _, ok := data["Report"]; !ok {
		data["Report"] = map[string]any{}
	}
	if _, ok := data["Reviews"]; !ok {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		reviews, err := a.service.All(ctx)
		if err != nil {
			// Only log the error and set the empty listing as it's an okay fallback instead of returning an error
			slog.Error("failed to fetch all reviews", "error", err)
		}
		cancel()
		data["Reviews"] = convertToHttpObjects(reviews)
	}

	page := htmx.NewComponent("templates/reviews/index.html").
		FS(templates).
		SetData(data).
		With(
			htmx.NewComponent("templates/reviews/_new.html").
				FS(templates).
				Attach("templates/reviews/__review-fields.html"),
			"New",
		).
		Wrap(baseContent(), "Body")

	_, err := h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render page", "page", "reviews/index", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}
}

func (a *reviewsHandler) Show(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.loadReview(r.Context(), h, reviewID)
	if err != nil {
		return
	}

	contributingCauses, err := a.loadContributingCauses(r.Context(), h)
	if err != nil {
		return
	}

	httpReview := convertToHttpObject(review)
	page := htmx.NewComponent("templates/reviews/show.html").
		FS(templates).
		AddData("Review", httpReview).
		With(
			contributingCausesComponent(httpReview.ID, contributingCauses, httpReview.ContributingCauses),
			"ContributingCauses",
		).
		Wrap(baseContent(), "Body")

	a.render(r.Context(), h, page)
}

func (a *reviewsHandler) Edit(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.service.Get(r.Context(), reviewID)
	if err != nil {
		slog.Info("get review error", "error", err)

		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return
		}
		slog.Error("error finding review", "error", err)
	}

	data := map[string]any{
		"Review": convertToHttpObject(review),
	}

	page := htmx.NewComponent("templates/reviews/edit.html").
		FS(templates).
		SetData(data).
		Attach("templates/reviews/__review-fields.html").
		Wrap(baseContent(), "Body")

	_, err = h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}

}

func (a *reviewsHandler) Update(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.service.Get(r.Context(), reviewID)
	if err != nil {
		slog.Info("get review error", "error", err)

		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return
		}
		slog.Error("error finding review", "error", err)
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inc ReviewBasic
	if err := a.decoder.Decode(&inc, r.Form); err != nil {
		slog.Error("failed to decode basic incident form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	// Now update the fetched review and save it
	review = review.Update(fromHttpObject(inc))
	_, err = a.service.Save(r.Context(), review)
	if err != nil {
		slog.Error("failed to save review", "id", reviewID, "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		h.JustWriteString(err.Error())
		return
	}

	// TODO: HTMX redirect so it doesn't reload the whole page and instead just loads the new content.
	h.Header().Add("Location", "/reviews/"+reviewID.String())
	h.WriteHeader(http.StatusSeeOther)
}

func (a *reviewsHandler) CreateContributingCause(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
	}

	reviewID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id for create contributing cause", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var causeBasic ReviewCauseForm
	if err := a.decoder.Decode(&causeBasic, r.PostForm); err != nil {
		slog.Error("failed to decode basic contributing cause form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	if err := a.service.AddContributingCause(r.Context(), reviewID, causeBasic.ContributingCauseID, causeBasic.Why); err != nil {
		slog.Error("failed to create contributing cause", "reviewID", reviewID, "error", err)
		h.WriteHeader(http.StatusBadRequest)
		return
	}

	review, err := a.loadReview(r.Context(), h, reviewID)
	if err != nil {
		return
	}

	contributingCauses, err := a.loadContributingCauses(r.Context(), h)
	if err != nil {
		return
	}

	httpReview := convertToHttpObject(review)
	page := contributingCausesComponent(httpReview.ID, contributingCauses, httpReview.ContributingCauses)

	a.render(r.Context(), h, page)
}

func (a *reviewsHandler) hasErrored(h *htmx.Handler, err error, status int, msg string, args ...any) bool {
	if err == nil {
		return false
	}

	slog.Error(msg, args...)
	h.WriteHeader(status)

	return true
}

func (a *reviewsHandler) loadReview(ctx context.Context, h *htmx.Handler, reviewID uuid.UUID) (reviewing.Review, error) {
	review, err := a.service.Get(ctx, reviewID)
	if err != nil {
		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return reviewing.Review{}, err
		}
		slog.Error("error finding review", "error", err)
	}
	return review, err
}

func (a *reviewsHandler) loadContributingCauses(ctx context.Context, h *htmx.Handler) ([]contributing.Cause, error) {
	contributingCauses, err := a.causeStore.All(ctx)
	if a.hasErrored(h, err, http.StatusInternalServerError, "failed to get all contributing causes", "error", err) {
		return nil, err
	}

	return contributingCauses, nil
}

func (a *reviewsHandler) render(ctx context.Context, h *htmx.Handler, page htmx.RenderableComponent) {
	_, err := h.Render(ctx, page)
	_ = a.hasErrored(h, err, http.StatusInternalServerError, "failed to render", "error", err)
}

func baseContent() htmx.RenderableComponent {
	return htmx.NewComponent("templates/reviews/base.html").FS(templates)
}

func convertToHttpObjects(rs []reviewing.Review) []ReviewBasic {
	ret := make([]ReviewBasic, 0, len(rs))

	for _, r := range rs {
		ret = append(ret, convertToHttpObject(r))
	}

	return ret
}

func convertToHttpObject(r reviewing.Review) ReviewBasic {
	causes := make([]ReviewCauseBasic, 0, len(r.ContributingCauses))
	for _, cause := range r.ContributingCauses {
		causes = append(causes, ReviewCauseBasic{
			Name:     cause.Cause.Name,
			Why:      cause.Why,
			Category: cause.Cause.Category,
		})
	}

	return ReviewBasic{
		ID:                  r.ID,
		URL:                 r.URL,
		Title:               r.Title,
		Description:         r.Description,
		Impact:              r.Impact,
		Where:               r.Where,
		ReportProximalCause: r.ReportProximalCause,
		ReportTrigger:       r.ReportTrigger,

		ContributingCauses: causes,

		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func convertContributingCauseToHttpObjects(ccs []contributing.Cause) map[string][]ContributingCauseBasic {
	ret := make(map[string][]ContributingCauseBasic)

	for _, cc := range ccs {
		ret[cc.Category] = append(ret[cc.Category], convertContributingCauseToHttpObject(cc))
	}

	return ret
}

func convertContributingCauseToHttpObject(cc contributing.Cause) ContributingCauseBasic {
	return ContributingCauseBasic{
		ID:          cc.ID,
		Name:        cc.Name,
		Description: cc.Description,
		Category:    cc.Category,
	}
}

// fromHttpObject takes all values from rb and assigns them to a new reviewing.Review.
func fromHttpObject(rb ReviewBasic) reviewing.Review {
	return reviewing.NewReview().
		Update(reviewing.Review{
			URL:                 rb.URL,
			Title:               rb.Title,
			Description:         rb.Description,
			Impact:              rb.Impact,
			Where:               rb.Where,
			ReportProximalCause: rb.ReportProximalCause,
			ReportTrigger:       rb.ReportTrigger,
		})
}
