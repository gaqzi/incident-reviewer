package normalized

import "context"

type ContributingCauseStorage interface {
	Get(ctx context.Context, ID int64) (ContributingCause, error)

	Save(ctx context.Context, cause ContributingCause) (ContributingCause, error)

	All(ctx context.Context) ([]ContributingCause, error)
}
