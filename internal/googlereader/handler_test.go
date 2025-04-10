package googlereader

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/model"
)

func Test_getStreamFilterModifiers_SortDirection(t *testing.T) {
	tests := []struct {
		name          string
		user          model.User
		req           string
		sortDirection string
	}{
		{
			name:          "SortDirection with user desc",
			user:          model.User{EntryDirection: "desc"},
			req:           "/",
			sortDirection: "desc",
		},
		{
			name:          "SortDirection with user asc",
			user:          model.User{EntryDirection: "asc"},
			req:           "/",
			sortDirection: "asc",
		},
		{
			name:          "SortDirection with req desc",
			user:          model.User{EntryDirection: "asc"},
			req:           "/?" + ParamStreamOrder + "=d",
			sortDirection: "desc",
		},
		{
			name:          "SortDirection with req asc",
			user:          model.User{EntryDirection: "desc"},
			req:           "/?" + ParamStreamOrder + "=o",
			sortDirection: "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", tt.req, nil)
			got, err := getStreamFilterModifiers(r, &tt.user)
			require.NoError(t, err)

			want := RequestModifiers{
				ExcludeTargets: []Stream{},
				FilterTargets:  []Stream{},
				Streams:        []Stream{},
				SortDirection:  tt.sortDirection,
			}
			assert.Equal(t, want, got)
		})
	}
}
