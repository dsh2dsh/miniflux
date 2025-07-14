// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package googlereader // import "miniflux.app/v2/internal/googlereader"

import (
	"net/http"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/model"
)

type RequestModifiers struct {
	ExcludeTargets    []Stream
	FilterTargets     []Stream
	Streams           []Stream
	Count             int
	Offset            int
	SortDirection     string
	StartTime         int64
	StopTime          int64
	ContinuationToken string
	UserID            int64
}

func (r RequestModifiers) String() string {
	streamStr := make([]string, len(r.Streams))
	for i, s := range r.Streams {
		streamStr[i] = s.String()
	}

	exclusions := make([]string, len(r.ExcludeTargets))
	for i, s := range r.ExcludeTargets {
		exclusions[i] = s.String()
	}

	filters := make([]string, len(r.FilterTargets))
	for i, s := range r.FilterTargets {
		filters[i] = s.String()
	}

	results := []string{
		"UserID: " + strconv.FormatInt(r.UserID, 10),
		"Streams: [" + strings.Join(streamStr, ", ") + "]",
		"Exclusions: [" + strings.Join(exclusions, ", ") + "]",
		"Filters: [" + strings.Join(filters, ", ") + "]",
		"Count: " + strconv.FormatInt(int64(r.Count), 10),
		"Offset: " + strconv.FormatInt(int64(r.Offset), 10),
		"Sort Direction: " + r.SortDirection,
		"Continuation Token: " + r.ContinuationToken,
		"Start Time: " + strconv.FormatInt(r.StartTime, 10),
		"Stop Time: " + strconv.FormatInt(r.StopTime, 10),
	}
	return strings.Join(results, "; ")
}

func parseStreamFilterFromRequest(r *http.Request, u *model.User,
) (RequestModifiers, error) {
	userID := u.ID
	result := RequestModifiers{SortDirection: u.EntryDirection, UserID: userID}

	switch r.URL.Query().Get(paramStreamOrder) {
	case "d":
		result.SortDirection = "desc"
	case "o":
		result.SortDirection = "asc"
	}

	var err error
	result.Streams, err = getStreams(
		request.QueryStringParamList(r, paramStreamID), userID)
	if err != nil {
		return RequestModifiers{}, err
	}

	result.ExcludeTargets, err = getStreams(
		request.QueryStringParamList(r, paramStreamExcludes), userID)
	if err != nil {
		return RequestModifiers{}, err
	}

	result.FilterTargets, err = getStreams(
		request.QueryStringParamList(r, paramStreamFilters), userID)
	if err != nil {
		return RequestModifiers{}, err
	}

	result.Count = request.QueryIntParam(r, paramStreamMaxItems, 0)
	result.Offset = request.QueryIntParam(r, paramContinuation, 0)
	result.StartTime = request.QueryInt64Param(r, paramStreamStartTime, 0)
	result.StopTime = request.QueryInt64Param(r, paramStreamStopTime, 0)
	return result, nil
}
